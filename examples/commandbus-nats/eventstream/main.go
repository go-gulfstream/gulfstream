package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	"github.com/nats-io/nats.go"

	commandbusnats "github.com/go-gulfstream/gulfstream/pkg/commandbus/nats"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func main() {
	storage := stream.NewStorage(blankPartyStream)
	mutation := stream.NewMutation(types.PartyStream, storage, customPublisher{})

	svc := newService()

	mutation.MountCommand(
		types.CreateNewEventCommand,
		CreateNewEventController(svc),
		stream.CreateMode())

	mutation.MountCommand(
		types.AddParticipantCommand,
		AddParticipantController(svc))

	commandbus := commandbusnats.NewServer(types.PartyStream, mutation,
		commandbusnats.WithServerErrorHandler(func(msg *nats.Msg, err error) {
			log.Printf("[ERR] msg:%s, %v\n", msg.Subject, err)
		}))

	opts := []nats.Option{nats.Name("name")}
	conn, err := nats.Connect("nats:4222", opts...)
	checkError(err)

	checkError(commandbus.Listen(conn))

	fmt.Println("running....")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func blankPartyStream() *stream.Stream {
	return stream.Blank(types.PartyStream, &state{})
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type customPublisher struct{}

func (customPublisher) Publish(_ context.Context, events []*event.Event) error {
	for _, e := range events {
		log.Printf("[SERVER:PUBLISHEVENT]=> publish{stream=%s, sid=%s, owner=%s eventName=%s, payload=%v}\n",
			e.StreamName(), e.StreamID(), e.Owner(), e.Name(), e.Payload())
	}
	return nil
}
