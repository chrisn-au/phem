package http

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/assumptions"
	"github.com/chrisneave/phem/api/internal/baseline"
	"github.com/chrisneave/phem/api/internal/scenarios"
)

type Deps struct {
	Pool        *pgxpool.Pool
	Baseline    *baseline.Service
	Scenarios   *scenarios.Engine
	Assumptions *assumptions.Store
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/healthz", healthz)

	// Baseline (read-only summaries)
	mux.HandleFunc("GET /api/baseline/summary", baselineSummary(d))
	mux.HandleFunc("GET /api/baseline/daily", baselineDaily(d))
	mux.HandleFunc("GET /api/baseline/hourly", baselineHourly(d))
	mux.HandleFunc("GET /api/baseline/monthly", baselineMonthly(d))

	// Data (intervals + ingestion)
	mux.HandleFunc("GET /api/data/intervals", dataIntervals(d))
	mux.HandleFunc("GET /api/data/health", dataHealth(d))
	mux.HandleFunc("GET /api/data/gas-bills", listGasBills(d))
	mux.HandleFunc("POST /api/data/gas-bills", createGasBill(d))
	mux.HandleFunc("POST /api/data/upload/solar", uploadSolar(d))
	mux.HandleFunc("POST /api/data/upload/consumption", uploadConsumption(d))

	// Scenarios
	mux.HandleFunc("GET /api/scenarios", listScenarios(d))
	mux.HandleFunc("POST /api/scenarios", upsertScenario(d))
	mux.HandleFunc("DELETE /api/scenarios/{id}", deleteScenario(d))
	mux.HandleFunc("POST /api/scenarios/{id}/compute", computeScenario(d))
	mux.HandleFunc("POST /api/scenarios/recompute-all", recomputeAll(d))
	mux.HandleFunc("GET /api/scenarios/explore", exploreScenarios(d))
	mux.HandleFunc("GET /api/scenarios/export.csv", exportScenariosCSV(d))

	// Assumptions
	mux.HandleFunc("GET /api/assumptions", listAssumptions(d))
	mux.HandleFunc("PUT /api/assumptions/{key}", updateAssumption(d))

	// Solar shading matrix (debug / advanced view)
	mux.HandleFunc("GET /api/solar/shading", solarShading(d))

	return Logger(CORS(mux))
}

func healthz(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, 200, map[string]string{"status": "ok"})
}
