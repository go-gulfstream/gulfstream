package stream

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type Publisher interface {
	Publish(event []*event.Event) error
}

type Subscriber interface {
	Subscribe(streamName string, h ...EventHandler)
}

type EventHandler interface {
	Match(eventName string) bool
	Handle(context.Context, *event.Event) error
	Rollback(context.Context, *event.Event) error
}

type EventErrorHandler interface {
	HandleError(ctx context.Context, e *event.Event, err error)
}
