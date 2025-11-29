package events

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

type BaseEvent struct {
	EventID   string
	EventType EventType
	Timestamp time.Time
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
