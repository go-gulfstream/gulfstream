package main

import (
	"time"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type state struct {
	EventName    string
	DateTime     time.Time
	Place        place
	Participants []participant
}

func (s *state) Mutate(e *event.Event) {
	switch payload := e.Payload().(type) {
	case *types.EventCreated:
		s.EventName = payload.EventName
		s.DateTime = payload.DateTime
		s.Place = place{
			Lat:     payload.Lat,
			Lon:     payload.Lon,
			Radius:  payload.Radius,
			Address: payload.Address,
		}
		s.Participants = []participant{}
	case *types.ParticipantAdded:
		s.Participants = append(s.Participants,
			participant{
				Name: payload.Name,
				Age:  payload.Age,
				Sex:  payload.Sex,
			})
	}
}

type place struct {
	Lat     float64
	Lon     float64
	Radius  float64
	Address string
}

type participant struct {
	Name string
	Age  uint16
	Sex  int8
}
