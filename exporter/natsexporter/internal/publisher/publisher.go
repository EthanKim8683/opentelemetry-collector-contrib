package publisher

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

type NatsPublisher struct {
	nc *nats.Conn
}

func (p *NatsPublisher) Publish(_ context.Context, subject string, data []byte) error {
	return p.nc.Publish(subject, data)
}

var _ Publisher = (*NatsPublisher)(nil)

func NewNatsPublisher(natsOptions *NatsOptions) (Publisher, error) {
	nc, err := natsOptions.buildOptions().Connect()
	if err != nil {
		return nil, err
	}

	return &NatsPublisher{
		nc: nc,
	}, nil
}

type JetStreamPublisher struct {
	js               jetstream.JetStream
	jetStreamOptions *JetStreamOptions
}

func (p *JetStreamPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := p.js.Publish(ctx, subject, data, p.jetStreamOptions.buildPublishOpts(data)...)
	return err
}

var _ Publisher = (*JetStreamPublisher)(nil)

func NewJetStreamPublisher(natsOptions *NatsOptions, jetStreamOptions *JetStreamOptions) (Publisher, error) {
	nc, err := natsOptions.buildOptions().Connect()
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &JetStreamPublisher{
		js:               js,
		jetStreamOptions: jetStreamOptions,
	}, nil
}
