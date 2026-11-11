package vehicle

import (
	"context"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSocFallbackSettings(t *testing.T) {
	socFallback := func(t *testing.T, v api.Vehicle) api.SocFallback {
		t.Helper()
		d, ok := api.Cap[api.SocFallbackDescriber](v)
		require.True(t, ok)
		return d.SocFallback()
	}

	t.Run("defaults", func(t *testing.T) {
		v, err := NewFromConfig(context.Background(), "template", map[string]any{
			"template": "offline", "title": "Born", "capacity": 58,
		})
		require.NoError(t, err)

		assert.Equal(t, api.SocFallback{
			Range: 150, Reserve: 10, ArrivalSoc: 5, Window: 30 * time.Minute, Unplugged: 90 * time.Minute, Consumption: 21,
		}, socFallback(t, v))
		assert.InDelta(t, 165, socFallback(t, v).FloorRange(), 1e-9)
	})

	t.Run("template passes the block through", func(t *testing.T) {
		// viper lowercases the keys
		v, err := NewFromConfig(context.Background(), "template", map[string]any{
			"template": "offline", "title": "Born", "capacity": 58,
			"socfallback": map[string]any{"range": 200, "window": "1h"},
		})
		require.NoError(t, err)

		res := socFallback(t, v)
		assert.Equal(t, 200.0, res.Range)
		assert.Equal(t, time.Hour, res.Window)
		assert.Equal(t, 21.0, res.Consumption, "unset keys take the defaults")
	})
}
