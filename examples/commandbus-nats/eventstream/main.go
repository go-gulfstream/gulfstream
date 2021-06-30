package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/go-gulfstream/gulfstream/pkg/storage"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	"github.com/nats-io/nats.go"

	commandbusnats "github.com/go-gulfstream/gulfstream/pkg/commandbus/nats"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

func main() {
	partyStreamStorage := storage.New(types.PartyStreamName, partyStreamFactory)
	mutator := stream.NewMutator(partyStreamStorage, customPublisher{})

	mutation := newMutation()

	mutator.AddCommandController(
		types.CreateNewPartyCommand,
		CreatePartyController(mutation),
		stream.WithCommandControllerCreateIfNotExists())

	mutator.AddCommandController(
		types.AddParticipantCommand,
		AddParticipantController(mutation))

	commandbus := commandbusnats.NewServer(types.PartyStreamName, mutator,
		commandbusnats.WithServerErrorHandler(func(msg *nats.Msg, err error) {
			log.Printf("[ERR] msg:%s, %v\n", msg.Subject, err)
		}))

	opts := []nats.Option{nats.Name("name")}
	conn, err := nats.Connect("nats:4222", opts...)
	checkError(err)
	defer conn.Close()

	checkError(commandbus.Listen(conn))

	fmt.Println("running....")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func partyStreamFactory() *stream.Stream {
	return stream.Blank(types.PartyStreamName, &state{})
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type customPublisher struct{}

func (customPublisher) Publish(events []*event.Event) error {
	for _, e := range events {
		log.Printf("[SERVER:PUBLISHEVENT]=> publish{stream=%s, sid=%s,  eventName=%s, payload=%v}\n",
			e.StreamName(), e.StreamID(), e.Name(), e.Payload())
	}
	return nil
}
