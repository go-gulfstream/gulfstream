package main

import (
	"context"
	"errors"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

type addToCartController struct {
	index *someLocalIndexInMem
}

func newAddToCartController(
	idx *someLocalIndexInMem,
) *addToCartController {
	return &addToCartController{
		index: idx,
	}
}

var errSome = errors.New("some error")

func (c *addToCartController) CommandSink(_ context.Context, ss *stream.Stream, cmd *command.Command) (*command.Reply, error) {
	someIndex := c.index.Load()
	if someIndex > 100 {
		return cmd.ReplyErr(errSome), nil
	}
	payload := cmd.Payload().(*addToCart)
	if payload.Price == 0 {
		return cmd.ReplyErr(errSome), nil
	}
	prevState := ss.State().(*order)
	ss.Mutate(addedToCartEvent, addedToCart{
		ShopID: payload.ShopID,
		Name:   payload.Name,
		Price:  payload.Price,
		Total:  prevState.Total + 1,
	})
	return cmd.ReplyOk(ss.Version()), nil
}
