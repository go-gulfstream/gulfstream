package loggingstd

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/stretchr/testify/assert"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"
	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"

	"github.com/golang/mock/gomock"
)

func TestEventLogging(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()

	ev := event.New("test", "test", uuid.New(), 11, nil)
	target := mockstream.NewMockEventSinker(ctrl)
	target.EXPECT().EventSink(ctx, ev).Return(nil)

	out := bytes.NewBuffer(nil)
	logger := log.New(out, "test", log.LstdFlags)

	sinker := stream.WithEventSinkerInterceptor(target, NewEventLogging(logger))
	err := sinker.EventSink(ctx, ev)
	assert.Nil(t, err)
	assert.Contains(t, out.String(), "[INFO] EventSink")
}
