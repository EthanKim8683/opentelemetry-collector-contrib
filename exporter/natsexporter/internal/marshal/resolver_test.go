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
			name:                 "resolves to JSON marshaler for OtlpJSONBuiltinMarshalerName",
			builtinMarshalerName: OtlpJSONBuiltinMarshalerName,
			wantLogsMarshaler:    &plog.JSONMarshaler{},
			wantMetricsMarshaler: &pmetric.JSONMarshaler{},
			wantTracesMarshaler:  &ptrace.JSONMarshaler{},
			wantError:            nil,
		},
		{
			name:                 "resolves to Protobuf marshaler for OtlpProtoBuiltinMarshalerName",
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

				logs := plog.NewLogs()
				metrics := pmetric.NewMetrics()
				traces := ptrace.NewTraces()

				wantLogs, err := tt.wantLogsMarshaler.MarshalLogs(logs)
				assert.NoError(t, err)
				wantMetrics, err := tt.wantMetricsMarshaler.MarshalMetrics(metrics)
				assert.NoError(t, err)
				wantTraces, err := tt.wantTracesMarshaler.MarshalTraces(traces)
				assert.NoError(t, err)

				haveLogs, err := builtinMarshaler.MarshalLogs(logs)
				assert.NoError(t, err)
				haveMetrics, err := builtinMarshaler.MarshalMetrics(metrics)
				assert.NoError(t, err)
				haveTraces, err := builtinMarshaler.MarshalTraces(traces)
				assert.NoError(t, err)

				assert.Equal(t, wantLogs, haveLogs)
				assert.Equal(t, wantMetrics, haveMetrics)
				assert.Equal(t, wantTraces, haveTraces)
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
