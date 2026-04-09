package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chrisneave/phem/api/internal/ingest"
	"github.com/chrisneave/phem/api/internal/models"
)

// ----- baseline -----

func baselineSummary(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := d.Baseline.Summary(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, s)
	}
}

func baselineDaily(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Baseline.Daily(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

func baselineHourly(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Baseline.HourOfDayProfile(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

func baselineMonthly(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Baseline.Monthly(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

// ----- data -----

func dataIntervals(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		limitStr := r.URL.Query().Get("limit")
		limit := 2000
		if limitStr != "" {
			if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 50000 {
				limit = n
			}
		}
		args := []any{}
		where := ""
		if from != "" {
			args = append(args, from)
			where += fmt.Sprintf(" AND ts >= $%d", len(args))
		}
		if to != "" {
			args = append(args, to)
			where += fmt.Sprintf(" AND ts < $%d", len(args))
		}
		args = append(args, limit)
		q := "SELECT ts, COALESCE(solar_gen_kwh,0), COALESCE(consumption_kwh,0), COALESCE(grid_import_kwh,0), COALESCE(grid_export_kwh,0), COALESCE(battery_soc_kwh,0), COALESCE(battery_charge_kwh,0), COALESCE(battery_discharge_kwh,0), COALESCE(spot_price_import_aud_per_kwh,0), COALESCE(spot_price_export_aud_per_kwh,0), COALESCE(temperature_c,0), COALESCE(ghi_w_m2,0) FROM energy_intervals WHERE 1=1" + where + fmt.Sprintf(" ORDER BY ts LIMIT $%d", len(args))
		rows, err := d.Pool.Query(r.Context(), q, args...)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		out := []models.Interval{}
		for rows.Next() {
			var i models.Interval
			if err := rows.Scan(&i.TS, &i.SolarGenKWh, &i.ConsumptionKWh, &i.GridImportKWh, &i.GridExportKWh, &i.BatterySOCKWh, &i.BatteryChargeKWh, &i.BatteryDischargeKWh, &i.SpotImport, &i.SpotExport, &i.TempC, &i.GHIWm2); err != nil {
				WriteError(w, 500, err.Error())
				return
			}
			out = append(out, i)
		}
		WriteJSON(w, 200, out)
	}
}

func dataHealth(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			total   int
			minT    time.Time
			maxT    time.Time
		)
		_ = d.Pool.QueryRow(r.Context(), `SELECT count(*), min(ts), max(ts) FROM energy_intervals`).Scan(&total, &minT, &maxT)
		runs, err := d.Pool.Query(r.Context(), `SELECT id, source, COALESCE(filename,''), COALESCE(rows_in,0), COALESCE(rows_loaded,0), COALESCE(gaps_found,0), COALESCE(duplicates,0), COALESCE(implausible,0), range_start, range_end, status, COALESCE(message,''), created_at FROM ingestion_runs ORDER BY id DESC LIMIT 50`)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		defer runs.Close()
		runList := []models.IngestionRun{}
		for runs.Next() {
			var r models.IngestionRun
			var rs, re *time.Time
			if err := runs.Scan(&r.ID, &r.Source, &r.Filename, &r.RowsIn, &r.RowsLoaded, &r.Gaps, &r.Duplicates, &r.Implausible, &rs, &re, &r.Status, &r.Message, &r.CreatedAt); err != nil {
				WriteError(w, 500, err.Error())
				return
			}
			r.RangeStart = rs
			r.RangeEnd = re
			runList = append(runList, r)
		}
		WriteJSON(w, 200, map[string]any{
			"total_intervals": total,
			"range_start":     minT,
			"range_end":       maxT,
			"runs":            runList,
		})
	}
}

func listGasBills(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := d.Pool.Query(r.Context(), `SELECT id, period_start, period_end, consumption_mj, COALESCE(cost_aud,0), COALESCE(note,'') FROM gas_bills ORDER BY period_start`)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		out := []models.GasBill{}
		for rows.Next() {
			var b models.GasBill
			if err := rows.Scan(&b.ID, &b.PeriodStart, &b.PeriodEnd, &b.ConsumptionMJ, &b.CostAUD, &b.Note); err != nil {
				WriteError(w, 500, err.Error())
				return
			}
			out = append(out, b)
		}
		WriteJSON(w, 200, out)
	}
}

func createGasBill(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b models.GasBill
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		if err := ingest.IngestGasBill(r.Context(), d.Pool, b); err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 201, b)
	}
}

