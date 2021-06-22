package main

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"
)

const orderStreamName = "order"

func main() {
	storage := stream.NewStorage()
	eventBus := stream.NewEventBus()

	mutation := stream.NewMutation(
		orderStreamName,
		storage,
		eventBus,
		blankStream,
	)
	mutation.FromCommand("addToCart",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, command *command.Command) (*command.Reply, error) {
			payload := command.Payload().(*addToCart)
			if payload.Price == 0 {
				return command.ReplyErr(), nil
			}
			prevState := s.State().(*order)
			if prevState.total > 30 {
				return command.ReplyErr(), nil
			}
			s.Mutate("addedToCart", addedToCart{
				ShopID: payload.ShopID,
				Name:   payload.Name,
				Price:  payload.Price,
				Total:  prevState.total + 1,
			})
			return nil, nil
		}), stream.Create())

	mutation.FromCommand("setupShopID",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return nil, nil
		}))

	mutation.FromCommand("activateOrder",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *command.Command) (*command.Reply, error) {
			return nil, nil
		}))
}

func blankStream() *stream.Stream {
	return stream.New(
		orderStreamName,
		uuid.UUID{},
		uuid.UUID{},
		&order{},
	)
}
