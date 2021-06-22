package stream

import (
	"context"
	"fmt"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/reply"

	"github.com/google/uuid"
)

type CommandController interface {
	CommandSink(context.Context, *Stream, *command.Command) (*reply.Reply, error)
}

type EventController interface {
	StreamIDFromEvent(*event.Event) uuid.UUID
	EventSink(context.Context, *Stream, *event.Event) error
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
	newStream          func() *Stream
	blacklistOfEvents  []string
}

func NewMutation(
	streamName string,
	storage Storage,
	publisher Publisher,
	newStream func() *Stream,
) *Mutation {
	return &Mutation{
		streamName:         streamName,
		storage:            storage,
		publisher:          publisher,
		newStream:          newStream,
		commandControllers: make(map[string]*commandController),
		eventControllers:   make(map[string]*eventController),
		blacklistOfEvents:  []string{},
	}
}

func (m *Mutation) CommandController(
	commandType string,
	ctrl CommandController,
	opts ...CommandControllerOption,
) {
	controller := &commandController{
		controller:  ctrl,
		commandType: commandType,
	}
	for _, opt := range opts {
		opt(controller)
	}
	m.commandControllers[commandType] = controller
}

func (m *Mutation) EventController(
	eventType string,
	ctrl EventController,
	opts ...EventControllerOption,
) {
	controller := &eventController{
		controller: ctrl,
		eventType:  eventType,
	}
	for _, opt := range opts {
		opt(controller)
	}
	m.eventControllers[eventType] = controller
}

func (m *Mutation) CommandSink(ctx context.Context, cmd *command.Command) (*reply.Reply, error) {
	if m.streamName != cmd.StreamName() {
		return nil, fmt.Errorf("stream mismatch: got %s, expected %s",
			cmd.StreamName(), m.streamName)
	}
	cc, found := m.commandControllers[cmd.Name()]
	if !found {
		return nil, fmt.Errorf("controller %s for command not found", cmd.Name())
	}
	var stream *Stream
	var err error
	if cc.assignNew {
		stream = m.newStream()
		// replace stream id from command if needed.
		if cmd.ID() != uuid.Nil {
			stream.id = cmd.ID()
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
	if len(stream.Changes()) == 0 {
		return r, nil
	}
	if err := m.storage.Persist(ctx, stream); err != nil {
		return nil, err
	}
	if err := m.publisher.Publish(ctx, stream.changes); err != nil {
		return nil, err
	}
	if err := m.storage.ConfirmVersion(ctx, stream.Version()); err != nil {
		return nil, err
	}
	stream.ClearChanges()
	return r, err
}

func (m *Mutation) SetBlacklistOfEvents(names ...string) {
	m.blacklistOfEvents = append(m.blacklistOfEvents, names...)
}

func (m *Mutation) EventSink(ctx context.Context, e *event.Event) error {
	ec, found := m.eventControllers[e.Name()]
	if !found {
		return fmt.Errorf("controller for event %s not found", e.Name())
	}
	for _, eventName := range m.blacklistOfEvents {
		if e.Name() == eventName {
			return nil
		}
	}
	streamID := ec.controller.StreamIDFromEvent(e)
	if streamID == uuid.Nil {
		return fmt.Errorf("unknown %s{%v}", m.streamName, streamID)
	}
	stream, err := m.storage.Load(ctx,
		m.streamName,
		streamID,
		e.Owner(),
	)
	if err != nil {
		return err
	}
	if err := ec.controller.EventSink(ctx, stream, e); err != nil {
		return err
	}
	if len(stream.changes) == 0 {
		return nil
	}
	if err := m.storage.Persist(ctx, stream); err != nil {
		return err
	}
	if err := m.publisher.Publish(ctx, stream.Changes()); err != nil {
		return err
	}
	if err := m.storage.ConfirmVersion(ctx, stream.Version()); err != nil {
		return err
	}
	stream.ClearChanges()
	return err
}

func AllowCreateStream() CommandControllerOption {
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
