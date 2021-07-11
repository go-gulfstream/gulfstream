package main

import (
	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/google/uuid"
)

const (
	addToCartCommand = "addToCart"
	activateCommand  = "activate"
)

type addToCart struct {
	ShopID uuid.UUID
	Name   string
	Price  float64
}

func (p *addToCart) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (p *addToCart) UnmarshalBinary([]byte) error {
	return nil
}

func newAddToCartCommand(p *addToCart) *command.Command {
	return command.New(addToCartCommand, orderStream, streamID, p)
}

func newActivateOrderCommand() *command.Command {
	return command.New(activateCommand, orderStream, streamID, nil)
}
