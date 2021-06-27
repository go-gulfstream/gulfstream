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
	PickStream(*event.Event) Picker
	EventSink(context.Context, *Stream, *event.Event) error
}

type Picker struct {
	StreamID  uuid.UUID
	StreamIDs []uuid.UUID
}

func (p Picker) isEmpty() bool {
	return len(p.StreamIDs) == 0 && p.StreamID == uuid.Nil
}

func (p Picker) hasOne() bool {
	return p.StreamID != uuid.Nil
}

func (p Picker) hasMany() bool {
	return p.StreamID == uuid.Nil && len(p.StreamIDs) > 0
}

func (p Picker) each(fn func(uuid.UUID) error) error {
	for _, streamID := range p.StreamIDs {
		if err := fn(streamID); err != nil {
			return err
		}
	}
	return nil
}

type ControllerFunc func(context.Context, *Stream, *command.Command) (*command.Reply, error)

func (fn ControllerFunc) CommandSink(ctx context.Context, s *Stream, c *command.Command) (*command.Reply, error) {
	return fn(ctx, s, c)
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
	strict             bool
	blacklistOfEvents  []string
}

func NewMutator(
	streamName string,
	storage Storage,
	publisher Publisher,
	opts ...MutatorOption,
) *Mutator {
	m := &Mutator{
		streamName:         streamName,
		storage:            storage,
		publisher:          publisher,
		commandControllers: make(map[string]*commandController),
		eventControllers:   make(map[string]*eventController),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type MutatorOption func(*Mutator)

func WithMutatorStrictMode() MutatorOption {
	return func(m *Mutator) {
		m.strict = true
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
	if cmd == nil {
		return nil, fmt.Errorf("command struct is nil pointer")
	}
	if m.streamName != cmd.StreamName() {
		return nil, fmt.Errorf("stream mismatch: got %s, expected %s",
			cmd.StreamName(), m.streamName)
	}
	cc, found := m.commandControllers[cmd.Name()]
	if !found {
		return nil, fmt.Errorf("mutator: controller for command %s.%s not found",
			cmd.StreamName(), cmd.Name())
	}
	var stream *Stream
	var err error
	if cc.createStream {
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
	if err := m.publisher.Publish(stream.changes); err != nil {
		return nil, multierror.Append(err, m.storage.MarkUnpublished(ctx, stream))
	}
	stream.ClearChanges()
	return r, err
}

func (m *Mutator) SetBlacklistOfEvents(eventNames ...string) {
	m.blacklistOfEvents = append(m.blacklistOfEvents, eventNames...)
}

func (m *Mutator) isMySelfEvent(e *event.Event) bool {
	if len(m.blacklistOfEvents) == 0 {
		return false
	}
	for _, eventName := range m.blacklistOfEvents {
		if eventName == e.Name() {
			return true
		}
	}
	return false
}

func (m *Mutator) EventSink(ctx context.Context, e *event.Event) (err error) {
	if e == nil {
		if m.strict {
			err = fmt.Errorf("event struct is nil pointer")
		}
		return
	}
	if m.isMySelfEvent(e) {
		if m.strict {
			err = fmt.Errorf("mutator: event cycles %s.%s",
				e.StreamName(), e.Name())
		}
		return
	}
	ec, found := m.eventControllers[e.Name()]
	if !found {
		if m.strict {
			err = fmt.Errorf("mutator: controller for event %s.%s not found",
				e.StreamName(), e.Name())
		}
		return
	}

	streamPicker := ec.controller.PickStream(e)
	if streamPicker.isEmpty() {
		if m.strict {
			err = fmt.Errorf("mutator: pick stream error %s.%s",
				m.streamName, e.Name())
		}
		return
	}
	if streamPicker.hasOne() {
		stream, err := m.loadStreamFromEvent(ctx, streamPicker.StreamID, ec.createStream)
		if err != nil {
			return err
		}
		if err := m.eventSink(ctx, ec, stream, e); err != nil {
			return err
		}
	}
	if streamPicker.hasMany() {
		return streamPicker.each(func(streamID uuid.UUID) error {
			stream, err := m.loadStreamFromEvent(ctx, streamID, ec.createStream)
			if err != nil {
				return err
			}
			if err := m.eventSink(ctx, ec, stream, e); err != nil {
				return err
			}
			return nil
		})
	}
	return
}

func (m *Mutator) loadStreamFromEvent(ctx context.Context, streamID uuid.UUID, createStream bool) (*Stream, error) {
	if createStream {
		return m.storage.BlankStream(), nil
	} else {
		return m.storage.Load(ctx, m.streamName, streamID)
	}
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
	if err := m.publisher.Publish(s.Changes()); err != nil {
		return multierror.Append(err, m.storage.MarkUnpublished(ctx, s))
	}
	s.ClearChanges()
	return nil
}

func WithCommandControllerCreateIfNotExists() CommandControllerOption {
	return func(ctrl *commandController) {
		ctrl.createStream = true
	}
}

func WithEventControllerCreateIfNotExists() EventControllerOption {
	return func(ctrl *eventController) {
		ctrl.createStream = true
	}
}

func EventControllerFunc(
	pickStream func(*event.Event) Picker,
	sink func(context.Context, *Stream, *event.Event) error,
) EventController {
	return eventControllerFunc{
		pick: pickStream,
		sink: sink,
	}
}

func (fn eventControllerFunc) PickStream(e *event.Event) Picker {
	return fn.pick(e)
}

func (fn eventControllerFunc) EventSink(ctx context.Context, s *Stream, e *event.Event) error {
	return fn.sink(ctx, s, e)
}

type eventControllerFunc struct {
	pick func(*event.Event) Picker
	sink func(context.Context, *Stream, *event.Event) error
}

type commandController struct {
	controller   CommandController
	commandType  string
	createStream bool
}

type eventController struct {
	controller   EventController
	eventType    string
	createStream bool
}
