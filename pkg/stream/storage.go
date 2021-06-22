package stream

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	Persist(ctx context.Context, stream *Stream) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID, owner uuid.UUID) (*Stream, error)
	ConfirmVersion(ctx context.Context, version int) error
}
