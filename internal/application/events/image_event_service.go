package events

import (
	"context"
	"log/slog"

	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/pkg/config"
	pkgErrors "github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageEventService struct {
	publisher  events.Publisher
	serializer events.EventSerializer
	logger     *slog.Logger
	cfg        *config.PubSubConfig
}

func NewImageEventService(
	publisher events.Publisher,
	serializer events.EventSerializer,
	logger *slog.Logger,
	cfg *config.PubSubConfig,
) *ImageEventService {
	return &ImageEventService{
		publisher:  publisher,
		serializer: serializer,
		logger:     logger,
		cfg:        cfg,
	}
}

func (s *ImageEventService) PublishImageProcessingCompleted(
	ctx context.Context,
	event events.ImageProcessingCompletedEvent,
) error {

	s.logger.Info("Publishing ImageProcessingCompletedEvent",
		"image_id", event.ImageID,
		"event_id", event.EventID,
	)

	data, err := s.serializer.Serialize(event)
	if err != nil {
		s.logger.Error("Failed to serialize ImageProcessingCompletedEvent",
			"image_id", event.ImageID,
			"event_id", event.EventID)

		return pkgErrors.WrapMessagingError(err, "Failed to serialize event")
	}

	attributes := map[string]string{
		"event_type": string(event.EventType),
		"event_id":   event.EventID,
		"image_id":   event.ImageID,
	}

	err = s.publisher.Publish(ctx, s.cfg.ImageProcessResultTopicID, data, attributes)
	if err != nil {
		s.logger.Error("Failed to publish event", "error", err)
		return pkgErrors.WrapMessagingError(err, "failed to publish event")
	}

	s.logger.Info("Successfully published image processing completed event",
		"event_id", event.EventID,
		"image_id", event.ImageID,
	)

	return nil
}

func (s *ImageEventService) PublishImageProcessingFailed(
	ctx context.Context,
	event events.ImageProcessingFailedEvent,
) error {
	s.logger.Info("Publishing image processing failed event",
		"event_id", event.EventID,
		"image_id", event.ImageID,
		"failure_reason", event.FailureReason,
	)

	data, err := s.serializer.Serialize(event)
	if err != nil {
		s.logger.Error("Failed to serialize event", "error", err)
		return pkgErrors.WrapMessagingError(err, "failed to serialize event")
	}

	attributes := map[string]string{
		"event_type":     string(event.EventType),
		"event_id":       event.EventID,
		"image_id":       event.ImageID,
		"failure_reason": event.FailureReason,
	}

	err = s.publisher.Publish(ctx, s.cfg.ImageProcessResultTopicID, data, attributes)
	if err != nil {
		s.logger.Error("Failed to publish event", "error", err)
		return pkgErrors.WrapMessagingError(err, "failed to publish event")
	}

	s.logger.Info("Successfully published image processing failed event",
		"event_id", event.EventID,
		"image_id", event.ImageID,
	)

	return nil
}
