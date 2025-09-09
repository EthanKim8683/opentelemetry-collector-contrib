// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grouper // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/grouper"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

type logsGrouper struct {
	valueExpression *ottl.ValueExpression[ottllog.TransformContext]
}

func (g *logsGrouper) Group(ctx context.Context, srcLogs plog.Logs) ([]Group[plog.Logs], error) {
	var errs error

	type destContext struct {
		logs            plog.Logs
		srcResourceLogs plog.ResourceLogs
		srcScopeLogs    plog.ScopeLogs
	}
	destBySubject := make(map[string]*destContext)

	for _, srcResourceLogs := range srcLogs.ResourceLogs().All() {
		var (
			srcResource       = srcResourceLogs.Resource()
			srcResourceSchema = srcResourceLogs.SchemaUrl()
		)

		for _, srcScopeLogs := range srcResourceLogs.ScopeLogs().All() {
			var (
				srcScope       = srcScopeLogs.Scope()
				srcScopeSchema = srcScopeLogs.SchemaUrl()
			)

			for _, srcLogRecord := range srcScopeLogs.LogRecords().All() {
				subjectAsAny, err := g.valueExpression.Eval(ctx, ottllog.NewTransformContext(
					srcLogRecord,
					srcScope,
					srcResource,
					srcScopeLogs,
					srcResourceLogs,
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
						logs: plog.NewLogs(),
					}
					destBySubject[subject] = dest
				}
				destLogs := dest.logs

				destResourceLogsSlice := destLogs.ResourceLogs()
				if dest.srcResourceLogs != srcResourceLogs {
					dest.srcResourceLogs = srcResourceLogs

					destResourceLogs := destResourceLogsSlice.AppendEmpty()
					srcResource.CopyTo(destResourceLogs.Resource())
					destResourceLogs.SetSchemaUrl(srcResourceSchema)
				}
				destResourceLogs := destResourceLogsSlice.At(destResourceLogsSlice.Len() - 1)

				destScopeLogsSlice := destResourceLogs.ScopeLogs()
				if dest.srcScopeLogs != srcScopeLogs {
					dest.srcScopeLogs = srcScopeLogs

					destScopeLogs := destScopeLogsSlice.AppendEmpty()
					srcScope.CopyTo(destScopeLogs.Scope())
					destScopeLogs.SetSchemaUrl(srcScopeSchema)
				}
				destScopeLogs := destScopeLogsSlice.At(destScopeLogsSlice.Len() - 1)

				destLogRecordSlice := destScopeLogs.LogRecords()
				srcLogRecord.CopyTo(destLogRecordSlice.AppendEmpty())
			}
		}
	}

	groups := make([]Group[plog.Logs], 0, len(destBySubject))
	for subject, dest := range destBySubject {
		groups = append(groups, Group[plog.Logs]{
			Subject: subject,
			Data:    dest.logs,
		})
	}
	return groups, errs
}

var _ Grouper[plog.Logs] = (*logsGrouper)(nil)

type LogsGrouperConfig struct {
	Subject string `mapstructure:"subject"`

	logsGrouper *logsGrouper
}

func (c *LogsGrouperConfig) Validate() error {
	if c.logsGrouper != nil {
		return nil
	}

	parser, err := ottllog.NewParser(
		ottlfuncs.StandardConverters[ottllog.TransformContext](),
		componenttest.NewNopTelemetrySettings(),
	)
	if err != nil {
		return fmt.Errorf("failed to create logs parser: %w", err)
	}

	valueExpression, err := parser.ParseValueExpression(c.Subject)
	if err != nil {
		return fmt.Errorf("failed to parse logs subject: %w", err)
	}

	c.logsGrouper = &logsGrouper{
		valueExpression: valueExpression,
	}
	return nil
}

func NewDefaultLogsGrouperConfig() LogsGrouperConfig {
	return LogsGrouperConfig{
		Subject: "\"otel_logs\"",
	}
}

func NewLogsGrouper(cfg *LogsGrouperConfig, telemetrySettings component.TelemetrySettings) (*logsGrouper, error) {
	if cfg.logsGrouper == nil {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}
	return cfg.logsGrouper, nil
}
