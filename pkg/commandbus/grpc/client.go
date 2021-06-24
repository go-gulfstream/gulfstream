package grpc

import (
	"context"
	"errors"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientRequestFunc func(*metadata.MD)
type ContextFunc func(ctx context.Context) context.Context

type Client struct {
	client       CommandBusClient
	callOpts     []grpc.CallOption
	requestFunc  []ClientRequestFunc
	contextFunc  []ContextFunc
	commandCodec *command.Codec
}

func NewClient(
	conn *grpc.ClientConn,
	opts ...ClientOption,
) *Client {
	c := &Client{
		client: NewCommandBusClient(conn),
	}
	for _, f := range opts {
		f(c)
	}
	return c
}

func WithClientContextFunc(fn ContextFunc) ClientOption {
	return func(cli *Client) {
		cli.contextFunc = append(cli.contextFunc, fn)
	}
}

func WithClientRequestFunc(fn ClientRequestFunc) ClientOption {
	return func(cli *Client) {
		cli.requestFunc = append(cli.requestFunc, fn)
	}
}

func WithClientCodec(c *command.Codec) ClientOption {
	return func(cli *Client) {
		cli.commandCodec = c
	}
}

func WithClientCallOptions(opts ...grpc.CallOption) ClientOption {
	return func(cli *Client) {
		cli.callOpts = append(cli.callOpts, opts...)
	}
}

type ClientOption func(*Client)

func (c *Client) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	data, err := c.encodeCommand(cmd)
	if err != nil {
		return nil, err
	}
	for _, ctxFunc := range c.contextFunc {
		ctx = ctxFunc(ctx)
	}
	md := metadata.MD{}
	for _, reqFunc := range c.requestFunc {
		reqFunc(&md)
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	resp, err := c.client.CommandSink(ctx, &Request{
		Message: data,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Error) > 0 {
		return nil, c.decodeError(resp.Error)
	}
	reply := new(command.Reply)
	if err := reply.UnmarshalBinary(resp.Message); err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) encodeCommand(cmd *command.Command) ([]byte, error) {
	if c.commandCodec != nil {
		return c.commandCodec.Encode(cmd)
	} else {
		return command.Encode(cmd)
	}
}

func (c *Client) decodeError(err string) error {
	return errors.New(err)
}
