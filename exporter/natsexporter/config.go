// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/grouper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publisher"
)

type PipelineConfig[
	T any,
	GrouperConfig grouper.GrouperConfig[T],
	MarshalerConfig marshaler.MarshalerConfig[T],
] struct {
	GrouperConfig   GrouperConfig   `mapstructure:",squash"`
	MarshalerConfig MarshalerConfig `mapstructure:",squash"`
}

func (c *PipelineConfig[T, GrouperConfig, MarshalerConfig]) Validate() error {
	var errs error
	errs = multierr.Append(errs, c.GrouperConfig.Validate())
	errs = multierr.Append(errs, c.MarshalerConfig.Validate())
	return errs
}

type LogsConfig PipelineConfig[
	plog.Logs,
	*grouper.LogsGrouperConfig,
	*marshaler.LogsMarshalerConfig,
]

type MetricsConfig PipelineConfig[
	pmetric.Metrics,
	*grouper.MetricsGrouperConfig,
	*marshaler.MetricsMarshalerConfig,
]

type TracesConfig PipelineConfig[
	ptrace.Traces,
	*grouper.TracesGrouperConfig,
	*marshaler.TracesMarshalerConfig,
]

type Config struct {
	ConnectorConfig publisher.ConnectorConfig `mapstructure:",squash"`

	Logs    LogsConfig    `mapstructure:"logs"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Traces  TracesConfig  `mapstructure:"traces"`

	BackOffConfig    configretry.BackOffConfig       `mapstructure:"retry_on_failure"`
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
}

func newDefaultConfig() Config {
	return Config{
		ConnectorConfig: publisher.NewDefaultConnectorConfig(),
		Logs:            newDefaultLogsConfig(),
		Metrics:         newDefaultMetricsConfig(),
		Traces:          newDefaultTracesConfig(),
	}
}

func (c *Config) Validate() error {
	var errs error
	errs = multierr.Append(errs, c.ConnectorConfig.Validate())
	errs = multierr.Append(errs, c.Logs.Validate())
	errs = multierr.Append(errs, c.Metrics.Validate())
	errs = multierr.Append(errs, c.Traces.Validate())
	return errs
}
