// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package marshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestPickMarshalLogs(t *testing.T) {
	t.Parallel()

	t.Run("returns MarshalLogs method for plog.Marshaler", func(t *testing.T) {
		genericMarshaler := &plog.JSONMarshaler{}
		marshal, err := pickMarshalLogs(genericMarshaler)
		assert.NoError(t, err)
		assert.NotNil(t, marshal)
	})

	t.Run("returns error for non-plog.Marshaler", func(t *testing.T) {
		genericMarshaler := &pmetric.JSONMarshaler{}
		_, err := pickMarshalLogs(genericMarshaler)
		assert.Error(t, err)
	})
}

func TestPickMarshalMetrics(t *testing.T) {
	t.Parallel()

	t.Run("returns MarshalMetrics method for pmetric.Marshaler", func(t *testing.T) {
		genericMarshaler := &pmetric.JSONMarshaler{}
		marshal, err := pickMarshalMetrics(genericMarshaler)
		assert.NoError(t, err)
		assert.NotNil(t, marshal)
	})

	t.Run("returns error for non-pmetric.Marshaler", func(t *testing.T) {
		genericMarshaler := &plog.JSONMarshaler{}
		_, err := pickMarshalMetrics(genericMarshaler)
		assert.Error(t, err)
	})
}

func TestPickMarshalTraces(t *testing.T) {
	t.Parallel()

	t.Run("returns MarshalTraces method for ptrace.Marshaler", func(t *testing.T) {
		genericMarshaler := &ptrace.JSONMarshaler{}
		marshal, err := pickMarshalTraces(genericMarshaler)
		assert.NoError(t, err)
		assert.NotNil(t, marshal)
	})

	t.Run("returns error for non-ptrace.Marshaler", func(t *testing.T) {
		genericMarshaler := &plog.JSONMarshaler{}
		_, err := pickMarshalTraces(genericMarshaler)
		assert.Error(t, err)
	})
}
