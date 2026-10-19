package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeAssistantConfig(t *testing.T) {
	newPlugin := func(t *testing.T, other map[string]any) *HomeAssistant {
		t.Helper()
		p, err := NewHomeAssistantPluginFromConfig(context.TODO(), other)
		require.NoError(t, err)
		return p.(*HomeAssistant)
	}

	t.Run("state", func(t *testing.T) {
		p := newPlugin(t, map[string]any{"uri": "http://ha:8123", "entity": "sensor.power"})
		assert.False(t, p.raw(), "typed state getters keep unit scaling")
		assert.Nil(t, p.pipeline)

		_, err := p.BoolSetter("enable")
		assert.NoError(t, err)
	})

	t.Run("attribute", func(t *testing.T) {
		p := newPlugin(t, map[string]any{
			"uri":       "http://ha:8123",
			"entity":    "sensor.house_load_forecast",
			"attribute": "forecast",
			"jq":        "map({start, end, value: .power}) | tostring",
		})
		assert.True(t, p.raw())
		assert.Equal(t, "forecast", p.attribute)
		assert.NotNil(t, p.pipeline)

		_, err := p.BoolSetter("enable")
		assert.EqualError(t, err, "attribute is read-only")
	})

	t.Run("pipeline on state", func(t *testing.T) {
		p := newPlugin(t, map[string]any{"uri": "http://ha:8123", "entity": "sensor.status", "regex": `(\d+)`})
		assert.True(t, p.raw(), "processed state is parsed as string")
	})

	t.Run("invalid jq", func(t *testing.T) {
		_, err := NewHomeAssistantPluginFromConfig(context.TODO(), map[string]any{"uri": "http://ha:8123", "entity": "sensor.foo", "jq": "map("})
		assert.Error(t, err)
	})
}
