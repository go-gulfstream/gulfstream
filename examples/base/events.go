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
