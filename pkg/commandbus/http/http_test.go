package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-gulfstream/gulfstream/pkg/storage"

	"github.com/stretchr/testify/assert"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func TestClientServer(t *testing.T) {
	// setup stream controllers
	mutation := newMutation()
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.CreateMode())

	// setup server side
	httpServer := NewServer(mutation)
	server := httptest.NewServer(httpServer)
	defer server.Close()

	// setup client side
	client := NewClient(server.URL)

	// request from client to server
	cmd := command.New("action", "order", uuid.New(), uuid.New(), nil)
	reply, err := client.CommandSink(context.Background(), cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())
}

func TestServerMiddleware(t *testing.T) {
	validOwnerID := uuid.New()
	invalidOwnerID := uuid.New()
	mutation := newMutation()
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.CreateMode())
	httpServer := NewServer(mutation)
	middleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner := r.Header.Get("X-Owner")
		if owner != validOwnerID.String() {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("forbidden"))
		} else {
			httpServer.ServeHTTP(w, r)
		}
	})
	srv := httptest.NewServer(middleware)
	defer srv.Close()
	httpClient := NewClient(srv.URL,
		WithClientRequestFunc(func(r *http.Request, c *command.Command) {
			r.Header.Set("X-Owner", c.Owner().String())
		}))

	// valid owner
	cmd := command.New("action", "order", uuid.New(), validOwnerID, nil)
	reply, err := httpClient.CommandSink(context.Background(), cmd)
	assert.NoError(t, err)
	assert.Equal(t, reply.Command(), cmd.ID())

	// invalid owner
	cmd = command.New("action", "order", uuid.New(), invalidOwnerID, nil)
	reply, err = httpClient.CommandSink(context.Background(), cmd)
	assert.Error(t, err)
	assert.Contains(t, "forbidden", err.Error())
	assert.Nil(t, reply)
}

func newMutation() *stream.Mutator {
	store := storage.New(func() *stream.Stream {
		return stream.Blank("one", &state{One: "one"})
	})
	return stream.NewMutator("order", store, mockPublisher{})
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
