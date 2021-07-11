package event

import (
	"fmt"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/codec"

	"github.com/google/uuid"
)

type Event struct {
	id         uuid.UUID
	streamID   uuid.UUID
	streamName string
	name       string
	payload    codec.Codec
	version    int
	createdAt  int64
}

func New(
	name string,
	streamName string,
	streamID uuid.UUID,
	version int,
	payload codec.Codec,
) *Event {
	return &Event{
		id:         uuid.New(),
		streamName: streamName,
		streamID:   streamID,
		name:       name,
		payload:    payload,
		version:    version,
		createdAt:  time.Now().Unix(),
	}
}

func (e *Event) String() string {
	return fmt.Sprintf("Event{ID:%s, Name:%s, Version:%d, StreamName:%s, StreamID:%s, CreatedAt:%d, Payload: %v}",
		e.id, e.name, e.version, e.streamName, e.streamID, e.createdAt, e.payload)
}

func (e *Event) ID() uuid.UUID {
	return e.id
}

func (e *Event) StreamID() uuid.UUID {
	return e.streamID
}

func (e *Event) StreamName() string {
	return e.streamName
}

func (e *Event) Version() int {
	return e.version
}

func (e *Event) Payload() codec.Codec {
	return e.payload
}

func (e *Event) Name() string {
	return e.name
}

func (e *Event) Unix() int64 {
	return e.createdAt
}
