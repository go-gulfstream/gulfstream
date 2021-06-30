package grpc

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"google.golang.org/grpc/metadata"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"google.golang.org/grpc"
)

type ServerRequestFunc func(metadata.MD)
type ServerErrorHandler func(err error)

type Server struct {
	UnimplementedCommandBusServer
	commandCodec command.Encoding
	mutator      stream.CommandSinker
	contextFunc  []ContextFunc
	requestFunc  []ServerRequestFunc
	errorHandler []ServerErrorHandler
}

func NewServer(
	mutator stream.CommandSinker,
	opts ...ServerOption,
) *Server {
	srv := &Server{
		mutator: mutator,
	}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

func (s *Server) Register(grpcSrv *grpc.Server) {
	RegisterCommandBusServer(grpcSrv, s)
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

func (s *Server) CommandSink(ctx context.Context, req *Request) (*Response, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, ctxFunc := range s.contextFunc {
		ctx = ctxFunc(ctx)
	}
	for _, reqFunc := range s.requestFunc {
		reqFunc(md)
	}

	cmd, err := s.decodeCommand(req.Data)
	if err != nil {
		return s.writeError(err), nil
	}
	reply, err := s.mutator.CommandSink(ctx, cmd)
	if err != nil {
		return s.writeError(err), nil
	}
	rawReply, err := reply.MarshalBinary()
	if err != nil {
		return s.writeError(err), nil
	}
	return s.write(rawReply), nil
}

func (s *Server) decodeCommand(data []byte) (*command.Command, error) {
	if s.commandCodec != nil {
		return s.commandCodec.Decode(data)
	} else {
		return command.Decode(data)
	}
}

func (s *Server) write(b []byte) *Response {
	return &Response{Data: b}
}

func (s *Server) writeError(err error) *Response {
	for _, errFunc := range s.errorHandler {
		errFunc(err)
	}
	return &Response{Error: err.Error()}
}
