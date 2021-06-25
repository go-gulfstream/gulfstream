package eventbus

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type Sinker interface {
	EventSink(ctx context.Context, e *event.Event) error
}
