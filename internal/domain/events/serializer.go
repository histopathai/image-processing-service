package events

import (
	"encoding/json"

	"github.com/histopathai/image-processing-service/pkg/errors"
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
		return nil, errors.NewInternalError("failed to serialize event").WithContext("error", err.Error())
	}
	return data, nil
}

func (s *JSONEventSerializer) Deserialize(data []byte, event interface{}) error {
	if err := json.Unmarshal(data, event); err != nil {
		return errors.NewInternalError("failed to deserialize event").WithContext("error", err.Error())
	}
	return nil
}
