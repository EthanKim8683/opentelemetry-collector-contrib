package publisher

import (
	"testing"
	"time"

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
	subject string,
	data []byte,
	wantPublishOptSlice []jetstream.PublishOpt,
	havePublishOptSlice []jetstream.PublishOpt,
) {
	var wantOpts, haveOpts any
	wantPublishOptSlice = append(wantPublishOptSlice, mockPublishOpt(&wantOpts))
	havePublishOptSlice = append(havePublishOptSlice, mockPublishOpt(&haveOpts))

	pov.js.Publish(pov.t.Context(), subject, data, wantPublishOptSlice...)
	pov.js.Publish(pov.t.Context(), subject, data, havePublishOptSlice...)
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

	tests := []struct {
		name                 string
		subject              string
		data                 []byte
		initJetStreamOptions func(jetStreamOptions *JetStreamOptions)
		wantPublishOptSlice  []jetstream.PublishOpt
	}{
		{
			name: "SetRetryWait",
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetRetryWait(time.Second)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{jetstream.WithRetryWait(time.Second)},
		},
		{
			name: "SetRetryAttempts",
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetRetryAttempts(3)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{jetstream.WithRetryAttempts(3)},
		},
		{
			name: "SetStallWait",
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetStallWait(time.Second)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{
				jetstream.WithStallWait(time.Second),
			},
		},
		{
			name: "SetDeduplicate",
			data: []byte("data"),
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetDeduplicate(true)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{
				jetstream.WithMsgID("b7119b48552d1da3"),
			},
		},
		{
			name: "SetDeduplicate",
			data: []byte("data"),
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetDeduplicate(false)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{
				jetstream.WithMsgID(""),
			},
		},
		{
			name: "SetDeduplicate",
			data: []byte("data"),
			initJetStreamOptions: func(jetStreamOptions *JetStreamOptions) {
				jetStreamOptions.SetDeduplicate(true)
				jetStreamOptions.SetDeduplicate(false)
			},
			wantPublishOptSlice: []jetstream.PublishOpt{
				jetstream.WithMsgID(""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jetStreamOptions JetStreamOptions
			tt.initJetStreamOptions(&jetStreamOptions)
			havePublishOptSlice := jetStreamOptions.buildPublishOptSlice(tt.data)

			pov.validatePublishOpt(
				tt.subject,
				tt.data,
				tt.wantPublishOptSlice,
				havePublishOptSlice,
			)
		})
	}
}
