package main

import (
	"encoding/json"
	"fmt"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
)

const orderStream = "order"

var owner = uuid.New()
var streamID = uuid.New()

type order struct {
	ShopID   uuid.UUID
	Items    []OrderItem
	Total    int
	Activate bool
}

func (o *order) String() string {
	return fmt.Sprintf("order{ShopID:%s, activate:%v, total:%d, items:%v}",
		o.ShopID, o.Activate, o.Total, o.Items,
	)
}

func (o *order) MarshalBinary() ([]byte, error) {
	return json.Marshal(o)
}

func (o *order) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, o)
}

func (o *order) Mutate(e *event.Event) {
	switch e.Name() {
	case activatedEvent:
		o.Activate = true
	case addedToCartEvent:
		payload := e.Payload().(addedToCart)
		o.ShopID = payload.ShopID
		o.Total = payload.Total
		o.Items = append(o.Items, OrderItem{
			Name:  payload.Name,
			Price: payload.Price,
		})
	}
}

type OrderItem struct {
	Name  string
	Price float64
}
