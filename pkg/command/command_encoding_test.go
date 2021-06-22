package command

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestCodec_Decode(t *testing.T) {
	id := uuid.New()
	buf := mockContainer(t, magicNumber, id, []byte(""))
	codec := NewCodec()
	command, err := codec.Decode(buf.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, id, command.ID())
	assert.Equal(t, id, command.StreamID())
	assert.Equal(t, id, command.Owner())
	assert.Equal(t, "name", command.Name())
	assert.Equal(t, "stream", command.StreamName())
	assert.Equal(t, int64(5), command.Unix())
}

func TestCodec_Encode(t *testing.T) {
	codec := NewCodec()
	addCardCommand := New("addCard", "account", uuid.New(), uuid.New(),
		addCard{
			Sum: 789,
		})

	codec.Register("addCard", addCardCodec{})
	data, err := codec.Encode(addCardCommand)
	assert.NoError(t, err)
	assert.NotZero(t, data)

	command2, err := codec.Decode(data)
	assert.NoError(t, err)
	assert.Equal(t, addCardCommand.ID(), command2.ID())
	assert.Equal(t, addCardCommand.StreamID(), command2.StreamID())
	assert.Equal(t, addCardCommand.Owner(), command2.Owner())
	assert.Equal(t, addCardCommand.Name(), command2.Name())
	assert.Equal(t, addCardCommand.StreamName(), command2.StreamName())
	assert.Equal(t, addCardCommand.Unix(), command2.Unix())
	assert.Equal(t, addCardCommand.Payload().(addCard).Sum, command2.Payload().(addCard).Sum)
}

func TestCodec_AddKnownType(t *testing.T) {
	codec := NewCodec()
	assert.Nil(t, codec.AddKnownType((*payload)(nil)))
	assert.Error(t, codec.AddKnownType(123))
}

func TestCodec_DecodeInvalidMagicNumber(t *testing.T) {
	id := uuid.New()
	badMagicNumber := uint16(6)
	buf := mockContainer(t, badMagicNumber, id, []byte(""))
	codec := NewCodec()
	command, err := codec.Decode(buf.Bytes())
	assert.Equal(t, ErrInvalidInputData, err)
	assert.Nil(t, command)
}

func TestCodec_DecodeInvalidInputData(t *testing.T) {
	codec := NewCodec()
	command, err := codec.Decode([]byte("some data"))
	assert.Equal(t, ErrInvalidInputData, err)
	assert.Nil(t, command)
	str := strings.Repeat("o", 512)
	command, err = codec.Decode([]byte(str))
	assert.Equal(t, ErrInvalidInputData, err)
	assert.Nil(t, command)
}

func mockContainer(t *testing.T, mn uint16, id uuid.UUID, payload []byte) *bytes.Buffer {
	buf := &bytes.Buffer{}
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, mn))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(len(payload))))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(4)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, uint32(6)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, id))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, []byte("name")))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, []byte("stream")))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, int64(5)))
	assert.Nil(t, binary.Write(buf, binary.LittleEndian, payload))
	return buf
}

type payload struct {
}

func (p *payload) UnmarshalBinary(data []byte) error {
	return nil
}

type addCardCodec struct{}

func (addCardCodec) Decode(data []byte) (interface{}, error) {
	var p addCard
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (addCardCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
