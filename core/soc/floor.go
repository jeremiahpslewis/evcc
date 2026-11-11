package soc

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	rangeReportsMaxAge = 28 * 24 * time.Hour // reports of recent weeks, so that the season's consumption counts
	minRangeReports    = 5                   // reports a line needs
	minRangeSpread     = 20                  // soc span in % the reports of a line must cover
)

// AddRangeReport adds a report to the recent ones, keeping the latest per soc percent, sorted by soc
func AddRangeReport(reports []api.RangeReport, r api.RangeReport) []api.RangeReport {
	res := slices.DeleteFunc(slices.Clone(reports), func(e api.RangeReport) bool {
		return e.Soc == r.Soc || r.Time.Sub(e.Time) > rangeReportsMaxAge
	})

	res = append(res, r)
	slices.SortFunc(res, func(a, b api.RangeReport) int {
		return cmp.Compare(a.Soc, b.Soc)
	})

	return res
}

// FloorSoc returns the soc in % at which the vehicle's range reads km, rounded up. It takes a line
// fitted through the recent range reports, else km at the consumption in kWh/100km of the capacity in kWh.
func FloorSoc(reports []api.RangeReport, now time.Time, km, consumption, capacity float64) float64 {
	var soc float64
	if a, b, ok := rangeLine(reports, now); ok {
		soc = (km - a) / b
	} else if capacity > 0 {
		soc = km * consumption / capacity
	}

	return min(math.Ceil(soc), 100)
}

// rangeLine fits range = a + b*soc through the recent reports by least squares
func rangeLine(reports []api.RangeReport, now time.Time) (float64, float64, bool) {
	var n, sx, sy, sxx, sxy float64
	lo, hi := 100, 0

	for _, r := range reports {
		if r.Range <= 0 || now.Sub(r.Time) > rangeReportsMaxAge {
			continue
		}

		x, y := float64(r.Soc), float64(r.Range)
		n++
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		lo, hi = min(lo, r.Soc), max(hi, r.Soc)
	}

	if n < minRangeReports || hi-lo < minRangeSpread {
		return 0, 0, false
	}

	b := (n*sxy - sx*sy) / (n*sxx - sx*sx)
	a := (sy - b*sx) / n

	return a, b, b > 0
}
