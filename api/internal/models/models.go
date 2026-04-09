package models

import "time"

// Interval is one 15-minute reading. Phase 2 forecast fields are nullable.
type Interval struct {
	TS                  time.Time `json:"ts"`
	SolarGenKWh         float64   `json:"solar_gen_kwh"`
	ConsumptionKWh      float64   `json:"consumption_kwh"`
	GridImportKWh       float64   `json:"grid_import_kwh"`
	GridExportKWh       float64   `json:"grid_export_kwh"`
	BatterySOCKWh       float64   `json:"battery_soc_kwh"`
	BatteryChargeKWh    float64   `json:"battery_charge_kwh"`
	BatteryDischargeKWh float64   `json:"battery_discharge_kwh"`
	SpotImport          float64   `json:"spot_price_import_aud_per_kwh"`
	SpotExport          float64   `json:"spot_price_export_aud_per_kwh"`
	TempC               float64   `json:"temperature_c"`
	GHIWm2              float64   `json:"ghi_w_m2"`
}

// Decomposition is one 15-minute decomposed-load row.
type Decomposition struct {
	TS                  time.Time `json:"ts"`
	HVACKWh             float64   `json:"hvac_kwh"`
	PoolPumpKWh         float64   `json:"pool_pump_kwh"`
	HotWaterGasEquivKWh float64   `json:"hot_water_gas_equiv_kwh"`
	CookingGasEquivKWh  float64   `json:"cooking_gas_equiv_kwh"`
	EVKWh               float64   `json:"ev_kwh"`
	BaseLoadKWh         float64   `json:"base_load_kwh"`
}

// GasBill — quarterly gas bill record.
type GasBill struct {
	ID             int       `json:"id"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	ConsumptionMJ  float64   `json:"consumption_mj"`
	CostAUD        float64   `json:"cost_aud"`
	Note           string    `json:"note,omitempty"`
}

// Scenario — user-defined upgrade combination.
type Scenario struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Upgrades     UpgradeToggles  `json:"upgrades"`
	DeviceParams map[string]any  `json:"device_params"`
	Dispatch     map[string]any  `json:"dispatch"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Result       *ScenarioResult `json:"result,omitempty"`
}

type UpgradeToggles struct {
	HPHWS     bool `json:"hphws"`
	Induction bool `json:"induction"`
	EV        bool `json:"ev"`
	Solar     bool `json:"solar"`
}

type ScenarioResult struct {
	CapexGrossAUD       float64        `json:"capex_gross_aud"`
	CapexNetAUD         float64        `json:"capex_net_aud"`
	AnnualSavingAUD     float64        `json:"annual_saving_aud"`
	PaybackYears        float64        `json:"payback_years"`
	AnnualCO2SavingKg   float64        `json:"annual_co2_saving_kg"`
	EmbodiedCO2Kg       float64        `json:"embodied_co2_kg"`
	CarbonPaybackYears  float64        `json:"carbon_payback_years"`
	CumulativeSavings   []YearPoint    `json:"cumulative_savings"`
	Breakdown           map[string]any `json:"breakdown"`
}

type YearPoint struct {
	Year       int     `json:"year"`
	NetSavingAUD float64 `json:"net_saving_aud"`
}

// Assumption — single key/value config item editable on the Assumptions screen.
type Assumption struct {
	Key         string `json:"key"`
	Category    string `json:"category"`
	Label       string `json:"label"`
	Value       any    `json:"value"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
}

// BaselineSummary — what the Baseline screen shows in cards.
type BaselineSummary struct {
	RangeStart           time.Time `json:"range_start"`
	RangeEnd             time.Time `json:"range_end"`
	IntervalCount        int       `json:"interval_count"`
	AnnualSolarKWh       float64   `json:"annual_solar_kwh"`
	AnnualConsumptionKWh float64   `json:"annual_consumption_kwh"`
	AnnualImportKWh      float64   `json:"annual_import_kwh"`
	AnnualExportKWh      float64   `json:"annual_export_kwh"`
	AnnualGasMJ          float64   `json:"annual_gas_mj"`
	AnnualElecCostAUD    float64   `json:"annual_elec_cost_aud"`
	AnnualGasCostAUD     float64   `json:"annual_gas_cost_aud"`
	AnnualCO2Kg          float64   `json:"annual_co2_kg"`
	SelfConsumptionPct   float64   `json:"self_consumption_pct"`
	SolarFractionPct     float64   `json:"solar_fraction_pct"`
}

// IngestionRun — audit row for the Data screen "data health" widget.
type IngestionRun struct {
	ID         int       `json:"id"`
	Source     string    `json:"source"`
	Filename   string    `json:"filename,omitempty"`
	RowsIn     int       `json:"rows_in"`
	RowsLoaded int       `json:"rows_loaded"`
	Gaps       int       `json:"gaps_found"`
	Duplicates int       `json:"duplicates"`
	Implausible int      `json:"implausible"`
	RangeStart *time.Time `json:"range_start,omitempty"`
	RangeEnd   *time.Time `json:"range_end,omitempty"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
