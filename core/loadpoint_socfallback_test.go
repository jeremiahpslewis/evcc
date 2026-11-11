package core

import (
	"strings"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newSocFallbackLoadpoint returns a loadpoint with a 58 kWh vehicle, a daily 06:45 plan to 80 % with the
// soc fallback switched on, plugged in now. The vehicle's range does not fit a line yet, so the floor
// is 165 km at 21 kWh/100km: 60 %.
func newSocFallbackLoadpoint(t *testing.T, fallback bool) (*Loadpoint, *clock.Mock, api.Vehicle) {
	t.Helper()

	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().Capacity().Return(58.0).AnyTimes()
	v.EXPECT().Features().Return(nil).AnyTimes()
	v.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()
	v.EXPECT().Phases().Return(0).AnyTimes()

	// a name per test, settings outlive a test whose database is gone
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	prefix := "vehicle." + name + "."
	require.NoError(t, settings.SetJson(prefix+keys.RepeatingPlans, []api.RepeatingPlan{
		{Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, Time: "06:45", Tz: "Europe/Berlin", Soc: 80, Active: true, SocFallback: fallback},
	}))
	t.Cleanup(func() {
		_ = settings.Delete(prefix + keys.RepeatingPlans)
		_ = settings.Delete(prefix + keys.RangeReports)
	})

	// plan occurrences follow the wall clock
	clk := clock.NewMock()
	clk.Set(time.Now())

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v
	lp.socEstimator = soc.NewEstimator(lp.log, v)
	lp.mode = api.ModeSmart
	lp.status = api.StatusB
	lp.connectedTime = clk.Now()

	return lp, clk, v
}

// read simulates the loadpoint reading the vehicle api at the clock's time
func read(lp *Loadpoint, v api.Vehicle, soc float64, rng int64) {
	lp.observeSocReport(v, &soc, &rng)
	lp.socEstimator.Soc(&soc, lp.getChargedEnergy())
	lp.vehicleSoc = soc
}

// The 10 November session: the soc froze at 86 % / 265 km at 07:55, the car was plugged in at 19:10
// and the plan looked met, so nothing was charged.
func TestSocFallbackFrozenSession(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, true)
	plugin := clk.Now()
	lp.unpluggedTime = plugin.Add(-11*time.Hour - 18*time.Minute) // 07:52

	clk.Set(plugin.Add(-11*time.Hour - 15*time.Minute)) // 07:55
	read(lp, v, 86, 265)
	for clk.Now().Before(plugin.Add(time.Hour)) {
		clk.Add(15 * time.Minute)
		read(lp, v, 86, 265)
	}

	lp.updateSocFallback()
	assert.False(t, lp.socConfirmed(v))

	p := lp.nextPlan()
	require.True(t, p.Fallback, "the stale soc must not make the plan look met")
	assert.Equal(t, 60, p.Soc, "plans to the floor only")

	fallbackSoc, ok := lp.GetSocFallback()
	require.True(t, ok)
	assert.Equal(t, 5.0, fallbackSoc, "assumes arrival at 5 %")

	// (60 - 5) % of 58 kWh at 85 %: about 37.5 kWh at the wallbox, 3.4 hours at 11 kW
	assert.InDelta(t, 37.5, soc.RemainingChargeEnergy(p.Soc, fallbackSoc, 58), 0.1)
	assert.InDelta(t, 3.4, lp.GetPlanRequiredDuration(60, 11e3).Hours(), 0.05)

	goal, socBased := lp.GetPlanGoal()
	assert.True(t, socBased)
	assert.Equal(t, 60.0, goal)

	// no plan's own goal is pursued while the soc is unconfirmed
	for _, g := range lp.GetSocPlanGoals() {
		assert.LessOrEqual(t, g.Soc, 60)
	}

	// the energy delivered, the car leaves with at least the floor
	lp.energyMetrics.totalKWh = 37.6
	assert.Equal(t, plan{}, lp.nextPlan())
	_, ok = lp.GetSocFallback()
	assert.False(t, ok)
}

