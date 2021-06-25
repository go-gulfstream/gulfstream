package types

import "time"

const (
	PartyCreatedEvent     = "PartyCreated"
	ParticipantAddedEvent = "ParticipantAdded"
)

type PartyCreated struct {
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
