-- PHEM schema — Phase 1
-- All energy quantities are in kWh per 15-minute interval unless suffixed.
-- All money is AUD. All temperatures are degrees Celsius.

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ----------------------------------------------------------------------------
-- Raw + enriched 15-minute interval timeseries
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS energy_intervals (
    ts                              TIMESTAMPTZ      NOT NULL,
    solar_gen_kwh                   DOUBLE PRECISION,
    consumption_kwh                 DOUBLE PRECISION,
    grid_import_kwh                 DOUBLE PRECISION,
    grid_export_kwh                 DOUBLE PRECISION,
    battery_soc_kwh                 DOUBLE PRECISION,
    battery_charge_kwh              DOUBLE PRECISION,
    battery_discharge_kwh           DOUBLE PRECISION,
    spot_price_import_aud_per_kwh   DOUBLE PRECISION,
    spot_price_export_aud_per_kwh   DOUBLE PRECISION,
    temperature_c                   DOUBLE PRECISION,
    ghi_w_m2                        DOUBLE PRECISION,
    PRIMARY KEY (ts)
);
SELECT create_hypertable('energy_intervals', 'ts', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_energy_intervals_ts ON energy_intervals (ts DESC);

-- ----------------------------------------------------------------------------
-- Decomposed loads (computed by baseline model). Synthetic data populates these
-- directly so the dashboard can compare components.
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS load_decomposition (
    ts                       TIMESTAMPTZ NOT NULL,
    hvac_kwh                 DOUBLE PRECISION,
    pool_pump_kwh            DOUBLE PRECISION,
    hot_water_gas_equiv_kwh  DOUBLE PRECISION,  -- gas-equivalent thermal kWh
    cooking_gas_equiv_kwh    DOUBLE PRECISION,  -- gas-equivalent thermal kWh
    ev_kwh                   DOUBLE PRECISION,  -- 0 in baseline
    base_load_kwh            DOUBLE PRECISION,
    PRIMARY KEY (ts)
);
SELECT create_hypertable('load_decomposition', 'ts', if_not_exists => TRUE);

-- ----------------------------------------------------------------------------
-- Quarterly gas bills (entered manually on Data screen, or seeded synthetic)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS gas_bills (
    id              SERIAL PRIMARY KEY,
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    consumption_mj  DOUBLE PRECISION NOT NULL,
    cost_aud        DOUBLE PRECISION,
    note            TEXT
);

-- ----------------------------------------------------------------------------
-- Scenarios — user-defined upgrade combinations
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scenarios (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    description   TEXT,
    upgrades      JSONB NOT NULL,        -- {"hphws":bool,"induction":bool,"ev":bool,"solar":bool}
    device_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    dispatch      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scenario_results (
    scenario_id            INT PRIMARY KEY REFERENCES scenarios(id) ON DELETE CASCADE,
    capex_gross_aud        DOUBLE PRECISION NOT NULL,
    capex_net_aud          DOUBLE PRECISION NOT NULL,
    annual_saving_aud      DOUBLE PRECISION NOT NULL,
    payback_years          DOUBLE PRECISION,
    annual_co2_saving_kg   DOUBLE PRECISION NOT NULL,
    embodied_co2_kg        DOUBLE PRECISION NOT NULL,
    carbon_payback_years   DOUBLE PRECISION,
    cumulative_savings     JSONB NOT NULL,        -- 20 yr cumulative net savings curve
    breakdown              JSONB NOT NULL,        -- per-upgrade contribution detail
    computed_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------------------
-- Editable assumptions / config (NFR-06 transparency)
-- Stored as a single key/value bag so the Assumptions screen can render and
-- save anything without schema migrations.
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS assumptions (
    key         TEXT PRIMARY KEY,
    category    TEXT NOT NULL,        -- 'cost' | 'rebate' | 'panel' | 'emission' | 'dispatch' | 'site' | 'usage'
    label       TEXT NOT NULL,
    value       JSONB NOT NULL,
    unit        TEXT,
    description TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------------------
-- Data ingestion audit log — what was uploaded, when, validation result
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ingestion_runs (
    id           SERIAL PRIMARY KEY,
    source       TEXT NOT NULL,        -- 'enphase_solar' | 'amber_consumption' | 'gas_bill' | 'synthetic'
    filename     TEXT,
    rows_in      INT,
    rows_loaded  INT,
    gaps_found   INT,
    duplicates   INT,
    implausible  INT,
    range_start  TIMESTAMPTZ,
    range_end    TIMESTAMPTZ,
    status       TEXT NOT NULL,        -- 'ok' | 'partial' | 'error'
    message      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
