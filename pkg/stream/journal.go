package stream

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
)

type WalkFunc func(*event.Event) error

type Journal interface {
	Append(ctx context.Context, e []*event.Event, expectedVersion int) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID) ([]*event.Event, error)
	Walk(ctx context.Context, streamName string, streamID uuid.UUID, fn WalkFunc) error
}
