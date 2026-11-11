package vehicle

import (
	"context"
	"maps"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewFromTemplateConfig)
}

// socFallbackKey is a vehicle setting outside the templates, passed to the rendered vehicle as is
const socFallbackKey = "socFallback"

func NewFromTemplateConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	other = maps.Clone(other)

	var socFallback any
	for k, v := range other {
		if strings.EqualFold(k, socFallbackKey) {
			socFallback = v
			delete(other, k)
		}
	}

	instance, err := templates.RenderInstance(templates.Vehicle, other)
	if err != nil {
		return nil, err
	}

	if socFallback != nil {
		instance.Other[socFallbackKey] = socFallback
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
