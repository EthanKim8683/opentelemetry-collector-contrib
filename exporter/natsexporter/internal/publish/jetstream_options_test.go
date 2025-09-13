// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package publish

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

func (v *publishOptValidator) validatePublishOpts(
	wantPublishOpts []jetstream.PublishOpt,
	havePublishOpts []jetstream.PublishOpt,
) {
	var wantOpts, haveOpts any
	wantPublishOpts = append(wantPublishOpts, mockPublishOpt(&wantOpts))
	havePublishOpts = append(havePublishOpts, mockPublishOpt(&haveOpts))

	v.js.Publish(v.t.Context(), "", nil, wantPublishOpts...) //nolint:errcheck
	v.js.Publish(v.t.Context(), "", nil, havePublishOpts...) //nolint:errcheck
	assert.Equal(v.t, wantOpts, haveOpts)
}

func newPublishOptValidator(t *testing.T) *publishOptValidator {
	s, err := server.NewServer(&server.Options{
		Port: server.RANDOM_PORT,
	})
	require.NoError(t, err)
	s.Start()
	defer s.Shutdown()

	nc, err := nats.Connect(s.ClientURL())
	require.NoError(t, err)

	js, err := jetstream.New(nc)
	require.NoError(t, err)

	return &publishOptValidator{t, js}
}

func TestJetStreamOptions(t *testing.T) {
	t.Parallel()

	t.Run("SetRetryWait", func(t *testing.T) {
		retryWait := time.Second

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetRetryWait(retryWait)

		validator := newPublishOptValidator(t)
		validator.validatePublishOpts(
			[]jetstream.PublishOpt{jetstream.WithRetryWait(retryWait)},
			jetStreamOptions.buildPublishOpts(nil),
		)
	})

	t.Run("SetRetryAttempts", func(t *testing.T) {
		retryAttempts := 1

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetRetryAttempts(retryAttempts)

		validator := newPublishOptValidator(t)
		validator.validatePublishOpts(
			[]jetstream.PublishOpt{jetstream.WithRetryAttempts(retryAttempts)},
			jetStreamOptions.buildPublishOpts(nil),
		)
	})

	t.Run("SetStallWait", func(t *testing.T) {
		stallWait := time.Second

		var jetStreamOptions JetStreamOptions
		jetStreamOptions.SetStallWait(stallWait)

		validator := newPublishOptValidator(t)
		validator.validatePublishOpts(
			[]jetstream.PublishOpt{jetstream.WithStallWait(stallWait)},
			jetStreamOptions.buildPublishOpts(nil),
		)
	})

	t.Run("SetDeduplication", func(t *testing.T) {
		data := []byte("data")
		msgID := strconv.FormatUint(xxhash.Sum64(data), 16)

		t.Run("true", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplication(true)

			validator := newPublishOptValidator(t)
			validator.validatePublishOpts(
				[]jetstream.PublishOpt{jetstream.WithMsgID(msgID)},
				jetStreamOptions.buildPublishOpts(data),
			)
		})

		t.Run("default", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions

			validator := newPublishOptValidator(t)
			validator.validatePublishOpts(
				nil,
				jetStreamOptions.buildPublishOpts(data),
			)
		})

		t.Run("false", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplication(false)

			validator := newPublishOptValidator(t)
			validator.validatePublishOpts(
				nil,
				jetStreamOptions.buildPublishOpts(data),
			)
		})

		t.Run("true false", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplication(true)
			jetStreamOptions.SetDeduplication(false)

			validator := newPublishOptValidator(t)
			validator.validatePublishOpts(
				nil,
				jetStreamOptions.buildPublishOpts(data),
			)
		})

		t.Run("true false true", func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			jetStreamOptions.SetDeduplication(true)
			jetStreamOptions.SetDeduplication(false)
			jetStreamOptions.SetDeduplication(true)

			validator := newPublishOptValidator(t)
			validator.validatePublishOpts(
				[]jetstream.PublishOpt{jetstream.WithMsgID(msgID)},
				jetStreamOptions.buildPublishOpts(data),
			)
		})
	})
}
