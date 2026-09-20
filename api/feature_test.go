package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqueFeatures(t *testing.T) {
	tc := []struct {
		name     string
		features []Feature
		want     []Feature
	}{
		{"nil", nil, nil},
		{"empty", []Feature{}, []Feature{}},
		{"single", []Feature{Heating}, []Feature{Heating}},
		{"no repeats", []Feature{Heating, Continuous}, []Feature{Heating, Continuous}},
		{"repeats", []Feature{Heating, Heating}, []Feature{Heating}},
		{"first-seen order", []Feature{Continuous, Heating, Continuous, IntegratedDevice, Heating}, []Feature{Continuous, Heating, IntegratedDevice}},
		{"all equal", []Feature{Cacheable, Cacheable, Cacheable}, []Feature{Cacheable}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, UniqueFeatures(tc.features))
		})
	}
}

func TestUniqueFeaturesDoesNotMutateInput(t *testing.T) {
	in := []Feature{Heating, Continuous, Heating, IntegratedDevice}

	res := UniqueFeatures(in)
	require.Equal(t, []Feature{Heating, Continuous, IntegratedDevice}, res)
	assert.Equal(t, []Feature{Heating, Continuous, Heating, IntegratedDevice}, in, "input must not be modified")

	// the result must not share storage with the input
	res[0] = Offline
	assert.Equal(t, Heating, in[0])
}
