package stream

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	BlankStream() *Stream
	Persist(ctx context.Context, s *Stream) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID, owner uuid.UUID) (*Stream, error)
	MarkUnpublished(ctx context.Context, s *Stream) error
	Walk(ctx context.Context, streamName string, streams []uuid.UUID, owner uuid.UUID, iter func(*Stream) error) error
}
