// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"

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

func TestNatsCoreExporter(t *testing.T) {
	t.Parallel()

	grouper := newFakeGrouper()
	resolver := newFakeResolver()
	marshaler := marshal.NewMarshaler(resolver, fakePick)
	publisher := newMockPublisher(t)
	exporter := newNatsExporter(grouper, marshaler, publisher)

	err := exporter.start(t.Context(), componenttest.NewNopHost())
	assert.NoError(t, err)

	err = exporter.export(t.Context(), "test")
	assert.NoError(t, err)
	err = exporter.export(t.Context(), "test")
	assert.NoError(t, err)

	err = exporter.shutdown(t.Context())
	assert.NoError(t, err)

	messages := publisher.replay(2)
	for _, messages := range messages {
		t.Logf("%s: %s", messages.subject, messages.data)
	}
}
