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
			ArrivalSoc: 5, Window: 30 * time.Minute, Unplugged: 90 * time.Minute,
		}, socFallback(t, v))
	})

	t.Run("template passes the block through", func(t *testing.T) {
		// viper lowercases the keys
		v, err := NewFromConfig(context.Background(), "template", map[string]any{
			"template": "offline", "title": "Born", "capacity": 58,
			"socfallback": map[string]any{"arrivalsoc": 10, "window": "1h"},
		})
		require.NoError(t, err)

		res := socFallback(t, v)
		assert.Equal(t, 10.0, res.ArrivalSoc)
		assert.Equal(t, time.Hour, res.Window)
		assert.Equal(t, 90*time.Minute, res.Unplugged, "unset keys take the defaults")
	})
}
