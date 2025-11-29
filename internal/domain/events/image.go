package events

const (
	EventTypeImageProcessingResult EventType = "image.processing.completed.v1"
)

type ImageProcessingResultEvent struct {
	BaseEvent
	ImageID       string  `json:"image-id"`
	Success       bool    `json:"success"`
	ProcessedPath *string `json:"processed-path,omitempty"`
	Width         *int    `json:"width,omitempty"`
	Height        *int    `json:"height,omitempty"`
	Size          *int64  `json:"size,omitempty"`
	Format        *string `json:"format,omitempty"`
	FailureReason *string `json:"failure-reason,omitempty"`
	Retryable     *bool   `json:"retryable,omitempty"`
	WorkerType    string  `json:"worker-type"` // small, medium, large
}

func NewImageProcessingResultEvent(
	imageID string,
	success bool,
	workerType string,
) *ImageProcessingResultEvent {
	return &ImageProcessingResultEvent{
		BaseEvent:  NewBaseEvent(EventTypeImageProcessingResult),
		ImageID:    imageID,
		Success:    success,
		WorkerType: workerType,
	}
}

func (e *ImageProcessingResultEvent) WithSuccess(processedPath string, width, height int, size int64, format string) *ImageProcessingResultEvent {
	e.ProcessedPath = &processedPath
	e.Width = &width
	e.Height = &height
	e.Size = &size
	e.Format = &format
	return e
}

func (e *ImageProcessingResultEvent) WithFailure(reason string, retryable bool) *ImageProcessingResultEvent {
	e.FailureReason = &reason
	e.Retryable = &retryable
	return e
}
