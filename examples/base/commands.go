package main

import (
	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/google/uuid"
)

type addToCart struct {
	ShopID uuid.UUID
	Name   string
	Price  float64
}

type setupShopID struct {
	ShopID uuid.UUID
}

func newAddToCartCommand(p *addToCart) *command.Command {
	return command.New("addToCart", "order", streamID, owner, p)
}
