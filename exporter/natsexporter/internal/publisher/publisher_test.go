package publisher

import (
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer(&server.Options{
		Port:      server.RANDOM_PORT,
		StoreDir:  t.TempDir(),
		JetStream: true,
	})
	require.NoError(t, err)
	s.Start()
	t.Cleanup(func() {
		s.Shutdown()
	})

	nc, err := nats.Connect(s.ClientURL())
	require.NoError(t, err)
	t.Cleanup(func() {
		nc.Close()
	})

	js, err := jetstream.New(nc)
	require.NoError(t, err)
	t.Cleanup(func() {
		js.CleanupPublisher()
	})

	js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name:     "STREAM",
		Subjects: []string{"test"},
	})
	require.NoError(t, err)

	t.Run("CoreNatsPublisher", func(t *testing.T) {
		var natsOptions NatsOptions
		natsOptions.SetURL(s.ClientURL())

		publisher, err := NewCoreNatsPublisher(&natsOptions)
		require.NoError(t, err)

		err = publisher.Connect()
		assert.NoError(t, err)

		err = publisher.Publish(t.Context(), "test", []byte("test"))
		assert.NoError(t, err)

		err = publisher.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("JetStreamPublisher", func(t *testing.T) {
		var natsOptions NatsOptions
		natsOptions.SetURL(s.ClientURL())

		var jetStreamOptions JetStreamOptions

		publisher, err := NewJetStreamPublisher(&natsOptions, &jetStreamOptions)
		require.NoError(t, err)

		err = publisher.Connect()
		assert.NoError(t, err)

		err = publisher.Publish(t.Context(), "test", []byte("test"))
		assert.NoError(t, err)

		err = publisher.Disconnect()
		assert.NoError(t, err)
	})
}
