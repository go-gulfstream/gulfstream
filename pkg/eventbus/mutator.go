package eventbus

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

func MutatorHandler(mutator stream.EventSinker) stream.EventHandler {
	return mutatorHandler{
		mutator: mutator,
	}
}

type mutatorHandler struct {
	mutator stream.EventSinker
}

func (mutatorHandler) Match(_ string) bool { return true }

func (eb mutatorHandler) Handle(ctx context.Context, e *event.Event) error {
	return eb.mutator.EventSink(ctx, e)
}
func (mutatorHandler) Rollback(context.Context, *event.Event) error { return nil }
