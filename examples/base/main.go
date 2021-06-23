package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"
)

func main() {
	storage := stream.NewStorage(blankStream)

	ctx := context.Background()
	idx := new(someLocalIndexInMem)

	eventBus := stream.NewEventBus()
	eventBus.Subscribe(ctx, orderStream,
		stream.EventHandlerFunc(addedToCartEvent,
			func(ctx context.Context, e *event.Event) error {
				idx.Increment()
				fmt.Printf("event: addedToCart%v\nversion: %d\nstream: %s\n",
					e.Payload().(addedToCart), e.Version(), e.StreamID())
				return nil
			}, nil),
	)

	eventBus.Subscribe(ctx, orderStream,
		stream.EventHandlerFunc(activatedEvent,
			func(ctx context.Context, e *event.Event) error {
				fmt.Printf("event: activated\nversion: %d\nstream: %s\n",
					e.Version(), e.StreamID())
				return nil
			}, nil),
	)

	go func() {
		checkError(eventBus.Listen(ctx))
	}()

	mutation := stream.NewMutation(
		orderStream,
		storage,
		eventBus,
	)

	mount(mutation, idx)

	addToCartCmd := newAddToCartCommand(&addToCart{
		ShopID: owner,
		Name:   "someProduct",
		Price:  2.33,
	})
	_, err := mutation.CommandSink(ctx, addToCartCmd)
	checkError(err)

	activateCmd := newActivateOrderCommand()
	_, err = mutation.CommandSink(ctx, activateCmd)
	checkError(err)

	currentStream, err := storage.Load(ctx, orderStream, streamID, owner)
	checkError(err)

	fmt.Printf("currentStream: %s\n", currentStream.State().(*order))
	<-time.After(2 * time.Second)

}

func blankStream() *stream.Stream {
	return stream.New(
		orderStream,
		uuid.UUID{},
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
