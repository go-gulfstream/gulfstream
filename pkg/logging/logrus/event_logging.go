package logginglogrus

import (
	"context"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/sirupsen/logrus"
)

func NewEventLogging(logger *logrus.Entry) stream.EventSinkerInterceptor {
	return func(sinker stream.EventSinker) stream.EventSinker {
		return eventLogging{
			next:   sinker,
			logger: logger,
		}
	}
}

func DefaultEventLogging() stream.EventSinkerInterceptor {
	return NewEventLogging(logrus.NewEntry(logrus.New()))
}

type eventLogging struct {
	next   stream.EventSinker
	logger *logrus.Entry
}

func (l eventLogging) EventSink(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.EventSink(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.logger.WithError(err).WithField("took", took).Error("Event Sink")
	} else {
		l.logger.WithField("event", e).WithField("took", took).Info("Event Sink")
	}
	return err
}
