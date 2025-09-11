// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter"

import (
	"context"
	"sync"

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
	marshaler *marshaler.Marshaler[T]
	publisher publisher.Publisher
}

func newNatsOptions(cfg *natsConfig, ctx context.Context) (*publisher.NatsOptions, error) {
	var errs error

	tlsConfig, err := cfg.TLS.LoadTLSConfig(ctx)
	errs = multierr.Append(errs, err)

	var natsOptions publisher.NatsOptions
	natsOptions.SetURL(cfg.Endpoint)
	natsOptions.SetTLS(tlsConfig)
	natsOptions.SetPedantic(cfg.Pedantic)
	natsOptions.SetToken(cfg.Auth.Token.Token)
	natsOptions.SetUser(cfg.Auth.User.Username, cfg.Auth.User.Password)
	errs = multierr.Append(errs, natsOptions.SetNkey(cfg.Auth.Nkey.Seed))
	errs = multierr.Append(errs, natsOptions.SetNkeyJWT(cfg.Auth.NkeyJWT.JWT, cfg.Auth.NkeyJWT.Seed))
	errs = multierr.Append(errs, natsOptions.SetNkeyUserFile(cfg.Auth.NkeyUserFile.UserFilePath))
	if errs != nil {
		return nil, errs
	}
	return &natsOptions, nil
}

func newJetStreamOptions(cfg *jetStreamConfig) *publisher.JetStreamOptions {
	var jetStreamOptions publisher.JetStreamOptions
	if cfg.RetryWait != nil {
		jetStreamOptions.SetRetryWait(*cfg.RetryWait)
	}
	if cfg.RetryAttempts != nil {
		jetStreamOptions.SetRetryAttempts(*cfg.RetryAttempts)
	}
	if cfg.StallWait != nil {
		jetStreamOptions.SetStallWait(*cfg.StallWait)
	}
	if cfg.Deduplicate != nil {
		jetStreamOptions.SetDeduplicate(*cfg.Deduplicate)
	}
	return &jetStreamOptions
}

func newNatsExporter[T any](
	set exporter.Settings,
	cfg *Config,
	grouper grouper.Grouper[T],
	marshaler *marshaler.Marshaler[T],
) *natsExporter[T] {
	return &natsExporter[T]{
		set:       set,
		cfg:       cfg,
		grouper:   grouper,
		marshaler: marshaler,
	}
}

func (e *natsExporter[T]) start(ctx context.Context, host component.Host) error {
	var errs error

	natsOptions, err := newNatsOptions(&e.cfg.natsConfig, ctx)
	errs = multierr.Append(errs, err)

	if e.cfg.JetStream != nil {
		jetStreamOptions := newJetStreamOptions(e.cfg.JetStream)
		errs = multierr.Append(errs, err)

		e.publisher = publisher.NewJetStreamPublisher(natsOptions, jetStreamOptions)
	} else {
		e.publisher = publisher.NewCoreNatsPublisher(natsOptions)
	}
	if errs != nil {
		return errs
	}

	errs = multierr.Append(errs, e.marshaler.Resolve(host))
	errs = multierr.Append(errs, e.publisher.Connect())
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
	return e.publisher.Disconnect()
}

func newResolver(cfg *resolverConfig) (marshaler.Resolver, error) {
	if cfg.EncodingExtensionName != nil {
		return marshaler.NewEncodingExtensionResolver(cfg.EncodingExtensionName)
	} else {
		return marshaler.NewBuiltinMarshalerResolver(cfg.MarshalerName)
	}
}

func newNatsLogsExporter(set exporter.Settings, cfg *Config) (*natsExporter[plog.Logs], error) {
	var errs error

	grouper, err := grouper.NewLogsGrouper(cfg.Logs.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.Logs.resolverConfig)
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalLogs)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}

func newNatsMetricsExporter(set exporter.Settings, cfg *Config) (*natsExporter[pmetric.Metrics], error) {
	var errs error

	grouper, err := grouper.NewMetricsGrouper(cfg.Metrics.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.Metrics.resolverConfig)
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalMetrics)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}

func newNatsTracesExporter(set exporter.Settings, cfg *Config) (*natsExporter[ptrace.Traces], error) {
	var errs error

	grouper, err := grouper.NewTracesGrouper(cfg.Traces.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.Traces.resolverConfig)
	errs = multierr.Append(errs, err)
	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalTraces)

	return newNatsExporter(set, cfg, grouper, marshaler), errs
}
