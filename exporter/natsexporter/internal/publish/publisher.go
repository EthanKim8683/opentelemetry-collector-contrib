// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package publish // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/natsexporter/internal/publish"

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

type CoreNatsPublisher struct {
	natsOptions *NatsOptions
	nc          *nats.Conn
}

func (p *CoreNatsPublisher) Connect() error {
	options := p.natsOptions.buildOptions()
	nc, err := options.Connect()
	if err != nil {
		return err
	}
	p.nc = nc

	return nil
}

func (p *CoreNatsPublisher) Publish(_ context.Context, subject string, data []byte) error {
	return p.nc.Publish(subject, data)
}

func (p *CoreNatsPublisher) Disconnect() error {
	if p.nc != nil {
		if err := p.nc.Drain(); err != nil {
			p.nc.Close()
			return err
		}
	}
	return nil
}

var _ Publisher = (*CoreNatsPublisher)(nil)

func NewCoreNatsPublisher(natsOptions *NatsOptions) Publisher {
	return &CoreNatsPublisher{
		natsOptions: natsOptions,
	}
}

type JetStreamPublisher struct {
	natsOptions      *NatsOptions
	jetStreamOptions *JetStreamOptions
	nc               *nats.Conn
	js               jetstream.JetStream
}

func (p *JetStreamPublisher) Connect() error {
	options := p.natsOptions.buildOptions()
	nc, err := options.Connect()
	if err != nil {
		return err
	}
	p.nc = nc

	js, err := jetstream.New(nc)
	if err != nil {
		return err
	}
	p.js = js

	return nil
}

func (p *JetStreamPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	publishOpts := p.jetStreamOptions.buildPublishOpts(data)
	if _, err := p.js.Publish(ctx, subject, data, publishOpts...); err != nil {
		return err
	}
	return nil
}

func (p *JetStreamPublisher) Disconnect() error {
	if p.js != nil {
		p.js.CleanupPublisher()
	}

	if p.nc != nil {
		if err := p.nc.Drain(); err != nil {
			p.nc.Close()
			return err
		}
	}

	return nil
}

var _ Publisher = (*JetStreamPublisher)(nil)

func NewJetStreamPublisher(natsOptions *NatsOptions, jetStreamOptions *JetStreamOptions) Publisher {
	return &JetStreamPublisher{
		natsOptions:      natsOptions,
		jetStreamOptions: jetStreamOptions,
	}
}
