package event

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	id         uuid.UUID
	streamID   uuid.UUID
	streamName string
	name       string
	payload    interface{}
	version    int
	createdAt  int64
}

func New(
	name string,
	streamName string,
	streamID uuid.UUID,
	version int,
	payload interface{},
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

func (e *Event) Payload() interface{} {
	return e.payload
}

func (e *Event) Name() string {
	return e.name
}

func (e *Event) Unix() int64 {
	return e.createdAt
}
