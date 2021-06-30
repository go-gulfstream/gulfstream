package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/google/uuid"
)

func New(streamName string, newStream func() *stream.Stream) stream.Storage {
	return &stateStorage{
		data:        make(map[uuid.UUID][]byte),
		versions:    make(map[uuid.UUID]int),
		blankStream: newStream,
		streamName:  streamName,
	}
}

type stateStorage struct {
	mu          sync.RWMutex
	data        map[uuid.UUID][]byte
	blankStream func() *stream.Stream
	versions    map[uuid.UUID]int
	streamName  string
}

func (s *stateStorage) NewStream() *stream.Stream {
	return s.blankStream()
}

func (s *stateStorage) StreamName() string {
	return s.streamName
}

func (s *stateStorage) Persist(_ context.Context, ss *stream.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ss.Name() != s.streamName {
		return fmt.Errorf("storage: mismatch stream names. got %s, expected %s",
			ss.Name(), s.streamName)
	}
	rawData, found := s.data[ss.ID()]
	if found {
		prev := s.blankStream()
		if err := prev.UnmarshalBinary(rawData); err != nil {
			return err
		}
		if prev.Version() != ss.PreviousVersion() {
			return fmt.Errorf("storage: mismatch stream version. got v%d, expected v%d",
				prev.Version(), ss.PreviousVersion())
		}
	}
	data, err := ss.MarshalBinary()
	if err != nil {
		return err
	}
	s.data[ss.ID()] = data
	return nil
}

func (s *stateStorage) Load(_ context.Context, streamID uuid.UUID) (*stream.Stream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rawData, found := s.data[streamID]
	if !found {
		return nil, fmt.Errorf("storage: %s{StreamID:%s} not found",
			s.streamName, streamID)
	}
	blankStream := s.blankStream()
	if err := blankStream.UnmarshalBinary(rawData); err != nil {
		return nil, err
	}
	return blankStream, nil
}

func (s *stateStorage) MarkUnpublished(_ context.Context, ss *stream.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ss.Name() != s.streamName {
		return fmt.Errorf("storage: mismatch stream names. got %s, expected %s",
			ss.Name(), s.streamName)
	}
	s.versions[ss.ID()] = ss.Version()
	return nil
}

func (s *stateStorage) Iter(_ context.Context, fn func(*stream.Stream) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rawData := range s.data {
		blankStream := s.blankStream()
		if err := blankStream.UnmarshalBinary(rawData); err != nil {
			return err
		}
		if err := fn(blankStream); err != nil {
			return err
		}
	}
	return nil
}
