// Package solar implements the clear-sky / shading model used to estimate
// upgrade panel yields. PRD §5.3 and §7.4: derive a per-(hour-of-day, month)
// shading factor from actual production vs theoretical clear-sky GHI, then
// apply it to upgraded panel specs to estimate incremental generation.
package solar

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
	lat  float64
	lon  float64
}

func New(pool *pgxpool.Pool, lat, lon float64) *Service {
	return &Service{pool: pool, lat: lat, lon: lon}
}

// ShadingMatrix is a [12 months][24 hours] grid of efficiency factors in 0..1.
// 1.0 means actual production matched theoretical, 0.0 means fully shaded.
type ShadingMatrix struct {
	Cells [12][24]float64 `json:"cells"`
}

// Compute scans energy_intervals + theoretical clear-sky GHI to derive a
// per-cell efficiency factor.
func (s *Service) Compute(ctx context.Context) (ShadingMatrix, error) {
	rows, err := s.pool.Query(ctx, `SELECT ts, solar_gen_kwh, ghi_w_m2 FROM energy_intervals`)
	if err != nil {
		return ShadingMatrix{}, err
	}
	defer rows.Close()
	var actual, theoretical [12][24]float64
	loc, _ := time.LoadLocation("Australia/Sydney")
	for rows.Next() {
		var ts time.Time
		var actKWh, ghi float64
		if err := rows.Scan(&ts, &actKWh, &ghi); err != nil {
			return ShadingMatrix{}, err
		}
		local := ts.In(loc)
		m := int(local.Month()) - 1
		h := local.Hour()
		actual[m][h] += actKWh
		// Theoretical 5 kW system at unshaded GHI: kWh per 15min = ghi/1000 * 5 * 0.25
		theoretical[m][h] += ghi / 1000.0 * 5 * 0.25
	}
	var sm ShadingMatrix
	for m := 0; m < 12; m++ {
		for h := 0; h < 24; h++ {
			if theoretical[m][h] > 0.01 {
				eff := actual[m][h] / theoretical[m][h]
				if eff > 1.1 {
					eff = 1.1
				}
				if eff < 0 {
					eff = 0
				}
				sm.Cells[m][h] = eff
			}
		}
	}
	return sm, rows.Err()
}

// PanelSpec describes one panel option from the assumptions table.
type PanelSpec struct {
	Watt          float64
	Eff           float64
	TempCoefPerC  float64
	LengthM       float64
	WidthM        float64
}

// EstimateAnnualYield applies the shading matrix to a panel array and returns
// estimated annual generation in kWh per year.
func (s *Service) EstimateAnnualYield(ctx context.Context, sm ShadingMatrix, spec PanelSpec, panelCount int) (float64, error) {
	// Walk a typical year of clear-sky GHI at the configured site, apply
	// shading per (month, hour), accumulate.
	loc, _ := time.LoadLocation("Australia/Sydney")
	year := 2025
	start := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(1, 0, 0)
	var total float64
	arrayKWPeak := float64(panelCount) * spec.Watt / 1000.0
	for ts := start; ts.Before(end); ts = ts.Add(15 * time.Minute) {
		alt := solarAltitude(ts, s.lat)
		if alt <= 0 {
			continue
		}
		sinAlt := math.Sin(alt)
		ghi := 1098 * sinAlt * math.Exp(-0.057/sinAlt)
		// Apply shading
		m := int(ts.Month()) - 1
		h := ts.Hour()
		shading := sm.Cells[m][h]
		if shading == 0 {
			shading = 0.85
		}
		// Convert GHI -> array kWh for 15 min
		// kWh = (GHI/1000) * arrayKWPeak * 0.25 * eff
		total += ghi / 1000.0 * arrayKWPeak * 0.25 * shading
	}
	// Temperature de-rate (rough): use mean -0.4% per °C above 25
	// Skipped per-interval for performance.
	return total, nil
}

func solarAltitude(ts time.Time, lat float64) float64 {
	doy := float64(ts.YearDay())
	decl := 23.44 * math.Pi / 180 * math.Sin(2*math.Pi*(284+doy)/365.0)
	hourFrac := float64(ts.Hour()) + float64(ts.Minute())/60
	hourAngle := (hourFrac - 12) * 15 * math.Pi / 180
	latRad := lat * math.Pi / 180
	sinAlt := math.Sin(latRad)*math.Sin(decl) + math.Cos(latRad)*math.Cos(decl)*math.Cos(hourAngle)
	if sinAlt < -1 {
		sinAlt = -1
	}
	if sinAlt > 1 {
		sinAlt = 1
	}
	return math.Asin(sinAlt)
}
