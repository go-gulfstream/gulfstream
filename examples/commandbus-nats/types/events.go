package types

import (
	"encoding/json"
	"time"
)

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

func (p *PartyCreated) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *PartyCreated) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
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

func (p *ParticipantAdded) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ParticipantAdded) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
