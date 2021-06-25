package eventbus

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type Sinker interface {
	EventSink(ctx context.Context, e *event.Event) error
}

func Mutator(mutator *stream.Mutator) stream.EventHandler {
	return eventBus{
		mutator: mutator,
	}
}

type eventBus struct {
	mutator *stream.Mutator
}

func (eb eventBus) Match(_ string) bool {
	return true
}

func (eb eventBus) Handle(ctx context.Context, e *event.Event) error {
	return eb.mutator.EventSink(ctx, e)
}

func (eb eventBus) Rollback(context.Context, *event.Event) error {
	return nil
}
