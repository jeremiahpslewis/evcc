package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/messenger"
)

const (
	evPlanFallback = "planfallback" // plan charges to the floor while the soc is unconfirmed

	// socFallbackRefusal is how long the vehicle may refuse power in a planned slot before it counts as full
	socFallbackRefusal = 5 * time.Minute
)

// socReport tracks the raw soc and range read from the vehicle api. A value that differs from
// the previous one is a report from the vehicle, a value unchanged may be a stale copy.
type socReport struct {
	vehicle  api.Vehicle
	soc      float64
	rng      int64     // km, -1 if unknown
	read     time.Time // last read
	reported time.Time // earliest the last report can have arrived: the read before it
	charger  bool      // the charger reads the soc from the vehicle itself
}

// socFallback is the session's state of the soc fallback
type socFallback struct {
	floor     float64   // soc in % the fallback charges to, 0 without a vehicle capacity
	idleSince time.Time // charging enabled in a planned slot while the vehicle draws nothing
	refused   time.Time // plan time the vehicle refused power for, which ends the fallback for that night
	notified  time.Time // plan time the fallback was announced for
}

// socFallbackSettings returns the vehicle's soc fallback settings
func socFallbackSettings(v api.Vehicle) api.SocFallback {
	if d, ok := api.Cap[api.SocFallbackDescriber](v); ok {
		return d.SocFallback()
	}
	return api.SocFallback{}.WithDefaults()
}

// observeSocReport tracks the soc and range read from the vehicle api. A change is a report,
// recorded with the range for the vehicle's floor.
func (lp *Loadpoint) observeSocReport(v api.Vehicle, socR *float64, rngR *int64) {
	if v == nil || socR == nil {
		return
	}

	now := lp.clock.Now()
	rng := int64(-1)
	if rngR != nil {
		rng = *rngR
	}

	lp.Lock()
	r := lp.socReport
	var report bool

	switch {
	case r.vehicle != v:
		r = socReport{vehicle: v, soc: *socR, rng: rng}

	case *socR != r.soc || rng >= 0 && r.rng >= 0 && rng != r.rng:
		r.reported = r.read
		r.soc = *socR
		r.rng = max(rng, r.rng)
		report = rng > 0

	case r.rng < 0:
		r.rng = rng
	}

	r.read = now
	r.charger = false
	lp.socReport = r
	lp.Unlock()

	if report {
		vs := vehicle.Settings(lp.log, v)
		reports := soc.AddRangeReport(vs.GetRangeReports(), api.RangeReport{Soc: int(math.Round(*socR)), Range: rng, Time: now})
		if err := vs.SetRangeReports(reports); err != nil {
			lp.log.ERROR.Printf("range reports: %v", err)
		}
	}
}

// observeChargerSoc marks the soc as read by the charger from the vehicle itself, which is never stale
func (lp *Loadpoint) observeChargerSoc() {
	lp.Lock()
	defer lp.Unlock()
	lp.socReport.charger = true
}

// socConfirmed returns true if the vehicle reported its soc since plug-in or within the window before it
func (lp *Loadpoint) socConfirmed(v api.Vehicle) bool {
	r := lp.socReport
	if r.charger {
		return true
	}
	return r.vehicle == v && !r.reported.IsZero() && !r.reported.Before(lp.connectedTime.Add(-socFallbackSettings(v).Window))
}

// socFallbackActive returns true while plans with the soc fallback switched on charge to the floor:
// connected after a long enough unplug, in a mode that honours plans, the soc unconfirmed (no mutex)
func (lp *Loadpoint) socFallbackActive(v api.Vehicle) bool {
	connected := lp.status == api.StatusB || lp.status == api.StatusC
	return v != nil && lp.socFallback.floor > 0 && connected && lp.mode != api.ModeOff &&
		lp.unpluggedLong(v) && !lp.socConfirmed(v)
}

// unpluggedLong returns true if the vehicle was unplugged longer than its setting before this session,
// or for how long is unknown since evcc started with the vehicle plugged in
func (lp *Loadpoint) unpluggedLong(v api.Vehicle) bool {
	return lp.unpluggedTime.IsZero() || lp.connectedTime.Sub(lp.unpluggedTime) > socFallbackSettings(v).Unplugged
}

// planFallback returns true if the next plan charges to the soc fallback's floor (no mutex)
func (lp *Loadpoint) planFallback() bool {
	return lp.socFallbackActive(lp.GetVehicle()) && lp.nextPlan().Fallback
}

