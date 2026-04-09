# PHEM — Personalised Home Energy Model

> Local desktop app for evaluating residential electrification upgrades against
> real solar / consumption / gas data, with payback and carbon analysis at
> 15-minute resolution. Built for a NSW homeowner running on Amber Electric.

PHEM models four candidate upgrades — heat pump hot water, induction cooktop,
electric vehicle, and a solar array upgrade — individually and in any
combination, against a baseline derived from 16 months of historical 15-minute
interval data. It produces simple payback (years), 20-year cumulative net
savings, annual CO₂ reduction, and embodied-carbon payback for each scenario,
and provides a brute-force "smart explorer" that ranks all 16 possible upgrade
combinations.

The full product requirements are in [`PHEM_PRD_v1.0.md`](./PHEM_PRD_v1.0.md).

---

## Stack

| Layer        | Tech                                                     |
| ------------ | -------------------------------------------------------- |
| Database     | **PostgreSQL 16 + TimescaleDB** (host port `12456`)      |
| Backend      | **Go 1.22**, `pgx/v5`, embedded SQL migrations           |
| Frontend     | **Next.js 14 (App Router)**, Tailwind, shadcn/ui, Recharts |
| Orchestration | Docker Compose                                          |

The PRD recommends Python + Streamlit. This implementation deliberately uses
Go + Postgres + Next.js for a more durable, dockerised stack that supports
the Phase 2 forecast / dispatch optimisation work without rework.

---

## Quick start

```sh
docker compose up --build
```

Then open <http://localhost:3000>.

On first boot the API generates **16 months of deterministic synthetic
15-minute data** (46 656 intervals + 5 quarterly gas bills + 5 starter
scenarios) so the dashboard renders end-to-end without you having to upload
real CSVs first. To wipe and reseed:

```sh
docker compose down -v && docker compose up --build
```

The Postgres instance is exposed on `localhost:12456` (`phem` / `phem` /
`phem`) so you can poke at it from DBeaver, psql, or any client of choice.

---

## Loading real data

The **Data** screen accepts:

- **Enphase Enlighten** solar production CSVs (15-min intervals)
- **Amber Electric / Enphase** whole-home consumption CSVs (15-min intervals)
- **Quarterly gas bills** entered manually (MJ + AUD)

CSV parsers detect timestamp and energy columns by header name and gracefully
handle Wh / kWh, several timestamp formats, gaps, duplicates, and implausible
readings. Validation results are reported back to the user before commit.

Every ingestion run is recorded in `ingestion_runs` for audit.

---

## Dashboard tour

| Screen          | Purpose                                                                              |
| --------------- | ------------------------------------------------------------------------------------ |
| **Data**        | Upload solar / consumption CSVs, enter gas bills, view ingestion audit log           |
| **Baseline**    | 8 stat cards + monthly load decomposition + hour-of-day profile + daily flows + spot price by hour |
| **Scenarios**   | Smart explorer (16-combo brute force) + scenario CRUD + 20-yr payback chart + per-upgrade contribution |
| **Assumptions** | 35 editable parameters across 7 categories (cost / rebate / panel / dispatch / emission / site / usage) |

### Smart explorer

A single click runs the upgrade engine across all 2⁴ = 16 possible upgrade
combinations against the loaded baseline data, then ranks the results and
tags the winners:

- **Fastest payback** — minimum years to break even
- **Most CO₂ cut** — biggest annual reduction in kg CO₂e
- **Best 20-yr value** — highest cumulative net savings at year 20
- **Cheapest entry** — lowest non-zero capex

Any combination can be saved as a named scenario in one click and then
fine-tuned (panel count, EV efficiency, smart-charge price thresholds, etc).

Because the search space is small the brute force is provably optimal — no
heuristics, no approximations, no LLM round-trips.

---

## Architecture

```
   ┌────────────┐    /api/* rewrite       ┌──────────────┐
   │  Next.js   │ ───────────────────────▶│   Go API     │
   │  :3000     │                         │   :8080      │
   └────────────┘                         └──────┬───────┘
        ▲                                        │ pgx
        │ browser                                ▼
        │                                 ┌──────────────┐
        │                                 │ TimescaleDB  │
        └─────────────────────────────────│  :12456      │
                                          └──────────────┘
```

The browser only ever talks to Next.js (`:3000`); Next.js rewrites `/api/*`
to the Go service inside the docker network. The Go service runs embedded SQL
migrations on startup, seeds synthetic data when the timeseries hypertable is
empty, and computes scenario results on demand.

### Project layout

