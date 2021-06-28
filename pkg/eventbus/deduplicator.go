package eventbus

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type Deduplicator interface {
	SetVisit(context.Context, *event.Event) error
	HasVisit(context.Context, *event.Event) (bool, error)
}
