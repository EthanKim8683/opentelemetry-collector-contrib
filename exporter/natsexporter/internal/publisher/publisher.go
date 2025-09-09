package publisher

import (
	"context"
)

type Publisher interface {
	Connect() error
	Publish(ctx context.Context, subject string, data []byte) error
	Disconnect() error
}
