package main

import (
	"context"
	"log"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	cmd "github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func CreateNewEventController(m Service) stream.CommandController {
	return stream.CommandCtrlFunc(func(ctx context.Context, s *stream.Stream, c *cmd.Command) (*cmd.Reply, error) {
		log.Println("invoke.CreateNewEventController")
		command := c.Payload().(*types.CreateNewEvent)
		if err := command.Validate(); err != nil {
			return c.ReplyErr(err), nil
		}
		currentState := s.State().(*state)
		event, err := m.CreateNewEvent(ctx, currentState, command)
		if err != nil {
			return c.ReplyErr(err), nil
		}
		s.Mutate(types.EventCreatedEvent, event)
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