```
em/
├── PHEM_PRD_v1.0.md            # Product requirements (the source of truth)
├── docker-compose.yml          # 3 services: timescale + api + web
├── README.md                   # this file
├── api/                        # Go backend
│   ├── cmd/server/main.go        # entrypoint, retry-connect, graceful shutdown
│   ├── internal/
│   │   ├── config/               # env config
│   │   ├── db/                   # pgxpool + embedded migrations
│   │   │   └── migrations/*.sql
│   │   ├── models/               # shared structs (mirrored in web/lib/api.ts)
│   │   ├── seed/                 # deterministic synthetic data generator
│   │   ├── ingest/               # CSV parsers (Enphase / Amber / gas)
│   │   ├── baseline/             # baseline annuals + load decomposition queries
│   │   ├── solar/                # clear-sky GHI + per-(month,hour) shading matrix
│   │   ├── scenarios/            # 4 upgrade modules + Compute + Explore engine
│   │   ├── assumptions/          # editable kv config store
│   │   └── http/                 # router, handlers, middleware
│   ├── go.mod
│   └── Dockerfile
└── web/                        # Next.js frontend
    ├── app/
    │   ├── layout.tsx
    │   ├── data/page.tsx
    │   ├── baseline/page.tsx
    │   ├── scenarios/page.tsx
    │   └── assumptions/page.tsx
    ├── components/
    │   ├── nav.tsx
    │   ├── stat-card.tsx
    │   ├── charts.tsx            # Recharts wrappers
    │   └── ui/                   # shadcn-style primitives
    ├── lib/
    │   ├── api.ts                # typed API client (mirrors api/internal/models)
    │   └── utils.ts              # cn() + AUD/kWh/kg/years/% formatters
    ├── package.json
    └── Dockerfile                # multi-stage standalone build
```

---

## API

| Method   | Path                              | Notes                                       |
| -------- | --------------------------------- | ------------------------------------------- |
| `GET`    | `/api/healthz`                    | liveness                                    |
| `GET`    | `/api/baseline/summary`           | annualised totals + cost + carbon           |
| `GET`    | `/api/baseline/daily`             | per-day rollups                             |
| `GET`    | `/api/baseline/hourly`            | hour-of-day average profile                 |
| `GET`    | `/api/baseline/monthly`           | per-month decomposed loads                  |
| `GET`    | `/api/data/intervals?from&to`     | raw 15-min data window                      |
| `GET`    | `/api/data/health`                | total intervals + ingestion runs            |
| `GET`    | `/api/data/gas-bills`             | list quarterly gas bills                    |
| `POST`   | `/api/data/gas-bills`             | add a gas bill                              |
| `POST`   | `/api/data/upload/solar`          | multipart Enphase solar CSV                 |
| `POST`   | `/api/data/upload/consumption`    | multipart Amber/Enphase consumption CSV     |
| `GET`    | `/api/scenarios`                  | list with cached results                    |
| `POST`   | `/api/scenarios`                  | upsert + immediate compute                  |
| `DELETE` | `/api/scenarios/{id}`             | delete scenario                             |
| `POST`   | `/api/scenarios/{id}/compute`     | force recompute                             |
| `POST`   | `/api/scenarios/recompute-all`    | recompute every saved scenario              |
| `GET`    | `/api/scenarios/explore`          | brute-force all 16 upgrade combinations     |
| `GET`    | `/api/scenarios/export.csv`       | comparison table CSV download               |
| `GET`    | `/api/assumptions`                | full kv bag                                 |
| `PUT`    | `/api/assumptions/{key}`          | update one (auto-recomputes scenarios)      |

---

## Database

7 tables, 2 of which are TimescaleDB hypertables:

- `energy_intervals` *(hypertable)* — raw + enriched 15-minute readings
- `load_decomposition` *(hypertable)* — per-load category breakdown
- `gas_bills` — manual quarterly entries
- `scenarios` + `scenario_results` — saved combos and their cached compute output
- `assumptions` — editable kv config (one row per parameter, JSONB value)
- `ingestion_runs` — audit log for CSV uploads / synthetic seed

Schema lives in [`api/internal/db/migrations`](./api/internal/db/migrations)
and is `go:embed`'d into the binary, so the API runs migrations on every
boot — no separate migration tool required.

---

## Configuration

Every model assumption is editable from the **Assumptions** screen and
persists to the `assumptions` table. Categories:

- **Site** — lat / lon / tz / roof azimuth / roof tilt
- **Usage** — gas split, annual km, daily hot water demand
- **Cost** — capex defaults for each upgrade + petrol price + tariff
- **Rebate** — NSW ESS / STC / EV rebate values (verify at install time)
- **Panel** — Standard (Trina-class) and Premium (AIKO-class) specs
- **Dispatch** — battery / smart-load price thresholds
- **Emission** — grid intensity, gas, petrol, embodied-carbon defaults

Saving any assumption automatically triggers a recompute of every saved
scenario so the dashboard reflects the new value immediately.

---

## Phase 2 (out of scope here)

The Phase 1 architecture is designed to extend without rework. Two interfaces
are referenced in the PRD but not implemented:

- `ForecastProvider` — accepts a date range, returns weather + spot price forecasts in the same schema as historical data
- `DispatchOptimiser` — accepts a 7-day forecast and returns a recommended dispatch schedule for battery / HPHWS / EV / pool pump

The `energy_intervals` table already includes nullable forecast columns so
Phase 2 can backfill them without a schema migration.

---

## Development

```sh
# Backend (no docker — needs running postgres)
cd api
go run ./cmd/server

# Frontend (no docker)
cd web
npm install
npm run dev
```

Backend defaults connect to `localhost:5432` if you're not using compose; override
with `PHEM_DB_HOST`, `PHEM_DB_PORT`, etc. (see `api/internal/config/config.go`).
