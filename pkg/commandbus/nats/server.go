package nats

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/nats-io/nats.go"
)

const errKey = "_e"

type ServerRequestFunc func(h nats.Header, c *command.Command)
type ServerResponseFunc func(h nats.Header, r *command.Reply)
type ContextFunc func(ctx context.Context) context.Context
type ServerErrorHandler func(msg *nats.Msg, err error)

type Server struct {
	subject      string
	mutator      stream.CommandSinker
	commandCodec command.Encoding
	requestFunc  []ServerRequestFunc
	responseFunc []ServerResponseFunc
	contextFunc  []ContextFunc
	errorHandler []ServerErrorHandler
}

func NewServer(
	subject string,
	mutator stream.CommandSinker,
	opts ...ServerOption,
) *Server {
	srv := &Server{
		subject: toSubj(subject),
		mutator: mutator,
	}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

type ServerOption func(*Server)

func WithServerCodec(c command.Encoding) ServerOption {
	return func(srv *Server) {
		srv.commandCodec = c
	}
}

func WithServerRequestFunc(fn ServerRequestFunc) ServerOption {
	return func(srv *Server) {
		srv.requestFunc = append(srv.requestFunc, fn)
	}
}

func WithServerResponseFunc(fn ServerResponseFunc) ServerOption {
	return func(srv *Server) {
		srv.responseFunc = append(srv.responseFunc, fn)
	}
}

func WithServerContextFunc(fn ContextFunc) ServerOption {
	return func(srv *Server) {
		srv.contextFunc = append(srv.contextFunc, fn)
	}
}

func WithServerErrorHandler(fn ServerErrorHandler) ServerOption {
	return func(srv *Server) {
		srv.errorHandler = append(srv.errorHandler, fn)
	}
}

func (s *Server) Listen(conn *nats.Conn) error {
	if _, err := conn.QueueSubscribe(s.subject, s.subject, func(msg *nats.Msg) {
		rawReply := s.handleMsg(msg)
		if err := msg.Respond(rawReply); err != nil {
			s.handleError(msg, err)
		}
	}); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	return conn.LastError()
}

func (s *Server) handleMsg(msg *nats.Msg) []byte {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, ctxFunc := range s.contextFunc {
		ctx = ctxFunc(ctx)
	}

	if msg.Header == nil {
		msg.Header = nats.Header{}
	}

	cmd, err := s.decodeCommand(msg.Data)
	if err != nil {
		return s.writeError(msg, err)
	}
	for _, reqFunc := range s.requestFunc {
		reqFunc(msg.Header, cmd)
	}

	reply, err := s.mutator.CommandSink(ctx, cmd)
	if err != nil {
		return s.writeError(msg, err)
	}

	for _, respFunc := range s.responseFunc {
		respFunc(msg.Header, reply)
	}

	rawReply, err := reply.MarshalBinary()
	if err != nil {
		return s.writeError(msg, err)
	}

	msg.Header.Del(errKey)

	return rawReply
}

func (s *Server) handleError(msg *nats.Msg, err error) {
	for _, errFunc := range s.errorHandler {
		errFunc(msg, err)
	}
}

func (s *Server) writeError(msg *nats.Msg, err error) []byte {
	s.handleError(msg, err)
	msg.Header.Set(errKey, errKey)
	return []byte(err.Error())
}

func (s *Server) decodeCommand(data []byte) (*command.Command, error) {
	if s.commandCodec != nil {
		return s.commandCodec.Decode(data)
	} else {
		return command.Decode(data)
	}
}
