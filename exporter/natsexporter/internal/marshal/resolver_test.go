// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshal

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestBuiltinMarshalerResolver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		builtinMarshalerName BuiltinMarshalerName
		wantLogsMarshaler    plog.Marshaler
		wantMetricsMarshaler pmetric.Marshaler
		wantTracesMarshaler  ptrace.Marshaler
		wantError            error
	}{
		{
			name:                 "resolves JSON marshaler for OtlpJSONBuiltinMarshalerName",
			builtinMarshalerName: OtlpJSONBuiltinMarshalerName,
			wantLogsMarshaler:    &plog.JSONMarshaler{},
			wantMetricsMarshaler: &pmetric.JSONMarshaler{},
			wantTracesMarshaler:  &ptrace.JSONMarshaler{},
			wantError:            nil,
		},
		{
			name:                 "resolves Protobuf marshaler for OtlpProtoBuiltinMarshalerName",
			builtinMarshalerName: OtlpProtoBuiltinMarshalerName,
			wantLogsMarshaler:    &plog.ProtoMarshaler{},
			wantMetricsMarshaler: &pmetric.ProtoMarshaler{},
			wantTracesMarshaler:  &ptrace.ProtoMarshaler{},
			wantError:            nil,
		},
		{
			name:                 "returns error for unsupported built-in marshaler",
			builtinMarshalerName: "unsupported",
			wantError:            errors.New("unsupported built-in marshaler"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewBuiltinMarshalerResolver(tt.builtinMarshalerName)
			if tt.wantError != nil {
				assert.ErrorContains(t, err, tt.wantError.Error())
			} else {
				assert.NoError(t, err)

				genericMarshaler, err := resolver.Resolve(componenttest.NewNopHost())
				assert.NoError(t, err)

				builtinMarshaler, ok := genericMarshaler.(*builtinMarshaler)
				assert.True(t, ok)

				assert.IsType(t, tt.wantLogsMarshaler, builtinMarshaler.logsMarshaler)
				assert.IsType(t, tt.wantMetricsMarshaler, builtinMarshaler.metricsMarshaler)
				assert.IsType(t, tt.wantTracesMarshaler, builtinMarshaler.tracesMarshaler)
			}
		})
	}
}

type fakeHost struct {
	extensions map[component.ID]component.Component
}

func (h *fakeHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

var _ component.Host = (*fakeHost)(nil)

type fakeExtension struct {
	component.Component
}

var _ component.Component = (*fakeExtension)(nil)

func TestEncodingExtensionResolver(t *testing.T) {
	t.Parallel()

	wantExtension := &fakeExtension{}
	host := &fakeHost{
		extensions: map[component.ID]component.Component{
			component.NewID(component.MustNewType("extension")): wantExtension,
		},
	}

	t.Run("resolves extension if found", func(t *testing.T) {
		resolver, err := NewEncodingExtensionResolver([]byte("extension"))
		assert.NoError(t, err)

		haveExtension, err := resolver.Resolve(host)
		assert.NoError(t, err)
		assert.Equal(t, wantExtension, haveExtension)
	})

	t.Run("returns error if extension not found", func(t *testing.T) {
		resolver, err := NewEncodingExtensionResolver([]byte("missing"))
		assert.NoError(t, err)

		_, err = resolver.Resolve(host)
		assert.ErrorContains(t, err, "encoding extension not found")
	})
}
