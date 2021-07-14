package logginglogrus

import (
	"context"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/sirupsen/logrus"
)

func NewCommandSinkerLogging(logger *logrus.Entry) stream.CommandSinkerInterceptor {
	return func(sinker stream.CommandSinker) stream.CommandSinker {
		return commandLogging{
			next:   sinker,
			logger: logger,
		}
	}
}

func DefaultCommandSinkerLogging() stream.CommandSinkerInterceptor {
	return NewCommandSinkerLogging(logrus.NewEntry(logrus.New()))
}

type commandLogging struct {
	next   stream.CommandSinker
	logger *logrus.Entry
}

func (l commandLogging) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	startTime := time.Now()
	reply, err := l.next.CommandSink(ctx, cmd)
	took := time.Since(startTime)
	if err != nil {
		l.logger.
			WithField("command", cmd).
			WithError(err).
			WithField("took", took).
			Error("CommandSink")
	} else {
		l.logger.
			WithField("command", cmd).
			WithField("reply", reply).
			WithField("took", took).
			Info("CommandSink")
	}
	return reply, err
}
