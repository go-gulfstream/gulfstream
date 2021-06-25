package types

import (
	"encoding/json"

	"github.com/go-gulfstream/gulfstream/pkg/command"
)

func init() {
	err := command.AddKnownType(
		(*CreateNewEvent)(nil),
		(*AddParticipant)(nil),
	)
	if err != nil {
		panic(err)
	}
}

func (c *CreateNewEvent) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CreateNewEvent) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &c)
}

func (c *AddParticipant) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *AddParticipant) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &c)
}
