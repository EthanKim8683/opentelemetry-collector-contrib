// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshaler"

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type marshalFunc[T any] func(data T) ([]byte, error)

type pickFunc[T any] func(genericMarshaler genericMarshaler) (marshalFunc[T], error)

func pickMarshalLogs(genericMarshaler genericMarshaler) (marshalFunc[plog.Logs], error) {
	logsMarshaler, ok := genericMarshaler.(plog.Marshaler)
	if !ok {
		return nil, errors.New("genericMarshaler does not implement plog.Marshaler")
	}
	return logsMarshaler.MarshalLogs, nil
}

var _ pickFunc[plog.Logs] = pickMarshalLogs

func pickMarshalMetrics(genericMarshaler genericMarshaler) (marshalFunc[pmetric.Metrics], error) {
	metricsMarshaler, ok := genericMarshaler.(pmetric.Marshaler)
	if !ok {
		return nil, errors.New("genericMarshaler does not implement pmetric.Marshaler")
	}
	return metricsMarshaler.MarshalMetrics, nil
}

var _ pickFunc[pmetric.Metrics] = pickMarshalMetrics

func pickMarshalTraces(genericMarshaler genericMarshaler) (marshalFunc[ptrace.Traces], error) {
	tracesMarshaler, ok := genericMarshaler.(ptrace.Marshaler)
	if !ok {
		return nil, errors.New("genericMarshaler does not implement ptrace.Marshaler")
	}
	return tracesMarshaler.MarshalTraces, nil
}

var _ pickFunc[ptrace.Traces] = pickMarshalTraces
