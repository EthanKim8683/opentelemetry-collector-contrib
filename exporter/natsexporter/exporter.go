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

func newNatsOptions(cfg *natsConfig, ctx context.Context) (*publisher.NatsOptions, error) {
	var errs error

	tlsConfig, err := cfg.TLS.LoadTLSConfig(ctx)
	errs = multierr.Append(errs, err)

	var natsOptions publisher.NatsOptions
	natsOptions.SetURL(cfg.Endpoint)
	natsOptions.SetTLS(tlsConfig)
	natsOptions.SetPedantic(cfg.Pedantic)
	natsOptions.SetToken(cfg.AuthConfig.Token.Token)
	natsOptions.SetUser(cfg.AuthConfig.User.Username, cfg.AuthConfig.User.Password)
	errs = multierr.Append(errs, natsOptions.SetNkey(cfg.AuthConfig.Nkey.Seed))
	errs = multierr.Append(errs, natsOptions.SetNkeyJWT(cfg.AuthConfig.NkeyJWT.JWT, cfg.AuthConfig.NkeyJWT.Seed))
	errs = multierr.Append(errs, natsOptions.SetNkeyUserFile(cfg.AuthConfig.NkeyUserFile.UserFilePath))

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
	if cfg.Dedup != nil {
		jetStreamOptions.SetDedup(*cfg.Dedup)
	}
	return &jetStreamOptions
}

func newPublisher(ctx context.Context, natsCfg *natsConfig, jetStreamCfg *jetStreamConfig) (publisher.Publisher, error) {
	var errs error

	natsOptions, err := newNatsOptions(natsCfg, ctx)
	errs = multierr.Append(errs, err)

	if jetStreamCfg != nil {
		jetStreamOptions := newJetStreamOptions(jetStreamCfg)
		errs = multierr.Append(errs, err)

		if errs != nil {
			return nil, errs
		}
		return publisher.NewJetStreamPublisher(natsOptions, jetStreamOptions), nil
	} else {
		if errs != nil {
			return nil, errs
		}
		return publisher.NewCoreNatsPublisher(natsOptions), nil
	}
}

func newResolver(cfg *resolverConfig) (marshaler.Resolver, error) {
	if cfg.EncodingExtensionName != nil {
		return marshaler.NewEncodingExtensionResolver(cfg.EncodingExtensionName)
	} else {
		return marshaler.NewBuiltinMarshalerResolver(cfg.MarshalerName)
	}
}

type natsExporter[T any] struct {
	set       exporter.Settings
	cfg       *Config
	grouper   grouper.Grouper[T]
	marshaler *marshaler.Marshaler[T]
	publisher publisher.Publisher
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

	publisher, err := newPublisher(ctx, &e.cfg.NatsConfig, e.cfg.JetStreamConfig)
	errs = multierr.Append(errs, err)
	e.publisher = publisher

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

func newNatsLogsExporter(set exporter.Settings, cfg *Config) (*natsExporter[plog.Logs], error) {
	var errs error

	grouper, err := grouper.NewLogsGrouper(cfg.LogsConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.LogsConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalLogs)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(set, cfg, grouper, marshaler), nil
}

func newNatsMetricsExporter(set exporter.Settings, cfg *Config) (*natsExporter[pmetric.Metrics], error) {
	var errs error

	grouper, err := grouper.NewMetricsGrouper(cfg.MetricsConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.MetricsConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalMetrics)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(set, cfg, grouper, marshaler), nil
}

func newNatsTracesExporter(set exporter.Settings, cfg *Config) (*natsExporter[ptrace.Traces], error) {
	var errs error

	grouper, err := grouper.NewTracesGrouper(cfg.TracesConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.TracesConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshaler.NewMarshaler(resolver, marshaler.PickMarshalTraces)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(set, cfg, grouper, marshaler), nil
}
