package loggingstd

import (
	"context"
	"log"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

type eventHandlerLogging struct {
	next   stream.EventHandler
	logger *log.Logger
}

func NewEventHandlerLogging(logger *log.Logger) stream.EventHandlerInterceptor {
	return func(sinker stream.EventHandler) stream.EventHandler {
		return eventHandlerLogging{
			next:   sinker,
			logger: logger,
		}
	}
}

func (l eventHandlerLogging) Match(eventName string) bool {
	return l.next.Match(eventName)
}

func (l eventHandlerLogging) Handle(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.Handle(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.logger.Printf("[ERROR] EventHandler %s, error=%v, took=%s\n", e, err, took)
	} else {
		l.logger.Printf("[INFO] EventHandler %s, took=%s", e, took)
	}
	return err
}

func (l eventHandlerLogging) Rollback(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.Handle(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.logger.Printf("[ERROR] EventHandler rollback %s, error=%v, took=%s\n", e, err, took)
	} else {
		l.logger.Printf("[INFO] EventHandler rollback %s, took=%s", e, took)
	}
	return err
}
