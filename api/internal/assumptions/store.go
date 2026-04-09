// Package assumptions provides typed access to the assumptions key/value table.
// All callers go through Get() / GetFloat() / Snapshot() so cost defaults,
// rebates, dispatch thresholds and emissions factors stay user-editable
// (NFR-06: every assumption visible and editable).
package assumptions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) All(ctx context.Context) ([]models.Assumption, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, category, label, value, COALESCE(unit,''), COALESCE(description,'') FROM assumptions ORDER BY category, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Assumption{}
	for rows.Next() {
		var a models.Assumption
		var raw []byte
		if err := rows.Scan(&a.Key, &a.Category, &a.Label, &raw, &a.Unit, &a.Description); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &a.Value)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) Set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE assumptions SET value = $2::jsonb, updated_at = now() WHERE key = $1`, key, raw)
	return err
}

func (s *Store) GetFloat(ctx context.Context, key string) (float64, error) {
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT value FROM assumptions WHERE key = $1`, key).Scan(&raw); err != nil {
		return 0, fmt.Errorf("missing assumption %s: %w", key, err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, fmt.Errorf("assumption %s not float: %w", key, err)
	}
	return v, nil
}

// Snapshot returns the entire bag as a typed struct so callers (baseline,
// scenarios) don't have to round-trip per assumption.
type Snapshot struct {
	GridKgPerKWh        float64
	GasKgPerKWhTh       float64
	PetrolKgPerL        float64
	PanelEmbodiedKg     float64
	EVEmbodiedKg        float64
	HPHWSEmbodiedKg     float64
	InductionEmbodiedKg float64

	HPHWSGross  float64
	HPHWSRebate float64
	IndGross    float64
	IndRebate   float64
	SolarGross  float64
	SolarRebate float64
	EVGross     float64
	EVRebate    float64
	PetrolPrice float64
	CX5LPer100  float64

	GasHWFraction float64
	AnnualKM      float64
	DailyHWL      float64

	BatteryChargeBelow    float64
	BatteryDischargeAbove float64
	SmartLoadThreshold    float64

	SupplyAUDPerDay float64
	ImportCap       float64
	ExportFloor     float64

	Horizon float64

	PanelStandard map[string]any
	PanelPremium  map[string]any
}

func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	all, err := s.All(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	m := map[string]any{}
	for _, a := range all {
		m[a.Key] = a.Value
	}
	getF := func(k string) float64 {
		if v, ok := m[k]; ok {
			if f, ok2 := v.(float64); ok2 {
				return f
			}
		}
		return 0
	}
	getMap := func(k string) map[string]any {
		if v, ok := m[k]; ok {
			if mm, ok2 := v.(map[string]any); ok2 {
				return mm
			}
		}
		return nil
	}
	return Snapshot{
		GridKgPerKWh:          getF("emission.grid_kg_per_kwh"),
		GasKgPerKWhTh:         getF("emission.gas_kg_per_kwh_th"),
		PetrolKgPerL:          getF("emission.petrol_kg_per_l"),
		PanelEmbodiedKg:       getF("emission.panel_kg_each"),
		EVEmbodiedKg:          getF("emission.ev_embodied_kg"),
		HPHWSEmbodiedKg:       getF("emission.hphws_embodied_kg"),
		InductionEmbodiedKg:   getF("emission.induction_embodied_kg"),
		HPHWSGross:            getF("cost.hphws_gross_aud"),
		HPHWSRebate:           getF("rebate.hphws_aud"),
		IndGross:              getF("cost.induction_gross_aud"),
		IndRebate:             getF("rebate.induction_aud"),
		SolarGross:            getF("cost.solar_upgrade_gross_aud"),
		SolarRebate:           getF("rebate.solar_upgrade_aud"),
		EVGross:               getF("cost.ev_gross_aud"),
		EVRebate:              getF("rebate.ev_aud"),
		PetrolPrice:           getF("cost.petrol_aud_per_l"),
		CX5LPer100:            getF("cost.cx5_l_per_100km"),
		GasHWFraction:         getF("usage.gas_hot_water_fraction"),
		AnnualKM:              getF("usage.annual_km"),
		DailyHWL:              getF("usage.daily_hot_water_l"),
		BatteryChargeBelow:    getF("dispatch.battery_charge_below"),
		BatteryDischargeAbove: getF("dispatch.battery_discharge_above"),
		SmartLoadThreshold:    getF("dispatch.smart_load_threshold"),
		SupplyAUDPerDay:       getF("tariff.supply_aud_per_day"),
		ImportCap:             getF("tariff.import_cap_aud_per_kwh"),
		ExportFloor:           getF("tariff.export_floor_aud_per_kwh"),
		Horizon:               getF("scenario.horizon_years"),
		PanelStandard:         getMap("panel.standard"),
		PanelPremium:          getMap("panel.premium"),
	}, nil
}
