package soc

import (
	"math"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddRangeReport(t *testing.T) {
	now := time.Now()

	var reports []api.RangeReport
	reports = AddRangeReport(reports, api.RangeReport{Soc: 60, Range: 165, Time: now.Add(-30 * 24 * time.Hour)})
	reports = AddRangeReport(reports, api.RangeReport{Soc: 40, Range: 106, Time: now.Add(-time.Hour)})
	reports = AddRangeReport(reports, api.RangeReport{Soc: 40, Range: 105, Time: now})

	// the report older than four weeks is dropped, the latest per soc kept
	require.Len(t, reports, 1)
	assert.Equal(t, int64(105), reports[0].Range)

	reports = AddRangeReport(reports, api.RangeReport{Soc: 20, Range: 47, Time: now})
	assert.Equal(t, []int{20, 40}, []int{reports[0].Soc, reports[1].Soc}, "sorted by soc")
}

func TestFloorSoc(t *testing.T) {
	now := time.Now()

	// this month's Born: about 2.95 km per % and zero near 4 %
	var reports []api.RangeReport
	for _, soc := range []int{22, 35, 48, 61, 74, 86} {
		reports = append(reports, api.RangeReport{Soc: soc, Range: int64(math.Round(2.95 * float64(soc-4))), Time: now})
	}
	assert.Equal(t, 60.0, FloorSoc(reports, now, 165, 21, 58), "line through the reports")

	// too few reports, too narrow a span or too old: 21 kWh/100km of 58 kWh
	assert.Equal(t, 60.0, FloorSoc(reports[:4], now, 165, 21, 58), "too few")
	assert.Equal(t, 60.0, FloorSoc([]api.RangeReport{
		{Soc: 50, Range: 136}, {Soc: 52, Range: 141}, {Soc: 54, Range: 147}, {Soc: 56, Range: 153}, {Soc: 58, Range: 159},
	}, now, 165, 21, 58), "too narrow")
	assert.Equal(t, 60.0, FloorSoc(reports, now.Add(29*24*time.Hour), 165, 21, 58), "too old")

	// winter: the range falls faster, the floor rises
	for i := range reports {
		reports[i].Range = int64(math.Round(2.1 * float64(reports[i].Soc-4)))
	}
	assert.Equal(t, 83.0, FloorSoc(reports, now, 165, 21, 58))

	// never above 100 %
	assert.Equal(t, 100.0, FloorSoc(nil, now, 500, 21, 58))
}
