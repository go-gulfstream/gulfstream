package logginglogrus

import (
	"bytes"
	"context"
	"testing"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestEventLogging(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()

	ev := event.New("test", "test", uuid.New(), 11, nil)
	target := mockstream.NewMockEventSinker(ctrl)
	target.EXPECT().EventSink(ctx, ev).Return(nil)

	out := bytes.NewBuffer(nil)
	logger := logrus.New()
	logger.Out = out

	sinker := stream.WithEventSinkerInterceptor(target, NewEventLogging(logger.WithField("component", "test")))
	err := sinker.EventSink(ctx, ev)
	assert.Nil(t, err)
	assert.NotEmpty(t, out.String())
}
