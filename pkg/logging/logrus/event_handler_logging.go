package logginglogrus

import (
	"context"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/sirupsen/logrus"
)

type eventHandlerLogging struct {
	next   stream.EventHandler
	logger *logrus.Entry
}

func NewEventHandlerLogging(logger *logrus.Entry) stream.EventHandlerInterceptor {
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
		l.logger.
			WithError(err).
			WithField("took", took).
			WithField("event", e).
			Error("EventHandler")
	} else {
		l.logger.
			WithField("took", took).
			WithField("event", e).
			Info("EventHandler")
	}
	return err
}

func (l eventHandlerLogging) Rollback(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.Handle(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.logger.
			WithError(err).
			WithField("took", took).
			WithField("event", e).
			Error("EventHandler rollback")
	} else {
		l.logger.
			WithField("took", took).
			WithField("event", e).
			Info("EventHandler rollback")
	}
	return err
}
