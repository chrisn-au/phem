# Personalised Home Energy Model
## Product Requirements Document
**Version 1.0 | April 2026**
**Owner: Residential Electrification Project | NSW, Australia**
**Status: DRAFT — FOR ENGINEERING REVIEW**

---

## Table of Contents

1. [Purpose and Scope](#1-purpose-and-scope)
2. [Goals and Success Criteria](#2-goals-and-success-criteria)
3. [Users](#3-users)
4. [Data Inputs](#4-data-inputs)
5. [Functional Requirements](#5-functional-requirements)
6. [Non-Functional Requirements](#6-non-functional-requirements)
7. [Technical Architecture Guidance](#7-technical-architecture-guidance)
8. [Dashboard Requirements](#8-dashboard-requirements)
9. [Default Cost and Rebate Assumptions](#9-default-cost-and-rebate-assumptions)
10. [Constraints and Boundary Conditions](#10-constraints-and-boundary-conditions)
11. [Carbon Methodology](#11-carbon-methodology)
12. [Phased Delivery Plan](#12-phased-delivery-plan)
13. [Open Questions and Assumptions Log](#13-open-questions-and-assumptions-log)
14. [Glossary](#14-glossary)

---

## 1. Purpose and Scope

### 1.1 Purpose

This document defines the requirements for a Personalised Home Energy Model (PHEM) — a local desktop application that enables a homeowner to evaluate the financial and environmental case for electrifying residential energy consumption and transport.

The primary user is a civil engineer with no software development background. The tool must be operable via a visual dashboard without editing configuration files or running terminal commands.

### 1.2 Project Background

The subject property is a home in New South Wales, Australia, with the following existing installed infrastructure:

- 5 kW rooftop solar (17 × Trina 295 W panels, Enphase micro-inverters)
- 43 kWh battery (Amber Electric automated dispatch, recently commissioned)
- 9 kW reverse-cycle air-conditioner — primary winter heating source
- Large swimming pool with Astral Pools pump on a timer; heat pump installed but unused
- Instantaneous natural gas hot water and gas cooktop; electric oven
- Wallbox EV charger (32 A max) already installed; no EV currently
- Electricity retailed via Amber Electric with wholesale spot price exposure (import and export)

### 1.3 Scope

PHEM shall support analysis of four electrification upgrades, individually and in combination:

- Heat pump hot water system (HPHWS) — replacing instantaneous gas hot water
- Induction cooktop — replacing gas cooktop
- Electric vehicle (EV) — replacing petrol Mazda CX5 (~8,000 km/year)
- Solar array upgrade — replacing existing 17-panel array with 14–15 higher-efficiency panels

The tool shall:

- Ingest and process all available historical data (solar production, whole-home consumption, gas bills)
- Model baseline and upgrade-scenario energy flows at 15-minute resolution
- Calculate simple financial payback and carbon payback for each scenario
- Present results in a visual, interactive dashboard
- Run fully offline on an Apple M2 Mac Mini with 16 GB RAM

**Out of scope for Phase 1 (reserved for Phase 2):**

- Live price-responsive scheduling using weather and NEM price forecasts
- Integration with live Amber API or Enphase API
- Multi-property modelling

---

## 2. Goals and Success Criteria

### 2.1 Primary Goals

- **Financial:** Identify which combination of electrification upgrades achieves simple payback within 5–7 years under realistic NSW energy pricing.
- **Carbon:** Quantify the carbon payback period for each upgrade, noting that low-mileage vehicle replacement has a higher carbon payback threshold to clear.
- **Transparency:** All model assumptions must be visible and adjustable by the user so that sensitivity can be tested without engineering assistance.

### 2.2 Success Criteria

| Criterion | Definition of Done |
|---|---|
| Data ingestion | All three historical data sources load without manual reformatting |
| Baseline accuracy | Modelled annual electricity spend within ±10% of actual Amber bills for the 16-month data window |
| Scenario comparison | User can compare up to 8 scenarios side-by-side with total capex, annual saving, and payback period |
| Interactivity | Toggling an upgrade component updates all results within 5 seconds |
| Hardware compliance | Peak RAM usage does not exceed 12 GB; application runs on M2 Mac Mini with 16 GB RAM |
| Carbon reporting | Each scenario reports kg CO₂-e saved per year and years to carbon payback |

---

## 3. Users

PHEM has a single primary user. The following profile captures relevant constraints for UX and engineering decisions:

| Attribute | Detail |
|---|---|
| Role | Homeowner and project decision-maker |
| Technical background | Civil engineer; comfortable with data and spreadsheets; no software development experience |
| Interface preference | Visual dashboard with toggles; does not want to edit config files or run terminal commands |
| Hardware | Apple M2 Mac Mini, 16 GB RAM, macOS |
| Data literacy | Understands energy units (kWh, MWh), financial metrics (payback), and basic carbon concepts |
| Future intent | Phase 2: weekly energy scheduling using weather and NEM price forecasts |

---

## 4. Data Inputs

### 4.1 Historical Data Sources

| Dataset | Period | Resolution | Format / Source |
|---|---|---|---|
| Solar production | 10 years | 15-minute | CSV export from Enphase Enlighten |
| Whole-home consumption | 16 months | 15-minute | CSV export from Amber Electric or Enphase; meter-level, includes all loads |
| Gas consumption | 16 months (4 quarterly bills) | Quarterly totals | Utility bill data (MJ or kWh equivalent) |
| Amber tariff structure | Current | Static | User-provided: supply charge, price cap, export pricing rules |

> **Assumption:** Gas split between hot water and cooking is unknown. The model shall default to 80% hot water / 20% cooking, adjustable by the user.

> **Assumption:** Gas seasonal distribution shall be inferred by the model from solar and consumption data seasonality (higher winter consumption correlates with heating and hot water demand).

### 4.2 Reference / External Data

- **Bureau of Meteorology (BOM) or Open-Meteo API** — historical hourly weather data (temperature, solar irradiance) for model calibration. Used offline after initial download.
- **NEM historical price data (AEMO)** — 5-minute or 30-minute spot prices for the NSW1 region, for the same window as consumption data. Used to validate Amber bill modelling.
- **NSW grid emissions intensity** — average kg CO₂-e per kWh (drawn from Australian Government or AEMO published factors).
- **Petrol emissions factor** — kg CO₂-e per litre (Australian standard factor).

### 4.3 User-Configurable Parameters

All parameters below shall be editable via the dashboard. Default values shall be pre-populated.

| Parameter | Default Value | Notes |
|---|---|---|
| Gas hot water fraction | 80% | Share of gas consumption attributed to hot water |
| Annual vehicle kilometres | 8,000 km | Matches current CX5 usage |
| EV efficiency | Parametric (e.g. 16 kWh/100km) | User selects from EV shortlist or inputs manually |
| EV battery capacity | Parametric (e.g. 60–82 kWh) | Drives charging load profile |
| HPHWS tank size | 250 L (4-bedroom default) | Adjustable; heat pump COP varies by ambient temp |
| HPHWS daily hot water usage | Parametric (L/day) | Default: 200 L/day for 2 occupants in 4-bed sizing |
| Solar panel option | Standard (e.g. Trina ~400 W) or Premium (e.g. AIKO ~440 W+) | Selectable; drives yield estimate |
| Panel count (upgrade) | 14 or 15 | Constrained by roof space and legacy panel footprint |
| Battery dispatch heuristic | Charge below X c/kWh, discharge above Y c/kWh | User sets thresholds; defaults based on Amber typical range |
| EV payback basis | Fuel saving only (default) | Toggle to include full vehicle replacement cost |

---

## 5. Functional Requirements

### 5.1 Data Ingestion Module

- **FR-01:** The system shall accept Enphase CSV exports for solar production (15-min) without any pre-processing by the user.
- **FR-02:** The system shall accept Amber Electric or Enphase CSV exports for whole-home consumption (15-min) without pre-processing.
- **FR-03:** The system shall accept manual entry of quarterly gas bill totals (MJ or kWh) via a form in the dashboard.
- **FR-04:** The system shall validate ingested data for gaps, duplicates, and implausible values, and report any issues to the user before proceeding.
- **FR-05:** The system shall automatically align all data series to a common UTC timestamp index.
- **FR-06:** Following successful ingestion, the system shall persist a processed data cache so that re-loading raw CSVs is not required on subsequent sessions.

### 5.2 Baseline Model

- **FR-07:** The system shall construct a 15-minute resolution baseline energy model covering the full 16-month consumption data window.
- **FR-08:** The baseline shall decompose whole-home consumption into estimated load categories: HVAC, pool pump, hot water (gas equivalent), cooking (gas equivalent), EV charging (currently zero), and residual / base load.
- **FR-09:** Load decomposition shall use a combination of time-of-use patterns, seasonal temperature correlations, and known device schedules (e.g. pool pump timer).
- **FR-10:** The system shall model battery dispatch using the user-configured heuristic (charge below threshold, discharge above threshold) applied to historical spot prices.
- **FR-11:** The baseline model shall calculate total annual electricity cost and gas cost, and display these against actual bill data for validation.
- **FR-12:** The baseline model shall estimate annual carbon emissions from: (a) net grid electricity import, (b) gas consumption, and (c) vehicle petrol use.

### 5.3 Solar Shading and Yield Estimation

- **FR-13:** The system shall estimate the shading loss profile of the existing array by comparing actual 15-minute production data against a clear-sky irradiance model (using site latitude/longitude and panel orientation).
- **FR-14:** The derived shading profile shall be applied to modelled yield estimates for the upgraded solar array.
- **FR-15:** The system shall support two upgrade panel specifications (Standard and Premium), parameterised by: rated power (W), efficiency (%), temperature coefficient, and dimensions. Defaults shall be pre-loaded for a representative Trina and AIKO panel.
- **FR-16:** Upgraded array yield shall be calculated for 14-panel and 15-panel configurations, using the same Enphase micro-inverter architecture (i.e. per-panel optimisation assumed).

### 5.4 Upgrade Scenario Modules

#### 5.4.1 Heat Pump Hot Water System

- **FR-17:** The system shall model HPHWS electricity consumption as a function of daily hot water demand (L), tank size (L), inlet water temperature (seasonal), and heat pump COP (which varies with ambient air temperature).
- **FR-18:** The model shall assume the HPHWS is installed outdoors or in a garage, with ambient temperature drawn from the historical weather dataset.
- **FR-19:** The model shall simulate preferential scheduling of the HPHWS during solar surplus windows or low spot-price windows, configurable by the user.
- **FR-20:** The model shall calculate: annual electricity consumption of HPHWS, annual gas offset, and net change in electricity cost.

#### 5.4.2 Induction Cooktop

- **FR-21:** The system shall model induction cooktop electricity consumption based on the gas cooking fraction (default 20% of gas bill), converted to electricity using a gas-to-induction efficiency ratio (default: induction is ~3× more efficient per unit of energy delivered).
- **FR-22:** Cooking load shall be treated as non-discretionary and distributed across typical meal-time windows (morning, midday, evening).

#### 5.4.3 Electric Vehicle

- **FR-23:** The system shall model EV charging load from user-configured parameters: annual kilometres, EV efficiency (kWh/100km), battery capacity, and charger power (Wallbox, max 7.4 kW on 32 A single-phase).
- **FR-24:** The model shall support two charging strategies: (a) Dumb charging — plug in evening, charge at full rate until full; (b) Smart charging — charge preferentially during solar surplus or low spot-price windows, within an overnight window.
- **FR-25:** The model shall calculate annual fuel cost saving (petrol replaced by electricity) and net change in electricity cost.
- **FR-26:** The model shall calculate annual carbon saving from EV adoption, accounting for both avoided petrol emissions and additional grid electricity emissions.

#### 5.4.4 Solar Array Upgrade

- **FR-27:** The system shall model the upgraded solar yield (14 or 15 panels × selected panel spec) using the shading profile derived in FR-13/FR-14, applied to the 10-year historical irradiance data.
- **FR-28:** The model shall calculate incremental generation versus the existing 5 kW system, and the financial value of that increment split between self-consumption and export.

### 5.5 Scenario Comparison Engine

- **FR-29:** The system shall support up to 8 named user-defined scenarios. Each scenario is defined by toggling any combination of the four upgrade components on or off.
- **FR-30:** For each scenario, the system shall calculate and display: total upfront capex (gross and net of NSW rebates), annual energy cost saving vs baseline, simple payback period (years), annual CO₂-e saving (kg), and carbon payback period (years).
- **FR-31:** The system shall display a scenario comparison table with all active scenarios side-by-side.
- **FR-32:** The system shall display a payback chart showing cumulative net saving over time (0–20 years) for all active scenarios on a single interactive line chart. A horizontal reference line at zero shall mark the break-even point.
- **FR-33:** The user shall be able to export the comparison table as a CSV file with one click.

### 5.6 Capex and Rebate Module

- **FR-34:** The system shall maintain a default cost database for each upgrade component, sourced from representative NSW market pricing at time of development. All defaults shall be user-overridable.
- **FR-35:** The system shall apply applicable NSW and federal rebates and incentives automatically, with a disclosure of which rebates are applied and their assumed values.

> **Note:** Rebate values to be confirmed against current NSW Energy Savings Scheme (ESS), Small-scale Technology Certificates (STCs) for solar, and any applicable EV incentives at time of engineering implementation. These change frequently; the model must make rebate values visible and editable.

### 5.7 Demand Response / Price Dispatch

- **FR-36:** Battery dispatch shall be modelled using the heuristic defined in Section 4.3 (charge below X, discharge above Y). Thresholds shall be configurable.
- **FR-37:** HPHWS and EV smart charging shall use a simple rule: operate during periods where spot price is below a user-defined threshold, subject to a daily must-complete constraint (e.g. hot water must be heated by 07:00; EV must be charged by 07:00).
- **FR-38:** Pool pump scheduling optimisation is deferred to Phase 2. The Phase 1 model shall treat the pool pump as operating on its existing timer schedule. The data model shall be designed to support dispatch optimisation in Phase 2 without structural changes.
- **FR-39:** Panasonic air-conditioner automated dispatch is deferred to Phase 2. Heating loads shall be treated as non-discretionary in Phase 1, except during spot price spike events above a user-configurable curtailment threshold.

---

## 6. Non-Functional Requirements

| ID | Category | Requirement |
|---|---|---|
| NFR-01 | Performance | Full scenario recalculation shall complete within 10 seconds on the target hardware (M2 Mac Mini, 16 GB RAM) |
| NFR-02 | Memory | Peak RAM usage shall not exceed 12 GB |
| NFR-03 | Platform | The application shall run on macOS (Apple Silicon) without requiring a paid software licence |
| NFR-04 | Offline operation | After initial data download (weather, NEM prices), all processing shall run fully offline |
| NFR-05 | Usability | A user with no software background shall be able to load data, configure a scenario, and read results without consulting documentation |
| NFR-06 | Transparency | Every model assumption shall be visible in the dashboard — no black-box parameters |
| NFR-07 | Maintainability | Rebate values, panel specs, and cost defaults shall be stored in a plain-text configuration file (JSON or YAML) that a non-developer can edit with a text editor |
| NFR-08 | Phase 2 readiness | The data pipeline and model output schema shall be designed to support live weather and NEM price forecast ingestion in Phase 2 without re-architecture |
| NFR-09 | Accuracy | Modelled annual electricity spend shall be within ±10% of actual billing data for the 16-month validation window |

---

## 7. Technical Architecture Guidance

This section provides guidance for the engineering team. Final technology choices are at the discretion of the implementing engineer, subject to the constraints in Section 6.

### 7.1 Stack (as built)

The implementation uses a dockerised, multi-service stack rather than a
single-process Python app. This is a deliberate departure from earlier
Streamlit-flavoured drafts: a real database and a typed API surface make the
Phase 2 forecast / dispatch work additive rather than a rewrite, and the
docker-compose stack runs identically on the target M2 Mac Mini and on any
linux box for collaborator review.

- **Database:** PostgreSQL 16 with the **TimescaleDB** extension. 15-minute timeseries are stored in hypertables; assumptions and scenarios live in regular relational tables. Postgres is exposed on host port `12456` for inspection via DBeaver / psql. All energy quantities are persisted as kWh per 15-min interval; all money is AUD; all temperatures are °C.
- **Backend language:** **Go 1.22** (`net/http` with the 1.22 method-aware ServeMux + `pgx/v5` for Postgres). No web framework — the standard library is sufficient at this scale and minimises supply chain surface area.
- **Migrations:** plain `*.sql` files embedded into the Go binary via `go:embed` and run on startup. No external migration tool.
- **Frontend:** **Next.js 14** (App Router) + TypeScript + Tailwind + shadcn/ui primitives + Recharts for charts. Output is a standalone Node server in a multi-stage Docker build.
- **Configuration:** all editable assumptions live in a `assumptions` table (one row per parameter, JSONB value) so the Assumptions screen can render and persist changes without schema migrations or YAML files (NFR-06: every assumption visible and editable in the UI).
- **Solar modelling:** implemented in-house in Go (Haurwitz clear-sky GHI + simple solar position) — no `pvlib-python`. The shading derivation is described in §7.4.
- **Orchestration:** `docker compose` with three services — `timescale`, `api`, `web`. Browser only talks to Next.js (`:3000`); Next.js rewrites `/api/*` to the Go service on the docker network.

### 7.2 Data Pipeline Architecture

- **Stage 1 — Ingest:** Enphase / Amber CSV uploads are parsed and validated by the Go ingest module (`internal/ingest`) and upserted into the `energy_intervals` hypertable. Gas bill totals are inserted into the `gas_bills` table via the dashboard form. Every ingestion run is recorded in `ingestion_runs` for audit. On first boot a deterministic synthetic generator (`internal/seed`) populates 16 months of data so the dashboard renders end-to-end without real CSVs.
- **Stage 2 — Enrich:** External reference data (weather, NEM prices, emissions factors) is joined into the same `energy_intervals` row at ingest time. The hypertable schema includes nullable forecast columns so Phase 2 weather / NEM forecasts can be backfilled without a schema migration.
- **Stage 3 — Model:** The baseline service (`internal/baseline`) issues SQL aggregates against `energy_intervals` + `load_decomposition` to build per-day, per-hour, and per-month rollups. The scenario engine (`internal/scenarios`) runs the four upgrade modules against an in-memory baseline summary plus an assumptions snapshot, persists results to `scenario_results`, and exposes a brute-force `Explore` method that evaluates all 16 upgrade combinations for ranking.
- **Stage 4 — Present:** The Next.js dashboard reads from the Go API. Saving any assumption automatically triggers a recompute of every saved scenario so the UI is always live; no separate refresh step is required.

### 7.3 Key Data Structures

The following structures shall be defined and documented in the codebase, forming the contract between pipeline stages and supporting Phase 2 extension:

**TimeseriesRecord**
```
timestamp (UTC)
solar_gen_kw
consumption_kw
spot_price_aud_kwh
battery_soc_kwh
battery_dispatch_kw
temperature_c
forecast_solar_gen_kw        # nullable — Phase 2
forecast_spot_price_aud_kwh  # nullable — Phase 2
```

**ScenarioConfig**
```
name
upgrades_enabled: dict[str, bool]
device_params: dict[str, dict]
dispatch_thresholds: dict[str, float]
```

**ScenarioResult**
```
scenario_name
capex_gross_aud
capex_net_rebates_aud
annual_saving_aud
payback_years
annual_co2_saving_kg
carbon_payback_years
```

### 7.4 Solar Shading Model

- Theoretical clear-sky GHI is calculated at site coordinates for every 15-minute interval using the Haurwitz model (`internal/solar`). The implementation is in-house Go — no `pvlib-python` dependency.
- Actual production from `energy_intervals.solar_gen_kwh` is compared to theoretical to derive a per-cell efficiency factor in a `[12 month][24 hour]` shading matrix.
- Future work: apply a rolling percentile filter (e.g. P90 over a 30-day window) to separate persistent shading from transient cloud cover. The Phase 1 implementation uses a straight ratio per cell.
- The resulting shading matrix is applied to upgrade scenario yield calculations via `Service.EstimateAnnualYield`.

### 7.5 Phase 2 Integration Points

The following interfaces shall be designed into Phase 1 architecture without being implemented:

- A `ForecastProvider` interface (abstract class) that accepts a date range and returns weather and spot price forecasts in the same schema as historical data
- A `DispatchOptimiser` interface that accepts a 7-day forecast and returns a recommended dispatch schedule for battery, HPHWS, EV, and pool pump
- The `TimeseriesRecord` structure shall include `forecast_solar_gen_kw` and `forecast_spot_price_aud_kwh` as nullable fields

---

## 8. Dashboard Requirements

### 8.1 Screen Structure

The dashboard shall be organised into four screens accessible via a top navigation bar:

| Screen | Purpose |
|---|---|
| 1. Data | Load and validate raw data files; view data health summary; enter gas bills; view baseline energy flow summary |
| 2. Baseline | View decomposed consumption, solar production, battery behaviour, and modelled vs actual costs for the historical period |
| 3. Scenarios | Define and compare upgrade scenarios; toggle components; adjust parameters; view payback and carbon results |
| 4. Assumptions | View and edit all model assumptions: cost defaults, rebates, panel specs, emissions factors, dispatch thresholds |

### 8.2 Scenarios Screen — Detailed Requirements

- The user shall be able to create a new scenario by entering a name and toggling any combination of the four upgrade components using on/off switches
- Each upgrade component, when toggled on, shall reveal a parameter panel with relevant device parameters (pre-populated with defaults)
- The scenario comparison table shall update within 5 seconds of any parameter change
- The payback chart shall show cumulative net saving (AUD) over 20 years for all active scenarios on a single Plotly line chart, with a horizontal reference line at zero marking the break-even point
- A carbon comparison panel shall show annual CO₂-e saving and carbon payback for each scenario
- The user shall be able to export the comparison table as a CSV file with one click

### 8.3 Data Screen — Detailed Requirements

- A file upload widget shall accept CSV files for solar and consumption data
- After upload, the system shall display: date range, record count, missing interval count, and a thumbnail time-series chart
- A manual entry form shall allow the user to enter up to 8 quarterly gas bills (date, MJ total or kWh equivalent)
- A "Run Baseline Model" button shall trigger Stage 2 and Stage 3 processing, with a progress bar

---

## 9. Default Cost and Rebate Assumptions

All values must be verified at implementation time and stored in an editable configuration file — not hard-coded.

| Item | Gross Cost (AUD) | Net of Rebates (AUD) | Rebate Notes |
|---|---|---|---|
| Heat pump hot water (250–315 L) | $2,500–$4,000 | ~$1,000–$2,500 | NSW ESS certificates + federal rebate (varies by COP and capacity) |
| Induction cooktop (supply + install) | $800–$1,800 | $800–$1,800 | No current rebate; gas disconnection cost may apply |
| Solar array upgrade (14–15 panels + labour) | $3,500–$6,000 | $2,000–$4,000 | STCs reduce cost; value depends on zone and deeming period |
| Electric vehicle (mid-range SUV, e.g. BYD Atto 3 / Ioniq 5) | $45,000–$65,000 | $43,000–$63,000 | NSW EV rebate phased out; confirm current status at implementation |
| EV charger (already installed) | $0 | $0 | Sunk cost — not included in payback calculation |

> **Note:** The EV payback model shall default to fuel-saving basis only (not total vehicle cost). A clearly labelled toggle shall allow the user to include full vehicle replacement cost vs. petrol equivalent.

> **Note:** NSW EV incentives were significantly revised in 2024–2025. Engineering team must confirm current rebate status at implementation time.

---

## 10. Constraints and Boundary Conditions

### 10.1 Physical Constraints

- The solar upgrade is constrained to 14 or 15 panels by available roof space and the legacy 1.6 × 1.0 m panel footprint. Newer panels (typically 1.75 × 1.13 m) reduce the count that fits.
- The existing Enphase micro-inverter architecture shall be assumed for the upgrade (one inverter per panel), retaining per-panel shading tolerance.
- The Wallbox EV charger operates at maximum 32 A single-phase = 7.4 kW. The model shall not assume three-phase charging.
- The HPHWS location is assumed to be outdoors or garage-adjacent, exposing it to ambient temperature variation. A fully internal installation would overstate COP — this assumption is flagged in the dashboard.

### 10.2 Operational Constraints

- Heating (reverse-cycle air-conditioner) shall be treated as non-discretionary. The model shall not curtail heating except at a user-configurable spot price spike threshold (default: do not curtail).
- Hot water shall be treated as non-discretionary with respect to daily completion — the tank must reach setpoint temperature before 07:00 regardless of price.
- Pool pump heating is disabled and shall be excluded from the model. Pool pump electricity consumption shall be modelled as fixed per the existing timer schedule.

### 10.3 Modelling Constraints

- The model operates on historical data only in Phase 1. It does not use live or forecast data.
- The consumption data does not sub-meter individual circuits. Load decomposition is statistical, not measured, and carries inherent uncertainty.
- The battery dispatch history prior to Amber automated dispatch commissioning is unavailable. The model shall apply the dispatch heuristic retroactively across the entire 16-month window as a consistent baseline assumption.

---

## 11. Carbon Methodology

### 11.1 Emissions Factors

| Source | Default Factor | Reference |
|---|---|---|
| Grid electricity (NSW) | ~0.79 kg CO₂-e/kWh | AEMO / Australian Government (confirm at implementation) |
| Natural gas | ~0.186 kg CO₂-e/kWh thermal (51.53 kg CO₂-e/GJ) | Australian Government NGER |
| Petrol | 2.31 kg CO₂-e/litre | Australian Government NGER |
| Solar panel manufacturing (embodied) | ~400 kg CO₂-e/panel | Default; adjustable |

### 11.2 Carbon Payback Calculation

For each upgrade scenario:

- **Annual carbon saving** = baseline emissions − scenario emissions (kg CO₂-e/year)
- **Upfront embodied carbon** (where applicable, e.g. new panels, new appliances) shall be estimated from a default database and disclosed in the dashboard
- **Carbon payback period** = upfront embodied carbon (kg) ÷ annual carbon saving (kg/year)

> **Note:** At ~8,000 km/year with a petrol CX5, avoided petrol emissions are modest (~1,400 kg CO₂-e/year). EV manufacturing embodied carbon (~8,000–10,000 kg CO₂-e for battery production) means carbon payback may be 6–8 years. The model shall calculate and display this transparently.

---

## 12. Phased Delivery Plan

| Phase | Scope | Exit Criteria |
|---|---|---|
| 1A | Data ingestion, validation, and baseline model (electricity only; gas modelled but not dispatched) | Modelled annual electricity cost within ±10% of actual bills; dashboard shows baseline energy flows |
| 1B | All four upgrade scenario modules; scenario comparison; payback and carbon calculations | User can compare all scenario combinations; payback chart renders correctly |
| 1C | Capex/rebate module; export to CSV; polish and usability testing with primary user | User can complete full analysis end-to-end without assistance |
| 2 (future) | Live weather and NEM price forecast ingestion; weekly dispatch scheduling; pool pump and AC dispatch | Out of scope for this PRD — Phase 1 architecture must support it |

---

## 13. Open Questions and Assumptions Log

| # | Question / Assumption | Owner | Status |
|---|---|---|---|
| 1 | Confirm current NSW rebate values for HPHWS, solar STCs, and EV at time of build | Engineer | Open |
| 2 | Confirm Amber tariff parameters (supply charge, import cap, export pricing) — user to provide | Owner | Open |
| 3 | Confirm Enphase CSV export column schema (field names, units, timestamp format) | Owner / Engineer | Open |
| 4 | Gas bills: confirm whether stated in MJ or kWh; confirm network distributor | Owner | Open |
| 5 | Assumed gas split 80/20 hot water/cooking — owner to confirm or adjust once model is running | Owner | Assumed |
| 6 | HPHWS location assumed outdoors/garage — owner to confirm at installation planning stage | Owner | Assumed |
| 7 | EV payback toggle defaults to fuel saving only — confirm with owner | Owner | Assumed |
| 8 | NSW grid emissions intensity factor — confirm latest published value from AEMO/Clean Energy Regulator | Engineer | Open |
| 9 | Pool pump power rating (kW) and daily timer schedule — needed for baseline load decomposition | Owner | Open |
| 10 | Site latitude/longitude and roof azimuth/tilt — needed for solar position and irradiance model | Owner | Open |

---

## 14. Glossary

| Term | Definition |
|---|---|
| Amber Electric | Australian electricity retailer providing real-time wholesale spot price exposure with a price cap |
| AEMO | Australian Energy Market Operator — governs the National Electricity Market (NEM) |
| BOM | Australian Bureau of Meteorology |
| COP | Coefficient of Performance — ratio of heat output to electrical energy input for a heat pump |
| ESS | NSW Energy Savings Scheme — provides financial certificates for energy efficiency upgrades |
| GHI | Global Horizontal Irradiance — total solar radiation received on a horizontal surface (W/m²) |
| HPHWS | Heat Pump Hot Water System |
| NEM | National Electricity Market — the interconnected electricity grid covering eastern Australia |
| NGER | National Greenhouse and Energy Reporting — Australian framework for emissions factors |
| PHEM | Personalised Home Energy Model — this application |
| Spot price | Real-time wholesale electricity price set by the NEM dispatch algorithm ($/MWh, quoted as c/kWh) |
| STC | Small-scale Technology Certificate — federal rebate mechanism for solar and heat pump installations |

---

*End of Document — PHEM PRD v1.0 — April 2026*
