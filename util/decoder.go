package util

import (
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
)

var (
	validate     = validator.New()
	configLog    = NewLogger("config")
	featuresType = reflect.TypeFor[[]api.Feature]()
)

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other, cc any) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
			uniqueFeaturesHook,
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	if err := decoder.Decode(other); err != nil {
		return &ConfigError{err}
	}

	// validate structs
	if rv := reflect.ValueOf(cc); rv.Kind() == reflect.Struct || rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct {
		return validate.Struct(cc)
	}

	return nil
}

// uniqueFeaturesHook normalizes every api.Feature list to an ordered set so that
// templates, handwritten yaml and programmatic configuration behave identically.
// Values that don't resolve to a feature are passed through for the decoder to reject.
var uniqueFeaturesHook = mapstructure.DecodeHookFuncType(func(_, to reflect.Type, data any) (any, error) {
	rv := reflect.ValueOf(data)
	if to != featuresType || !rv.IsValid() || rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return data, nil
	}

	features := make([]api.Feature, 0, rv.Len())
	for i := range rv.Len() {
		f, err := toFeature(rv.Index(i).Interface())
		if err != nil {
			return data, nil
		}
		features = append(features, f)
	}

	unique := api.UniqueFeatures(features)
	if len(unique) < len(features) {
		configLog.WARN.Printf("ignoring duplicate features: %s", strings.Join(repeatedFeatures(features), ", "))
	}

	return unique, nil
})

// toFeature resolves a raw configuration value to a feature
func toFeature(v any) (api.Feature, error) {
	switch v := v.(type) {
	case api.Feature:
		return v, nil
	case string:
		return api.FeatureString(v)
	default:
		return 0, errors.New("not a feature")
	}
}

// repeatedFeatures returns the names of features given more than once, in first-seen order
func repeatedFeatures(features []api.Feature) []string {
	var res []string

	seen := make(map[api.Feature]bool, len(features))
	for _, f := range features {
		if name := strings.ToLower(f.String()); seen[f] && !slices.Contains(res, name) {
			res = append(res, name)
		}
		seen[f] = true
	}

	return res
}

// ConfigError wraps yaml configuration errors from mapstructure
type ConfigError struct {
	err error
}

func NewConfigError(err error) error {
	return &ConfigError{err}
}

func (e ConfigError) Error() string {
	return e.err.Error()
}

func (e ConfigError) Unwrap() error {
	return e.err
}
