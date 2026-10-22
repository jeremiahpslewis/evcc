package tariff

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomForecastParts(t *testing.T) {
	start := time.Now().Truncate(SlotDuration)
	rates := func(value float64) string {
		b, _ := json.Marshal(api.Rates{{Start: start, End: start.Add(SlotDuration), Value: value}})
		return string(b)
	}

	tf, err := NewConfigurableFromConfig(context.TODO(), map[string]any{
		"tariff":   "load",
		"forecast": map[string]any{"source": "const", "value": rates(1500)},
		"parts": map[string]any{
			"heatpump": map[string]any{"source": "const", "value": rates(900)},
		},
	})
	require.NoError(t, err)

	household, err := tf.Rates()
	require.NoError(t, err)
	assert.Equal(t, 1500.0, household[0].Value)

	parts := tf.(interface{ Parts() map[string]api.Rates })
	require.Eventually(t, func() bool { return len(parts.Parts()["heatpump"]) == 1 }, time.Second, 10*time.Millisecond)
	assert.Equal(t, 900.0, parts.Parts()["heatpump"][0].Value)
}

func TestCustomPartsRequireForecast(t *testing.T) {
	_, err := NewConfigurableFromConfig(context.TODO(), map[string]any{
		"price": map[string]any{"source": "const", "value": "0.3"},
		"parts": map[string]any{"heatpump": map[string]any{"source": "const", "value": "[]"}},
	})
	assert.EqualError(t, err, "parts require a forecast")
}
