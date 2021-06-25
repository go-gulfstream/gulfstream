package types

import (
	"fmt"
	"time"
)

const (
	CreateNewEventCommand = "CreateNewEvent"
	AddParticipantCommand = "AddParticipant"
)

type CreateNewEvent struct {
	EventName      string
	DateTime       time.Time
	Lat            float64
	Lon            float64
	Radius         float64
	Address        string
	MaxParticipant int
}

func (c *CreateNewEvent) Validate() error {
	if len(c.EventName) < 3 {
		return fmt.Errorf("CreateNewEvent.Name too short")
	}
	return nil
}

type AddParticipant struct {
	Name string
	Age  uint16
	Sex  int8
}
