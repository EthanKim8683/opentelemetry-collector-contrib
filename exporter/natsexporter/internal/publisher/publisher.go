package publisher

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Publisher interface {
	Connect() error
	Publish(ctx context.Context, subject string, data []byte) error
	Disconnect() error
}

type NatsPublisher struct {
	natsOptions *NatsOptions
	nc          *nats.Conn
}

func (p *NatsPublisher) Connect() error {
	nc, err := p.natsOptions.buildOptions().Connect()
	if err != nil {
		return err
	}
	p.nc = nc

	return nil
}

func (p *NatsPublisher) Publish(_ context.Context, subject string, data []byte) error {
	return p.nc.Publish(subject, data)
}

func (p *NatsPublisher) Disconnect() error {
	// TODO: Figure out Drain
	// return p.nc.Drain()
	p.nc.Close()
	return nil
}

var _ Publisher = (*NatsPublisher)(nil)

func NewNatsPublisher(natsOptions *NatsOptions) (Publisher, error) {
	return &NatsPublisher{
		natsOptions: natsOptions,
	}, nil
}

type JetStreamPublisher struct {
	natsOptions      *NatsOptions
	jetStreamOptions *JetStreamOptions
	nc               *nats.Conn
	js               jetstream.JetStream
}

func (p *JetStreamPublisher) Connect() error {
	nc, err := p.natsOptions.buildOptions().Connect()
	if err != nil {
		return err
	}
	p.nc = nc

	js, err := jetstream.New(nc, p.jetStreamOptions.buildJetStreamOptSlice()...)
	if err != nil {
		return err
	}
	p.js = js

	return nil
}

func (p *JetStreamPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := p.js.Publish(ctx, subject, data, p.jetStreamOptions.buildPublishOptSlice(data)...)
	return err
}

func (p *JetStreamPublisher) Disconnect() error {
	p.js.CleanupPublisher()
	// TODO: Figure out Drain
	// return p.nc.Drain()
	p.nc.Close()
	return nil
}

var _ Publisher = (*JetStreamPublisher)(nil)

func NewJetStreamPublisher(natsOptions *NatsOptions, jetStreamOptions *JetStreamOptions) (Publisher, error) {
	return &JetStreamPublisher{
		natsOptions:      natsOptions,
		jetStreamOptions: jetStreamOptions,
	}, nil
}
