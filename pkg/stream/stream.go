package stream

import (
	"reflect"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/google/uuid"
)

type Stream struct {
	id      uuid.UUID
	state   State
	version int
	name    string
	owner   uuid.UUID
	changes []*event.Event
}

func New(name string, id uuid.UUID, owner uuid.UUID, state State) *Stream {
	rv := reflect.ValueOf(state)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("stream: New(non-pointer " + rv.String() + ")")
	}
	return &Stream{
		id:      id,
		state:   state,
		name:    name,
		owner:   owner,
		version: -1,
	}
}

func (s *Stream) Owner() uuid.UUID {
	return s.owner
}

func (s *Stream) Changes() []*event.Event {
	return s.changes
}

func (s *Stream) ClearChanges() {
	s.changes = []*event.Event{}
}

func (s *Stream) ID() uuid.UUID {
	return s.id
}

func (s *Stream) Name() string {
	return s.name
}

func (s *Stream) State() State {
	return s.state
}

func (s *Stream) Version() int {
	return s.version + len(s.changes)
}

func (s *Stream) PreviousVersion() int {
	return s.version
}

func (s *Stream) Mutate(eventName string, payload interface{}) {
	s.version++
	e := event.NewEvent(eventName, s.name, s.id, s.owner, s.version, payload)
	s.state.Mutate(e)
	s.changes = append(s.changes, e)
}

func RestoreFromEvent(s *Stream, event *event.Event) {
	s.state.Mutate(event)
	s.version = event.Version()
}
