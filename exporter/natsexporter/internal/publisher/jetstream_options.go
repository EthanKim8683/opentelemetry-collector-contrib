package publisher

import (
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nats-io/nats.go/jetstream"
)

type buildJetStreamPubOptionFunc func(data []byte) jetstream.PublishOpt

type JetStreamOptions struct {
	buildJetStreamPubOptions []buildJetStreamPubOptionFunc
}

func (po *JetStreamOptions) SetRetryWait(retryWait time.Duration) {
	po.buildJetStreamPubOptions = append(po.buildJetStreamPubOptions, func(_ []byte) jetstream.PublishOpt {
		return jetstream.WithRetryWait(retryWait)
	})
}

func (po *JetStreamOptions) SetRetryAttempts(retryAttempts int) {
	po.buildJetStreamPubOptions = append(po.buildJetStreamPubOptions, func(_ []byte) jetstream.PublishOpt {
		return jetstream.WithRetryAttempts(retryAttempts)
	})
}

func (po *JetStreamOptions) SetStallWait(stallWait time.Duration) {
	po.buildJetStreamPubOptions = append(po.buildJetStreamPubOptions, func(_ []byte) jetstream.PublishOpt {
		return jetstream.WithStallWait(stallWait)
	})
}

func (po *JetStreamOptions) SetDeduplicate(deduplicate bool) {
	if !deduplicate {
		return
	}

	po.buildJetStreamPubOptions = append(po.buildJetStreamPubOptions, func(data []byte) jetstream.PublishOpt {
		hash := xxhash.Sum64(data)
		msgID := strconv.FormatUint(hash, 16)

		return jetstream.WithMsgID(msgID)
	})
}

func (po *JetStreamOptions) buildPublishOpts(data []byte) []jetstream.PublishOpt {
	publishOpts := make([]jetstream.PublishOpt, 0, len(po.buildJetStreamPubOptions))
	for _, buildPublishOpts := range po.buildJetStreamPubOptions {
		publishOpts = append(publishOpts, buildPublishOpts(data))
	}
	return publishOpts
}
