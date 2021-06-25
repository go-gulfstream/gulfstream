package nats

import (
	"context"
	"errors"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/nats-io/nats.go"
)

const defaultClientTimeout = 15 * time.Second

type ClientResponseFunc func(h nats.Header, c *command.Reply)
type ClientRequestFunc func(h nats.Header, c *command.Command)

type Client struct {
	subject      string
	conn         *nats.Conn
	commandCodec *command.Codec
	requestFunc  []ClientRequestFunc
	responseFunc []ClientResponseFunc
	contextFunc  []ContextFunc
	timeout      time.Duration
}

func NewClient(
	subject string,
	conn *nats.Conn,
	opts ...ClientOption,
) *Client {
	c := &Client{
		subject: toSubj(subject),
		conn:    conn,
		timeout: defaultClientTimeout,
	}
	for _, f := range opts {
		f(c)
	}
	return c
}

type ClientOption func(*Client)

func WithClientCodec(c *command.Codec) ClientOption {
	return func(cli *Client) {
		cli.commandCodec = c
	}
}

func WithClientTimeout(dur time.Duration) ClientOption {
	return func(cli *Client) {
		cli.timeout = dur
	}
}

func WithClientRequestFunc(fn ClientRequestFunc) ClientOption {
	return func(cli *Client) {
		cli.requestFunc = append(cli.requestFunc, fn)
	}
}

func WithClientContextFunc(fn ContextFunc) ClientOption {
	return func(cli *Client) {
		cli.contextFunc = append(cli.contextFunc, fn)
	}
}

func WithClientResponseFunc(fn ClientResponseFunc) ClientOption {
	return func(cli *Client) {
		cli.responseFunc = append(cli.responseFunc, fn)
	}
}

func (c *Client) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	if c.conn.Status() != nats.CONNECTED {
		return nil, nats.ErrConnectionClosed
	}

	data, err := c.encodeCommand(cmd)
	if err != nil {
		return nil, err
	}
	for _, ctxFunc := range c.contextFunc {
		ctx = ctxFunc(ctx)
	}

	inMsg := nats.NewMsg(c.subject)
	inMsg.Data = data
	for _, reqFunc := range c.requestFunc {
		reqFunc(inMsg.Header, cmd)
	}
	outMsg, err := c.conn.RequestMsg(inMsg, c.timeout)
	if err != nil {
		return nil, c.conn.LastError()
	}
	if outMsg.Header.Get(errKey) == errKey {
		return nil, errors.New(string(outMsg.Data))
	}
	reply := new(command.Reply)
	if err := reply.UnmarshalBinary(outMsg.Data); err != nil {
		return nil, err
	}
	for _, respFunc := range c.responseFunc {
		respFunc(outMsg.Header, reply)
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

func toSubj(s string) string {
	return s + "-gulfstream"
}
