// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/metadata"
)

const (
	defaultLogsSubject      = "\"otel_logs\""
	defaultLogsMarshaler    = marshaler.OtlpProtoBuiltinMarshalerName
	defaultMetricsSubject   = "\"otel_metrics\""
	defaultMetricsMarshaler = marshaler.OtlpProtoBuiltinMarshalerName
	defaultTracesSubject    = "\"otel_spans\""
	defaultTracesMarshaler  = marshaler.OtlpProtoBuiltinMarshalerName
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		newDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
	)
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
