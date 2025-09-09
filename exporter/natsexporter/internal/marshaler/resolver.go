// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type GenericMarshaler any

type BuiltinMarshalerName string

const (
	OtlpProtoBuiltinMarshalerName BuiltinMarshalerName = "otlp_proto"
	OtlpJSONBuiltinMarshalerName  BuiltinMarshalerName = "otlp_json"
)

type Resolver interface {
	Resolve(host component.Host) (GenericMarshaler, error)
}

type resolverConfig interface {
	Validate() error
	resolver() Resolver
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
	genericMarshaler GenericMarshaler
}

func (r *builtinMarshalerResolver) Resolve(host component.Host) (GenericMarshaler, error) {
	return r.genericMarshaler, nil
}

var _ Resolver = (*builtinMarshalerResolver)(nil)

type builtinMarshalerResolverConfig struct {
	builtinMarshalerName BuiltinMarshalerName

	builtinMarshalerResolver *builtinMarshalerResolver
}

func (c *builtinMarshalerResolverConfig) Validate() error {
	var genericMarshaler GenericMarshaler
	switch c.builtinMarshalerName {
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
		return fmt.Errorf("unsupported built-in marshaler: %s", c.builtinMarshalerName)
	}

	c.builtinMarshalerResolver = &builtinMarshalerResolver{
		genericMarshaler: genericMarshaler,
	}
	return nil
}

func (c *builtinMarshalerResolverConfig) resolver() Resolver {
	return c.builtinMarshalerResolver
}

var _ resolverConfig = (*builtinMarshalerResolverConfig)(nil)

type encodingExtensionResolver struct {
	id component.ID
}

func (r *encodingExtensionResolver) Resolve(host component.Host) (GenericMarshaler, error) {
	encodingExtension, ok := host.GetExtensions()[r.id]
	if !ok {
		return nil, fmt.Errorf("encoding extension not found: %s", r.id)
	}
	return encodingExtension, nil
}

var _ Resolver = (*encodingExtensionResolver)(nil)

type encodingExtensionResolverConfig struct {
	encodingExtensionName []byte

	encodingExtensionResolver *encodingExtensionResolver
}

func (c *encodingExtensionResolverConfig) Validate() error {
	var id component.ID
	if err := id.UnmarshalText(c.encodingExtensionName); err != nil {
		return fmt.Errorf("failed to unmarshal encoding extension name: %w", err)
	}

	c.encodingExtensionResolver = &encodingExtensionResolver{
		id: id,
	}
	return nil
}

func (c *encodingExtensionResolverConfig) resolver() Resolver {
	return c.encodingExtensionResolver
}

var _ resolverConfig = (*encodingExtensionResolverConfig)(nil)

type ResolverConfig struct {
	builtinMarshalerResolverConfig  *builtinMarshalerResolverConfig
	encodingExtensionResolverConfig *encodingExtensionResolverConfig

	resolver Resolver
}

func (c *ResolverConfig) Validate() error {
	if c.builtinMarshalerResolverConfig != nil &&
		c.encodingExtensionResolverConfig != nil {
		return errors.New("marshaler configured more than once")
	}

	var resolverConfig resolverConfig
	if c.builtinMarshalerResolverConfig != nil {
		resolverConfig = c.builtinMarshalerResolverConfig
	} else if c.encodingExtensionResolverConfig != nil {
		resolverConfig = c.encodingExtensionResolverConfig
	} else {
		return errors.New("marshaler not configured")
	}
	if err := resolverConfig.Validate(); err != nil {
		return err
	}

	c.resolver = resolverConfig.resolver()
	return nil
}

func NewResolver(cfg *ResolverConfig) (Resolver, error) {
	if cfg.resolver == nil {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}
	return cfg.resolver, nil
}
