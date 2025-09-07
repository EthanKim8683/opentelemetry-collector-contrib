package publisher

import (
	"context"

	"github.com/nats-io/nats.go"
)

type CoreNatsPublisher struct {
	nc *nats.Conn
}

func (p *CoreNatsPublisher) Publish(_ context.Context, subject string, data []byte) error {
	return p.nc.Publish(subject, data)
}

func NewCoreNatsPublisher(nc *nats.Conn) *CoreNatsPublisher {
	return &CoreNatsPublisher{nc: nc}
}
