package stream

import (
	"context"
	"fmt"

	"github.com/go-gulfstream/gulfstream/pkg/event"
)

type Projection struct {
	handlers   map[string]projectionHandler
	strictMode bool
}

type EventHandlerFunc func(ctx context.Context, e *event.Event) error

func NewProjection() *Projection {
	return &Projection{
		handlers:   make(map[string]projectionHandler),
		strictMode: true,
	}
}

func (p *Projection) SkipUnhandledEvent() {
	p.strictMode = false
}

func (p *Projection) AddEventController(eventName string, fn EventHandlerFunc) {
	p.handlers[eventName] = projectionHandler{handler: fn}
}

func (p *Projection) AddEventControllerWithRollback(eventName string, fn, rollback EventHandlerFunc) {
	p.handlers[eventName] = projectionHandler{handler: fn, rollback: rollback}
}

func (p *Projection) Match(eventName string) bool {
	_, found := p.handlers[eventName]
	return found
}

func (p *Projection) Handle(ctx context.Context, e *event.Event) error {
	proj, found := p.handlers[e.Name()]
	if !found {
		if p.strictMode {
			return fmt.Errorf("stream: %s projection not found", e.Name())
		}
		return nil
	}
	return proj.handler(ctx, e)
}

func (p *Projection) Rollback(ctx context.Context, e *event.Event) error {
	proj, found := p.handlers[e.Name()]
	if !found {
		if p.strictMode {
			return fmt.Errorf("stream: %s rollback projection not found", e.Name())
		}
		return nil
	}
	return proj.rollback(ctx, e)
}

type projectionHandler struct {
	handler  EventHandlerFunc
	rollback EventHandlerFunc
}
