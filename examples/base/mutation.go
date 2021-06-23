package main

import (
	"context"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func mount(mutation *stream.Mutation) {
	mutation.FromCommand("addToCart",
		stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, command *command.Command) (*command.Reply, error) {
			payload := command.Payload().(*addToCart)
			if payload.Price == 0 {
				return command.ReplyErr(), nil
			}
			prevState := s.State().(*order)
			if prevState.Total > 30 {
				return command.ReplyErr(), nil
			}
			s.Mutate("addedToCart", addedToCart{
				ShopID: payload.ShopID,
				Name:   payload.Name,
				Price:  payload.Price,
				Total:  prevState.Total + 1,
			})
			return nil, nil
		}), stream.OnlyCreateMode())
}
