package stream

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestStream_UnmarshalBinary(t *testing.T) {
	state := &myState{One: "one", Two: "two"}
	payload, err := state.MarshalBinary()
	assert.NoError(t, err)
	payloadSize := uint32(len(payload))
	name := []byte("stream")
	nameSize := uint32(len(name))
	updatedAt := int64(20)
	buf := &bytes.Buffer{}
	id := uuid.New()
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, magicNumber))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, payloadSize))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, nameSize))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, name))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, int64(10)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, updatedAt))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, payload))
	s := new(Stream)
	s.state = state
	assert.Nil(t, s.UnmarshalBinary(buf.Bytes()))
	assert.Equal(t, id, s.ID())
	assert.Equal(t, id, s.Owner())
	assert.Equal(t, 10, s.Version())
	assert.Equal(t, updatedAt, s.Unix())
	assert.Equal(t, state.One, s.State().(*myState).One)
	assert.Equal(t, state.Two, s.State().(*myState).Two)
}

func TestStream_MarshalBinary(t *testing.T) {
	stream1 := New("name", uuid.New(), uuid.New(),
		&myState{
			One: "one",
			Two: "two",
		})
	data, err := stream1.MarshalBinary()
	assert.NoError(t, err)
	assert.NotZero(t, data)
	stream2 := Blank("name", &myState{})
	assert.NoError(t, stream2.UnmarshalBinary(data))
	assert.Equal(t, stream1.ID(), stream2.ID())
	assert.Equal(t, stream1.Owner(), stream2.Owner())
	assert.Equal(t, stream1.Name(), stream2.Name())
	assert.Equal(t, stream1.Version(), stream2.Version())
	assert.Equal(t, stream1.Unix(), stream2.Unix())
}

type myState struct {
	One string
	Two string
}

func (s *myState) Mutate(e *event.Event) {

}

func (s *myState) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *myState) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
