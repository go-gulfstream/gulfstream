package stream

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-gulfstream/gulfstream/pkg/util"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

const (
	DefaultPartitions = 32
	MinPartitions     = 1
	MaxPartitions     = 1024
)

type Publisher interface {
	Publish(ctx context.Context, event []*event.Event) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, streamName string, h EventHandler)
}

type EventHandler interface {
	Match(eventName string) bool
	Handle(context.Context, *event.Event) error
	Rollback(context.Context, *event.Event) error
}

type EventErrorHandler interface {
	HandleError(ctx context.Context, e *event.Event, err error)
}

type eventHandlerFunc struct {
	handler         func(context.Context, *event.Event) error
	rollbackHandler func(context.Context, *event.Event) error
	eventName       string
}

func (fn eventHandlerFunc) Match(eventName string) bool {
	return eventName == fn.eventName
}

func (fn eventHandlerFunc) Handle(ctx context.Context, e *event.Event) error {
	if fn.handler == nil {
		return nil
	}
	return fn.handler(ctx, e)
}

func (fn eventHandlerFunc) Rollback(ctx context.Context, e *event.Event) error {
	if fn.rollbackHandler == nil {
		return nil
	}
	return fn.rollbackHandler(ctx, e)
}

func EventHandlerFunc(eventName string, handler, rollbackHandler func(context.Context, *event.Event) error) EventHandler {
	return eventHandlerFunc{
		handler:         handler,
		rollbackHandler: rollbackHandler,
		eventName:       eventName,
	}
}

type EventBus struct {
	channels     map[string]*channel
	errorHandler EventErrorHandler
	wg           *sync.WaitGroup
	closed       bool
	partitions   int
}

type Option func(*EventBus)

func NewEventBus(o ...Option) *EventBus {
	eb := &EventBus{
		partitions: DefaultPartitions,
		channels:   make(map[string]*channel),
		wg:         new(sync.WaitGroup),
	}
	for _, f := range o {
		f(eb)
	}
	return eb
}

func WithEventBusPartitions(n int) Option {
	return func(eb *EventBus) {
		if n > MinPartitions && n <= MaxPartitions {
			eb.partitions = n
		}
	}
}

func WithEventBusErrorHandler(h EventErrorHandler) Option {
	return func(eb *EventBus) {
		eb.errorHandler = h
	}
}

func (b *EventBus) Publish(_ context.Context, events []*event.Event) error {
	for _, e := range events {
		channel, ok := b.channels[e.StreamName()]
		if !ok {
			return fmt.Errorf("channel for stream %s not found",
				e.StreamName())
		}
		channel.publish(e)
	}
	return nil
}

func (b *EventBus) Subscribe(_ context.Context, streamName string, h EventHandler) {
	channel, ok := b.channels[streamName]
	if !ok {
		channel = newChannel(b.partitions, streamName)
		b.wg.Add(1)
	}
	channel.addRecv(h)
	b.channels[streamName] = channel
}

func (b *EventBus) Listen(ctx context.Context) error {
	for _, channel := range b.channels {
		channel.setErrorHandler(b.errorHandler)
		channel.listen(ctx)
	}
	b.wg.Wait()
	return nil
}

func (b *EventBus) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	for _, channel := range b.channels {
		channel.close()
	}
	b.wg.Done()
	return nil
}

type channel struct {
	topic      string
	recv       []EventHandler
	partitions []chan *event.Event
	pn         int
	closeSig   chan struct{}
	seed       uint32
	once       sync.Once
	wg         sync.WaitGroup
	eh         EventErrorHandler
}

func newChannel(partitions int, topic string) *channel {
	ch := &channel{
		pn:         partitions,
		seed:       util.SeedUint32(),
		topic:      topic,
		recv:       []EventHandler{},
		closeSig:   make(chan struct{}),
		partitions: make([]chan *event.Event, partitions),
	}
	for i := 0; i < partitions; i++ {
		ch.partitions[i] = make(chan *event.Event, 1)
		ch.wg.Add(1)
	}
	return ch
}

func (ch *channel) addRecv(handler EventHandler) *channel {
	ch.recv = append(ch.recv, handler)
	return ch
}

func (ch *channel) setErrorHandler(h EventErrorHandler) *channel {
	if h == nil {
		return ch
	}
	ch.eh = h
	return ch
}

func (ch *channel) close() {
	ch.once.Do(func() {
		close(ch.closeSig)
	})
	ch.wg.Wait()
}

func (ch *channel) listen(ctx context.Context) {
	for i, partition := range ch.partitions {
		go ch.listenPartition(ctx, i, partition)
	}
}

func (ch *channel) listenPartition(ctx context.Context, pn int, channel chan *event.Event) {
	defer ch.wg.Done()
	var closed bool
	for {
		select {
		case <-ctx.Done():
			closed = true
			if len(ch.partitions[pn]) == 0 {
				return
			}
			return
		case <-ch.closeSig:
			closed = true
			if len(ch.partitions[pn]) == 0 {
				return
			}
		case e := <-channel:
			rollback := -1
			for i, recv := range ch.recv {
				if !recv.Match(e.Name()) {
					continue
				}
				if err := recv.Handle(ctx, e); err != nil {
					rollback = i
					if ch.eh != nil {
						ch.eh.HandleError(ctx, e, err)
					}
					break
				}
			}
			if rollback >= 0 {
				for i := rollback; i >= 0; i-- {
					recv := ch.recv[i]
					if !recv.Match(e.Name()) {
						continue
					}
					if err := recv.Rollback(ctx, e); err != nil {
						if ch.eh != nil {
							err = fmt.Errorf("eventHandler rollback: %w", err)
							ch.eh.HandleError(ctx, e, err)
						}
					}
				}
			}
			if len(ch.partitions[pn]) == 0 && closed {
				return
			}
		}
	}
}

func (ch *channel) publish(e *event.Event) {
	key := e.StreamID().String() + e.StreamName()
	idx := util.DJB2(ch.seed, key) % uint32(ch.pn)
	ch.partitions[idx] <- e
}
