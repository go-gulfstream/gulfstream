package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"
)

type Mutation interface {
	CreateParty(ctx context.Context, currentState *state, command *types.CreateNewParty) (*types.PartyCreated, error)
	AddParticipant(ctx context.Context, currentState *state, command *types.AddParticipant) (*types.ParticipantAdded, error)
}

type mutation struct {
}

func newMutation() Mutation {
	return &mutation{}
}

func (m *mutation) CreateParty(ctx context.Context, currentState *state, command *types.CreateNewParty) (*types.PartyCreated, error) {
	radius := command.Radius / 2
	if radius > 5000 {
		return nil, fmt.Errorf("max radius exceeded")
	}
	if command.DateTime.IsZero() {
		command.DateTime = time.Now().Add(10 * time.Hour)
	}
	return &types.PartyCreated{
		EventName: command.EventName,
		DateTime:  command.DateTime,
		Lat:       command.Lat,
		Lon:       command.Lon,
		Radius:    radius,
		Address:   command.Address,
	}, nil
}

func (m *mutation) AddParticipant(ctx context.Context, currentState *state, command *types.AddParticipant) (*types.ParticipantAdded, error) {
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
