package command

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestReply_UnmarshalBinary(t *testing.T) {
	id := uuid.New()
	reply := newReply(id, 100, nil)
	data, err := reply.MarshalBinary()
	assert.Nil(t, err)
	assert.NotZero(t, data)
	reply2 := new(Reply)
	assert.Nil(t, reply2.UnmarshalBinary(data))
	assert.Equal(t, reply.Command(), reply2.Command())
	assert.Equal(t, reply.Unix(), reply2.Unix())
	assert.Equal(t, reply.Err(), reply2.Err())
	assert.Equal(t, reply.StreamVersion(), reply2.StreamVersion())
}

func TestReply_UnmarshalBinaryWithError(t *testing.T) {
	id := uuid.New()
	reply := newReply(id, 100, errors.New("error"))
	data, err := reply.MarshalBinary()
	assert.Nil(t, err)
	assert.NotZero(t, data)
	reply2 := new(Reply)
	assert.Nil(t, reply2.UnmarshalBinary(data))
	assert.Equal(t, reply.Err(), reply2.Err())
}
