package loggingstd

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func TestCommandLogging(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cmd := command.New("test", "test", uuid.New(), nil)
	reply := cmd.ReplyOk(11)

	target := mockstream.NewMockCommandSinker(ctrl)
	target.EXPECT().CommandSink(ctx, cmd).Return(reply, nil)

	out := bytes.NewBuffer(nil)
	logger := log.New(out, "test", log.LstdFlags)

	sinker := stream.WithCommandSinkerInterceptor(target, NewCommandSinkerLogging(logger))
	replyOk, err := sinker.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, reply, replyOk)
	assert.Contains(t, out.String(), "[INFO] CommandSink")
}
