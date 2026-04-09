// Package scenarios runs the four upgrade modules and produces the
// payback / carbon results that drive the Scenarios screen comparison table
// and chart. Each upgrade is a delta against the baseline annuals.
//
// All numbers are intentionally simple — every input comes from the
// assumptions store so the user can dial things in from the dashboard.
package scenarios

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/assumptions"
	"github.com/chrisneave/phem/api/internal/baseline"
	"github.com/chrisneave/phem/api/internal/models"
	"github.com/chrisneave/phem/api/internal/solar"
)

type Engine struct {
	pool     *pgxpool.Pool
	asm      *assumptions.Store
	baseline *baseline.Service
	solar    *solar.Service
}

func New(pool *pgxpool.Pool, asm *assumptions.Store, b *baseline.Service, sol *solar.Service) *Engine {
	return &Engine{pool: pool, asm: asm, baseline: b, solar: sol}
}

// ----------------------------------------------------------------------------
// CRUD
// ----------------------------------------------------------------------------

func (e *Engine) List(ctx context.Context) ([]models.Scenario, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT s.id, s.name, COALESCE(s.description,''), s.upgrades, s.device_params, s.dispatch, s.created_at, s.updated_at,
		       r.capex_gross_aud, r.capex_net_aud, r.annual_saving_aud, r.payback_years,
		       r.annual_co2_saving_kg, r.embodied_co2_kg, r.carbon_payback_years,
		       r.cumulative_savings, r.breakdown
		FROM scenarios s
		LEFT JOIN scenario_results r ON r.scenario_id = s.id
		ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Scenario{}
	for rows.Next() {
		var s models.Scenario
		var ups, dev, disp []byte
		var cg, cn, sav, pb, co2, emb, cpb *float64
		var cum, brk []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &ups, &dev, &disp, &s.CreatedAt, &s.UpdatedAt,
			&cg, &cn, &sav, &pb, &co2, &emb, &cpb, &cum, &brk); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(ups, &s.Upgrades)
		_ = json.Unmarshal(dev, &s.DeviceParams)
		_ = json.Unmarshal(disp, &s.Dispatch)
		if cg != nil {
			r := &models.ScenarioResult{
				CapexGrossAUD:      *cg,
				CapexNetAUD:        *cn,
				AnnualSavingAUD:    *sav,
				AnnualCO2SavingKg:  *co2,
				EmbodiedCO2Kg:      *emb,
			}
			if pb != nil {
				r.PaybackYears = *pb
			}
			if cpb != nil {
				r.CarbonPaybackYears = *cpb
			}
			_ = json.Unmarshal(cum, &r.CumulativeSavings)
			_ = json.Unmarshal(brk, &r.Breakdown)
			s.Result = r
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (e *Engine) Upsert(ctx context.Context, s *models.Scenario) error {
	ups, _ := json.Marshal(s.Upgrades)
	dev, _ := json.Marshal(s.DeviceParams)
	disp, _ := json.Marshal(s.Dispatch)
	if s.ID == 0 {
		err := e.pool.QueryRow(ctx, `
			INSERT INTO scenarios (name, description, upgrades, device_params, dispatch)
			VALUES ($1,$2,$3::jsonb,$4::jsonb,$5::jsonb) RETURNING id, created_at, updated_at`,
			s.Name, s.Description, ups, dev, disp).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
		return err
	}
	_, err := e.pool.Exec(ctx, `
		UPDATE scenarios SET name=$2, description=$3, upgrades=$4::jsonb, device_params=$5::jsonb, dispatch=$6::jsonb, updated_at=now()
		WHERE id=$1`, s.ID, s.Name, s.Description, ups, dev, disp)
	return err
}

func (e *Engine) Delete(ctx context.Context, id int) error {
	_, err := e.pool.Exec(ctx, `DELETE FROM scenarios WHERE id=$1`, id)
	return err
}

// ----------------------------------------------------------------------------
// Compute
// ----------------------------------------------------------------------------

func (e *Engine) Compute(ctx context.Context, scenarioID int) (*models.ScenarioResult, error) {
	scenarios, err := e.List(ctx)
	if err != nil {
		return nil, err
	}
	var s *models.Scenario
	for i := range scenarios {
		if scenarios[i].ID == scenarioID {
			s = &scenarios[i]
			break
		}
	}
	if s == nil {
		return nil, fmt.Errorf("scenario %d not found", scenarioID)
	}
	asm, err := e.asm.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	base, err := e.baseline.Summary(ctx)
	if err != nil {
		return nil, err
	}
	res, err := e.computeFromScenario(ctx, s, asm, base)
	if err != nil {
		return nil, err
	}
	if err := e.persistResult(ctx, scenarioID, res); err != nil {
		return nil, err
	}
	return res, nil
}

// computeFromScenario runs the upgrade modules against an in-memory scenario
// and returns a result without persisting. Used by both Compute (which then
// persists) and Explore (which evaluates all 16 combinations transiently).
func (e *Engine) computeFromScenario(
	ctx context.Context,
	s *models.Scenario,
	asm assumptions.Snapshot,
	base models.BaselineSummary,
) (*models.ScenarioResult, error) {
	annualSaving := 0.0
	annualCO2Saving := 0.0
	embodied := 0.0
	capexGross := 0.0
	capexNet := 0.0
	breakdown := map[string]any{}

	// ---- HPHWS ----
	if s.Upgrades.HPHWS {
		// Annual gas hot water = AnnualGasMJ * GasHWFraction (MJ -> kWh thermal /3.6)
		gasHWThermalKWh := base.AnnualGasMJ * asm.GasHWFraction / 3.6
		// Average COP across the year (rough)
		cop := paramFloat(s.DeviceParams, "hphws_cop", 3.5)
		hphwsElecKWh := gasHWThermalKWh / cop
		// Saving: gas cost saved - electricity cost added (use mid-day cheap power assumption ~ $0.10)
		gasCostSaved := gasHWThermalKWh * asm.GasKgPerKWhTh / asm.GasKgPerKWhTh * 0.045 * 3.6 // = MJ saved * $0.045
		// Simpler: gas saved $ = MJ * 0.045
		gasMJSaved := gasHWThermalKWh * 3.6
		gasCostSaved = gasMJSaved * 0.045
		smartPrice := paramFloat(s.DeviceParams, "hphws_avg_price", 0.10)
		elecCostAdded := hphwsElecKWh * smartPrice
		hphwsSaving := gasCostSaved - elecCostAdded
		// CO2: avoided gas emissions - added grid emissions (smart charged so use 0.4*grid)
		gridFactor := asm.GridKgPerKWh * 0.5
		hphwsCO2 := gasHWThermalKWh*asm.GasKgPerKWhTh - hphwsElecKWh*gridFactor

		annualSaving += hphwsSaving
		annualCO2Saving += hphwsCO2
		capexGross += asm.HPHWSGross
		capexNet += asm.HPHWSGross - asm.HPHWSRebate
		embodied += asm.HPHWSEmbodiedKg
		breakdown["hphws"] = map[string]any{
			"annual_saving_aud":     round2(hphwsSaving),
			"annual_co2_saving_kg":  round2(hphwsCO2),
			"capex_gross_aud":       asm.HPHWSGross,
			"capex_net_aud":         asm.HPHWSGross - asm.HPHWSRebate,
			"elec_kwh":              round2(hphwsElecKWh),
			"gas_mj_avoided":        round2(gasMJSaved),
			"cop":                   cop,
		}
	}

	// ---- Induction ----
	if s.Upgrades.Induction {
		gasCookThermalKWh := base.AnnualGasMJ * (1 - asm.GasHWFraction) / 3.6
		// Induction is ~3x more efficient delivered, so electric kWh ≈ thermal/3
		ratio := paramFloat(s.DeviceParams, "induction_eff_ratio", 3.0)
		elecKWh := gasCookThermalKWh / ratio
		// Cooking happens at meal times — use average residential rate ~ $0.25/kWh
		elecPrice := paramFloat(s.DeviceParams, "induction_avg_price", 0.25)
		gasMJ := gasCookThermalKWh * 3.6
		gasCostSaved := gasMJ * 0.045
		elecCostAdded := elecKWh * elecPrice
		saving := gasCostSaved - elecCostAdded
		co2 := gasCookThermalKWh*asm.GasKgPerKWhTh - elecKWh*asm.GridKgPerKWh

		annualSaving += saving
		annualCO2Saving += co2
		capexGross += asm.IndGross
		capexNet += asm.IndGross - asm.IndRebate
		embodied += asm.InductionEmbodiedKg
		breakdown["induction"] = map[string]any{
			"annual_saving_aud":    round2(saving),
			"annual_co2_saving_kg": round2(co2),
			"capex_gross_aud":      asm.IndGross,
			"capex_net_aud":        asm.IndGross - asm.IndRebate,
			"elec_kwh":             round2(elecKWh),
		}
	}

	// ---- EV ----
	if s.Upgrades.EV {
		evEff := paramFloat(s.DeviceParams, "ev_kwh_per_100km", 16.0)
		annualKM := asm.AnnualKM
		evKWh := annualKM * evEff / 100.0
		// Petrol replaced
		litres := annualKM * asm.CX5LPer100 / 100.0
		fuelCostSaved := litres * asm.PetrolPrice
		// Smart charging avg ~ $0.08/kWh
		evPrice := paramFloat(s.DeviceParams, "ev_avg_price", 0.08)
		elecCostAdded := evKWh * evPrice
		// Whether to include vehicle cost or fuel-only
		includeVehicle := paramBool(s.DeviceParams, "ev_include_vehicle", false)
		saving := fuelCostSaved - elecCostAdded
		// CO2
		co2 := litres*asm.PetrolKgPerL - evKWh*asm.GridKgPerKWh*0.6

		annualSaving += saving
		annualCO2Saving += co2
		if includeVehicle {
			capexGross += asm.EVGross
			capexNet += asm.EVGross - asm.EVRebate
		}
		embodied += asm.EVEmbodiedKg
		breakdown["ev"] = map[string]any{
			"annual_saving_aud":    round2(saving),
			"annual_co2_saving_kg": round2(co2),
			"capex_gross_aud":      asm.EVGross,
			"capex_net_aud":        asm.EVGross - asm.EVRebate,
			"include_vehicle":      includeVehicle,
			"ev_kwh":               round2(evKWh),
			"litres_avoided":       round2(litres),
		}
	}

	// ---- Solar upgrade ----
	if s.Upgrades.Solar {
		panelKey := paramString(s.DeviceParams, "solar_panel", "premium")
		panelCount := int(paramFloat(s.DeviceParams, "solar_panel_count", 15))
		var spec solar.PanelSpec
		var src map[string]any
		if panelKey == "standard" {
			src = asm.PanelStandard
		} else {
			src = asm.PanelPremium
		}
		spec = panelSpecFromMap(src)
		sm, err := e.solar.Compute(ctx)
		if err == nil {
			yield, _ := e.solar.EstimateAnnualYield(ctx, sm, spec, panelCount)
			incremental := yield - base.AnnualSolarKWh
			if incremental < 0 {
				incremental = 0
			}
			// 70% self-consumed at $0.30 avoided, 30% exported at $0.06
			selfCons := incremental * 0.7
			exported := incremental * 0.3
			saving := selfCons*0.30 + exported*0.06
			co2 := incremental * asm.GridKgPerKWh
			annualSaving += saving
			annualCO2Saving += co2
			capexGross += asm.SolarGross
			capexNet += asm.SolarGross - asm.SolarRebate
			embodied += float64(panelCount) * asm.PanelEmbodiedKg
			breakdown["solar"] = map[string]any{
				"annual_saving_aud":    round2(saving),
				"annual_co2_saving_kg": round2(co2),
				"capex_gross_aud":      asm.SolarGross,
				"capex_net_aud":        asm.SolarGross - asm.SolarRebate,
				"new_array_kwh":        round2(yield),
				"incremental_kwh":      round2(incremental),
				"panel":                panelKey,
				"panel_count":          panelCount,
			}
		}
	}

	res := &models.ScenarioResult{
		CapexGrossAUD:     round2(capexGross),
		CapexNetAUD:       round2(capexNet),
		AnnualSavingAUD:   round2(annualSaving),
		AnnualCO2SavingKg: round2(annualCO2Saving),
		EmbodiedCO2Kg:     round2(embodied),
		Breakdown:         breakdown,
	}
	if annualSaving > 0 {
		res.PaybackYears = round2(capexNet / annualSaving)
	}
	if annualCO2Saving > 0 {
		res.CarbonPaybackYears = round2(embodied / annualCO2Saving)
	}

	// 20-year cumulative saving curve
	horizon := int(asm.Horizon)
	if horizon < 5 {
		horizon = 20
	}
	cum := make([]models.YearPoint, 0, horizon+1)
	for y := 0; y <= horizon; y++ {
		net := -capexNet + annualSaving*float64(y)
		cum = append(cum, models.YearPoint{Year: y, NetSavingAUD: round2(net)})
	}
	res.CumulativeSavings = cum

	return res, nil
}

func (e *Engine) persistResult(ctx context.Context, scenarioID int, res *models.ScenarioResult) error {
	cumJSON, _ := json.Marshal(res.CumulativeSavings)
	brkJSON, _ := json.Marshal(res.Breakdown)
	_, err := e.pool.Exec(ctx, `
		INSERT INTO scenario_results (scenario_id, capex_gross_aud, capex_net_aud, annual_saving_aud, payback_years, annual_co2_saving_kg, embodied_co2_kg, carbon_payback_years, cumulative_savings, breakdown, computed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb, now())
		ON CONFLICT (scenario_id) DO UPDATE SET
		  capex_gross_aud=EXCLUDED.capex_gross_aud,
		  capex_net_aud=EXCLUDED.capex_net_aud,
		  annual_saving_aud=EXCLUDED.annual_saving_aud,
		  payback_years=EXCLUDED.payback_years,
		  annual_co2_saving_kg=EXCLUDED.annual_co2_saving_kg,
		  embodied_co2_kg=EXCLUDED.embodied_co2_kg,
		  carbon_payback_years=EXCLUDED.carbon_payback_years,
		  cumulative_savings=EXCLUDED.cumulative_savings,
		  breakdown=EXCLUDED.breakdown,
		  computed_at=now()`,
		scenarioID, res.CapexGrossAUD, res.CapexNetAUD, res.AnnualSavingAUD, nullableFloat(res.PaybackYears),
		res.AnnualCO2SavingKg, res.EmbodiedCO2Kg, nullableFloat(res.CarbonPaybackYears), cumJSON, brkJSON)
	return err
}

// ----------------------------------------------------------------------------
// Explore — brute-force all 16 upgrade combinations and rank them. With 4
// boolean upgrades the search space is small enough to evaluate in full
// (well under 100 ms on the target hardware), so we don't need a heuristic.
// ----------------------------------------------------------------------------

type ExploreCombo struct {
	Upgrades         models.UpgradeToggles `json:"upgrades"`
	Label            string                `json:"label"`
	CapexNetAUD      float64               `json:"capex_net_aud"`
	AnnualSavingAUD  float64               `json:"annual_saving_aud"`
	PaybackYears     float64               `json:"payback_years"`
	AnnualCO2Kg      float64               `json:"annual_co2_saving_kg"`
	EmbodiedCO2Kg    float64               `json:"embodied_co2_kg"`
	CarbonPaybackYrs float64               `json:"carbon_payback_years"`
	NPV20            float64               `json:"npv_20yr_aud"` // 20-yr cumulative net saving (capex paid back from year 0)
	Tags             []string              `json:"tags"`         // "best_payback", "best_carbon", etc.
}

type ExploreResult struct {
	Combos      []ExploreCombo `json:"combos"`
	BestPayback int            `json:"best_payback_idx"`
	BestCarbon  int            `json:"best_carbon_idx"`
	BestNPV     int            `json:"best_npv_idx"`
	Cheapest    int            `json:"cheapest_idx"`
}

func (e *Engine) Explore(ctx context.Context) (*ExploreResult, error) {
	asm, err := e.asm.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	base, err := e.baseline.Summary(ctx)
	if err != nil {
		return nil, err
	}

	// Sensible defaults — match the user's interactive sliders.
	defaultParams := map[string]any{
		"hphws_cop":           3.5,
		"hphws_avg_price":     0.10,
		"induction_eff_ratio": 3.0,
		"induction_avg_price": 0.25,
		"ev_kwh_per_100km":    16.0,
		"ev_avg_price":        0.08,
		"ev_include_vehicle":  false,
		"solar_panel":         "premium",
		"solar_panel_count":   15.0,
	}

	combos := []ExploreCombo{}
	for mask := 0; mask < 16; mask++ {
		ups := models.UpgradeToggles{
			HPHWS:     mask&1 != 0,
			Induction: mask&2 != 0,
			EV:        mask&4 != 0,
			Solar:     mask&8 != 0,
		}
		s := &models.Scenario{
			Name:         labelFor(ups),
			Upgrades:     ups,
			DeviceParams: defaultParams,
			Dispatch:     map[string]any{},
		}
		res, err := e.computeFromScenario(ctx, s, asm, base)
		if err != nil {
			return nil, err
		}
		npv := -res.CapexNetAUD + res.AnnualSavingAUD*20
		combos = append(combos, ExploreCombo{
			Upgrades:         ups,
			Label:            s.Name,
			CapexNetAUD:      res.CapexNetAUD,
			AnnualSavingAUD:  res.AnnualSavingAUD,
			PaybackYears:     res.PaybackYears,
			AnnualCO2Kg:      res.AnnualCO2SavingKg,
			EmbodiedCO2Kg:    res.EmbodiedCO2Kg,
			CarbonPaybackYrs: res.CarbonPaybackYears,
			NPV20:            round2(npv),
		})
	}

	// Rank
	best := ExploreResult{Combos: combos, BestPayback: -1, BestCarbon: -1, BestNPV: -1, Cheapest: -1}
	bestPB, bestCO2, bestNPV, cheap := 1e9, -1.0, -1e9, 1e9
	for i, c := range combos {
		if c.PaybackYears > 0 && c.PaybackYears < bestPB {
			bestPB = c.PaybackYears
			best.BestPayback = i
		}
		if c.AnnualCO2Kg > bestCO2 {
			bestCO2 = c.AnnualCO2Kg
			best.BestCarbon = i
		}
		if c.NPV20 > bestNPV {
			bestNPV = c.NPV20
			best.BestNPV = i
		}
		// Cheapest non-zero
		if c.CapexNetAUD > 0 && c.CapexNetAUD < cheap {
			cheap = c.CapexNetAUD
			best.Cheapest = i
		}
	}
	if best.BestPayback >= 0 {
		best.Combos[best.BestPayback].Tags = append(best.Combos[best.BestPayback].Tags, "best_payback")
	}
	if best.BestCarbon >= 0 {
		best.Combos[best.BestCarbon].Tags = append(best.Combos[best.BestCarbon].Tags, "best_carbon")
	}
	if best.BestNPV >= 0 {
		best.Combos[best.BestNPV].Tags = append(best.Combos[best.BestNPV].Tags, "best_20yr_value")
	}
	if best.Cheapest >= 0 {
		best.Combos[best.Cheapest].Tags = append(best.Combos[best.Cheapest].Tags, "cheapest_entry")
	}
	return &best, nil
}

// labelFor produces a short human label like "HPHWS + Induction + Solar".
func labelFor(u models.UpgradeToggles) string {
	parts := []string{}
	if u.HPHWS {
		parts = append(parts, "HPHWS")
	}
	if u.Induction {
		parts = append(parts, "Induction")
	}
	if u.EV {
		parts = append(parts, "EV")
	}
	if u.Solar {
		parts = append(parts, "Solar+")
	}
	if len(parts) == 0 {
		return "Do nothing"
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += " + " + p
	}
	return out
}

// ComputeAll recomputes every scenario (used after assumptions are edited).
func (e *Engine) ComputeAll(ctx context.Context) error {
	scenarios, err := e.List(ctx)
	if err != nil {
		return err
	}
	for _, s := range scenarios {
		if _, err := e.Compute(ctx, s.ID); err != nil {
			return err
		}
	}
	return nil
}

// SeedDefaultScenarios populates a useful starter set if scenarios is empty.
func (e *Engine) SeedDefaultScenarios(ctx context.Context) error {
	var n int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM scenarios`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	defaults := []models.Scenario{
		{Name: "Do nothing", Description: "Baseline reference", Upgrades: models.UpgradeToggles{}, DeviceParams: map[string]any{}, Dispatch: map[string]any{}},
		{Name: "Hot water + induction", Description: "Cheapest electrification first", Upgrades: models.UpgradeToggles{HPHWS: true, Induction: true}, DeviceParams: map[string]any{"hphws_cop": 3.5}, Dispatch: map[string]any{}},
		{Name: "All gas removal", Description: "HPHWS + induction + new solar", Upgrades: models.UpgradeToggles{HPHWS: true, Induction: true, Solar: true}, DeviceParams: map[string]any{"solar_panel": "premium", "solar_panel_count": 15}, Dispatch: map[string]any{}},
		{Name: "Full electrification", Description: "Everything inc. EV (fuel only)", Upgrades: models.UpgradeToggles{HPHWS: true, Induction: true, EV: true, Solar: true}, DeviceParams: map[string]any{"ev_kwh_per_100km": 16, "ev_include_vehicle": false, "solar_panel": "premium", "solar_panel_count": 15}, Dispatch: map[string]any{}},
		{Name: "Full + EV total cost", Description: "Includes EV vehicle replacement capex", Upgrades: models.UpgradeToggles{HPHWS: true, Induction: true, EV: true, Solar: true}, DeviceParams: map[string]any{"ev_kwh_per_100km": 16, "ev_include_vehicle": true, "solar_panel": "premium", "solar_panel_count": 15}, Dispatch: map[string]any{}},
	}
	for i := range defaults {
		if err := e.Upsert(ctx, &defaults[i]); err != nil {
			return err
		}
		if _, err := e.Compute(ctx, defaults[i].ID); err != nil {
			return err
		}
	}
	return nil
}

// ----- helpers -----

func paramFloat(m map[string]any, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		switch x := v.(type) {
		case float64:
			return x
		case int:
			return float64(x)
		case json.Number:
			f, _ := x.Float64()
			return f
		}
	}
	return def
}

func paramBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok2 := v.(bool); ok2 {
			return b
		}
	}
	return def
}

func paramString(m map[string]any, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return def
}

func panelSpecFromMap(m map[string]any) solar.PanelSpec {
	return solar.PanelSpec{
		Watt:         paramFloat(m, "watt", 400),
		Eff:          paramFloat(m, "eff", 0.21),
		TempCoefPerC: paramFloat(m, "temp_coef_per_c", -0.003),
		LengthM:      paramFloat(m, "length_m", 1.7),
		WidthM:       paramFloat(m, "width_m", 1.1),
	}
}

func nullableFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
