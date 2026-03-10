package handoff

import (
	"encoding/json"
	"os"
)

// Payload is written by hook mode and read by review mode.
type Payload struct {
	Plan      string `json:"plan"`
	Diff      string `json:"diff"`
	SessionID string `json:"session_id"`
}

func WritePayload(path string, p Payload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func ReadPayload(path string) (Payload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Payload{}, err
	}
	var p Payload
	return p, json.Unmarshal(b, &p)
}
