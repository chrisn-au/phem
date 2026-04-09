// Package baseline computes summary stats and decomposed annual rollups
// from the energy_intervals + load_decomposition tables. The values feed both
// the Baseline screen and the scenario engine (which uses baseline annuals as
// the "do nothing" reference point).
package baseline

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/assumptions"
	"github.com/chrisneave/phem/api/internal/models"
)

type Service struct {
	pool *pgxpool.Pool
	asm  *assumptions.Store
}

func New(pool *pgxpool.Pool, asm *assumptions.Store) *Service {
	return &Service{pool: pool, asm: asm}
}

// Summary returns the headline numbers shown on the Baseline screen.
func (s *Service) Summary(ctx context.Context) (models.BaselineSummary, error) {
	asm, err := s.asm.Snapshot(ctx)
	if err != nil {
		return models.BaselineSummary{}, err
	}
	var sum models.BaselineSummary
	row := s.pool.QueryRow(ctx, `
		SELECT
		  min(ts), max(ts), count(*),
		  COALESCE(sum(solar_gen_kwh),0),
		  COALESCE(sum(consumption_kwh),0),
		  COALESCE(sum(grid_import_kwh),0),
		  COALESCE(sum(grid_export_kwh),0),
		  COALESCE(sum(grid_import_kwh * spot_price_import_aud_per_kwh),0)
		FROM energy_intervals`)
	var sumSpotImport float64
	if err := row.Scan(&sum.RangeStart, &sum.RangeEnd, &sum.IntervalCount,
		&sum.AnnualSolarKWh, &sum.AnnualConsumptionKWh,
		&sum.AnnualImportKWh, &sum.AnnualExportKWh, &sumSpotImport); err != nil {
		return sum, err
	}

	// Annualise
	days := sum.RangeEnd.Sub(sum.RangeStart).Hours() / 24
	if days < 1 {
		days = 1
	}
	scale := 365.0 / days
	sum.AnnualSolarKWh *= scale
	sum.AnnualConsumptionKWh *= scale
	sum.AnnualImportKWh *= scale
	sum.AnnualExportKWh *= scale

	// Annual gas from gas_bills (sum of MJ in window, scaled)
	var totalMJ float64
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(sum(consumption_mj),0) FROM gas_bills`).Scan(&totalMJ); err == nil {
		sum.AnnualGasMJ = totalMJ * (365.0 / max(daysOfBills(ctx, s.pool), 90))
	}

	// Cost: spot import * import + supply charge per day - export earnings
	// Use Amber-style: if spot above cap, cap; if export below floor, floor.
	// We approximate by recomputing per-interval (cheap because it's a few rolls).
	var importCost, exportEarn float64
	rows, err := s.pool.Query(ctx, `SELECT grid_import_kwh, grid_export_kwh, spot_price_import_aud_per_kwh, spot_price_export_aud_per_kwh FROM energy_intervals`)
	if err == nil {
		for rows.Next() {
			var imp, exp, sImp, sExp float64
			if err := rows.Scan(&imp, &exp, &sImp, &sExp); err == nil {
				if sImp > asm.ImportCap {
					sImp = asm.ImportCap
				}
				if sExp < asm.ExportFloor {
					sExp = asm.ExportFloor
				}
				importCost += imp * sImp
				exportEarn += exp * sExp
			}
		}
		rows.Close()
	}
	supply := asm.SupplyAUDPerDay * days
	sum.AnnualElecCostAUD = (importCost + supply - exportEarn) * scale
	// Gas cost ~ $0.045/MJ + $90/quarter standing
	sum.AnnualGasCostAUD = sum.AnnualGasMJ*0.045 + 90*4

	// Carbon
	sum.AnnualCO2Kg = sum.AnnualImportKWh*asm.GridKgPerKWh + sum.AnnualGasMJ/3.6*asm.GasKgPerKWhTh

	if sum.AnnualConsumptionKWh > 0 {
		sum.SelfConsumptionPct = (sum.AnnualConsumptionKWh - sum.AnnualImportKWh) / sum.AnnualConsumptionKWh * 100
		sum.SolarFractionPct = (sum.AnnualSolarKWh - sum.AnnualExportKWh) / sum.AnnualConsumptionKWh * 100
	}
	return sum, nil
}

func daysOfBills(ctx context.Context, pool *pgxpool.Pool) float64 {
	var minD, maxD time.Time
	if err := pool.QueryRow(ctx, `SELECT min(period_start), max(period_end) FROM gas_bills`).Scan(&minD, &maxD); err != nil {
		return 0
	}
	if minD.IsZero() {
		return 0
	}
	return maxD.Sub(minD).Hours() / 24
}

// Daily aggregates the 15-min energy_intervals table into per-day rows for chart rendering.
type DailyRow struct {
	Day             time.Time `json:"day"`
	SolarKWh        float64   `json:"solar_kwh"`
	ConsumptionKWh  float64   `json:"consumption_kwh"`
	GridImportKWh   float64   `json:"grid_import_kwh"`
	GridExportKWh   float64   `json:"grid_export_kwh"`
	BatteryDisKWh   float64   `json:"battery_discharge_kwh"`
	AvgSpotImport   float64   `json:"avg_spot_import"`
	AvgTempC        float64   `json:"avg_temp_c"`
}

func (s *Service) Daily(ctx context.Context) ([]DailyRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date_trunc('day', ts) AS day,
		       sum(solar_gen_kwh),
		       sum(consumption_kwh),
		       sum(grid_import_kwh),
		       sum(grid_export_kwh),
		       sum(battery_discharge_kwh),
		       avg(spot_price_import_aud_per_kwh),
		       avg(temperature_c)
		FROM energy_intervals
		GROUP BY 1
		ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DailyRow{}
	for rows.Next() {
		var r DailyRow
		if err := rows.Scan(&r.Day, &r.SolarKWh, &r.ConsumptionKWh, &r.GridImportKWh, &r.GridExportKWh, &r.BatteryDisKWh, &r.AvgSpotImport, &r.AvgTempC); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// HourOfDayProfile returns the typical-day profile averaged over the full window.
type HourRow struct {
	Hour          int     `json:"hour"`
	SolarKWh      float64 `json:"solar_kwh"`
	ConsumptionKWh float64 `json:"consumption_kwh"`
	HVACKWh       float64 `json:"hvac_kwh"`
	PoolKWh       float64 `json:"pool_kwh"`
	BaseKWh       float64 `json:"base_kwh"`
	HotWaterKWh   float64 `json:"hot_water_gas_equiv_kwh"`
	CookingKWh    float64 `json:"cooking_gas_equiv_kwh"`
	EVKWh         float64 `json:"ev_kwh"`
	SpotImport    float64 `json:"spot_import"`
}

func (s *Service) HourOfDayProfile(ctx context.Context) ([]HourRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT EXTRACT(hour FROM e.ts)::int AS h,
		       avg(e.solar_gen_kwh)*4,
		       avg(e.consumption_kwh)*4,
		       avg(d.hvac_kwh)*4,
		       avg(d.pool_pump_kwh)*4,
		       avg(d.base_load_kwh)*4,
		       avg(d.hot_water_gas_equiv_kwh)*4,
		       avg(d.cooking_gas_equiv_kwh)*4,
		       avg(d.ev_kwh)*4,
		       avg(e.spot_price_import_aud_per_kwh)
		FROM energy_intervals e
		JOIN load_decomposition d USING (ts)
		GROUP BY 1
		ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HourRow{}
	for rows.Next() {
		var r HourRow
		if err := rows.Scan(&r.Hour, &r.SolarKWh, &r.ConsumptionKWh, &r.HVACKWh, &r.PoolKWh, &r.BaseKWh, &r.HotWaterKWh, &r.CookingKWh, &r.EVKWh, &r.SpotImport); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MonthlyDecomposition returns total kWh per month grouped by load category.
type MonthRow struct {
	Month         time.Time `json:"month"`
	SolarKWh      float64   `json:"solar_kwh"`
	HVACKWh       float64   `json:"hvac_kwh"`
	PoolKWh       float64   `json:"pool_kwh"`
	BaseKWh       float64   `json:"base_kwh"`
	HotWaterKWh   float64   `json:"hot_water_gas_equiv_kwh"`
	CookingKWh    float64   `json:"cooking_gas_equiv_kwh"`
	EVKWh         float64   `json:"ev_kwh"`
	GridImportKWh float64   `json:"grid_import_kwh"`
	GridExportKWh float64   `json:"grid_export_kwh"`
	AvgTempC      float64   `json:"avg_temp_c"`
}

func (s *Service) Monthly(ctx context.Context) ([]MonthRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date_trunc('month', e.ts) AS m,
		       sum(e.solar_gen_kwh),
		       sum(d.hvac_kwh),
		       sum(d.pool_pump_kwh),
		       sum(d.base_load_kwh),
		       sum(d.hot_water_gas_equiv_kwh),
		       sum(d.cooking_gas_equiv_kwh),
		       sum(d.ev_kwh),
		       sum(e.grid_import_kwh),
		       sum(e.grid_export_kwh),
		       avg(e.temperature_c)
		FROM energy_intervals e
		JOIN load_decomposition d USING (ts)
		GROUP BY 1
		ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MonthRow{}
	for rows.Next() {
		var r MonthRow
		if err := rows.Scan(&r.Month, &r.SolarKWh, &r.HVACKWh, &r.PoolKWh, &r.BaseKWh, &r.HotWaterKWh, &r.CookingKWh, &r.EVKWh, &r.GridImportKWh, &r.GridExportKWh, &r.AvgTempC); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
