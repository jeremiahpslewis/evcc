package util

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeFeaturesUnique(t *testing.T) {
	tc := []struct {
		name string
		in   any
		want []api.Feature
	}{
		{"repeated values", []string{"heating", "heating"}, []api.Feature{api.Heating}},
		{"first-seen order kept", []string{"continuous", "heating", "continuous"}, []api.Feature{api.Continuous, api.Heating}},
		{"mixed casing", []string{"heating", "Heating"}, []api.Feature{api.Heating}},
		{"typed input", []api.Feature{api.Offline, api.Offline, api.Retryable}, []api.Feature{api.Offline, api.Retryable}},
		{"any slice", []any{"streaming", "streaming"}, []api.Feature{api.Streaming}},
		{"empty", []string{}, []api.Feature{}},
		{"no repeats", []string{"average", "cacheable"}, []api.Feature{api.Average, api.Cacheable}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			var cc struct {
				Features []api.Feature `mapstructure:"features"`
			}

			require.NoError(t, DecodeOther(map[string]any{"features": tc.in}, &cc))
			assert.Equal(t, tc.want, cc.Features)
		})
	}
}

// TestDecodeFeaturesValidation ensures normalization does not swallow invalid features
func TestDecodeFeaturesValidation(t *testing.T) {
	var cc struct {
		Features []api.Feature `mapstructure:"features"`
	}

	err := DecodeOther(map[string]any{"features": []string{"heating", "nosuchfeature"}}, &cc)
	require.Error(t, err)
	assert.ErrorContains(t, err, "nosuchfeature")
}

// TestDecodeFeaturesInputImmutable ensures the caller's configuration map and slice survive decoding
func TestDecodeFeaturesInputImmutable(t *testing.T) {
	features := []string{"heating", "heating", "continuous"}
	other := map[string]any{"features": features}

	var cc struct {
		Features []api.Feature `mapstructure:"features"`
	}

	require.NoError(t, DecodeOther(other, &cc))
	assert.Equal(t, []api.Feature{api.Heating, api.Continuous}, cc.Features)

	assert.Equal(t, []string{"heating", "heating", "continuous"}, features, "input slice must not be modified")
	assert.Equal(t, map[string]any{"features": []string{"heating", "heating", "continuous"}}, other, "input map must not be modified")
}

// TestDecodeOtherListsUnchanged ensures only feature lists are treated as sets
func TestDecodeOtherListsUnchanged(t *testing.T) {
	var cc struct {
		Identifiers []string      `mapstructure:"identifiers"`
		Features    []api.Feature `mapstructure:"features"`
	}

	require.NoError(t, DecodeOther(map[string]any{
		"identifiers": []string{"abc", "abc", "def"},
		"features":    []string{"offline", "offline"},
	}, &cc))

	assert.Equal(t, []string{"abc", "abc", "def"}, cc.Identifiers, "non-feature lists keep duplicates")
	assert.Equal(t, []api.Feature{api.Offline}, cc.Features)
}
