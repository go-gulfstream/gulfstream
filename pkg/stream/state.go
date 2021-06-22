package stream

import (
	"encoding"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type State interface {
	Mutate(*event.Event)
	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}
