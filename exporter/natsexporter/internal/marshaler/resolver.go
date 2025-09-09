// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type genericMarshaler any

type resolver interface {
	resolve(host component.Host) (genericMarshaler, error)
}

type builtinMarshaler struct {
	logsMarshaler    plog.Marshaler
	metricsMarshaler pmetric.Marshaler
	tracesMarshaler  ptrace.Marshaler
}

func (g *builtinMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return g.logsMarshaler.MarshalLogs(ld)
}

func (g *builtinMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	return g.metricsMarshaler.MarshalMetrics(md)
}

func (g *builtinMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return g.tracesMarshaler.MarshalTraces(td)
}

type builtinMarshalerResolver struct {
	genericMarshaler genericMarshaler
}

func (r *builtinMarshalerResolver) resolve(host component.Host) (genericMarshaler, error) {
	return r.genericMarshaler, nil
}

var _ resolver = (*builtinMarshalerResolver)(nil)

type BuiltinMarshalerName string

const (
	OtlpProtoBuiltinMarshalerName BuiltinMarshalerName = "otlp_proto"
	OtlpJSONBuiltinMarshalerName  BuiltinMarshalerName = "otlp_json"
)

func newBuiltinMarshalerResolver(builtinMarshalerName BuiltinMarshalerName) (resolver, error) {
	var genericMarshaler genericMarshaler
	switch builtinMarshalerName {
	case OtlpProtoBuiltinMarshalerName:
		genericMarshaler = &builtinMarshaler{
			logsMarshaler:    &plog.ProtoMarshaler{},
			metricsMarshaler: &pmetric.ProtoMarshaler{},
			tracesMarshaler:  &ptrace.ProtoMarshaler{},
		}
	case OtlpJSONBuiltinMarshalerName:
		genericMarshaler = &builtinMarshaler{
			logsMarshaler:    &plog.JSONMarshaler{},
			metricsMarshaler: &pmetric.JSONMarshaler{},
			tracesMarshaler:  &ptrace.JSONMarshaler{},
		}
	default:
		return nil, fmt.Errorf("unsupported built-in marshaler: %s", builtinMarshalerName)
	}

	return &builtinMarshalerResolver{
		genericMarshaler: genericMarshaler,
	}, nil
}

type encodingExtensionResolver struct {
	id component.ID
}

func (r *encodingExtensionResolver) resolve(host component.Host) (genericMarshaler, error) {
	encodingExtension, ok := host.GetExtensions()[r.id]
	if !ok {
		return nil, fmt.Errorf("encoding extension not found: %s", r.id)
	}
	return encodingExtension, nil
}

var _ resolver = (*encodingExtensionResolver)(nil)

func newEncodingExtensionResolver(encodingExtensionName []byte) (resolver, error) {
	var id component.ID
	if err := id.UnmarshalText(encodingExtensionName); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding extension name: %w", err)
	}

	return &encodingExtensionResolver{
		id: id,
	}, nil
}
