// Package seed populates the DB with 16 months of synthetic 15-minute energy
// data when energy_intervals is empty. The synthetic data is deterministic
// (fixed RNG seed) so each fresh boot produces an identical dataset.
//
// The intent is to make the dashboard render fully end-to-end without
// requiring real Enphase / Amber / gas CSVs. Real CSV upload via the Data
// screen replaces this data.
package seed

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	intervalMinutes = 15
	intervalsPerDay = 24 * 60 / intervalMinutes
)

// SeedIfEmpty populates the DB with synthetic data if energy_intervals has 0 rows.
func SeedIfEmpty(ctx context.Context, pool *pgxpool.Pool, siteLat, siteLon float64) error {
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM energy_intervals").Scan(&count); err != nil {
		return fmt.Errorf("count energy_intervals: %w", err)
	}
	if count > 0 {
		log.Printf("seed: energy_intervals already has %d rows, skipping", count)
		return nil
	}
	return generate(ctx, pool, siteLat, siteLon)
}

func generate(ctx context.Context, pool *pgxpool.Pool, siteLat, siteLon float64) error {
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		loc = time.UTC
	}
	// 16 months ending close to "today" so the dashboard shows recent dates.
	end := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
	start := end.AddDate(0, -16, 0)
	totalIntervals := int(end.Sub(start).Minutes()) / intervalMinutes
	log.Printf("seed: generating %d 15-min intervals from %s to %s", totalIntervals, start.Format("2006-01-02"), end.Format("2006-01-02"))

	rng := rand.New(rand.NewSource(42))

	type row struct {
		ts                                                                                                          time.Time
		solar, cons, gridImport, gridExport, batSOC, batCh, batDis, spotImp, spotExp, temp, ghi                     float64
		hvac, pool, hwGas, cookGas, evKWh, base                                                                     float64
	}
	rows := make([]row, 0, totalIntervals)

	batSOC := 21.5 // start half-charged on a 43 kWh battery
	const batCap = 43.0
	const batMaxKW = 10.0 // 10 kW peak charge/discharge

	for i := 0; i < totalIntervals; i++ {
		ts := start.Add(time.Duration(i*intervalMinutes) * time.Minute)
		local := ts.In(loc)

		// ---- weather (seasonal, NSW Sydney-ish) ----
		dayOfYear := float64(local.YearDay())
		// Seasonal mean temp: ~22 summer, ~13 winter (S-hemisphere phase: Jan warm, July cold)
		seasonalMean := 17.5 + 5.0*math.Cos(2*math.Pi*(dayOfYear-15)/365.0)
		// Diurnal swing
		hourFrac := float64(local.Hour())*60 + float64(local.Minute())
		diurnal := 5.0 * math.Cos(2*math.Pi*(hourFrac-840)/1440) // peak ~14:00
		temp := seasonalMean + diurnal + rng.NormFloat64()*1.2

		// ---- clear-sky GHI (very simple solar position) ----
		// Solar declination
		decl := 23.44 * math.Pi / 180 * math.Sin(2*math.Pi*(284+float64(local.YearDay()))/365.0)
		latRad := siteLat * math.Pi / 180
		hourAngle := (hourFrac/60 - 12) * 15 * math.Pi / 180
		sinAlt := math.Sin(latRad)*math.Sin(decl) + math.Cos(latRad)*math.Cos(decl)*math.Cos(hourAngle)
		alt := math.Asin(sinAlt)
		var ghi float64
		if alt > 0 {
			// Haurwitz clear-sky model: GHI = 1098 * sin(alt) * exp(-0.057/sin(alt))
			ghi = 1098 * sinAlt * math.Exp(-0.057/sinAlt)
		}
		// Cloud noise — random multiplier 0.4–1.0 with persistence
		cloud := 0.6 + 0.4*math.Abs(math.Sin(float64(i)/13.7))
		if rng.Float64() < 0.05 {
			cloud *= 0.3 // occasional heavy cloud
		}
		ghi *= cloud
		if ghi < 0 {
			ghi = 0
		}

		// ---- solar generation (5 kW array, ~80% of clear-sky theoretical at noon) ----
		// At GHI 1000 W/m², 5 kW array roughly produces 5 kWh per hour = 1.25 kWh per 15 min.
		solarKWh := ghi / 1000.0 * 1.25 * 0.94 // shading/soiling
		if solarKWh < 0 {
			solarKWh = 0
		}

		// ---- HVAC: cooling + heating, mostly winter heating ----
		hvacKWh := 0.0
		// Heating threshold 16C, cooling threshold 26C
		if temp < 16 {
			hvacKWh = (16 - temp) * 0.06 // up to ~0.5 kWh / 15min on cold morning
			if local.Hour() < 6 || local.Hour() > 22 {
				hvacKWh *= 0.4
			}
		} else if temp > 26 {
			hvacKWh = (temp - 26) * 0.05
			if local.Hour() < 9 || local.Hour() > 21 {
				hvacKWh *= 0.3
			}
		}
		hvacKWh += rng.Float64() * 0.05

		// ---- Pool pump: scheduled 10:00-15:00, ~1.0 kW ----
		poolKWh := 0.0
		if local.Hour() >= 10 && local.Hour() < 15 {
			poolKWh = 0.25 // 1 kW * 15min
		}

		// ---- Hot water (gas equiv): morning 06-08 peak, evening 18-21 ----
		hwKWh := 0.0
		switch local.Hour() {
		case 6, 7:
			hwKWh = 0.8 + rng.Float64()*0.2
		case 18, 19, 20:
			hwKWh = 0.6 + rng.Float64()*0.2
		default:
			hwKWh = 0.05 // standby
		}

		// ---- Cooking (gas equiv): meal windows ----
		cookKWh := 0.0
		switch local.Hour() {
		case 7:
			cookKWh = 0.15
		case 12:
			cookKWh = 0.2
		case 18, 19:
			cookKWh = 0.4
		}

		// ---- EV: zero (baseline) ----
		evKWh := 0.0

		// ---- Base load (~250W constant) ----
		baseKWh := 0.06 + rng.NormFloat64()*0.005

		// Total electrical consumption excludes gas-equivalent loads (hwGas, cookGas)
		// because hot water is gas, cooking is gas in the baseline.
		consumption := hvacKWh + poolKWh + evKWh + baseKWh
		if consumption < 0 {
			consumption = 0
		}

		// ---- Spot price (sinusoid + noise + spikes) ----
		// Day average ~ 0.10 AUD/kWh, evening peak higher, occasional spikes
		spot := 0.08 + 0.06*math.Sin(2*math.Pi*(hourFrac-18*60)/1440)
		if local.Hour() >= 17 && local.Hour() <= 20 {
			spot += 0.10
		}
		if local.Hour() >= 11 && local.Hour() <= 14 && solarKWh > 0.5 {
			spot -= 0.06 // solar surplus depresses spot
		}
		spot += rng.NormFloat64() * 0.02
		if rng.Float64() < 0.002 {
			spot += 0.4 + rng.Float64()*0.6 // rare spike
		}
		if spot < -0.05 {
			spot = -0.05
		}
		if spot > 0.95 {
			spot = 0.95
		}
		spotExp := spot - 0.02
		if spotExp < -0.05 {
			spotExp = -0.05
		}

		// ---- Battery dispatch (heuristic: charge below 0.05, discharge above 0.30) ----
		batCh := 0.0
		batDis := 0.0
		net := consumption - solarKWh // positive => need import
		if net < 0 {
			// Solar surplus → charge battery first
			surplus := -net
			room := batCap - batSOC
			ch := math.Min(surplus, math.Min(room, batMaxKW*0.25))
			batCh = ch
			batSOC += ch
			net += ch // remaining surplus exports
		} else {
			// Need power. If price is high enough, discharge.
			if spot > 0.30 && batSOC > 2.0 {
				dis := math.Min(net, math.Min(batSOC-2.0, batMaxKW*0.25))
				batDis = dis
				batSOC -= dis
				net -= dis
			}
			// Also charge cheaply at night even with no surplus, if price < 0.05
			if spot < 0.05 && batSOC < batCap-1 {
				ch := math.Min(batCap-batSOC, batMaxKW*0.25)
				batCh = ch
				batSOC += ch
				net += ch
			}
		}

		gridImport := 0.0
		gridExport := 0.0
		if net > 0 {
			gridImport = net
		} else {
			gridExport = -net
		}

		rows = append(rows, row{
			ts:         ts,
			solar:      round3(solarKWh),
			cons:       round3(consumption),
			gridImport: round3(gridImport),
			gridExport: round3(gridExport),
			batSOC:     round3(batSOC),
			batCh:      round3(batCh),
			batDis:     round3(batDis),
			spotImp:    round4(spot),
			spotExp:    round4(spotExp),
			temp:       round2(temp),
			ghi:        round1(ghi),
			hvac:       round3(hvacKWh),
			pool:       round3(poolKWh),
			hwGas:      round3(hwKWh),
			cookGas:    round3(cookKWh),
			evKWh:      round3(evKWh),
			base:       round3(baseKWh),
		})
	}

	// COPY into energy_intervals
	intervalsCols := []string{"ts", "solar_gen_kwh", "consumption_kwh", "grid_import_kwh", "grid_export_kwh", "battery_soc_kwh", "battery_charge_kwh", "battery_discharge_kwh", "spot_price_import_aud_per_kwh", "spot_price_export_aud_per_kwh", "temperature_c", "ghi_w_m2"}
	intervalsSrc := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
		r := rows[i]
		return []any{r.ts, r.solar, r.cons, r.gridImport, r.gridExport, r.batSOC, r.batCh, r.batDis, r.spotImp, r.spotExp, r.temp, r.ghi}, nil
	})
	if _, err := pool.CopyFrom(ctx, pgx.Identifier{"energy_intervals"}, intervalsCols, intervalsSrc); err != nil {
		return fmt.Errorf("copy energy_intervals: %w", err)
	}

	// COPY into load_decomposition
	decompCols := []string{"ts", "hvac_kwh", "pool_pump_kwh", "hot_water_gas_equiv_kwh", "cooking_gas_equiv_kwh", "ev_kwh", "base_load_kwh"}
	decompSrc := pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
		r := rows[i]
		return []any{r.ts, r.hvac, r.pool, r.hwGas, r.cookGas, r.evKWh, r.base}, nil
	})
	if _, err := pool.CopyFrom(ctx, pgx.Identifier{"load_decomposition"}, decompCols, decompSrc); err != nil {
		return fmt.Errorf("copy load_decomposition: %w", err)
	}

	// Quarterly gas bills derived from total hot water + cooking gas equiv kWh per quarter.
	// 1 kWh = 3.6 MJ. Cost rough proxy at $0.04/MJ supply + 0.025 var.
	type qBill struct {
		startQ time.Time
		endQ   time.Time
		mj     float64
	}
	qBills := []qBill{}
	for q := 0; q < 6; q++ {
		startQ := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, q*3, 0)
		endQ := startQ.AddDate(0, 3, 0)
		if endQ.After(end) {
			break
		}
		var sumKWh float64
		for _, r := range rows {
			if !r.ts.Before(startQ) && r.ts.Before(endQ) {
				sumKWh += r.hwGas + r.cookGas
			}
		}
		// Convert kWh thermal -> MJ
		qBills = append(qBills, qBill{startQ, endQ, sumKWh * 3.6})
	}
	for _, b := range qBills {
		_, err := pool.Exec(ctx, `INSERT INTO gas_bills (period_start, period_end, consumption_mj, cost_aud, note) VALUES ($1,$2,$3,$4,$5)`,
			b.startQ, b.endQ.AddDate(0, 0, -1), b.mj, b.mj*0.045+90, "synthetic")
		if err != nil {
			return fmt.Errorf("insert gas bill: %w", err)
		}
	}

	// Audit row
	_, err = pool.Exec(ctx,
		`INSERT INTO ingestion_runs (source, rows_in, rows_loaded, range_start, range_end, status, message) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		"synthetic", len(rows), len(rows), start, end, "ok", "Generated synthetic baseline (deterministic seed=42)")
	if err != nil {
		return fmt.Errorf("audit row: %w", err)
	}

	log.Printf("seed: inserted %d intervals, %d gas bills", len(rows), len(qBills))
	return nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
