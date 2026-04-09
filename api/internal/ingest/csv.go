// Package ingest parses CSV uploads from Enphase Enlighten (solar) and
// Amber/Enphase consumption exports, plus accepts manual gas bill entries.
//
// Schemas vary slightly between exporters; the parsers are forgiving — they
// detect the header row, find a timestamp column and a kWh/Wh column by name
// or position, and emit one row per 15-min interval. Validation is reported
// back to the caller as gap/duplicate/implausible counts.
package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chrisneave/phem/api/internal/models"
)

type Result struct {
	Source     string    `json:"source"`
	Filename   string    `json:"filename"`
	RowsIn     int       `json:"rows_in"`
	RowsLoaded int       `json:"rows_loaded"`
	Gaps       int       `json:"gaps_found"`
	Duplicates int       `json:"duplicates"`
	Implausible int      `json:"implausible"`
	RangeStart time.Time `json:"range_start"`
	RangeEnd   time.Time `json:"range_end"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
}

type sample struct {
	ts  time.Time
	val float64
}

// IngestSolar parses an Enphase solar production CSV and writes the values
// into energy_intervals.solar_gen_kwh, replacing any existing rows in range.
func IngestSolar(ctx context.Context, pool *pgxpool.Pool, filename string, r io.Reader) (*Result, error) {
	samples, raw, err := parseTSValueCSV(r)
	if err != nil {
		return nil, err
	}
	res := analyse("enphase_solar", filename, raw, samples)
	if len(samples) == 0 {
		res.Status = "error"
		res.Message = "no data rows"
		return res, nil
	}
	// Upsert into energy_intervals.solar_gen_kwh
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, s := range samples {
		_, err := tx.Exec(ctx, `
			INSERT INTO energy_intervals (ts, solar_gen_kwh)
			VALUES ($1, $2)
			ON CONFLICT (ts) DO UPDATE SET solar_gen_kwh = EXCLUDED.solar_gen_kwh`, s.ts, s.val)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	logRun(ctx, pool, res)
	return res, nil
}

// IngestConsumption parses Amber/Enphase whole-home consumption CSV.
func IngestConsumption(ctx context.Context, pool *pgxpool.Pool, filename string, r io.Reader) (*Result, error) {
	samples, raw, err := parseTSValueCSV(r)
	if err != nil {
		return nil, err
	}
	res := analyse("amber_consumption", filename, raw, samples)
	if len(samples) == 0 {
		res.Status = "error"
		res.Message = "no data rows"
		return res, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	for _, s := range samples {
		_, err := tx.Exec(ctx, `
			INSERT INTO energy_intervals (ts, consumption_kwh)
			VALUES ($1, $2)
			ON CONFLICT (ts) DO UPDATE SET consumption_kwh = EXCLUDED.consumption_kwh`, s.ts, s.val)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	logRun(ctx, pool, res)
	return res, nil
}

// IngestGasBill inserts a manually-entered gas bill record.
func IngestGasBill(ctx context.Context, pool *pgxpool.Pool, b models.GasBill) error {
	_, err := pool.Exec(ctx, `INSERT INTO gas_bills (period_start, period_end, consumption_mj, cost_aud, note) VALUES ($1,$2,$3,$4,$5)`,
		b.PeriodStart, b.PeriodEnd, b.ConsumptionMJ, b.CostAUD, b.Note)
	return err
}

// ----- parsing helpers -----

func parseTSValueCSV(r io.Reader) ([]sample, int, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	out := []sample{}
	rowsIn := 0
	tsIdx, valIdx := -1, -1
	headerSeen := false

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// skip bad lines, count toward rowsIn so caller knows
			rowsIn++
			continue
		}
		rowsIn++
		if !headerSeen {
			tsIdx, valIdx = detectColumns(rec)
			if tsIdx >= 0 {
				headerSeen = true
				continue
			}
			// No header — assume col 0 = ts, col 1 = value
			if len(rec) >= 2 {
				tsIdx, valIdx = 0, 1
				headerSeen = true
				// fall through and parse this row
			} else {
				continue
			}
		}
		if tsIdx >= len(rec) || valIdx >= len(rec) {
			continue
		}
		ts, ok := parseTS(rec[tsIdx])
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(rec[valIdx]), 64)
		if err != nil {
			continue
		}
		// Heuristic: if value > 50 we probably got Wh, convert to kWh
		if v > 50 {
			v = v / 1000.0
		}
		out = append(out, sample{ts: ts, val: v})
	}
	return out, rowsIn, nil
}

func detectColumns(header []string) (int, int) {
	tsIdx, valIdx := -1, -1
	for i, h := range header {
		hl := strings.ToLower(strings.TrimSpace(h))
		switch {
		case tsIdx < 0 && (strings.Contains(hl, "time") || strings.Contains(hl, "date") || strings.Contains(hl, "interval")):
			tsIdx = i
		case valIdx < 0 && (strings.Contains(hl, "kwh") || strings.Contains(hl, "wh") || strings.Contains(hl, "energy") || strings.Contains(hl, "production") || strings.Contains(hl, "consumption")):
			valIdx = i
		}
	}
	return tsIdx, valIdx
}

func parseTS(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"02/01/2006 15:04",
		"01/02/2006 15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func analyse(source, filename string, rowsIn int, samples []sample) *Result {
	res := &Result{
		Source:     source,
		Filename:   filename,
		RowsIn:     rowsIn,
		RowsLoaded: len(samples),
		Status:     "ok",
	}
	if len(samples) == 0 {
		return res
	}
	// Sort by ts (assume already chronological — quick check)
	// Detect duplicates and gaps
	seen := map[time.Time]int{}
	var minT, maxT time.Time
	for _, s := range samples {
		if minT.IsZero() || s.ts.Before(minT) {
			minT = s.ts
		}
		if maxT.IsZero() || s.ts.After(maxT) {
			maxT = s.ts
		}
		if seen[s.ts] > 0 {
			res.Duplicates++
		}
		seen[s.ts]++
		// Implausibility heuristic: < -1 or > 50 kWh per 15-min
		if s.val < -1 || s.val > 50 {
			res.Implausible++
		}
		if math.IsNaN(s.val) || math.IsInf(s.val, 0) {
			res.Implausible++
		}
	}
	res.RangeStart = minT
	res.RangeEnd = maxT
	expectedIntervals := int(maxT.Sub(minT).Minutes()/15) + 1
	if expectedIntervals > len(samples) {
		res.Gaps = expectedIntervals - len(samples)
	}
	if res.Gaps > 0 || res.Duplicates > 0 || res.Implausible > 0 {
		res.Status = "partial"
	}
	return res
}

func logRun(ctx context.Context, pool *pgxpool.Pool, r *Result) {
	_, _ = pool.Exec(ctx, `INSERT INTO ingestion_runs (source, filename, rows_in, rows_loaded, gaps_found, duplicates, implausible, range_start, range_end, status, message) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		r.Source, r.Filename, r.RowsIn, r.RowsLoaded, r.Gaps, r.Duplicates, r.Implausible, r.RangeStart, r.RangeEnd, r.Status, r.Message)
}

// pgx helper to silence unused-import warning if compiled standalone
var _ = pgx.NamedArgs{}

// fmt is used in error formatting paths; keep import alive
var _ = fmt.Sprintf
