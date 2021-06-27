package kafka

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/hashicorp/go-multierror"

	"github.com/google/uuid"

	"github.com/Shopify/sarama"
	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

var _ stream.Subscriber = (*Subscriber)(nil)

type Subscriber struct {
	brokers       []string
	handlers      map[string][]stream.EventHandler
	eventCodec    event.Encoding
	consumerGroup sarama.ConsumerGroup
	conf          *sarama.Config
	contextFunc   []func(context.Context) context.Context
	exitFunc      []func()
	setupFunc     []func(sarama.ConsumerGroupSession) error
	cleanupFunc   []func(sarama.ConsumerGroupSession) error
	errorFunc     []func(*event.Event, error)
	beforeFunc    []func(*sarama.ConsumerMessage) (bool, error)
	group         string
	ready         chan error
}

func NewSubscriber(
	addr []string,
	conf *sarama.Config,
	opts ...SubscriberOption,
) *Subscriber {
	if conf == nil {
		conf = DefaultConfig()
	}
	s := &Subscriber{
		brokers:  addr,
		conf:     conf,
		handlers: make(map[string][]stream.EventHandler),
		ready:    make(chan error),
	}
	for _, f := range opts {
		f(s)
	}
	return s
}

type SubscriberOption func(*Subscriber)

func WithSubscriberGroupName(groupName string) SubscriberOption {
	return func(s *Subscriber) {
		s.group = groupName
	}
}

func WithSubscriberExitFunc(fn func()) SubscriberOption {
	return func(s *Subscriber) {
		s.exitFunc = append(s.exitFunc, fn)
	}
}

func WithSubscriberSetupFunc(fn func(sarama.ConsumerGroupSession) error) SubscriberOption {
	return func(s *Subscriber) {
		s.setupFunc = append(s.setupFunc, fn)
	}
}

func WithSubscriberCleanupFunc(fn func(sarama.ConsumerGroupSession) error) SubscriberOption {
	return func(s *Subscriber) {
		s.cleanupFunc = append(s.cleanupFunc, fn)
	}
}

func WithSubscriberContextFunc(fn func(context.Context) context.Context) SubscriberOption {
	return func(s *Subscriber) {
		s.contextFunc = append(s.contextFunc, fn)
	}
}

func WithSubscriberErrorHandler(fn func(*event.Event, error)) SubscriberOption {
	return func(s *Subscriber) {
		s.errorFunc = append(s.errorFunc, fn)
	}
}

func WithSubscriberBeforeFunc(fn func(*sarama.ConsumerMessage) (bool, error)) SubscriberOption {
	return func(s *Subscriber) {
		s.beforeFunc = append(s.beforeFunc, fn)
	}
}

func WithSubscriberCodec(codec event.Encoding) SubscriberOption {
	return func(s *Subscriber) {
		s.eventCodec = codec
	}
}

func (s *Subscriber) Subscribe(streamName string, h ...stream.EventHandler) {
	s.handlers[streamName] = append(s.handlers[streamName], h...)
}

func (s *Subscriber) Listen(ctx context.Context) (err error) {
	if len(s.group) == 0 {
		s.group = "gulfstream." + uuid.New().String()
	}
	s.consumerGroup, err = sarama.NewConsumerGroup(s.brokers, s.group, s.conf)
	if err != nil {
		return err
	}
	topics := make([]string, 0, len(s.handlers))
	for streamName := range s.handlers {
		topics = append(topics, streamName)
	}
	go func() {
		defer func() {
			for _, exitFunc := range s.exitFunc {
				exitFunc()
			}
		}()
		for {
			if err := s.consumerGroup.Consume(ctx, topics, s); err != nil {
				select {
				case s.ready <- err:
				default:
				}
				return
			}
			if ctx.Err() != nil {
				return
			}
			s.ready = make(chan error)
		}
	}()
	return <-s.ready
}

func (s *Subscriber) Close() error {
	return s.consumerGroup.Close()
}

func (s *Subscriber) Setup(sess sarama.ConsumerGroupSession) error {
	close(s.ready)
	for _, setupFunc := range s.setupFunc {
		if err := setupFunc(sess); err != nil {
			return err
		}
	}
	return nil
}

func (s *Subscriber) Cleanup(sess sarama.ConsumerGroupSession) error {
	for _, cleanupFunc := range s.cleanupFunc {
		if err := cleanupFunc(sess); err != nil {
			return err
		}
	}
	return nil
}

func (s *Subscriber) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := session.Context()
	for _, ctcFunc := range s.contextFunc {
		ctx = ctcFunc(ctx)
	}
	for message := range claim.Messages() {
		for _, beforeFunc := range s.beforeFunc {
			if mark, err := beforeFunc(message); err != nil {
				if mark {
					session.MarkMessage(message, "")
				}
				continue
			}
		}
		streamName, eventName := findMetaInfoFromHeaders(message.Headers)
		if len(streamName) == 0 || len(eventName) == 0 {
			session.MarkMessage(message, "")
			continue
		}
		handlers, found := s.handlers[message.Topic]
		if !found {
			session.MarkMessage(message, "")
			continue
		}
		var matched bool
		for _, handler := range handlers {
			if handler.Match(eventName) {
				matched = true
				break
			}
		}
		if !matched {
			session.MarkMessage(message, "")
			continue
		}
		e, err := s.decodeEvent(message.Value)
		if err != nil {
			s.errorHandle(nil, err)
			continue
		}
		rollback := -1
		for i, recv := range handlers {
			if !recv.Match(e.Name()) {
				continue
			}
			if er := recv.Handle(ctx, e); er != nil {
				rollback = i
				err = multierror.Append(err, er)
				s.errorHandle(e, err)
				break
			}
		}
		if rollback >= 0 {
			for i := rollback; i >= 0; i-- {
				recv := handlers[i]
				if !recv.Match(e.Name()) {
					continue
				}
				if er := recv.Rollback(ctx, e); er != nil {
					err = fmt.Errorf("receiver rollback: %w", er)
					s.errorHandle(e, err)
					err = multierror.Append(err, er)
				}
			}
		}
		if err == nil {
			session.MarkMessage(message, "")
		}
	}
	return nil
}

func (s *Subscriber) decodeEvent(data []byte) (*event.Event, error) {
	if s.eventCodec != nil {
		return s.eventCodec.Decode(data)
	} else {
		return event.Decode(data)
	}
}

func (s *Subscriber) errorHandle(msg *event.Event, err error) {
	if len(s.errorFunc) == 0 {
		return
	}
	for _, errFunc := range s.errorFunc {
		errFunc(msg, err)
	}
}

func findMetaInfoFromHeaders(headers []*sarama.RecordHeader) (s string, e string) {
	for _, header := range headers {
		key := byte2String(header.Key)
		switch key {
		case "_stream":
			s = byte2String(header.Value)
		case "_event":
			e = byte2String(header.Value)
		}
	}
	if len(s) == 0 {
		return s, e
	}
	return
}

func byte2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
