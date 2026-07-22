package charger

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/meters"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.Add("switchsocket", NewSwitchSocketFromConfig)
}

type SwitchSocketConfig struct {
	Switch meter.Config
	Standby meter.Config
}

func NewSwitchSocketFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		embeds.Embed[embed.Config]
		Switch meter.Config
		Standby meter.Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	enabled, err := cc.Enabled.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	power, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, err
	}

	soc, err := cc.Soc.BoolGetter(ctx)
	if err != nil {
		return nil, err
	}

	// ... rest of config parsing ...

	c := &SwitchSocket{
		enabled: enabled,
		power:   power,
		soc:     soc,
	}

	return c, nil
}

type SwitchSocket struct {
	enabled func() (bool, error)
	power   func() (float64, error)
	soc     func() (bool, error)
	phases  int
}

func (v *SwitchSocket) Phases() int {
	return max(v.phases, 1)
}
