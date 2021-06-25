package commandbus

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/command"
)

type Sinker interface {
	CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error)
}
