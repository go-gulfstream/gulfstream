package http

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/go-gulfstream/gulfstream/pkg/command"

	"github.com/go-gulfstream/gulfstream/pkg/commandbus"
)

type ServerRequestFunc func(r *http.Request)
type ServerResponseFunc func(w http.ResponseWriter)
type ContextFunc func(ctx context.Context) context.Context

type Server struct {
	mutation     commandbus.CommandBus
	commandCodec *command.Codec
	requestFunc  []ServerRequestFunc
	responseFunc []ServerResponseFunc
	contextFunc  []ContextFunc
}

func NewServer(
	mutation commandbus.CommandBus,
) *Server {
	return &Server{
		mutation:     mutation,
		requestFunc:  []ServerRequestFunc{},
		responseFunc: []ServerResponseFunc{},
	}
}

type ServerOption func(*Server)

func WithServerCodec(c *command.Codec) ServerOption {
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	for _, ctxFunc := range s.contextFunc {
		ctx = ctxFunc(ctx)
	}

	for _, reqFunc := range s.requestFunc {
		reqFunc(r)
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, err)
		return
	}
	defer r.Body.Close()

	cmd, err := s.decodeCommand(data)
	if err != nil {
		s.writeError(w, err)
		return
	}
	reply, err := s.mutation.CommandSink(ctx, cmd)
	if err != nil {
		s.writeError(w, err)
		return
	}

	rawReply, err := reply.MarshalBinary()
	if err != nil {
		s.writeError(w, err)
		return
	}
	for _, respFunc := range s.responseFunc {
		respFunc(w)
	}
	_, _ = w.Write(rawReply)
}

func (s *Server) decodeCommand(data []byte) (*command.Command, error) {
	if s.commandCodec != nil {
		return s.commandCodec.Decode(data)
	} else {
		return command.Decode(data)
	}
}

func (s *Server) writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(err.Error()))
}