// Without the switch the stale soc keeps the plan met, as before.
func TestSocFallbackSwitchedOff(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, false)

	clk.Add(-11 * time.Hour)
	read(lp, v, 86, 265)
	clk.Add(11 * time.Hour)
	read(lp, v, 86, 265)

	lp.updateSocFallback()
	assert.Equal(t, plan{}, lp.nextPlan())
}

func TestSocFallbackConfirmed(t *testing.T) {
	t.Run("report 10 minutes after plug-in", func(t *testing.T) {
		lp, clk, v := newSocFallbackLoadpoint(t, true)
		plugin := clk.Now()

		clk.Set(plugin.Add(-11 * time.Hour))
		read(lp, v, 86, 265)
		clk.Set(plugin)
		read(lp, v, 86, 265) // the value shown at plug-in lags

		lp.updateSocFallback()
		assert.True(t, lp.nextPlan().Fallback)

		clk.Add(10 * time.Minute)
		read(lp, v, 22, 53)

		lp.updateSocFallback()
		assert.True(t, lp.socConfirmed(v))

		p := lp.nextPlan()
		assert.False(t, p.Fallback)
		assert.Equal(t, 80, p.Soc, "the plan's own goal from the reported soc")
	})

	t.Run("report within the window before plug-in", func(t *testing.T) {
		lp, clk, v := newSocFallbackLoadpoint(t, true)
		plugin := clk.Now()

		clk.Set(plugin.Add(-21 * time.Minute))
		read(lp, v, 30, 77)
		clk.Add(time.Minute)
		read(lp, v, 29, 74)
		clk.Set(plugin)
		read(lp, v, 29, 74)

		lp.updateSocFallback()
		assert.True(t, lp.socConfirmed(v))
		assert.False(t, lp.nextPlan().Fallback)
	})

	t.Run("plugged in hours after the last report", func(t *testing.T) {
		lp, clk, v := newSocFallbackLoadpoint(t, true)
		plugin := clk.Now()

		clk.Set(plugin.Add(-4 * time.Hour))
		read(lp, v, 40, 106)
		clk.Add(time.Minute)
		read(lp, v, 38, 100)
		clk.Set(plugin)
		read(lp, v, 38, 100)

		lp.updateSocFallback()
		assert.False(t, lp.socConfirmed(v), "a report hours before plug-in may be stale")
		assert.True(t, lp.nextPlan().Fallback)

		// reports once charging wakes the car
		lp.status = api.StatusC
		lp.energyMetrics.totalKWh = 2
		clk.Add(3 * time.Hour)
		read(lp, v, 41, 109)

		lp.updateSocFallback()
		assert.True(t, lp.socConfirmed(v))
		assert.False(t, lp.nextPlan().Fallback)
	})

	t.Run("the charger reads the soc", func(t *testing.T) {
		lp, _, v := newSocFallbackLoadpoint(t, true)

		lp.observeChargerSoc()
		lp.updateSocFallback()
		assert.True(t, lp.socConfirmed(v))
	})
}

// A report arriving in the first charging slot: planning resumes from it, the energy charged
// before it is not counted twice.
func TestSocFallbackReportWhileCharging(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, true)
	plugin := clk.Now()

	clk.Set(plugin.Add(-11 * time.Hour))
	read(lp, v, 86, 265)
	clk.Set(plugin)
	read(lp, v, 86, 265)

	lp.updateSocFallback()
	require.True(t, lp.nextPlan().Fallback)

	// first slot: 5 kWh charged, then the car reports
	lp.status = api.StatusC
	lp.energyMetrics.totalKWh = 5
	clk.Add(30 * time.Minute)
	read(lp, v, 40, 106)

	lp.updateSocFallback()
	p := lp.nextPlan()
	assert.False(t, p.Fallback)
	assert.Equal(t, 80, p.Soc)
	assert.Equal(t, soc.RemainingChargeDuration(80, 11e3, 40, 58), lp.GetPlanRequiredDuration(80, 11e3))
}

