package stream

import (
	"reflect"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/google/uuid"
)

const placeholderThreshold = 100

type Stream struct {
	id        uuid.UUID
	name      string
	version   int
	updatedAt int64
	state     State
	changes   []*event.Event
}

func New(name string, id uuid.UUID, initState State) *Stream {
	if len(name) == 0 {
		panic("no stream name")
	}
	checkPtr(initState)
	return &Stream{
		id:    id,
		state: initState,
		name:  name,
	}
}

func Blank(name string, initState State) *Stream {
	return New(name, uuid.UUID{}, initState)
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

func (s *Stream) Unix() int64 {
	return s.updatedAt
}

func (s *Stream) Mutate(eventName string, payload interface{}) {
	version := s.Version() + 1
	if s.isPlaceholderVersion() {
		version++
	}
	e := event.New(eventName, s.name, s.id, version, payload)
	s.state.Mutate(e)
	s.changes = append(s.changes, e)
	s.updatedAt = time.Now().Unix()
}

func (s *Stream) isPlaceholderVersion() bool {
	return s.Version()+1%placeholderThreshold == 0
}

func RestoreFromEvent(s *Stream, event *event.Event) {
	s.state.Mutate(event)
	s.version = event.Version()
}

func checkPtr(state State) {
	rv := reflect.ValueOf(state)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("stream: New(non-pointer " + rv.String() + ")")
	}
}
