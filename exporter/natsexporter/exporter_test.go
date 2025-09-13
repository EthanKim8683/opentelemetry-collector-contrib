// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package natsexporter

import (
	"context"
	"errors"
	"math/rand/v2"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/group"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/marshal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publish"
)

const (
	fakeMarshalError = "marshal error"
	fakePublishError = "publish error"
)

type fakeGrouper struct{}

func (g *fakeGrouper) Group(ctx context.Context, data string) ([]group.Group[string], error) {
	tokens := strings.Split(data, ",")
	groups := make([]group.Group[string], len(tokens))
	for i, token := range tokens {
		groups[i] = group.Group[string]{
			Subject: token,
			Data:    token,
		}
	}
	return groups, nil
}

var _ group.Grouper[string] = (*fakeGrouper)(nil)

func newFakeGrouper() group.Grouper[string] {
	return &fakeGrouper{}
}

type fakeGenericMarshaler struct{}

func (m *fakeGenericMarshaler) MarshalString(sd string) ([]byte, error) {
	if sd == fakeMarshalError {
		return nil, errors.New(fakeMarshalError)
	}
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
	mu       sync.Mutex
	messages []message
}

func (m *mockPublisher) Connect() error {
	return nil
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	if subject == fakePublishError {
		return errors.New(fakePublishError)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, message{subject: subject, data: data})
	return nil
}

func (m *mockPublisher) Disconnect() error {
	return nil
}

func (m *mockPublisher) replay(count int) []message {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	t.Run("composes grouper marshaler and publisher", func(t *testing.T) {
		tokens := make([]string, 100)
		for i := range tokens {
			tokens[i] = uuid.NewString()
		}

		data := strings.Join(tokens, ",")

		wantMessages := make([]message, 0, len(tokens))
		for _, token := range tokens {
			wantMessages = append(wantMessages, message{
				subject: token,
				data:    []byte(token),
			})
		}

		grouper := newFakeGrouper()
		resolver := newFakeResolver()
		marshaler := marshal.NewMarshaler(resolver, fakePick)
		publisher := newMockPublisher(t)
		exporter := newNatsExporter(grouper, marshaler, publisher)

		err := exporter.start(t.Context(), componenttest.NewNopHost())
		assert.NoError(t, err)

		err = exporter.export(t.Context(), data)
		assert.NoError(t, err)

		err = exporter.shutdown(t.Context())
		assert.NoError(t, err)

		haveMessages := publisher.replay(len(wantMessages))
		assert.ElementsMatch(t, wantMessages, haveMessages)
	})

	t.Run("catches and returns errors", func(t *testing.T) {
		tokens := make([]string, 100)
		for i := range tokens {
			tokens[i] = []string{
				uuid.NewString(),
				fakeMarshalError,
				fakePublishError,
			}[rand.IntN(3)]
		}

		data := strings.Join(tokens, ",")

		wantErrorsLen := 0
		wantMessages := make([]message, 0, len(tokens))
		for _, token := range tokens {
			switch token {
			case fakeMarshalError, fakePublishError:
				wantErrorsLen++
			default:
				wantMessages = append(wantMessages, message{
					subject: token,
					data:    []byte(token),
				})
			}
		}

		grouper := newFakeGrouper()
		resolver := newFakeResolver()
		marshaler := marshal.NewMarshaler(resolver, fakePick)
		publisher := newMockPublisher(t)
		exporter := newNatsExporter(grouper, marshaler, publisher)

		err := exporter.start(t.Context(), componenttest.NewNopHost())
		assert.NoError(t, err)

		err = exporter.export(t.Context(), data)
		assert.Equal(t, wantErrorsLen, len(multierr.Errors(err)))

		err = exporter.shutdown(t.Context())
		assert.NoError(t, err)

		haveMessages := publisher.replay(len(wantMessages))
		assert.ElementsMatch(t, wantMessages, haveMessages)
	})
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
			EncodingExtensionName: &[]string{"encoding"}[0],
		}

		haveResolver, err := newResolver(&cfg)
		assert.NoError(t, err)

		wantResolver, err := marshal.NewEncodingExtensionResolver([]byte(*cfg.EncodingExtensionName))
		require.NoError(t, err)

		assert.IsType(t, wantResolver, haveResolver)
	})
}

func TestNewNatsOptions(t *testing.T) {
	t.Parallel()

	_, seed := createNkey(t)
	userJWT, userSeed := createNkeyJWT(t, 5*time.Minute)
	userFilePath := createNkeyUserFile(t, 5*time.Minute)

	cfg := &NatsConfig{
		Endpoint:    "nats://localhost:4222",
		TLS:         configtls.NewDefaultClientConfig(),
		Pedantic:    true,
		Compression: true,
		AuthConfig: AuthConfig{
			TokenConfig: &TokenConfig{
				Token: "token",
			},
			UserConfig: &UserConfig{
				Username: "user",
				Password: "password",
			},
			NkeyConfig: &NkeyConfig{
				Seed: string(seed),
			},
			NkeyJWTConfig: &NkeyJWTConfig{
				UserJWT: userJWT,
				Seed:    string(userSeed),
			},
			NkeyUserFileConfig: &NkeyUserFileConfig{
				UserFilePath: userFilePath,
			},
		},
	}

	natsOptions, err := newNatsOptions(cfg)
	assert.NoError(t, err)

	value := reflect.ValueOf(*natsOptions)
	assert.Equal(t, 9, value.FieldByName("setOptionFuncs").Len())
}

func TestNewJetStreamOptions(t *testing.T) {
	t.Parallel()

	cfg := &JetStreamConfig{
		RetryWait:     &[]time.Duration{1 * time.Second}[0],
		RetryAttempts: &[]int{10}[0],
		StallWait:     &[]time.Duration{1 * time.Second}[0],
		Deduplication: &[]bool{true}[0],
	}

	jetStreamOptions := newJetStreamOptions(cfg)

	value := reflect.ValueOf(*jetStreamOptions)
	assert.Equal(t, 4, value.FieldByName("buildPublishOptFuncs").Len())
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
