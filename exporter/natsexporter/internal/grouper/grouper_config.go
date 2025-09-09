package grouper

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type GrouperConfig[T any] interface {
	NewGrouper(telemetrySettings component.TelemetrySettings) (Grouper[T], error)
	Validate() error
}

type LogsGrouperConfig struct {
	Subject string `mapstructure:"subject"`
}

func (c *LogsGrouperConfig) NewGrouper(telemetrySettings component.TelemetrySettings) (Grouper[plog.Logs], error) {
	return newLogsGrouper(c.Subject, telemetrySettings)
}

func (c *LogsGrouperConfig) Validate() error {
	_, err := c.NewGrouper(componenttest.NewNopTelemetrySettings())
	return err
}

var _ GrouperConfig[plog.Logs] = (*LogsGrouperConfig)(nil)

func NewDefaultLogsGrouperConfig() LogsGrouperConfig {
	return LogsGrouperConfig{
		Subject: "\"otel_logs\"",
	}
}

type MetricsGrouperConfig struct {
	Subject string `mapstructure:"subject"`
}

func (c *MetricsGrouperConfig) NewGrouper(telemetrySettings component.TelemetrySettings) (Grouper[pmetric.Metrics], error) {
	return newMetricsGrouper(c.Subject, telemetrySettings)
}

func (c *MetricsGrouperConfig) Validate() error {
	_, err := c.NewGrouper(componenttest.NewNopTelemetrySettings())
	return err
}

var _ GrouperConfig[pmetric.Metrics] = (*MetricsGrouperConfig)(nil)

func NewDefaultMetricsGrouperConfig() MetricsGrouperConfig {
	return MetricsGrouperConfig{
		Subject: "\"otel_metrics\"",
	}
}

type TracesGrouperConfig struct {
	Subject string `mapstructure:"subject"`
}

func (c *TracesGrouperConfig) NewGrouper(telemetrySettings component.TelemetrySettings) (Grouper[ptrace.Traces], error) {
	return newTracesGrouper(c.Subject, telemetrySettings)
}

func (c *TracesGrouperConfig) Validate() error {
	_, err := c.NewGrouper(componenttest.NewNopTelemetrySettings())
	return err
}

var _ GrouperConfig[ptrace.Traces] = (*TracesGrouperConfig)(nil)

func NewDefaultTracesGrouperConfig() TracesGrouperConfig {
	return TracesGrouperConfig{
		Subject: "\"otel_traces\"",
	}
}
