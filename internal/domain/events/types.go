package events

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeImageProcessingRequested EventType = "image.processing.requested.v1"
	EventTypeImageProcessingCompleted EventType = "image.processing.completed.v1"
	EventTypeImageProcessingFailed    EventType = "image.processing.failed.v1"
)

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

type ImageProcessingRequestedEvent struct {
	BaseEvent
	ImageID    string `json:"image-id"`
	OriginPath string `json:"origin-path"`
}

func NewImageProcessingRequestedEvent(imageID, originPath string) ImageProcessingRequestedEvent {
	return ImageProcessingRequestedEvent{
		BaseEvent:  NewBaseEvent(EventTypeImageProcessingRequested),
		ImageID:    imageID,
		OriginPath: originPath,
	}
}

// YENİ: İşleme tamamlandı eventi
type ImageProcessingCompletedEvent struct {
	BaseEvent
	ImageID       string `json:"image-id"`
	ProcessedPath string `json:"processed-path"` // DZI, tiles, thumbnail root path
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Size          int64  `json:"size"`
}

func NewImageProcessingCompletedEvent(imageID, processedPath string, width int, height int, size int64) ImageProcessingCompletedEvent {
	return ImageProcessingCompletedEvent{
		BaseEvent:     NewBaseEvent(EventTypeImageProcessingCompleted),
		ImageID:       imageID,
		ProcessedPath: processedPath,
		Width:         width,
		Height:        height,
		Size:          size,
	}
}

type ImageProcessingFailedEvent struct {
	BaseEvent
	ImageID       string `json:"image-id"`
	FailureReason string `json:"failure-reason"`
}

func NewImageProcessingFailedEvent(imageID, failureReason string) ImageProcessingFailedEvent {
	return ImageProcessingFailedEvent{
		BaseEvent:     NewBaseEvent(EventTypeImageProcessingFailed),
		ImageID:       imageID,
		FailureReason: failureReason,
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
