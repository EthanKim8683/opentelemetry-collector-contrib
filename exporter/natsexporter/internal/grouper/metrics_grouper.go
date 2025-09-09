// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grouper // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/grouper"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

type metricsGrouper struct {
	valueExpression *ottl.ValueExpression[ottlmetric.TransformContext]
}

func (g *metricsGrouper) Group(ctx context.Context, srcMetrics pmetric.Metrics) ([]Group[pmetric.Metrics], error) {
	var errs error

	type destContext struct {
		metrics            pmetric.Metrics
		srcResourceMetrics pmetric.ResourceMetrics
		srcScopeMetrics    pmetric.ScopeMetrics
	}
	destBySubject := make(map[string]*destContext)

	for _, srcResourceMetrics := range srcMetrics.ResourceMetrics().All() {
		var (
			srcResource       = srcResourceMetrics.Resource()
			srcResourceSchema = srcResourceMetrics.SchemaUrl()
		)

		for _, srcScopeMetrics := range srcResourceMetrics.ScopeMetrics().All() {
			var (
				srcScope       = srcScopeMetrics.Scope()
				srcScopeSchema = srcScopeMetrics.SchemaUrl()
				srcMetricSlice = srcScopeMetrics.Metrics()
			)

			for _, srcMetric := range srcMetricSlice.All() {
				subjectAsAny, err := g.valueExpression.Eval(ctx, ottlmetric.NewTransformContext(
					srcMetric,
					srcMetricSlice,
					srcScope,
					srcResource,
					srcScopeMetrics,
					srcResourceMetrics,
				))
				if err != nil {
					errs = multierr.Append(errs, err)
					continue
				}

				subject, ok := subjectAsAny.(string)
				if !ok {
					errs = multierr.Append(errs, errors.New("subject is not a string"))
					continue
				}

				dest, ok := destBySubject[subject]
				if !ok {
					dest = &destContext{
						metrics: pmetric.NewMetrics(),
					}
					destBySubject[subject] = dest
				}
				destMetrics := dest.metrics

				destResourceMetricsSlice := destMetrics.ResourceMetrics()
				if dest.srcResourceMetrics != srcResourceMetrics {
					dest.srcResourceMetrics = srcResourceMetrics

					destResourceMetrics := destResourceMetricsSlice.AppendEmpty()
					srcResource.CopyTo(destResourceMetrics.Resource())
					destResourceMetrics.SetSchemaUrl(srcResourceSchema)
				}
				destResourceMetrics := destResourceMetricsSlice.At(destResourceMetricsSlice.Len() - 1)

				destScopeMetricsSlice := destResourceMetrics.ScopeMetrics()
				if dest.srcScopeMetrics != srcScopeMetrics {
					dest.srcScopeMetrics = srcScopeMetrics

					destScopeMetrics := destScopeMetricsSlice.AppendEmpty()
					srcScope.CopyTo(destScopeMetrics.Scope())
					destScopeMetrics.SetSchemaUrl(srcScopeSchema)
				}
				destScopeMetrics := destScopeMetricsSlice.At(destScopeMetricsSlice.Len() - 1)

				destMetricSlice := destScopeMetrics.Metrics()
				srcMetric.CopyTo(destMetricSlice.AppendEmpty())
			}
		}
	}

	groups := make([]Group[pmetric.Metrics], 0, len(destBySubject))
	for subject, dest := range destBySubject {
		groups = append(groups, Group[pmetric.Metrics]{
			Subject: subject,
			Data:    dest.metrics,
		})
	}
	return groups, errs
}

var _ Grouper[pmetric.Metrics] = (*metricsGrouper)(nil)

type MetricsGrouperConfig struct {
	Subject *string `mapstructure:"subject"`

	metricsGrouper *metricsGrouper
}

func (c *MetricsGrouperConfig) Validate() error {
	if c.Subject == nil {
		return errors.New("subject not configured")
	}

	parser, err := ottlmetric.NewParser(
		ottlfuncs.StandardConverters[ottlmetric.TransformContext](),
		componenttest.NewNopTelemetrySettings(),
	)
	if err != nil {
		return fmt.Errorf("failed to create metrics parser: %w", err)
	}

	valueExpression, err := parser.ParseValueExpression(*c.Subject)
	if err != nil {
		return fmt.Errorf("failed to parse metrics subject: %w", err)
	}

	c.metricsGrouper = &metricsGrouper{
		valueExpression: valueExpression,
	}
	return nil
}

func NewDefaultMetricsGrouperConfig() MetricsGrouperConfig {
	subject := "\"otel_logs\""
	return MetricsGrouperConfig{
		Subject: &subject,
	}
}

func NewMetricsGrouper(cfg *MetricsGrouperConfig, telemetrySettings component.TelemetrySettings) (*metricsGrouper, error) {
	if cfg.metricsGrouper == nil {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}
	return cfg.metricsGrouper, nil
}
