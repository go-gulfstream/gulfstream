package main

import "github.com/google/uuid"

const (
	addedToCartEvent = "addedToCart"
	activatedEvent   = "activated"
)

type addedToCart struct {
	ShopID uuid.UUID
	Name   string
	Price  float64
	Total  int
}

func (p *addedToCart) UnmarshalBinary([]byte) error {
	return nil
}

func (p *addedToCart) MarshalBinary() ([]byte, error) {
	return nil, nil
}
