package publisher

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStreamPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func (p *JetStreamPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := p.js.Publish(ctx, subject, data)
	return err
}

func NewJetStreamPublisher(nc *nats.Conn) (*JetStreamPublisher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &JetStreamPublisher{nc: nc, js: js}, nil
}
