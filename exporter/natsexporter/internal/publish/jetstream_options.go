package publish

import (
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nats-io/nats.go/jetstream"
)

type buildJetStreamPublishOptFunc func(data []byte) jetstream.PublishOpt

type JetStreamOptions struct {
	buildPublishOptFuncs []buildJetStreamPublishOptFunc
}

func (jso *JetStreamOptions) SetRetryWait(retryWait time.Duration) {
	jso.buildPublishOptFuncs = append(jso.buildPublishOptFuncs, func(data []byte) jetstream.PublishOpt {
		return jetstream.WithRetryWait(retryWait)
	})
}

func (jso *JetStreamOptions) SetRetryAttempts(retryAttempts int) {
	jso.buildPublishOptFuncs = append(jso.buildPublishOptFuncs, func(data []byte) jetstream.PublishOpt {
		return jetstream.WithRetryAttempts(retryAttempts)
	})
}

func (jso *JetStreamOptions) SetStallWait(stallWait time.Duration) {
	jso.buildPublishOptFuncs = append(jso.buildPublishOptFuncs, func(data []byte) jetstream.PublishOpt {
		return jetstream.WithStallWait(stallWait)
	})
}

func (jso *JetStreamOptions) SetDedup(dedup bool) {
	var buildPublishOptFunc buildJetStreamPublishOptFunc
	if dedup {
		buildPublishOptFunc = func(data []byte) jetstream.PublishOpt {
			hash := xxhash.Sum64(data)
			msgID := strconv.FormatUint(hash, 16)
			return jetstream.WithMsgID(msgID)
		}
	} else {
		buildPublishOptFunc = func(data []byte) jetstream.PublishOpt {
			return jetstream.WithMsgID("")
		}
	}
	jso.buildPublishOptFuncs = append(jso.buildPublishOptFuncs, buildPublishOptFunc)
}

func (jso *JetStreamOptions) buildPublishOpts(data []byte) []jetstream.PublishOpt {
	publishOpts := make([]jetstream.PublishOpt, 0, len(jso.buildPublishOptFuncs))
	for _, buildPublishOptFunc := range jso.buildPublishOptFuncs {
		publishOpts = append(publishOpts, buildPublishOptFunc(data))
	}
	return publishOpts
}
