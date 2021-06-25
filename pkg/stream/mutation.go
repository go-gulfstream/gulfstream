package stream

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/command"

	"github.com/google/uuid"
)

type CommandController interface {
	CommandSink(context.Context, *Stream, *command.Command) (*command.Reply, error)
}

type EventController interface {
	StreamIDsFromEvent(*event.Event) []uuid.UUID
	EventSink(context.Context, *Stream, *event.Event) error
}

type CommandCtrlFunc func(context.Context, *Stream, *command.Command) (*command.Reply, error)

func (fn CommandCtrlFunc) CommandSink(ctx context.Context, s *Stream, c *command.Command) (*command.Reply, error) {
	return fn(ctx, s, c)
}

type (
	CommandControllerOption func(*commandController)
	EventControllerOption   func(*eventController)
)

type Mutation struct {
	streamName         string
	storage            Storage
	publisher          Publisher
	commandControllers map[string]*commandController
	eventControllers   map[string]*eventController
	blacklistOfEvents  []string
}

func NewMutation(
	streamName string,
	storage Storage,
	publisher Publisher,
) *Mutation {
	return &Mutation{
		streamName:         streamName,
		storage:            storage,
		publisher:          publisher,
		commandControllers: make(map[string]*commandController),
		eventControllers:   make(map[string]*eventController),
		blacklistOfEvents:  []string{},
	}
}

func (m *Mutation) MountCommand(
	commandName string,
	ctrl CommandController,
	opts ...CommandControllerOption,
) {
	controller := &commandController{
		controller:  ctrl,
		commandType: commandName,
	}
	for _, opt := range opts {
		opt(controller)
	}
	m.commandControllers[commandName] = controller
}

func (m *Mutation) MountEvent(
	eventName string,
	ctrl EventController,
	opts ...EventControllerOption,
) {
	controller := &eventController{
		controller: ctrl,
		eventType:  eventName,
	}
	for _, opt := range opts {
		opt(controller)
	}
	m.eventControllers[eventName] = controller
}

func (m *Mutation) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	if m.streamName != cmd.StreamName() {
		return nil, fmt.Errorf("stream mismatch: got %s, expected %s",
			cmd.StreamName(), m.streamName)
	}
	cc, found := m.commandControllers[cmd.Name()]
	if !found {
		return nil, fmt.Errorf("mutation controller for command %s not found", cmd.Name())
	}
	var stream *Stream
	var err error
	if cc.assignNew {
		stream = m.storage.BlankStream()
		// replace stream id from command if needed.
		if cmd.StreamID() != uuid.Nil {
			stream.id = cmd.StreamID()
		}
		// replace owner from command if needed.
		if cmd.Owner() != uuid.Nil {
			stream.owner = cmd.Owner()
		}
	} else {
		stream, err = m.storage.Load(ctx,
			cmd.StreamName(),
			cmd.StreamID(),
			cmd.Owner(),
		)
		if err != nil {
			return nil, err
		}
	}
	r, err := cc.controller.CommandSink(ctx, stream, cmd)
	if err != nil {
		return nil, err
	}
	if r == nil {
		r = cmd.ReplyOk(stream.Version())
	}
	if len(stream.Changes()) == 0 {
		return r, nil
	}
	if err := m.storage.Persist(ctx, stream); err != nil {
		return nil, err
	}
	if err := m.publisher.Publish(ctx, stream.changes); err != nil {
		return nil, multierror.Append(err, m.storage.MarkUnpublished(ctx, stream))
	}
	stream.ClearChanges()
	return r, err
}

func (m *Mutation) SetBlacklistOfEvents(names ...string) {
	m.blacklistOfEvents = append(m.blacklistOfEvents, names...)
}

func (m *Mutation) isMySelfEvent(e *event.Event) bool {
	for _, eventName := range m.blacklistOfEvents {
		if e.Name() == eventName {
			return true
		}
	}
	return false
}

func (m *Mutation) EventSink(ctx context.Context, e *event.Event) error {
	ec, found := m.eventControllers[e.Name()]
	if !found {
		return fmt.Errorf("controller for event %s not found", e.Name())
	}

	if m.isMySelfEvent(e) {
		return nil
	}

	streamIDs := ec.controller.StreamIDsFromEvent(e)
	if len(streamIDs) == 0 {
		return nil
	}

	return m.storage.Walk(ctx, m.streamName, streamIDs, e.Owner(),
		func(s *Stream) error {
			if err := ec.controller.EventSink(ctx, s, e); err != nil {
				return err
			}
			if len(s.Changes()) == 0 {
				return nil
			}
			if err := m.storage.Persist(ctx, s); err != nil {
				return err
			}
			if err := m.publisher.Publish(ctx, s.Changes()); err != nil {
				return multierror.Append(err, m.storage.MarkUnpublished(ctx, s))
			}
			s.ClearChanges()
			return nil
		})
}

func CreateMode() CommandControllerOption {
	return func(ctrl *commandController) {
		ctrl.assignNew = true
	}
}

type commandController struct {
	controller  CommandController
	commandType string
	assignNew   bool
}

type eventController struct {
	controller EventController
	eventType  string
}
