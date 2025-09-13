// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreNatsPublisher(t *testing.T) {
	t.Parallel()

	t.Run("delivers published messages end-to-end", func(t *testing.T) {
		subject := "test"
		data := []byte("test")

		s, err := server.NewServer(&server.Options{
			Port: server.RANDOM_PORT,
		})
		require.NoError(t, err)
		s.Start()
		t.Cleanup(func() {
			s.Shutdown()
		})

		nc, err := nats.Connect(s.ClientURL())
		require.NoError(t, err)

		subscription, err := nc.SubscribeSync(subject)
		require.NoError(t, err)

		var natsOptions NatsOptions
		natsOptions.SetURL(s.ClientURL())

		publisher := NewCoreNatsPublisher(&natsOptions)

		err = publisher.Publish(t.Context(), subject, data)
		assert.NoError(t, err)

		msg, err := subscription.NextMsgWithContext(t.Context())
		require.NoError(t, err)
		assert.Equal(t, subject, msg.Subject)
		assert.Equal(t, data, msg.Data)

		err = publisher.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("disconnects gracefully if not connected", func(t *testing.T) {
		publisher := NewCoreNatsPublisher(&NatsOptions{})

		err := publisher.Disconnect()
		assert.NoError(t, err)
	})
}

func TestJetStreamPublisher(t *testing.T) {
	t.Parallel()

	t.Run("delivers published messages end-to-end", func(t *testing.T) {
		subject := "test"
		data := []byte("test")

		s, err := server.NewServer(&server.Options{
			Port:      server.RANDOM_PORT,
			JetStream: true,
			StoreDir:  t.TempDir(),
		})
		require.NoError(t, err)
		s.Start()
		t.Cleanup(func() {
			s.Shutdown()
		})

		nc, err := nats.Connect(s.ClientURL())
		require.NoError(t, err)

		js, err := jetstream.New(nc)
		require.NoError(t, err)

		stream, err := js.CreateStream(t.Context(), jetstream.StreamConfig{
			Name:     "STREAM",
			Subjects: []string{subject},
		})
		require.NoError(t, err)

		consumer, err := stream.CreateConsumer(t.Context(), jetstream.ConsumerConfig{})
		require.NoError(t, err)

		messagesContext, err := consumer.Messages()
		require.NoError(t, err)

		var natsOptions NatsOptions
		natsOptions.SetURL(s.ClientURL())

		var jetStreamOptions JetStreamOptions

		publisher := NewJetStreamPublisher(&natsOptions, &jetStreamOptions)

		err = publisher.Publish(t.Context(), subject, data)
		assert.NoError(t, err)

		message, err := messagesContext.Next()
		require.NoError(t, err)
		assert.Equal(t, subject, message.Subject())
		assert.Equal(t, data, message.Data())

		err = publisher.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("disconnects gracefully if not connected", func(t *testing.T) {
		publisher := NewJetStreamPublisher(&NatsOptions{}, &JetStreamOptions{})

		err := publisher.Disconnect()
		assert.NoError(t, err)
	})
}
