package events

import "github.com/histopathai/image-processing-service/internal/domain/model"

const (
	EventTypeImageProcessingResult EventType = "image.processing.completed.v1"
)

type ProcessResult struct {
	Width  int
	Height int
	Size   int64
}

type ImageProcessCompleteEvent struct {
	BaseEvent
	ImageID           string
	ProcessingVersion string
	Contents          []model.Content

	Success       bool
	Result        *ProcessResult
	FailureReason string
	Retryable     bool
}
