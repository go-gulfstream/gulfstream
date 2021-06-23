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

const orderStreamName = "order"

var owner = uuid.New()
var streamID = uuid.New()

func main() {
	storage := stream.NewStorage()

	ctx := context.Background()

	eventBus := stream.NewEventBus()
	eventBus.Subscribe(context.TODO(), "order",
		stream.EventHandlerFunc("addedToCart",
			func(ctx context.Context, e *event.Event) error {
				fmt.Printf("event: addedToCart%v\nversion: %d\nstream: %s\n",
					e.Payload().(addedToCart), e.Version(), e.StreamID())
				return nil
			}, nil),
	)

	go func() {
		if err := eventBus.Listen(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	mutation := stream.NewMutation(
		orderStreamName,
		storage,
		eventBus,
		blankStream,
	)

	mount(mutation)

	addToCartCommand := newAddToCartCommand(&addToCart{
		ShopID: owner,
		Name:   "someProduct",
		Price:  2.33,
	})

	_, err := mutation.CommandSink(ctx, addToCartCommand)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	<-time.After(3 * time.Second)

}

func blankStream() *stream.Stream {
	return stream.New(
		orderStreamName,
		uuid.UUID{},
		uuid.UUID{},
		&order{},
	)
}
