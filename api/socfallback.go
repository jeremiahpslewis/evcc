package api

import "time"

// SocFallback configures how a plan charges while the vehicle's soc is unconfirmed, i.e. the
// vehicle has not reported since it was plugged in. Zero values take the defaults.
type SocFallback struct {
	ArrivalSoc float64       `mapstructure:"arrivalSoc"` // soc assumed at plug-in, %
	Window     time.Duration `mapstructure:"window"`     // a report this long before plug-in confirms the soc
	Unplugged  time.Duration `mapstructure:"unplugged"`  // the fallback applies once the vehicle was unplugged longer
}

// SocFallbackDescriber returns the vehicle's soc fallback settings
type SocFallbackDescriber interface {
	SocFallback() SocFallback
}

// WithDefaults returns the settings with zero values replaced by the defaults
func (f SocFallback) WithDefaults() SocFallback {
	if f.ArrivalSoc == 0 {
		f.ArrivalSoc = 5
	}
	if f.Window == 0 {
		f.Window = 30 * time.Minute
	}
	if f.Unplugged == 0 {
		f.Unplugged = 90 * time.Minute
	}
	return f
}