// socFallbackSoc returns the soc the fallback plans from: the arrival soc plus the session's charged energy
func (lp *Loadpoint) socFallbackSoc(v api.Vehicle) float64 {
	return socFallbackSettings(v).ArrivalSoc + lp.getChargedEnergy()/(v.Capacity()*1e3/soc.ChargeEfficiency)*100
}

// GetSocFallback returns the soc the plan assumes and true while it charges to the floor
func (lp *Loadpoint) GetSocFallback() (float64, bool) {
	lp.RLock()
	defer lp.RUnlock()

	v := lp.GetVehicle()
	if v == nil || !lp.socBasedPlanning() || !lp.planFallback() {
		return 0, false
	}

	return lp.socFallbackSoc(v), true
}

// updateSocFallback computes the vehicle's floor, ends the fallback for a night the vehicle refuses
// power and announces a night the fallback takes over. Runs once the vehicle was read.
func (lp *Loadpoint) updateSocFallback() {
	v := lp.GetVehicle()

	var floor float64
	if v != nil && v.Capacity() > 0 {
		cfg := socFallbackSettings(v)
		reports := vehicle.Settings(lp.log, v).GetRangeReports()
		floor = soc.FloorSoc(reports, lp.clock.Now(), cfg.FloorRange(), cfg.Consumption, v.Capacity())
	}

	lp.Lock()
	event := lp.stepSocFallback(v, floor)
	lp.Unlock()

	if event != nil {
		lp.pushEventAttributes(evPlanFallback, event)
	}
}

// stepSocFallback updates the fallback state and returns the attributes of an announcement (no mutex)
func (lp *Loadpoint) stepSocFallback(v api.Vehicle, floor float64) map[string]any {
	lp.socFallback.floor = floor

	connected := lp.status == api.StatusB || lp.status == api.StatusC
	lp.publish(keys.VehicleSocConfirmed, v != nil && connected && lp.socConfirmed(v))

	// a locked fallback goal is stale once the soc is confirmed or the vehicle refused power
	if l := lp.planLocked; l.Fallback && (!lp.socFallbackActive(v) || l.Time.Equal(lp.socFallback.refused)) {
		lp.clearPlanLock()
	}

	var p plan
	if lp.socBasedPlanning() && lp.socFallbackActive(v) {
		p = lp.nextPlan()
	}
	lp.publish(keys.PlanFallback, p.Fallback)

	if !p.Fallback {
		lp.socFallback.idleSince = time.Time{}
		return nil
	}

	now := lp.clock.Now()

	// a vehicle drawing nothing in a planned slot is full or at its own limit
	if lp.planActive && lp.enabled && lp.status == api.StatusB {
		if lp.socFallback.idleSince.IsZero() {
			lp.socFallback.idleSince = now
		}

		if now.Sub(lp.socFallback.idleSince) >= socFallbackRefusal {
			lp.log.INFO.Printf("plan: vehicle refuses power, soc fallback ends for %v", p.End.Round(time.Minute).Local())
			lp.socFallback.refused = p.End
			lp.socFallback.idleSince = time.Time{}
			lp.clearPlanLock()
			lp.publish(keys.PlanFallback, false)
		}

		return nil
	}
	lp.socFallback.idleSince = time.Time{}

	// announce once the window after plug-in has passed, a healthy vehicle usually reports within it
	if p.End.Equal(lp.socFallback.notified) || now.Sub(lp.connectedTime) < socFallbackSettings(v).Window {
		return nil
	}

	energy := soc.RemainingChargeEnergy(p.Soc, lp.socFallbackSoc(v), v.Capacity())
	lp.log.INFO.Printf("plan: soc unconfirmed, charging %.1fkWh to %d%% by %v", energy, p.Soc, p.End.Round(time.Minute).Local())
	lp.socFallback.notified = p.End

	return map[string]any{
		"planFallbackTime":   p.End,
		"planFallbackSoc":    p.Soc,
		"planFallbackEnergy": energy,
	}
}

// pushEventAttributes sends an event with event-specific template attributes
func (lp *Loadpoint) pushEventAttributes(event string, attributes map[string]any) {
	// test helper
	if lp.pushChan == nil {
		return
	}
	lp.pushChan <- messenger.Event{Event: event, Attributes: attributes}
}
