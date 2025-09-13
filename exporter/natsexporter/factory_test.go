// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/metadata"
)

func TestCreateDefaultConfig(t *testing.T) {
	t.Parallel()

	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, cfg.(*Config).Validate())
}

func TestFactory(t *testing.T) {
	t.Parallel()

	t.Run("CreateLogs", func(t *testing.T) {
		factory := NewFactory()

		_, err := factory.CreateLogs(
			t.Context(),
			exportertest.NewNopSettings(metadata.Type),
			factory.CreateDefaultConfig(),
		)
		assert.NoError(t, err)
	})

	t.Run("CreateMetrics", func(t *testing.T) {
		factory := NewFactory()

		_, err := factory.CreateMetrics(
			t.Context(),
			exportertest.NewNopSettings(metadata.Type),
			factory.CreateDefaultConfig(),
		)
		assert.NoError(t, err)
	})

	t.Run("CreateTraces", func(t *testing.T) {
		factory := NewFactory()

		_, err := factory.CreateTraces(
			t.Context(),
			exportertest.NewNopSettings(metadata.Type),
			factory.CreateDefaultConfig(),
		)
		assert.NoError(t, err)
	})
}
