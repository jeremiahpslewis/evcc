package api

import "time"

// SocFallback configures the floor a plan charges to while the vehicle's soc is unconfirmed,
// i.e. the vehicle has not reported since it was plugged in. Zero values take the defaults.
type SocFallback struct {
	Range       float64       `mapstructure:"range"`       // km the floor must cover
	Reserve     float64       `mapstructure:"reserve"`     // % added to the range
	ArrivalSoc  float64       `mapstructure:"arrivalSoc"`  // soc assumed at plug-in, %
	Window      time.Duration `mapstructure:"window"`      // a report this long before plug-in confirms the soc
	Unplugged   time.Duration `mapstructure:"unplugged"`   // the fallback applies once the vehicle was unplugged longer
	Consumption float64       `mapstructure:"consumption"` // kWh/100km, for the floor while the range does not fit a line
}

// SocFallbackDescriber returns the vehicle's soc fallback settings
type SocFallbackDescriber interface {
	SocFallback() SocFallback
}

// WithDefaults returns the settings with zero values replaced by the defaults
func (f SocFallback) WithDefaults() SocFallback {
	if f.Range == 0 {
		f.Range = 150
	}
	if f.Reserve == 0 {
		f.Reserve = 10
	}
	if f.ArrivalSoc == 0 {
		f.ArrivalSoc = 5
	}
	if f.Window == 0 {
		f.Window = 30 * time.Minute
	}
	if f.Unplugged == 0 {
		f.Unplugged = 90 * time.Minute
	}
	if f.Consumption == 0 {
		f.Consumption = 21
	}
	return f
}

// FloorRange returns the range in km the floor must cover, reserve included
func (f SocFallback) FloorRange() float64 {
	return f.Range * (1 + f.Reserve/100)
}

// RangeReport is a soc and range the vehicle reported together
type RangeReport struct {
	Soc   int       `json:"soc"`   // %
	Range int64     `json:"range"` // km
	Time  time.Time `json:"time"`
}
