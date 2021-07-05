package storage

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

const streamName = "test"

type state struct {
	One string
	Two string
}

func (s *state) Mutate(e *event.Event) {

}

func (s *state) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *state) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

func blankStream() *stream.Stream {
	return stream.New(streamName, uuid.New(), &state{One: "One", Two: "Two"})
}
