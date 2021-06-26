package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestCodec_Decode(t *testing.T) {
	e := New("somePayload", "party", uuid.New(), 1,
		&somePayload{Test: "Test"},
	)
	c := NewCodec()
	assert.Nil(t, c.AddKnownType((*somePayload)(nil)))
	data, err := c.Encode(e)
	assert.Nil(t, err)
	assert.NotZero(t, data)
	e2, err := c.Decode(data)
	assert.Nil(t, err)
	assert.Equal(t, e.ID(), e2.ID())
	assert.Equal(t, e.StreamID(), e2.StreamID())
	assert.Equal(t, e.StreamName(), e2.StreamName())
	assert.Equal(t, e.Version(), e2.Version())
	assert.Equal(t, e.Name(), e2.Name())
	assert.Equal(t, e.Unix(), e2.Unix())
	assert.Equal(t, e.Payload().(*somePayload).Test, e2.Payload().(*somePayload).Test)
}

type somePayload struct {
	Test string
}

func (p *somePayload) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *somePayload) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
