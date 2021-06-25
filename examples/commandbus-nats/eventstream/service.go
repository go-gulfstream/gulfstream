package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"
)

type Service interface {
	CreateNewEvent(ctx context.Context, currentState *state, command *types.CreateNewEvent) (*types.EventCreated, error)
	AddParticipant(ctx context.Context, currentState *state, command *types.AddParticipant) (*types.ParticipantAdded, error)
}

type service struct {
}

func newService() Service {
	return &service{}
}

func (m *service) CreateNewEvent(ctx context.Context, currentState *state, command *types.CreateNewEvent) (*types.EventCreated, error) {
	radius := command.Radius / 2
	if radius > 5000 {
		return nil, fmt.Errorf("max radius exceeded")
	}
	if command.DateTime.IsZero() {
		command.DateTime = time.Now().Add(10 * time.Hour)
	}
	return &types.EventCreated{
		EventName: command.EventName,
		DateTime:  command.DateTime,
		Lat:       command.Lat,
		Lon:       command.Lon,
		Radius:    radius,
		Address:   command.Address,
	}, nil
}

func (m *service) AddParticipant(ctx context.Context, currentState *state, command *types.AddParticipant) (*types.ParticipantAdded, error) {
	return &types.ParticipantAdded{
		EventName: currentState.EventName,
		DateTime:  currentState.DateTime,
		Lat:       currentState.Place.Lat,
		Lon:       currentState.Place.Lon,
		Radius:    currentState.Place.Radius,
		Address:   currentState.Place.Address,
		Name:      command.Name,
		Age:       command.Age,
		Sex:       command.Sex,
	}, nil
}
