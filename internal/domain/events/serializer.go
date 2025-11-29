package events

import (
	"encoding/json"
	"fmt"
)

type EventSerializer interface {
	Serialize(event interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

type JSONEventSerializer struct{}

func NewJSONEventSerializer() *JSONEventSerializer {
	return &JSONEventSerializer{}
}

func (s *JSONEventSerializer) Serialize(event interface{}) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}
	return data, nil
}

func (s *JSONEventSerializer) Deserialize(data []byte, event interface{}) error {
	if err := json.Unmarshal(data, event); err != nil {
		return fmt.Errorf("failed to deserialize event: %w", err)
	}
	return nil
}
