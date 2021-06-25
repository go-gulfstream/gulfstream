package types

import "time"

const (
	EventCreatedEvent     = "EventCreated"
	ParticipantAddedEvent = "ParticipantAdded"
)

type EventCreated struct {
	EventName string
	DateTime  time.Time
	Lat       float64
	Lon       float64
	Radius    float64
	Address   string
}

type ParticipantAdded struct {
	EventName string
	DateTime  time.Time
	Lat       float64
	Lon       float64
	Radius    float64
	Address   string
	Name      string
	Age       uint16
	Sex       int8
}
