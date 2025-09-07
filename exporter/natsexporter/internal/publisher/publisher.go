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

func NewNatsPublisher(nc *nats.Conn) *NatsPublisher {
	return &NatsPublisher{nc: nc}
}

type JetStreamPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func (p *JetStreamPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := p.js.Publish(ctx, subject, data, jetstream.WithRetryAttempts(0))
	return err
}

func NewJetStreamPublisher(nc *nats.Conn) (*JetStreamPublisher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &JetStreamPublisher{nc: nc, js: js}, nil
}
