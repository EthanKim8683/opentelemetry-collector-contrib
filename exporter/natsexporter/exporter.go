// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/grouper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publisher"
)

type natsExporter[T any] struct {
	set       exporter.Settings
	cfg       *Config
	grouper   grouper.Grouper[T]
	marshaler marshaler.Marshaler[T]
	publisher publisher.Publisher
}

func newNatsExporter[T any](
	set exporter.Settings,
	cfg *Config,
	grouper grouper.Grouper[T],
	marshaler marshaler.Marshaler[T],
) *natsExporter[T] {
	return &natsExporter[T]{
		set:       set,
		cfg:       cfg,
		grouper:   grouper,
		marshaler: marshaler,
	}
}

func (e *natsExporter[T]) start(_ context.Context, host component.Host) error {
	var errs error
	errs = multierr.Append(errs, e.marshaler.Resolve(host))
	errs = multierr.Append(errs, e.publisher.Start())
	return errs
}

func (e *natsExporter[T]) export(ctx context.Context, data T) error {
	var errs error

	groups, err := e.grouper.Group(ctx, data)
	errs = multierr.Append(errs, err)

	for _, group := range groups {
		bytes, err := e.marshaler.Marshal(group.Data)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}

		err = e.publisher.Publish(ctx, group.Subject, bytes)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

func (e *natsExporter[T]) shutdown(_ context.Context) error {
	return e.publisher.Shutdown()
}

func createResolver(cfg *SignalConfig) (marshaler.Resolver, error) {
	if cfg.BuiltinMarshalerName != "" {
		return marshaler.NewBuiltinMarshalerResolver(cfg.BuiltinMarshalerName)
	} else if cfg.EncodingExtensionName != "" {
		return marshaler.NewEncodingExtensionResolver(cfg.EncodingExtensionName)
	} else {
		return nil, errors.New("no built-in marshaler or encoding extension configured")
	}
}

func newNatsLogsExporter(set exporter.Settings, cfg *Config) (*natsExporter[plog.Logs], error) {
	var errs error

	grouper, err := grouper.NewLogsGrouper(cfg.Logs.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := createResolver((*SignalConfig)(&cfg.Logs))
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalLogs)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}

func newNatsMetricsExporter(set exporter.Settings, cfg *Config) (*natsExporter[pmetric.Metrics], error) {
	var errs error

	grouper, err := grouper.NewMetricsGrouper(cfg.Metrics.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := createResolver((*SignalConfig)(&cfg.Metrics))
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalMetrics)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}

func newNatsTracesExporter(set exporter.Settings, cfg *Config) (*natsExporter[ptrace.Traces], error) {
	var errs error

	grouper, err := grouper.NewTracesGrouper(cfg.Traces.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := createResolver((*SignalConfig)(&cfg.Traces))
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalTraces)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}
