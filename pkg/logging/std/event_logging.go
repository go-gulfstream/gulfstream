package loggingstd

import (
	"context"
	"log"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func NewEventLogging(logger *log.Logger) stream.EventSinkerInterceptor {
	return func(sinker stream.EventSinker) stream.EventSinker {
		return eventLogging{
			next:   sinker,
			logger: logger,
		}
	}
}

func DefaultEventLogging() stream.EventSinkerInterceptor {
	return NewEventLogging(log.Default())
}

type eventLogging struct {
	next   stream.EventSinker
	logger *log.Logger
}

func (l eventLogging) EventSink(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.EventSink(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.logger.Printf("[ERROR] EventSink %s, error=%v, took=%s\n", e, err, took)
	} else {
		l.logger.Printf("[INFO] EventSink %s, took=%s\n", e, took)
	}
	return err
}
