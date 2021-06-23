package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func TestClientServer(t *testing.T) {
	storage := stream.NewStorage(func() *stream.Stream {
		return stream.Blank(&state{One: "one"})
	})
	mutation := stream.NewMutation("order", storage, mockPublisher{})
	mutation.FromCommand("action",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.CreateMode())
	httpServer := NewServer(mutation)
	srv := httptest.NewServer(httpServer)
	defer srv.Close()
	httpClient := NewClient(srv.URL)
	cmd := command.New("action", "order", uuid.New(), uuid.New(), nil)
	reply, err := httpClient.CommandSink(context.Background(), cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())
}

type state struct {
	One string
	Two string
}

func (s *state) Mutate(*event.Event) {}

func (s *state) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *state) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

type mockPublisher struct{}

func (mockPublisher) Publish(ctx context.Context, event []*event.Event) error {
	return nil
}
