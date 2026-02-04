package events

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

type BaseEvent struct {
	EventID   string    `json:"event_id"`
	EventType EventType `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

func NewBaseEvent(eventType EventType) BaseEvent {
	return BaseEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now(),
	}
}

type Event interface {
	GetEventID() string
	GetEventType() EventType
	GetTimestamp() time.Time
}

func (e BaseEvent) GetEventID() string {
	return e.EventID
}

func (e BaseEvent) GetEventType() EventType {
	return e.EventType
}

func (e BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}
