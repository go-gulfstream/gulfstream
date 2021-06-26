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
	PickOwner(*event.Event) []Owner
	EventSink(context.Context, *Stream, *event.Event) error
}

type ControllerFunc func(context.Context, *Stream, *command.Command) (*command.Reply, error)

func (fn ControllerFunc) CommandSink(ctx context.Context, s *Stream, c *command.Command) (*command.Reply, error) {
	return fn(ctx, s, c)
}

type Owner struct {
	StreamID uuid.UUID
	Owner    uuid.UUID
}

type (
	CommandControllerOption func(*commandController)
	EventControllerOption   func(*eventController)
)

type Mutator struct {
	streamName         string
	storage            Storage
	publisher          Publisher
	commandControllers map[string]*commandController
	eventControllers   map[string]*eventController
	blacklistOfEvents  []string
}

func NewMutator(
	streamName string,
	storage Storage,
	publisher Publisher,
) *Mutator {
	return &Mutator{
		streamName:         streamName,
		storage:            storage,
		publisher:          publisher,
		commandControllers: make(map[string]*commandController),
		eventControllers:   make(map[string]*eventController),
		blacklistOfEvents:  []string{},
	}
}

func (m *Mutator) AddCommandController(
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

func (m *Mutator) AddEventController(
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

func (m *Mutator) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
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
	} else {
		stream, err = m.storage.Load(ctx, cmd.StreamName(), cmd.StreamID())
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

func (m *Mutator) SetBlacklistOfEvents(names ...string) {
	m.blacklistOfEvents = append(m.blacklistOfEvents, names...)
}

func (m *Mutator) isMySelfEvent(e *event.Event) bool {
	for _, eventName := range m.blacklistOfEvents {
		if e.Name() == eventName {
			return true
		}
	}
	return false
}

func (m *Mutator) EventSink(ctx context.Context, e *event.Event) error {
	ec, found := m.eventControllers[e.Name()]
	if !found || m.isMySelfEvent(e) {
		return nil
	}
	owners := ec.controller.PickOwner(e)
	if len(owners) == 0 {
		return nil
	}
	for _, owner := range owners {
		stream, err := m.storage.Load(ctx, m.streamName, owner.StreamID)
		if err != nil {
			return err
		}
		if err := m.eventSink(ctx, ec, stream, e); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mutator) eventSink(ctx context.Context, ec *eventController, s *Stream, e *event.Event) error {
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
