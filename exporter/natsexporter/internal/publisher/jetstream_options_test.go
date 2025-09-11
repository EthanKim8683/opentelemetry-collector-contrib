package publisher

import (
	"strconv"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wrapPublishOpt[T any](publishOpt func(opts *T) error, optsRef *any) func(opts *T) error {
	return func(opts *T) error {
		*optsRef = *opts
		return publishOpt(opts)
	}
}

func mockPublishOpt(optsRef *any) jetstream.PublishOpt {
	return wrapPublishOpt(jetstream.WithMsgID(""), optsRef)
}

type publishOptValidator struct {
	t  *testing.T
	js jetstream.JetStream
}

func (pov *publishOptValidator) validatePublishOpt(
	wantPublishOptSlice []jetstream.PublishOpt,
	havePublishOptSlice []jetstream.PublishOpt,
) {
	var wantOpts, haveOpts any
	wantPublishOptSlice = append(wantPublishOptSlice, mockPublishOpt(&wantOpts))
	havePublishOptSlice = append(havePublishOptSlice, mockPublishOpt(&haveOpts))

	pov.js.Publish(pov.t.Context(), "", nil, wantPublishOptSlice...)
	pov.js.Publish(pov.t.Context(), "", nil, havePublishOptSlice...)
	assert.Equal(pov.t, wantOpts, haveOpts)
}

func newPublishOptValidator(t *testing.T) *publishOptValidator {
	s, err := server.NewServer(&server.Options{})
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

	return &publishOptValidator{t, js}
}

func TestJetStreamOptions(t *testing.T) {
	t.Parallel()

	pov := newPublishOptValidator(t)

	t.Run("SetRetryWait", func(t *testing.T) {
		retryWait := time.Second

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetRetryWait(retryWait)

		pov.validatePublishOpt(
			[]jetstream.PublishOpt{jetstream.WithRetryWait(retryWait)},
			jetStreamOptions.buildPublishOptSlice(nil),
		)
	})

	t.Run("SetRetryAttempts", func(t *testing.T) {
		retryAttempts := 1

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetRetryAttempts(retryAttempts)

		pov.validatePublishOpt(
			[]jetstream.PublishOpt{jetstream.WithRetryAttempts(retryAttempts)},
			jetStreamOptions.buildPublishOptSlice(nil),
		)
	})

	t.Run("SetStallWait", func(t *testing.T) {
		stallWait := time.Second

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetStallWait(stallWait)

		pov.validatePublishOpt(
			[]jetstream.PublishOpt{jetstream.WithStallWait(stallWait)},
			jetStreamOptions.buildPublishOptSlice(nil),
		)
	})

	t.Run("SetDeduplicate", func(t *testing.T) {
		data := []byte("data")
		msgID := strconv.FormatUint(xxhash.Sum64(data), 16)

		t.Run("true", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplicate(true)

			pov.validatePublishOpt(
				[]jetstream.PublishOpt{jetstream.WithMsgID(msgID)},
				jetStreamOptions.buildPublishOptSlice(data),
			)
		})

		t.Run("false", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplicate(false)

			pov.validatePublishOpt(
				[]jetstream.PublishOpt{jetstream.WithMsgID("")},
				jetStreamOptions.buildPublishOptSlice(data),
			)
		})

		t.Run("true then false", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplicate(true)
			jetStreamOptions.SetDeduplicate(false)

			pov.validatePublishOpt(
				[]jetstream.PublishOpt{jetstream.WithMsgID("")},
				jetStreamOptions.buildPublishOptSlice(data),
			)
		})
	})
}
