package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/storage"

	"github.com/go-gulfstream/gulfstream/pkg/command"

	"github.com/go-gulfstream/gulfstream/pkg/eventbus"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"
)

func main() {
	orderStreamStorage := storage.New(orderStream, orderStreamFactory)

	ctx := context.Background()
	idx := new(someLocalIndexInMem)

	channel := eventbus.NewChannel()
	channel.Subscribe(orderStream,
		eventbus.HandlerFunc(addedToCartEvent,
			func(ctx context.Context, e *event.Event) error {
				idx.Increment()
				fmt.Printf("event: addedToCart%v\nversion: %d\nstream: %s\n",
					e.Payload().(addedToCart), e.Version(), e.StreamID())
				return nil
			}, nil),
	)

	channel.Subscribe(orderStream,
		eventbus.HandlerFunc(activatedEvent,
			func(ctx context.Context, e *event.Event) error {
				fmt.Printf("event: activated\nversion: %d\nstream: %s\n",
					e.Version(), e.StreamID())
				return nil
			}, nil),
	)

	go func() {
		checkError(channel.Listen(ctx))
	}()

	mutator := stream.NewMutator(orderStreamStorage, channel)

	mutator.AddCommandController(addToCartCommand,
		newAddToCartController(idx), stream.WithCommandControllerCreateIfNotExists())

	mutator.AddCommandController(activateCommand,
		stream.ControllerFunc(func(ctx context.Context, s *stream.Stream, command *command.Command) (*command.Reply, error) {
			s.Mutate(activatedEvent, nil)
			return nil, nil
		}))

	addToCartCmd := newAddToCartCommand(&addToCart{
		ShopID: owner,
		Name:   "someProduct",
		Price:  2.33,
	})
	_, err := mutator.CommandSink(ctx, addToCartCmd)
	checkError(err)

	activateCmd := newActivateOrderCommand()
	_, err = mutator.CommandSink(ctx, activateCmd)
	checkError(err)

	currentStream, err := orderStreamStorage.Load(ctx, streamID)
	checkError(err)

	fmt.Printf("currentStream: %s\n", currentStream.State().(*order))
	<-time.After(2 * time.Second)

}

func orderStreamFactory() *stream.Stream {
	return stream.New(
		orderStream,
		uuid.UUID{},
		&order{},
	)
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
