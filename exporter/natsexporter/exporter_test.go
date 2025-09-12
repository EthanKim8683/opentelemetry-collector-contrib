// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/group"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publish"
)

type fakeGrouper struct{}

func (g *fakeGrouper) Group(ctx context.Context, data string) ([]group.Group[string], error) {
	return []group.Group[string]{{Subject: data, Data: data}}, nil
}

var _ group.Grouper[string] = (*fakeGrouper)(nil)

func newFakeGrouper() group.Grouper[string] {
	return &fakeGrouper{}
}

type fakeGenericMarshaler struct{}

func (m *fakeGenericMarshaler) MarshalString(sd string) ([]byte, error) {
	return []byte(sd), nil
}

var _ marshal.GenericMarshaler = (*fakeGenericMarshaler)(nil)

type fakeResolver struct{}

func (r *fakeResolver) Resolve(host component.Host) (marshal.GenericMarshaler, error) {
	return &fakeGenericMarshaler{}, nil
}

var _ marshal.Resolver = (*fakeResolver)(nil)

func newFakeResolver() marshal.Resolver {
	return &fakeResolver{}
}

func fakePick(genericMarshaler marshal.GenericMarshaler) (marshal.MarshalFunc[string], error) {
	return genericMarshaler.(*fakeGenericMarshaler).MarshalString, nil
}

var _ marshal.PickFunc[string] = fakePick

type message struct {
	subject string
	data    []byte
}

type mockPublisher struct {
	t        *testing.T
	messages []message
}

func (m *mockPublisher) Connect() error {
	return nil
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	m.messages = append(m.messages, message{subject: subject, data: data})
	return nil
}

func (m *mockPublisher) Disconnect() error {
	return nil
}

func (m *mockPublisher) replay(count int) []message {
	require.GreaterOrEqual(m.t, len(m.messages), count)

	var messages []message
	messages, m.messages = m.messages[:count], m.messages[count:]
	return messages
}

var _ publish.Publisher = (*mockPublisher)(nil)

func newMockPublisher(t *testing.T) *mockPublisher {
	return &mockPublisher{t: t}
}

func TestNatsExporter(t *testing.T) {
	t.Parallel()

	data := make([]string, 64)
	for i := range data {
		data[i] = uuid.NewString()
	}

	grouper := newFakeGrouper()
	resolver := newFakeResolver()
	marshaler := marshal.NewMarshaler(resolver, fakePick)
	publisher := newMockPublisher(t)
	exporter := newNatsExporter(grouper, marshaler, publisher)

	err := exporter.start(t.Context(), componenttest.NewNopHost())
	assert.NoError(t, err)

	for _, data := range data {
		err = exporter.export(t.Context(), data)
		assert.NoError(t, err)
	}

	err = exporter.shutdown(t.Context())
	assert.NoError(t, err)

	messages := publisher.replay(len(data))
	for i, messages := range messages {
		assert.Equal(t, data[i], messages.subject)
		assert.Equal(t, []byte(data[i]), messages.data)
	}
}

func TestNewResolver(t *testing.T) {
	t.Parallel()

	t.Run("builtinMarshalerResolver", func(t *testing.T) {
		cfg := ResolverConfig{
			MarshalerName: marshal.OtlpProtoBuiltinMarshalerName,
		}

		haveResolver, err := newResolver(&cfg)
		assert.NoError(t, err)

		wantResolver, err := marshal.NewBuiltinMarshalerResolver(cfg.MarshalerName)
		require.NoError(t, err)

		assert.IsType(t, wantResolver, haveResolver)
	})

	t.Run("encodingExtensionResolver", func(t *testing.T) {
		cfg := ResolverConfig{
			MarshalerName:         marshal.OtlpProtoBuiltinMarshalerName,
			EncodingExtensionName: []byte("encoding"),
		}

		haveResolver, err := newResolver(&cfg)
		assert.NoError(t, err)

		wantResolver, err := marshal.NewEncodingExtensionResolver(cfg.EncodingExtensionName)
		require.NoError(t, err)

		assert.IsType(t, wantResolver, haveResolver)
	})
}

func TestNewPublisher(t *testing.T) {
	t.Parallel()

	t.Run("CoreNatsPublisher", func(t *testing.T) {
		havePublisher, err := newPublisher(&NatsConfig{}, nil)
		assert.NoError(t, err)

		wantPublisher := publish.NewCoreNatsPublisher(&publish.NatsOptions{})

		assert.IsType(t, wantPublisher, havePublisher)
	})

	t.Run("JetStreamPublisher", func(t *testing.T) {
		havePublisher, err := newPublisher(&NatsConfig{}, &JetStreamConfig{})
		assert.NoError(t, err)

		wantPublisher := publish.NewJetStreamPublisher(&publish.NatsOptions{}, &publish.JetStreamOptions{})

		assert.IsType(t, wantPublisher, havePublisher)
	})
}

func TestNewNatsLogsExporter(t *testing.T) {
	t.Parallel()

	set := exportertest.NewNopSettings(typ)
	cfg := createDefaultConfig().(*Config)

	exporter, err := newNatsLogsExporter(set, cfg)
	assert.NoError(t, err)

	wantGrouper, err := group.NewLogsGrouper(cfg.LogsConfig.Subject, set.TelemetrySettings)
	require.NoError(t, err)

	assert.IsType(t, wantGrouper, exporter.grouper)
}

func TestNewNatsMetricsExporter(t *testing.T) {
	t.Parallel()

	set := exportertest.NewNopSettings(typ)
	cfg := createDefaultConfig().(*Config)

	exporter, err := newNatsMetricsExporter(set, cfg)
	assert.NoError(t, err)

	wantGrouper, err := group.NewMetricsGrouper(cfg.MetricsConfig.Subject, set.TelemetrySettings)
	require.NoError(t, err)

	assert.IsType(t, wantGrouper, exporter.grouper)
}

func TestNewNatsTracesExporter(t *testing.T) {
	t.Parallel()

	set := exportertest.NewNopSettings(typ)
	cfg := createDefaultConfig().(*Config)

	exporter, err := newNatsTracesExporter(set, cfg)
	assert.NoError(t, err)

	wantGrouper, err := group.NewTracesGrouper(cfg.TracesConfig.Subject, set.TelemetrySettings)
	require.NoError(t, err)

	assert.IsType(t, wantGrouper, exporter.grouper)
}
