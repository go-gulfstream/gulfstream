package main

import (
	"context"
	"log"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	cmd "github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func CreatePartyController(m Service) stream.CommandController {
	return stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *cmd.Command) (*cmd.Reply, error) {
		log.Println("invoke.CreatePartyController")
		command := c.Payload().(*types.CreateNewParty)
		if err := command.Validate(); err != nil {
			return c.ReplyErr(err), nil
		}
		currentState := s.State().(*state)
		event, err := m.CreateParty(ctx, currentState, command)
		if err != nil {
			return c.ReplyErr(err), nil
		}
		s.Mutate(types.PartyCreatedEvent, event)
		return c.ReplyOk(s.Version()), nil
	})
}

func AddParticipantController(m Service) stream.CommandController {
	return stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *cmd.Command) (*cmd.Reply, error) {
		log.Println("invoke.AddParticipantController")
		command := c.Payload().(*types.AddParticipant)
		currentState := s.State().(*state)
		event, err := m.AddParticipant(ctx, currentState, command)
		if err != nil {
			return c.ReplyErr(err), nil
		}
		s.Mutate(types.ParticipantAddedEvent, event)
		return c.ReplyOk(s.Version()), nil
	})
}
