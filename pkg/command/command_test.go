package command

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	var (
		name     = "addCard"
		streamID = uuid.New()
		stream   = "account"
	)
	addCardCommand := New(name, stream, streamID,
		addCard{
			Sum: 5000,
		})
	assert.NotNil(t, addCardCommand)
	assert.False(t, addCardCommand.IsEmptyStreamID())
	assert.Equal(t, stream, addCardCommand.StreamName())
	assert.Equal(t, streamID, addCardCommand.StreamID())
	assert.NotZero(t, addCardCommand.Unix())
	assert.Equal(t, name, addCardCommand.Name())
	assert.Equal(t, 5000, addCardCommand.Payload().(addCard).Sum)
	assert.False(t, uuid.Nil == addCardCommand.ID())
}

type addCard struct {
	Sum int
}
