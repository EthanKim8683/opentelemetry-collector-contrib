package marshaler

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type MarshalerConfig[T any] interface {
	NewMarshaler() (*Marshaler[T], error)
	Validate() error
}

type LogsMarshalerConfig struct {
	resolverConfig `mapstructure:",squash"`
}

func (c *LogsMarshalerConfig) NewMarshaler() (*Marshaler[plog.Logs], error) {
	resolver, err := c.resolverConfig.newResolver()
	if err != nil {
		return nil, err
	}

	return newMarshaler(resolver, pickMarshalLogs), nil
}

func (c *LogsMarshalerConfig) Validate() error {
	_, err := c.NewMarshaler()
	return err
}

var _ MarshalerConfig[plog.Logs] = (*LogsMarshalerConfig)(nil)

func NewDefaultLogsMarshalerConfig() LogsMarshalerConfig {
	return LogsMarshalerConfig{
		resolverConfig: newDefaultResolverConfig(),
	}
}

type MetricsMarshalerConfig struct {
	resolverConfig `mapstructure:",squash"`
}

func (c *MetricsMarshalerConfig) NewMarshaler() (*Marshaler[pmetric.Metrics], error) {
	resolver, err := c.resolverConfig.newResolver()
	if err != nil {
		return nil, err
	}

	return newMarshaler(resolver, pickMarshalMetrics), nil
}

func (c *MetricsMarshalerConfig) Validate() error {
	_, err := c.NewMarshaler()
	return err
}

var _ MarshalerConfig[pmetric.Metrics] = (*MetricsMarshalerConfig)(nil)

func NewDefaultMetricsMarshalerConfig() MetricsMarshalerConfig {
	return MetricsMarshalerConfig{
		resolverConfig: newDefaultResolverConfig(),
	}
}

type TracesMarshalerConfig struct {
	resolverConfig `mapstructure:",squash"`
}

func (c *TracesMarshalerConfig) NewMarshaler() (*Marshaler[ptrace.Traces], error) {
	resolver, err := c.resolverConfig.newResolver()
	if err != nil {
		return nil, err
	}

	return newMarshaler(resolver, pickMarshalTraces), nil
}

func (c *TracesMarshalerConfig) Validate() error {
	_, err := c.NewMarshaler()
	return err
}

var _ MarshalerConfig[ptrace.Traces] = (*TracesMarshalerConfig)(nil)

func NewDefaultTracesMarshalerConfig() TracesMarshalerConfig {
	return TracesMarshalerConfig{
		resolverConfig: newDefaultResolverConfig(),
	}
}
