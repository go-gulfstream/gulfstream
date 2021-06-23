package main

import (
	"encoding/json"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
)

type order struct {
	ShopID   uuid.UUID
	Items    []orderItem
	Total    int
	Activate bool
}

func (o *order) MarshalBinary() ([]byte, error) {
	return json.Marshal(o)
}

func (o *order) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, o)
}

func (o *order) Mutate(e *event.Event) {
	switch payload := e.Payload().(type) {
	case addedToCart:
		o.ShopID = payload.ShopID
		o.Total = payload.Total
		o.Items = append(o.Items, orderItem{
			name:  payload.Name,
			price: payload.Price,
		})
	}
}

type orderItem struct {
	name  string
	price float64
}
