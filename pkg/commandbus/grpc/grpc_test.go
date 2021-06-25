package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/command"

	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

func TestClientServer(t *testing.T) {
	// setup stream controllers
	mutation := newMutation()
	mutation.MountCommand("action",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.CreateMode())

	ctx := context.Background()

	// setup server side
	addr, lis := listen(t)
	defer lis.Close()
	grpcSrv := grpc.NewServer()
	defer grpcSrv.GracefulStop()
	server := NewServer(mutation)
	server.Register(grpcSrv)
	go func() {
		assert.Nil(t, grpcSrv.Serve(lis))
	}()
	time.Sleep(100 * time.Millisecond)

	// setup client side
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	assert.Nil(t, err)
	defer conn.Close()
	client := NewClient(conn)

	// request from client to server
	cmd := command.New("action", "order", uuid.New(), uuid.New(), nil)
	reply, err := client.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())
}

func TestServerInterceptors(t *testing.T) {
	validOwnerID := uuid.New()
	invalidOwnerID := uuid.New()

	// setup stream controllers
	mutation := newMutation()
	mutation.MountCommand("action",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.CreateMode())

	ctx := context.Background()
	schema := "bearer"

	// setup server side
	addr, lis := listen(t)
	defer lis.Close()
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			token, err := authFromMD(ctx, schema)
			if err != nil {
				return nil, err
			}
			if token != validOwnerID.String() {
				return nil, status.Errorf(codes.Unauthenticated, "Bad authorization string")
			}
			return handler(ctx, req)
		}),
	)
	defer grpcSrv.GracefulStop()
	server := NewServer(mutation)
	server.Register(grpcSrv)
	go func() {
		assert.Nil(t, grpcSrv.Serve(lis))
	}()
	time.Sleep(100 * time.Millisecond)

	// setup client side
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	assert.Nil(t, err)
	defer conn.Close()
	client := NewClient(conn,
		WithClientRequestFunc(
			func(md metadata.MD, c *command.Command) {
				token := schema + " " + c.Owner().String()
				md.Append(schema, token)
			}))

	// request from client to server
	// valid owner
	cmd := command.New("action", "order", uuid.New(), validOwnerID, nil)
	reply, err := client.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())

	// invalid owner
	cmd = command.New("action", "order", uuid.New(), invalidOwnerID, nil)
	reply, err = client.CommandSink(ctx, cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad authorization string")
	assert.Nil(t, reply)
}

func freePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func listen(t *testing.T) (string, net.Listener) {
	p := freePort(t)
	addr := fmt.Sprintf(":%d", p)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return addr, l
}

func newMutation() *stream.Mutation {
	storage := stream.NewStorage(func() *stream.Stream {
		return stream.Blank(&state{One: "one", Two: "two"})
	})
	return stream.NewMutation("order", storage, mockPublisher{})
}

// current state of the order stream.
// this is a fundamental part of the stream.
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

func authFromMD(ctx context.Context, expectedScheme string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	val := getKey(md, expectedScheme)
	if val == "" {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	splits := strings.SplitN(val, " ", 2)
	if len(splits) < 2 {
		return "", status.Errorf(codes.Unauthenticated, "Bad authorization string")
	}
	if !strings.EqualFold(splits[0], expectedScheme) {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	return splits[1], nil
}

func getKey(md metadata.MD, key string) string {
	k := strings.ToLower(key)
	vv, ok := md[k]
	if !ok {
		return ""
	}
	return vv[0]
}
