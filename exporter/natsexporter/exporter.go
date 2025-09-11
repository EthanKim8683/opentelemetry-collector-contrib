// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
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
	options   *nats.Options
	grouper   grouper.Grouper[T]
	marshaler *marshaler.Marshaler[T]
	publisher publisher.Publisher
}

func newNatsExporter[T any](
	set exporter.Settings,
	cfg *Config,
	options *nats.Options,
	grouper grouper.Grouper[T],
	marshaler *marshaler.Marshaler[T],
) *natsExporter[T] {
	return &natsExporter[T]{
		set:       set,
		cfg:       cfg,
		options:   options,
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

	var wg sync.WaitGroup
	errCh := make(chan error)
	for _, group := range groups {
		var (
			subject = group.Subject
			data    = group.Data
		)

		wg.Go(func() {
			bytes, err := e.marshaler.Marshal(data)
			if err != nil {
				errCh <- err
				return
			}

			err = e.publisher.Publish(ctx, subject, bytes)
			if err != nil {
				errCh <- err
				return
			}
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		errs = multierr.Append(errs, err)
	}

	return errs
}

func (e *natsExporter[T]) shutdown(_ context.Context) error {
	return e.publisher.Shutdown()
}

func newNatsLogsExporter(set exporter.Settings, cfg *Config) (*natsExporter[plog.Logs], error) {
	var errs error
	grouper, err := cfg.Logs.GrouperConfig.NewGrouper(set.TelemetrySettings)
	errs = multierr.Append(errs, err)
	marshaler, err := cfg.Logs.MarshalerConfig.NewMarshaler()
	errs = multierr.Append(errs, err)
	return newNatsExporter(set, cfg, options, grouper, marshaler), errs
}

func newNatsMetricsExporter(set exporter.Settings, cfg *Config) (*natsExporter[pmetric.Metrics], error) {
	// var errs error

	// grouper, err := grouper.NewMetricsGrouper(cfg.Metrics.Subject, set.TelemetrySettings)
	// errs = multierr.Append(errs, err)

	// resolver, err := createResolver((*SignalConfig)(&cfg.Metrics))
	// errs = multierr.Append(errs, err)
	// marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalMetrics)

	// return newNatsExporter(set, cfg, grouper, marshaler), errs

	return nil, nil
}

func newNatsTracesExporter(set exporter.Settings, cfg *Config) (*natsExporter[ptrace.Traces], error) {
	// var errs error

	// grouper, err := grouper.NewTracesGrouper(cfg.Traces.Subject, set.TelemetrySettings)
	// errs = multierr.Append(errs, err)

	// resolver, err := createResolver((*SignalConfig)(&cfg.Traces))
	// errs = multierr.Append(errs, err)
	// marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalTraces)

	// return newNatsExporter(set, cfg, grouper, marshaler), errs

	return nil, nil
}
