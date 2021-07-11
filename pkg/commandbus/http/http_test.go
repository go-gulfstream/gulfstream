package commandbushttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func TestClientServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	// setup stream controllers
	mutation := newMutation(ctrl)
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.WithCommandControllerCreateIfNotExists())

	// setup server side
	httpServer := NewServer(mutation)
	server := httptest.NewServer(httpServer)
	defer server.Close()

	// setup client side
	client := NewClient(server.URL)

	// request from client to server
	cmd := command.New("action", "order", uuid.New(), nil)
	reply, err := client.CommandSink(context.Background(), cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())
}

func TestServerMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	validID := uuid.New()
	invalidID := uuid.New()
	mutation := newMutation(ctrl)
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.WithCommandControllerCreateIfNotExists())
	httpServer := NewServer(mutation)
	middleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner := r.Header.Get("X-Owner")
		if owner != validID.String() {
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
			r.Header.Set("X-Owner", c.StreamID().String())
		}))

	// valid owner
	cmd := command.New("action", "order", validID, nil)
	reply, err := httpClient.CommandSink(context.Background(), cmd)
	assert.NoError(t, err)
	assert.Equal(t, reply.Command(), cmd.ID())

	// invalid owner
	cmd = command.New("action", "order", invalidID, nil)
	reply, err = httpClient.CommandSink(context.Background(), cmd)
	assert.Error(t, err)
	assert.Contains(t, "forbidden", err.Error())
	assert.Nil(t, reply)
}

func newMutation(ctrl *gomock.Controller) *stream.Mutator {
	publisher := mockstream.NewMockPublisher(ctrl)
	state := mockstream.NewMockState(ctrl)
	store := stream.NewStorage("order", func() *stream.Stream {
		return stream.Blank("one", state)
	})
	return stream.NewMutator(store, publisher)
}
