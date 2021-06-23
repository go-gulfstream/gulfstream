package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/command"
)

const defaultClientTimeout = 15 * time.Second

type ClientResponseFunc func(w *http.Response)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	endpoint     string
	client       Doer
	commandCodec *command.Codec
	requestFunc  []RequestFunc
	responseFunc []ClientResponseFunc
}

type ClientOption func(*Client)

func NewClient(
	addr string,
	opts ...ClientOption,
) *Client {
	c := &Client{
		client:   &http.Client{Timeout: defaultClientTimeout},
		endpoint: strings.TrimRight(addr, "/"),
	}
	for _, f := range opts {
		f(c)
	}
	return c
}

func WithClientTransport(c Doer) ClientOption {
	return func(cli *Client) {
		cli.client = c
	}
}

func WithClientCodec(c *command.Codec) ClientOption {
	return func(cli *Client) {
		cli.commandCodec = c
	}
}

func WithClientRequestFunc(fn RequestFunc) ClientOption {
	return func(cli *Client) {
		cli.requestFunc = append(cli.requestFunc, fn)
	}
}

func WithClientResponseFunc(fn ClientResponseFunc) ClientOption {
	return func(cli *Client) {
		cli.responseFunc = append(cli.responseFunc, fn)
	}
}

func (c *Client) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	data, err := c.encodeCommand(cmd)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	for _, reqFunc := range c.requestFunc {
		ctx = reqFunc(ctx, req)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	for _, respFunc := range c.responseFunc {
		respFunc(resp)
	}
	rawResp, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusInternalServerError {
		return nil, c.decodeError(rawResp)
	}
	reply := new(command.Reply)
	if err := reply.UnmarshalBinary(rawResp); err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) decodeError(b []byte) error {
	return errors.New(string(b))
}

func (c *Client) encodeCommand(cmd *command.Command) ([]byte, error) {
	if c.commandCodec != nil {
		return c.commandCodec.Encode(cmd)
	} else {
		return command.Encode(cmd)
	}
}
