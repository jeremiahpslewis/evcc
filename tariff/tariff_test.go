package tariff

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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

func TestCustomForecastGeneratedAt(t *testing.T) {
	start := time.Now().Truncate(SlotDuration)
	rates := func(value float64) string {
		b, _ := json.Marshal(api.Rates{{Start: start, End: start.Add(SlotDuration), Value: value}})
		return string(b)
	}

	// read with its generatedAt as configured
	tf, err := NewConfigurableFromConfig(context.TODO(), map[string]any{
		"tariff":      "load",
		"forecast":    map[string]any{"source": "const", "value": rates(1500)},
		"generatedAt": map[string]any{"source": "const", "value": "2026-10-29T10:15:03+00:00"},
	})
	require.NoError(t, err)
	_, issue, err := tf.(*Tariff).ForecastIssue()
	require.NoError(t, err)
	assert.Equal(t, "2026-10-29T10:15:03+00:00", issue)

	// the forecast is issued anew while the first read runs: read again
	issued := []string{"A", "B", "B", "B", "B"}
	value := 1500.0
	tr := &Tariff{
		embed: new(embed),
		log:   util.NewLogger("test"),
		data:  util.NewMonitor[api.Rates](time.Hour),
		forecastG: func() (string, error) {
			value += 100
			return rates(value), nil
		},
		generatedAtG: func() (string, error) {
			s := issued[0]
			issued = issued[1:]
			return s, nil
		},
	}

	require.NoError(t, tr.Refresh())
	data, issue, err := tr.ForecastIssue()
	require.NoError(t, err)
	assert.Equal(t, "B", issue)
	assert.Equal(t, 1700.0, data[0].Value)

	issue, err = tr.GeneratedAt()
	require.NoError(t, err)
	assert.Equal(t, "B", issue)

	_, err = NewConfigurableFromConfig(context.TODO(), map[string]any{
		"price":       map[string]any{"source": "const", "value": "0.3"},
		"generatedAt": map[string]any{"source": "const", "value": "x"},
	})
	assert.EqualError(t, err, "generatedAt requires a forecast")
}
