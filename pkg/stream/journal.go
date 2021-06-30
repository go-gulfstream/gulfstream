package stream

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
)

type Journal interface {
	Append(ctx context.Context, e []*event.Event, expectedVersion int) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID) ([]*event.Event, error)
}
