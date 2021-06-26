package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/go-gulfstream/gulfstream/examples/commandbus-nats/types"

	"github.com/google/uuid"

	"github.com/go-gulfstream/gulfstream/pkg/command"

	commandbusnats "github.com/go-gulfstream/gulfstream/pkg/commandbus/nats"

	"github.com/nats-io/nats.go"
)

func main() {
	opts := []nats.Option{nats.Name("name")}
	conn, err := nats.Connect("nats:4222", opts...)
	checkError(err)
	defer conn.Close()

	ctx := context.Background()

	commandbus := commandbusnats.NewClient(types.PartyStream, conn)

	go func() {
		for {
			// create new party-event
			createPartyCmd := createParty(types.CreateNewParty{
				EventName: "golang fest",
				DateTime:  time.Now().Add(time.Hour),
				Lat:       1,
				Lon:       1,
				Radius:    3000,
				Address:   "google",
			})
			log.Printf("[CLIENT:COMMANDSINK] id=%s, name=%s",
				createPartyCmd.ID(),
				createPartyCmd.Name(),
			)
			reply, err := commandbus.CommandSink(ctx, createPartyCmd)
			checkError(err)
			if reply == nil {
				os.Exit(1)
			}

			log.Printf("[CLIENT:REPLY]=> %s stream=%s, sid=%s, commandName=%s, v=%d\n",
				status(reply.Command() == createPartyCmd.ID()),
				createPartyCmd.StreamName(),
				createPartyCmd.StreamID(),
				createPartyCmd.Name(),
				reply.StreamVersion())

			// add participant
			addParticipantCommand := addParticipant(
				createPartyCmd.StreamID(),
				types.AddParticipant{
					Name: "user",
					Age:  16,
					Sex:  -1,
				})
			reply, err = commandbus.CommandSink(ctx, addParticipantCommand)
			checkError(err)
			if reply == nil {
				os.Exit(1)
			}

			log.Printf("[CLIENT:REPLY]=> %s stream=%s, sid=%s, commandName=%s, v=%d\n",
				status(reply.Command() == addParticipantCommand.ID()),
				addParticipantCommand.StreamName(),
				addParticipantCommand.StreamID(),
				addParticipantCommand.Name(),
				reply.StreamVersion())

			time.Sleep(2 * time.Second)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func status(f bool) string {
	if f {
		return "OK"
	}
	return "FAIL"
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createParty(p types.CreateNewParty) *command.Command {
	return command.New(types.CreateNewPartyCommand, types.PartyStream, uuid.New(), &p)
}

func addParticipant(streamID uuid.UUID, p types.AddParticipant) *command.Command {
	return command.New(types.AddParticipantCommand, types.PartyStream, streamID, &p)
}
