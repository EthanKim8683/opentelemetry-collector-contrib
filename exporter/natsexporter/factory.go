// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/metadata"
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		NatsConfig: NatsConfig{
			Endpoint:   nats.DefaultURL,
			TLS:        configtls.NewDefaultClientConfig(),
			Pedantic:   true,
			AuthConfig: AuthConfig{},
		},
		LogsConfig: LogsConfig{
			Subject: "\"otel_logs\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
		MetricsConfig: MetricsConfig{
			Subject: "\"otel_metrics\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
		TracesConfig: TracesConfig{
			Subject: "\"otel_traces\"",
			ResolverConfig: ResolverConfig{
				MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
			},
		},
		QueueBatchConfig: exporterhelper.NewDefaultQueueConfig(),
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	natsCfg := cfg.(*Config)

	exporter, err := newNatsLogsExporter(set, natsCfg)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		exporter.export,
		exporterhelper.WithStart(exporter.start),
		exporterhelper.WithShutdown(exporter.shutdown),
		exporterhelper.WithQueueBatch(natsCfg.QueueBatchConfig, exporterhelper.NewLogsQueueBatchSettings()),
	)
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	natsCfg := cfg.(*Config)

	exporter, err := newNatsMetricsExporter(set, natsCfg)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		exporter.export,
		exporterhelper.WithStart(exporter.start),
		exporterhelper.WithShutdown(exporter.shutdown),
		exporterhelper.WithQueueBatch(natsCfg.QueueBatchConfig, exporterhelper.NewMetricsQueueBatchSettings()),
	)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	natsCfg := cfg.(*Config)

	exporter, err := newNatsTracesExporter(set, natsCfg)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		exporter.export,
		exporterhelper.WithStart(exporter.start),
		exporterhelper.WithShutdown(exporter.shutdown),
		exporterhelper.WithQueueBatch(natsCfg.QueueBatchConfig, exporterhelper.NewTracesQueueBatchSettings()),
	)
}
