package main

import (
	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
)

type order struct {
	shopID   uuid.UUID
	items    []orderItem
	total    int
	activate bool
}

func (o *order) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (o *order) UnmarshalBinary(data []byte) error {
	return nil
}

func (o *order) Mutate(e *event.Event) {

}

type orderItem struct {
	name  string
	price float64
}
