package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestEmbedUniqueFeatures(t *testing.T) {
	var embed embed

	other := map[string]any{
		"features": []string{"streaming", "streaming", "welcomecharge"},
	}

	require.NoError(t, util.DecodeOther(other, &embed))
	require.Equal(t, []api.Feature{api.Streaming, api.WelcomeCharge}, embed.Features())
}

// TestWrapperFeatures covers the overlap of configured and wrapper-supplied features
func TestWrapperFeatures(t *testing.T) {
	for _, features := range [][]string{
		nil,
		{"offline"},
		{"offline", "retryable"},
		{"streaming", "offline", "streaming"},
	} {
		other := map[string]any{"title": "foo"}
		if features != nil {
			other["features"] = features
		}

		v := NewWrapper("foo", "bar", other, api.ErrNotAvailable)
		require.Equal(t, api.UniqueFeatures(v.Features()), v.Features(), "features: %v", features)
		require.Contains(t, v.Features(), api.Offline)
		require.Contains(t, v.Features(), api.Retryable)
	}
}
