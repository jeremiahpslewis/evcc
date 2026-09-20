package api

import "slices"

type Feature int

//go:generate go tool enumer -type Feature -text
const (
	_                  Feature = iota
	CoarseCurrent              // charger
	IntegratedDevice           // charger - always connected - no vehicle, no charging sessions
	SwitchDevice               // charger - no current control - heat pumps or switch sockets
	Heating                    // charger - heating device - soc ist temperature (°C)
	DemandWeekday              // charger - demand forecast: same-weekday average over past 4 weeks (warm water)
	DemandTemperature          // charger - demand forecast: daily avg scaled by outdoor temp (room heating)
	Continuous                 // charger - heating device where disabled means "normal operation"
	Average                    // tariff
	Cacheable                  // tariff
	Offline                    // vehicle
	Retryable                  // vehicle
	Streaming                  // vehicle
	WelcomeCharge              // vehicle
	ClimaterDisabled           // vehicle - ignore climater state for charge control
	AutodetectDisabled         // vehicle - do not try to identify vehicle by status
	WakeUpDisabled             // vehicle - do not send wake-up calls
)

// UniqueFeatures returns the features as an ordered set. Repeated values are
// dropped, the first occurrence determines the position. The input is not modified.
func UniqueFeatures(features []Feature) []Feature {
	if len(features) == 0 {
		return features
	}

	res := make([]Feature, 0, len(features))
	for _, f := range features {
		if !slices.Contains(res, f) {
			res = append(res, f)
		}
	}

	return res
}
