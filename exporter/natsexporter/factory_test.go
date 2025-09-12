// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/metadata"
	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory()
	assert.Equal(t, metadata.Type, factory.Type())
}
