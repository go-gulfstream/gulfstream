package command

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Reply struct {
	command   uuid.UUID
	createdAt int64
	err       error
	version   int
}

func newReply(commandID uuid.UUID, version int, err error) *Reply {
	return &Reply{
		version:   version,
		err:       err,
		command:   commandID,
		createdAt: time.Now().Unix(),
	}
}

func (r *Reply) String() string {
	return fmt.Sprintf("CommandReply{Command:%s, Version:%d, Err:%v}",
		r.command, r.version, r.err)
}

func (r *Reply) StreamVersion() int {
	return r.version
}

func (r *Reply) Command() uuid.UUID {
	return r.command
}

func (r *Reply) Unix() int64 {
	return r.createdAt
}

func (r *Reply) Err() error {
	return r.err
}
