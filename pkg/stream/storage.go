package stream

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Storage interface {
	StreamName() string
	NewStream() *Stream
	Persist(ctx context.Context, s *Stream) error
	Load(ctx context.Context, streamID uuid.UUID) (*Stream, error)
	Drop(ctx context.Context, streamID uuid.UUID) error
}

func NewStorage(streamName string, newStream func() *Stream) Storage {
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
	blankStream func() *Stream
	versions    map[uuid.UUID]int
	streamName  string
}

func (s *stateStorage) NewStream() *Stream {
	return s.blankStream()
}

func (s *stateStorage) StreamName() string {
	return s.streamName
}

func (s *stateStorage) Persist(_ context.Context, ss *Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.Compare(s.streamName, ss.Name()) != 0 {
		return fmt.Errorf("storage: different stream names got %s, expected %s",
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

func (s *stateStorage) Load(_ context.Context, streamID uuid.UUID) (*Stream, error) {
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

func (s *stateStorage) Drop(_ context.Context, streamID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, streamID)
	return nil
}
