package command

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCodec_Decode(t *testing.T) {
	c := NewCodec()
	c.Register("some", &some{})
	payload := &some{One: "one", Two: "two"}
	data, err := payload.MarshalBinary()
	id := uuid.New()
	assert.NoError(t, err)
	buf := mockContainer(t, id, data)
	cmd, err := c.Decode(buf.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, id, cmd.ID())
	assert.Equal(t, "some", cmd.Name())
	assert.Equal(t, "one", cmd.Payload().(*some).One)
}

func TestCodec_Encode(t *testing.T) {
	c := NewCodec()
	c.Register("some", &some{})
	id := uuid.New()
	payload := &some{One: "one", Two: "two"}
	cmd := New("some", "some", id, payload)
	data, err := c.Encode(cmd)
	assert.NoError(t, err)
	assert.NotZero(t, data)
	cmd2, err := c.Decode(data)
	assert.NoError(t, err)
	assert.NotNil(t, cmd2)
	assert.Equal(t, cmd.ID(), cmd2.ID())
	assert.Equal(t, cmd.Payload().(*some).One, cmd2.Payload().(*some).One)
}

type some struct {
	One string
	Two string
}

func (s *some) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *some) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

func mockContainer(t *testing.T, id uuid.UUID, payload []byte) *bytes.Buffer {
	buf := &bytes.Buffer{}
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, commandMagicNumber))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(len(payload))))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(4)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(6)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, []byte("some")))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, []byte("stream")))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, int64(5)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, payload))
	return buf
}
