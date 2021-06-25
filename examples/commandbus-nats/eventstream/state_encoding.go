package main

import "encoding/json"

func (s *state) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *state) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
