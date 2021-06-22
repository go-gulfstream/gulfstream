package command

import (
	"time"

	"github.com/google/uuid"
)

type Command struct {
	id         uuid.UUID
	streamID   uuid.UUID
	owner      uuid.UUID
	name       string
	streamName string
	createdAt  int64
	payload    interface{}
}

func New(
	name string,
	streamName string,
	streamID uuid.UUID,
	owner uuid.UUID,
	payload interface{},
) *Command {
	return &Command{
		id:         uuid.New(),
		name:       name,
		streamID:   streamID,
		streamName: streamName,
		owner:      owner,
		createdAt:  time.Now().Unix(),
		payload:    payload,
	}
}

func (c *Command) ReplyOk() *Reply {
	return &Reply{}
}

func (c *Command) ReplyErr() *Reply {
	return &Reply{}
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

func (c *Command) Owner() uuid.UUID {
	return c.owner
}

func (c *Command) Payload() interface{} {
	return c.payload
}

func (c *Command) Unix() int64 {
	return c.createdAt
}
