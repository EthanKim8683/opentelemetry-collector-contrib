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

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/group"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publish"
)

type natsExporter[T any] struct {
	grouper   group.Grouper[T]
	marshaler *marshal.Marshaler[T]
	publisher publish.Publisher
}

func newNatsExporter[T any](
	grouper group.Grouper[T],
	marshaler *marshal.Marshaler[T],
	publisher publish.Publisher,
) *natsExporter[T] {
	return &natsExporter[T]{
		grouper:   grouper,
		marshaler: marshaler,
		publisher: publisher,
	}
}

func (e *natsExporter[T]) start(_ context.Context, host component.Host) error {
	var errs error
	errs = multierr.Append(errs, e.marshaler.Resolve(host))
	return errs
}

func (e *natsExporter[T]) export(ctx context.Context, data T) error {
	var errs error

	groups, err := e.grouper.Group(ctx, data)
	errs = multierr.Append(errs, err)

	var wg sync.WaitGroup
	errCh := make(chan error, len(groups))
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

			if err = e.publisher.Publish(ctx, subject, bytes); err != nil {
				errCh <- err
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

func newResolver(cfg *ResolverConfig) (marshal.Resolver, error) {
	if cfg.EncodingExtensionName != nil {
		return marshal.NewEncodingExtensionResolver([]byte(*cfg.EncodingExtensionName))
	}
	return marshal.NewBuiltinMarshalerResolver(cfg.MarshalerName)
}

func newNatsOptions(cfg *NatsConfig) (*publish.NatsOptions, error) {
	var errs error

	tlsConfig, err := cfg.TLS.LoadTLSConfig(context.Background())
	errs = multierr.Append(errs, err)

	var natsOptions publish.NatsOptions

	natsOptions.SetURL(cfg.Endpoint)
	natsOptions.SetTLS(tlsConfig)
	natsOptions.SetPedantic(cfg.Pedantic)
	natsOptions.SetCompression(cfg.Compression)

	if cfg.AuthConfig.TokenConfig != nil {
		natsOptions.SetToken(cfg.AuthConfig.TokenConfig.Token)
	}
	if cfg.AuthConfig.UserConfig != nil {
		natsOptions.SetUser(
			cfg.AuthConfig.UserConfig.Username,
			cfg.AuthConfig.UserConfig.Password,
		)
	}
	if cfg.AuthConfig.NkeyConfig != nil {
		errs = multierr.Append(errs, natsOptions.SetNkey(
			[]byte(cfg.AuthConfig.NkeyConfig.Seed),
		))
	}
	if cfg.AuthConfig.NkeyJWTConfig != nil {
		errs = multierr.Append(errs, natsOptions.SetNkeyJWT(
			cfg.AuthConfig.NkeyJWTConfig.UserJWT,
			[]byte(cfg.AuthConfig.NkeyJWTConfig.Seed),
		))
	}
	if cfg.AuthConfig.NkeyUserFileConfig != nil {
		errs = multierr.Append(errs, natsOptions.SetNkeyUserFile(
			cfg.AuthConfig.NkeyUserFileConfig.UserFilePath,
		))
	}

	if errs != nil {
		return nil, errs
	}
	return &natsOptions, nil
}

func newJetStreamOptions(cfg *JetStreamConfig) *publish.JetStreamOptions {
	var jetStreamOptions publish.JetStreamOptions
	if cfg.RetryWait != nil {
		jetStreamOptions.SetRetryWait(*cfg.RetryWait)
	}
	if cfg.RetryAttempts != nil {
		jetStreamOptions.SetRetryAttempts(*cfg.RetryAttempts)
	}
	if cfg.StallWait != nil {
		jetStreamOptions.SetStallWait(*cfg.StallWait)
	}
	if cfg.Deduplication != nil {
		jetStreamOptions.SetDeduplication(*cfg.Deduplication)
	}
	return &jetStreamOptions
}

func newPublisher(natsCfg *NatsConfig, jetStreamCfg *JetStreamConfig) (publish.Publisher, error) {
	var errs error

	natsOptions, err := newNatsOptions(natsCfg)
	errs = multierr.Append(errs, err)

	var publisher publish.Publisher
	if jetStreamCfg != nil {
		jetStreamOptions := newJetStreamOptions(jetStreamCfg)
		errs = multierr.Append(errs, err)

		publisher = publish.NewJetStreamPublisher(natsOptions, jetStreamOptions)
	} else {
		publisher = publish.NewCoreNatsPublisher(natsOptions)
	}

	if errs != nil {
		return nil, errs
	}
	return publisher, nil
}

func newNatsLogsExporter(set exporter.Settings, cfg *Config) (*natsExporter[plog.Logs], error) {
	var errs error

	grouper, err := group.NewLogsGrouper(cfg.LogsConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.LogsConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshal.NewMarshaler(resolver, marshal.PickMarshalLogs)

	publisher, err := newPublisher(&cfg.NatsConfig, cfg.JetStreamConfig)
	errs = multierr.Append(errs, err)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(grouper, marshaler, publisher), nil
}

func newNatsMetricsExporter(set exporter.Settings, cfg *Config) (*natsExporter[pmetric.Metrics], error) {
	var errs error

	grouper, err := group.NewMetricsGrouper(cfg.MetricsConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.MetricsConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshal.NewMarshaler(resolver, marshal.PickMarshalMetrics)

	publisher, err := newPublisher(&cfg.NatsConfig, cfg.JetStreamConfig)
	errs = multierr.Append(errs, err)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(grouper, marshaler, publisher), nil
}

func newNatsTracesExporter(set exporter.Settings, cfg *Config) (*natsExporter[ptrace.Traces], error) {
	var errs error

	grouper, err := group.NewTracesGrouper(cfg.TracesConfig.Subject, set.TelemetrySettings)
	errs = multierr.Append(errs, err)

	resolver, err := newResolver(&cfg.TracesConfig.ResolverConfig)
	errs = multierr.Append(errs, err)

	marshaler := marshal.NewMarshaler(resolver, marshal.PickMarshalTraces)

	publisher, err := newPublisher(&cfg.NatsConfig, cfg.JetStreamConfig)
	errs = multierr.Append(errs, err)

	if errs != nil {
		return nil, errs
	}
	return newNatsExporter(grouper, marshaler, publisher), nil
}
