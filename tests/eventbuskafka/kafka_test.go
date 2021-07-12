package eventbuskafka

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Shopify/sarama"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"

	eventbuskafka "github.com/go-gulfstream/gulfstream/pkg/eventbus/kafka"
	"github.com/go-gulfstream/gulfstream/tests"

	"github.com/stretchr/testify/suite"
)

func TestEventbus_Kafka(t *testing.T) {
	tests.SkipIfNotIntegration(t)

	suite.Run(t, &KafkaSuite{
		addr: []string{tests.KafkaAddr},
	})
}

type KafkaSuite struct {
	suite.Suite
	addr  []string
	topic string
}

func (s *KafkaSuite) SetupTest() {
	s.topic = uuid.New().String()
}

func (s *KafkaSuite) TearDownTest() {
}

func (s *KafkaSuite) TestPublishSubscriber() {
	waitSubscriber := make(chan struct{})
	var total uint32
	sub := eventbuskafka.NewSubscriber(s.addr, nil,
		eventbuskafka.WithSubscriberSetupFunc(
			func(session sarama.ConsumerGroupSession) error {
				close(waitSubscriber)
				return nil
			}),
	)
	defer sub.Close()
	proj := stream.NewProjection()
	proj.AddEventController("event1",
		func(ctx context.Context, e *event.Event) error {
			atomic.AddUint32(&total, 1)
			return nil
		})
	proj.AddEventController("event2",
		func(ctx context.Context, e *event.Event) error {
			atomic.AddUint32(&total, 1)
			return nil
		})
	sub.Subscribe(s.topic, proj)
	go func() {
		s.NoError(sub.Listen(context.Background()))
	}()
	select {
	case <-waitSubscriber:
	case <-time.After(5 * time.Second):
		s.T().Fatal("timeout")
	}
	publisher := eventbuskafka.NewPublisher(s.addr, nil)
	defer publisher.Close()
	s.NoError(publisher.Connect())
	s.NoError(publisher.Publish([]*event.Event{
		event.New("event1", s.topic, uuid.New(), 1, nil),
		event.New("event2", s.topic, uuid.New(), 2, nil),
	}))
	<-time.After(2 * time.Second)
	s.Equal(uint32(2), total)
}
