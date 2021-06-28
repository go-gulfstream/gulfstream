package command

import (
	"time"

	"github.com/google/uuid"
)

type Command struct {
	id         uuid.UUID
	streamID   uuid.UUID
	name       string
	streamName string
	createdAt  int64
	payload    interface{}
}

func New(
	name string,
	streamName string,
	streamID uuid.UUID,
	payload interface{},
) *Command {
	return &Command{
		id:         uuid.New(),
		name:       name,
		streamID:   streamID,
		streamName: streamName,
		createdAt:  time.Now().Unix(),
		payload:    payload,
	}
}

func (c *Command) ReplyOk(version int) *Reply {
	return newReply(c.id, version, nil)
}

func (c *Command) ReplyErr(err error) *Reply {
	return newReply(c.id, 0, err)
}

func (c *Command) ID() uuid.UUID {
	return c.id
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) IsEmptyStreamID() bool {
	return c.streamID == uuid.Nil
}

func (c *Command) StreamID() uuid.UUID {
	return c.streamID
}

func (c *Command) StreamName() string {
	return c.streamName
}

func (c *Command) Payload() interface{} {
	return c.payload
}

func (c *Command) Unix() int64 {
	return c.createdAt
}
