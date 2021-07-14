package loggingstd

import (
	"context"
	"log"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func NewCommandSinkerLogging(logger *log.Logger) stream.CommandSinkerInterceptor {
	return func(sinker stream.CommandSinker) stream.CommandSinker {
		return commandLogging{
			next:   sinker,
			logger: logger,
		}
	}
}

func DefaultCommandSinkerLogging() stream.CommandSinkerInterceptor {
	return NewCommandSinkerLogging(log.Default())
}

type commandLogging struct {
	next   stream.CommandSinker
	logger *log.Logger
}

func (l commandLogging) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	startTime := time.Now()
	reply, err := l.next.CommandSink(ctx, cmd)
	took := time.Since(startTime)
	if err != nil {
		l.logger.Printf("[ERROR] CommandSink %s, error=%v, took=%s\n", cmd, err, took)
	} else {
		l.logger.Printf("[INFO] CommandSink %s => %s, took=%s", cmd, reply, took)
	}
	return reply, err
}
