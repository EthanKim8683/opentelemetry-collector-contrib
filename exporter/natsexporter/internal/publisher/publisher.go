package publisher

import (
	"context"
)

type Publisher interface {
	Start() error
	Publish(ctx context.Context, subject string, data []byte) error
	Shutdown() error
}
