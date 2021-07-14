package logginglogrus

import (
	"bytes"
	"context"
	"testing"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCommandLogging(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	cmd := command.New("test", "test", uuid.New(), nil)
	reply := cmd.ReplyOk(11)

	target := mockstream.NewMockCommandSinker(ctrl)
	target.EXPECT().CommandSink(ctx, cmd).Return(reply, nil)

	out := bytes.NewBuffer(nil)
	logger := logrus.New()
	logger.Out = out
	sinker := stream.WithCommandSinkerInterceptor(target, NewCommandSinkerLogging(logger.WithField("component", "test")))
	replyOk, err := sinker.CommandSink(ctx, cmd)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, reply, replyOk)
	assert.NotEmpty(t, out.String())
}
