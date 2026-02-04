package events

import "github.com/histopathai/image-processing-service/internal/domain/model"

const (
	ImageProcessCompleteEventType EventType = "image.process.complete.v1"
)

type ProcessResult struct {
	Width  int   `json:"width"`
	Height int   `json:"height"`
	Size   int64 `json:"size"`
}

type ImageProcessCompleteEvent struct {
	BaseEvent
	ImageID           string          `json:"image_id"`
	ProcessingVersion string          `json:"processing_version"`
	Contents          []model.Content `json:"contents"`

	Success       bool           `json:"success"`
	Result        *ProcessResult `json:"result,omitempty"`
	FailureReason string         `json:"failure_reason,omitempty"`
	Retryable     bool           `json:"retryable"`
}
