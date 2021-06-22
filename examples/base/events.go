package main

import "github.com/google/uuid"

type addedToCart struct {
	ShopID uuid.UUID
	Name   string
	Price  float64
	Total  int
}

type configured struct {
	ShopID uuid.UUID
	Total  int
}
