package commandbusgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/golang/mock/gomock"

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
	ctrl := gomock.NewController(t)
	// setup stream controllers
	mutation := newMutation(ctrl)
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.WithCommandControllerCreateIfNotExists())

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
	cmd := command.New("action", "order", uuid.New(), nil)
	reply, err := client.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())
}

func TestServerInterceptors(t *testing.T) {
	ctrl := gomock.NewController(t)

	validID := uuid.New()
	invalidID := uuid.New()

	// setup stream controllers
	mutation := newMutation(ctrl)
	mutation.AddCommandController("action",
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return c.ReplyOk(12), nil
		}), stream.WithCommandControllerCreateIfNotExists())

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
			if token != validID.String() {
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
				token := schema + " " + c.StreamID().String()
				md.Append(schema, token)
			}))

	// request from client to server
	// valid owner
	cmd := command.New("action", "order", validID, nil)
	reply, err := client.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, 12, reply.StreamVersion())
	assert.Equal(t, cmd.ID(), reply.Command())
	assert.Nil(t, reply.Err())

	// invalid owner
	cmd = command.New("action", "order", invalidID, nil)
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

func newMutation(ctrl *gomock.Controller) *stream.Mutator {
	publisher := mockstream.NewMockPublisher(ctrl)
	stor := stream.NewStorage("order", func() *stream.Stream {
		return stream.Blank("one", &state{One: "one", Two: "two"})
	})
	return stream.NewMutator(stor, publisher)
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
