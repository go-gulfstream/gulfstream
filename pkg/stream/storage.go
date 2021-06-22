package stream

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Storage interface {
	Persist(ctx context.Context, stream *Stream) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID, owner uuid.UUID) (*Stream, error)
	ConfirmVersion(ctx context.Context, version int) error
}

func NewStorage() Storage {
	return &stateStorage{
		data: make(map[key][]byte),
	}
}

type stateStorage struct {
	mu          sync.RWMutex
	data        map[key][]byte
	blankStream func() *Stream
}

type key struct {
	streamType string
	streamID   uuid.UUID
	owner      uuid.UUID
}

func (s *stateStorage) Persist(_ context.Context, stream *Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key{streamType: stream.Name(), owner: stream.Owner(), streamID: stream.ID()}
	rawData, found := s.data[k]
	if found {
		prev := s.blankStream()
		if err := prev.UnmarshalBinary(rawData); err != nil {
			return err
		}
		if prev.Version() != stream.PreviousVersion() {
			return fmt.Errorf("mismatch stream version. got %d, expected %d",
				prev.Version(), stream.PreviousVersion())
		}
	}
	stream.version += len(stream.changes)
	data, err := stream.MarshalBinary()
	if err != nil {
		return err
	}
	s.data[k] = data
	return nil
}

func (s *stateStorage) Load(_ context.Context, streamType string, streamID uuid.UUID, owner uuid.UUID) (*Stream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k := key{streamType: streamType, owner: owner, streamID: streamID}
	rawData, found := s.data[k]
	if !found {
		return nil, fmt.Errorf("%s{ID:%s,Owner:%s} not found",
			streamType, streamID, owner)
	}
	stream := s.blankStream()
	if err := stream.UnmarshalBinary(rawData); err != nil {
		return nil, err
	}
	return stream, nil
}

func (s *stateStorage) ConfirmVersion(ctx context.Context, version int) error {
	return nil
}
