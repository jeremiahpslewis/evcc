package plugin

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant reads and writes Home Assistant entity states. With an attribute
// or a pipeline configured, values are read as string and numbers and booleans
// parsed from it: the entity's unit then no longer applies.
type HomeAssistant struct {
	*getter
	conn      *homeassistant.Connection
	entity    string
	attribute string
	pipeline  *pipeline.Pipeline
}

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantPluginFromConfig)
}

// NewHomeAssistantPluginFromConfig creates a Home Assistant plugin
func NewHomeAssistantPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		homeassistant.Config `mapstructure:",squash"`
		pipeline.Settings    `mapstructure:",squash"`
		Entity               string
		Attribute            string // read this attribute instead of the state
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Entity == "" {
		return nil, errors.New("missing entity")
	}

	log := util.NewLogger("homeassistant")

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home_, cc.Insecure)
	if err != nil {
		return nil, err
	}

	p := &HomeAssistant{
		conn:      conn,
		entity:    cc.Entity,
		attribute: cc.Attribute,
	}

	// an empty pipeline still converts xml, only create it when configured
	if cc.Settings != (pipeline.Settings{}) {
		if p.pipeline, err = pipeline.New(log, cc.Settings); err != nil {
			return nil, err
		}
	}

	p.getter = defaultGetters(p, 1)

	return p, nil
}

// raw reports whether values are parsed from the string value instead of the typed state
func (p *HomeAssistant) raw() bool {
	return p.attribute != "" || p.pipeline != nil
}

// value returns the configured attribute or the entity state
func (p *HomeAssistant) value() (string, error) {
	if p.attribute != "" {
		return p.conn.GetAttribute(p.entity, p.attribute)
	}

	res, err := p.conn.GetState(p.entity)
	return res.State, err
}

var _ StringGetter = (*HomeAssistant)(nil)

// StringGetter returns the entity state or attribute, processed by the pipeline
func (p *HomeAssistant) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		s, err := p.value()
		if err != nil || p.pipeline == nil {
			return s, err
		}

		b, err := p.pipeline.Process([]byte(s))
		return string(b), err
	}, nil
}

var _ FloatGetter = (*HomeAssistant)(nil)

// FloatGetter returns the entity state as float
func (p *HomeAssistant) FloatGetter() (func() (float64, error), error) {
	if p.raw() {
		return p.getter.FloatGetter()
	}

	return func() (float64, error) {
		return p.conn.GetFloatState(p.entity)
	}, nil
}

var _ IntGetter = (*HomeAssistant)(nil)

// IntGetter returns the entity state as int
func (p *HomeAssistant) IntGetter() (func() (int64, error), error) {
	if p.raw() {
		return p.getter.IntGetter()
	}

	return func() (int64, error) {
		return p.conn.GetIntState(p.entity)
	}, nil
}

var _ BoolGetter = (*HomeAssistant)(nil)

// BoolGetter returns the entity state as bool
func (p *HomeAssistant) BoolGetter() (func() (bool, error), error) {
	if p.raw() {
		return p.getter.BoolGetter()
	}

	return func() (bool, error) {
		return p.conn.GetBoolState(p.entity)
	}, nil
}

var _ BoolSetter = (*HomeAssistant)(nil)

// BoolSetter invokes the entity's turn_on/turn_off service
func (p *HomeAssistant) BoolSetter(param string) (func(bool) error, error) {
	if p.attribute != "" {
		return nil, errors.New("attribute is read-only")
	}

	return func(enable bool) error {
		return p.conn.CallSwitchService(p.entity, enable)
	}, nil
}
