package eventbus

import (
	"context"
	stdlog "log"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type stdLoggerHandler struct {
	logger *stdlog.Logger
}

func NewStdLoggerHandler(logger *stdlog.Logger) stream.EventHandler {
	stdlog.Print()
	if logger == nil {
		logger = stdlog.Default()
	}
	return stdLoggerHandler{logger: logger}
}

func (stdLoggerHandler) Match(_ string) bool { return true }

func (h stdLoggerHandler) Handle(_ context.Context, e *event.Event) error {
	h.logger.Printf("[EVENT] name=%s, stream=%s, streamID=%s, owner=%s, version=%d, timestamp=%d, payload=%v\n",
		e.Name(), e.StreamName(), e.StreamID(), e.Owner(), e.Version(), e.Unix(), e.Payload())
	return nil
}
func (stdLoggerHandler) Rollback(context.Context, *event.Event) error { return nil }