func TestSocFallbackRefusal(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, true)

	clk.Add(-11 * time.Hour)
	read(lp, v, 86, 265)
	clk.Add(11 * time.Hour)
	read(lp, v, 86, 265)

	lp.updateSocFallback()
	p := lp.nextPlan()
	require.True(t, p.Fallback)

	// in a planned slot, enabled, the car full
	lp.planActive = true
	lp.enabled = true
	lp.lockPlanGoal(p.End, p.Soc, p.Id, true)

	lp.updateSocFallback()
	clk.Add(4 * time.Minute)
	lp.updateSocFallback()
	assert.True(t, lp.nextPlan().Fallback, "4 minutes are not a refusal")

	// drawing power in between restarts the count
	lp.status = api.StatusC
	lp.updateSocFallback()
	lp.status = api.StatusB
	lp.updateSocFallback()
	clk.Add(4 * time.Minute)
	lp.updateSocFallback()
	assert.True(t, lp.nextPlan().Fallback)

	clk.Add(time.Minute)
	lp.updateSocFallback()
	assert.Equal(t, plan{}, lp.nextPlan(), "the fallback ends for the night")
	assert.Equal(t, PlanLock{}, lp.planLocked)
}

// A replug without a drive: the same value and no report. Within 90 minutes the fallback stays out,
// after a longer unplug it runs until a report confirms the soc, the accepted false alarm.
func TestSocFallbackReplug(t *testing.T) {
	for _, tc := range []struct {
		unplugged time.Duration
		fallback  bool
	}{
		{time.Hour, false},
		{90 * time.Minute, false},
		{2 * time.Hour, true},
	} {
		t.Run(tc.unplugged.String(), func(t *testing.T) {
			lp, clk, v := newSocFallbackLoadpoint(t, true)

			read(lp, v, 70, 195)
			clk.Add(10 * time.Minute)
			read(lp, v, 71, 198)

			lp.updateSocFallback()
			require.True(t, lp.socConfirmed(v))

			lp.unpluggedTime = clk.Now()
			clk.Add(tc.unplugged)
			read(lp, v, 71, 198)
			lp.evVehicleConnectHandler()
			read(lp, v, 71, 198)

			lp.updateSocFallback()
			assert.False(t, lp.socConfirmed(v))
			assert.Equal(t, tc.fallback, lp.nextPlan().Fallback)
		})
	}
}

func TestSocFallbackEvent(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, true)
	events := make(chan messenger.Event, 2)
	lp.pushChan = events

	clk.Add(-11 * time.Hour)
	read(lp, v, 86, 265)
	clk.Add(11 * time.Hour)
	read(lp, v, 86, 265)

	// a healthy car usually reports within the window after plug-in
	lp.updateSocFallback()
	clk.Add(29 * time.Minute)
	lp.updateSocFallback()
	assert.Empty(t, events)

	clk.Add(time.Minute)
	lp.updateSocFallback()
	require.Len(t, events, 1)

	ev := <-events
	assert.Equal(t, evPlanFallback, ev.Event)
	assert.Equal(t, lp.nextPlan().End, ev.Attributes["planFallbackTime"])
	assert.Equal(t, 60, ev.Attributes["planFallbackSoc"])
	assert.InDelta(t, 37.5, ev.Attributes["planFallbackEnergy"], 0.1)

	// once per night
	clk.Add(time.Minute)
	lp.updateSocFallback()
	assert.Empty(t, events)
}

// The floor follows the line through the vehicle's reported range.
func TestSocFallbackFloorFromRange(t *testing.T) {
	lp, clk, v := newSocFallbackLoadpoint(t, true)

	// winter: 2.1 km per % and zero at 4 %
	clk.Add(-48 * time.Hour)
	read(lp, v, 90, 181)
	for s := 89; s >= 20; s -= 7 {
		clk.Add(time.Hour)
		read(lp, v, float64(s), int64(2.1*float64(s-4)+0.5))
	}

	lp.updateSocFallback()
	assert.Equal(t, 83.0, lp.socFallback.floor)
}

// The disconnect evcc reports at startup is no unplug: the vehicle may have been away all day.
func TestSocFallbackUnpluggedAtStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Status().Return(api.StatusA, nil).AnyTimes()

	clk := clock.NewMock()
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.charger = charger

	require.NoError(t, lp.updateChargerStatus())
	assert.True(t, lp.unpluggedTime.IsZero(), "startup")

	lp.status = api.StatusB
	require.NoError(t, lp.updateChargerStatus())
	assert.Equal(t, clk.Now(), lp.unpluggedTime, "unplugged")
}
