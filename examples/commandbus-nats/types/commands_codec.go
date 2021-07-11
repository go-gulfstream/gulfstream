package types

import (
	"encoding/json"

	"github.com/go-gulfstream/gulfstream/pkg/codec"

	"github.com/go-gulfstream/gulfstream/pkg/command"
)

func init() {
	command.RegisterCodecs(map[string]codec.Codec{
		"CreateNewParty": &CreateNewParty{},
		"AddParticipant": &AddParticipant{},
	})
}

func (c *CreateNewParty) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *CreateNewParty) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &c)
}

func (c *AddParticipant) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *AddParticipant) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &c)
}
