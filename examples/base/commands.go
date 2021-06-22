package main

import "github.com/google/uuid"

type addToCart struct {
	ShopID uuid.UUID
	Name   string
	Price  float64
}

type setupShopID struct {
	ShopID uuid.UUID
}
