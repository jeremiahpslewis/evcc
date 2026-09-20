package templates

import (
	"maps"
	"regexp"
	"slices"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// featuresKeyRe matches a top-level features mapping key
var featuresKeyRe = regexp.MustCompile(`(?m)^features:`)

func TestFeatureList(t *testing.T) {
	tc := []struct {
		name   string
		values []any
		want   []string
	}{
		{"empty", nil, []string{}},
		{"nil entries", []any{nil, nil}, []string{}},
		{"disabled options", []any{false, "", nil}, []string{}},
		{"plain names", []any{"heating", "continuous"}, []string{"heating", "continuous"}},
		{"repeated names", []any{"heating", "heating"}, []string{"heating"}},
		{"nested lists", []any{[]any{"cacheable", "average"}, "cacheable"}, []string{"cacheable", "average"}},
		{"typed nested list", []any{[]string{"streaming"}, "streaming"}, []string{"streaming"}},
		{"first-seen order", []any{"b", []any{"a", "b"}, "c"}, []string{"b", "a", "c"}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, featureList(tc.values...))
		})
	}
}

func TestFeatureListDoesNotMutateInput(t *testing.T) {
	in := []string{"heating", "heating"}

	assert.Equal(t, []string{"heating"}, featureList(in))
	assert.Equal(t, []string{"heating", "heating"}, in, "input must not be modified")
}

// featureParams returns the template's optional feature toggles
func featureParams(tmpl Template) []string {
	var res []string
	for _, p := range tmpl.Params {
		if p.Type != TypeBool {
			continue
		}
		if _, err := api.FeatureString(p.Name); err == nil {
			res = append(res, p.Name)
		}
	}
	return res
}

// renderValues builds a renderable parameter set for the given template and usage
func renderValues(tmpl Template, usage string) map[string]any {
	values := tmpl.Defaults(RenderModeUnitTest)

	if values[ParamModbus] != nil {
		choices := tmpl.ModbusChoices()
		switch {
		case slices.Contains(choices, ModbusChoiceTCPIP):
			values[ModbusKeyTCPIP] = true
		case slices.Contains(choices, ModbusChoiceUDP):
			values[ModbusKeyUDP] = true
		default:
			values[ModbusKeyRS485TCPIP] = true
		}
		tmpl.ModbusValues(RenderModeUnitTest, values)
	}

	values["template"] = tmpl.Template
	values["host"] = "localhost"

	if usage != "" {
		values[ParamUsage] = usage
	}

	return values
}

// assertUniqueFeatures renders the template and verifies the emitted feature list
func assertUniqueFeatures(t *testing.T, class Class, tmpl Template, values map[string]any) {
	t.Helper()

	b, _, err := tmpl.RenderResult(class, RenderModeInstance, values)
	require.NoError(t, err, string(b))

	assert.LessOrEqual(t, len(featuresKeyRe.FindAll(b, -1)), 1, "at most one features key:\n%s", string(b))

	// duplicate mapping keys are a parse error, so this also guards the rendered document
	var instance struct {
		Features []api.Feature  `yaml:"features"`
		Other    map[string]any `yaml:",inline"`
	}
	require.NoError(t, yaml.Unmarshal(b, &instance), string(b))

	assert.Equal(t, api.UniqueFeatures(instance.Features), instance.Features, "features must not repeat:\n%s", string(b))
}

// TestTemplateFeaturesUnique renders every template of every feature-aware class with
// no, each single and all optional features enabled
func TestTemplateFeaturesUnique(t *testing.T) {
	t.Parallel()

	for _, class := range []Class{Charger, Meter, Vehicle, Tariff} {
		for _, tmpl := range ByClass(class, WithDeprecated()) {
			usages := tmpl.Usages()
			if len(usages) == 0 {
				usages = []string{""}
			}

			params := featureParams(tmpl)

			// no optional feature, each single one, and all combined
			combinations := [][]string{nil}
			for _, p := range params {
				combinations = append(combinations, []string{p})
			}
			if len(params) > 1 {
				combinations = append(combinations, params)
			}

			for _, usage := range usages {
				name := tmpl.Template
				if usage != "" {
					name += "/" + usage
				}

				t.Run(class.String()+"/"+name, func(t *testing.T) {
					t.Parallel()

					for _, enabled := range combinations {
						values := maps.Clone(renderValues(tmpl, usage))
						for _, p := range enabled {
							values[p] = true
						}
						assertUniqueFeatures(t, class, tmpl, values)
					}
				})
			}
		}
	}
}

// TestVoltcastFeatures covers the built-in/optional overlap that duplicated the features key
func TestVoltcastFeatures(t *testing.T) {
	tmpl, err := ByName(Tariff, "voltcast")
	require.NoError(t, err)

	for _, average := range []bool{false, true} {
		values := renderValues(tmpl, "")
		values["average"] = average

		b, _, err := tmpl.RenderResult(Tariff, RenderModeInstance, values)
		require.NoError(t, err, string(b))

		require.Len(t, featuresKeyRe.FindAll(b, -1), 1, "exactly one features key:\n%s", string(b))

		var instance struct {
			Features []api.Feature  `yaml:"features"`
			Other    map[string]any `yaml:",inline"`
		}
		require.NoError(t, yaml.Unmarshal(b, &instance), string(b))

		want := []api.Feature{api.Cacheable}
		if average {
			want = append(want, api.Average)
		}
		assert.Equal(t, want, instance.Features)
	}
}

// TestDuplicateFeatureKeysRejected documents that duplicate mapping keys stay a parse error
func TestDuplicateFeatureKeysRejected(t *testing.T) {
	var instance Instance
	err := yaml.Unmarshal([]byte("type: custom\nfeatures: [\"cacheable\"]\nfeatures:\n- average\n"), &instance)
	require.Error(t, err)
	assert.ErrorContains(t, err, `mapping key "features" already defined`)
}