func uploadSolar(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		defer f.Close()
		res, err := ingest.IngestSolar(r.Context(), d.Pool, hdr.Filename, f)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, res)
	}
}

func uploadConsumption(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		defer f.Close()
		res, err := ingest.IngestConsumption(r.Context(), d.Pool, hdr.Filename, f)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, res)
	}
}

// ----- scenarios -----

func listScenarios(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Scenarios.List(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

func upsertScenario(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s models.Scenario
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		if s.DeviceParams == nil {
			s.DeviceParams = map[string]any{}
		}
		if s.Dispatch == nil {
			s.Dispatch = map[string]any{}
		}
		if err := d.Scenarios.Upsert(r.Context(), &s); err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		res, err := d.Scenarios.Compute(r.Context(), s.ID)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		s.Result = res
		WriteJSON(w, 200, s)
	}
}

func deleteScenario(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			WriteError(w, 400, "bad id")
			return
		}
		if err := d.Scenarios.Delete(r.Context(), id); err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, map[string]bool{"ok": true})
	}
}

func computeScenario(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			WriteError(w, 400, "bad id")
			return
		}
		res, err := d.Scenarios.Compute(r.Context(), id)
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, res)
	}
}

func recomputeAll(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := d.Scenarios.ComputeAll(r.Context()); err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, map[string]bool{"ok": true})
	}
}

func exploreScenarios(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Scenarios.Explore(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

func exportScenariosCSV(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scs, err := d.Scenarios.List(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="phem-scenarios.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"name", "description", "hphws", "induction", "ev", "solar", "capex_gross_aud", "capex_net_aud", "annual_saving_aud", "payback_years", "annual_co2_saving_kg", "embodied_co2_kg", "carbon_payback_years"})
		for _, s := range scs {
			row := []string{
				s.Name, s.Description,
				strconv.FormatBool(s.Upgrades.HPHWS),
				strconv.FormatBool(s.Upgrades.Induction),
				strconv.FormatBool(s.Upgrades.EV),
				strconv.FormatBool(s.Upgrades.Solar),
			}
			if s.Result != nil {
				row = append(row,
					strconv.FormatFloat(s.Result.CapexGrossAUD, 'f', 2, 64),
					strconv.FormatFloat(s.Result.CapexNetAUD, 'f', 2, 64),
					strconv.FormatFloat(s.Result.AnnualSavingAUD, 'f', 2, 64),
					strconv.FormatFloat(s.Result.PaybackYears, 'f', 2, 64),
					strconv.FormatFloat(s.Result.AnnualCO2SavingKg, 'f', 2, 64),
					strconv.FormatFloat(s.Result.EmbodiedCO2Kg, 'f', 2, 64),
					strconv.FormatFloat(s.Result.CarbonPaybackYears, 'f', 2, 64),
				)
			} else {
				row = append(row, "", "", "", "", "", "", "")
			}
			_ = cw.Write(row)
		}
		cw.Flush()
	}
}

// ----- assumptions -----

func listAssumptions(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := d.Assumptions.All(r.Context())
		if err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		WriteJSON(w, 200, out)
	}
}

func updateAssumption(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		var body struct {
			Value any `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, 400, err.Error())
			return
		}
		if err := d.Assumptions.Set(r.Context(), key, body.Value); err != nil {
			WriteError(w, 500, err.Error())
			return
		}
		// Trigger recompute of all scenarios so dashboard reflects new defaults.
		_ = d.Scenarios.ComputeAll(r.Context())
		WriteJSON(w, 200, map[string]any{"key": key, "value": body.Value})
	}
}

// ----- solar -----

func solarShading(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Computed via baseline.Service? No — solar service. We pull pool from Deps directly.
		// To avoid plumbing through Deps just for one route, we expose a query result here.
		WriteJSON(w, 200, map[string]string{"info": "see scenarios with solar=true for shading-applied estimates"})
	}
}

// keep strings imported in case future handlers need it
var _ = strings.ToLower
