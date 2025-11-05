package handlers

import (
	"context"
	"log/slog"
	"path/filepath"

	appEvents "github.com/histopathai/image-processing-service/internal/application/events"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/service"
	"github.com/histopathai/image-processing-service/pkg/config"
	pkgErrors "github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageHandlerService struct {
	processor    *service.ImageProcessor
	eventService *appEvents.ImageEventService
	serializer   events.EventSerializer
	cfg          *config.Config
	logger       *slog.Logger
}

func NewImageHandlerService(
	processor *service.ImageProcessor,
	eventService *appEvents.ImageEventService,
	serializer events.EventSerializer,
	cfg *config.Config,
	logger *slog.Logger,
) *ImageHandlerService {
	return &ImageHandlerService{
		processor:    processor,
		eventService: eventService,
		serializer:   serializer,
		cfg:          cfg,
		logger:       logger,
	}
}

func (h *ImageHandlerService) HandleImageProcessingRequest(
	ctx context.Context,
	data []byte,
	attributes map[string]string,
) error {
	eventType := attributes["event_type"]

	h.logger.Info("Handling image processing request",
		"event_type", eventType,
		"attributes", attributes,
	)

	if eventType != string(events.EventTypeImageProcessingRequested) {
		h.logger.Warn("Unexpected event type", "event_type", eventType)
		return pkgErrors.NewValidationError("unexpected event type").
			WithContext("event_type", eventType)
	}

	var requestEvent events.ImageProcessingRequestedEvent
	if err := h.serializer.Deserialize(data, &requestEvent); err != nil {
		h.logger.Error("Failed to deserialize event", "error", err)
		return pkgErrors.WrapMessagingError(err, "failed to deserialize event")
	}

	h.logger.Info("Processing image request",
		"event_id", requestEvent.EventID,
		"image_id", requestEvent.ImageID,
		"origin_path", requestEvent.OriginPath,
	)

	if err := h.processImage(ctx, requestEvent); err != nil {
		h.logger.Error("Failed to process image",
			"image_id", requestEvent.ImageID,
			"error", err,
		)

		failedEvent := events.NewImageProcessingFailedEvent(
			requestEvent.ImageID,
			err.Error(),
		)

		if publishErr := h.eventService.PublishImageProcessingFailed(ctx, failedEvent); publishErr != nil {
			h.logger.Error("Failed to publish failure event", "error", publishErr)
		}

		return err
	}

	h.logger.Info("Successfully processed image request",
		"event_id", requestEvent.EventID,
		"image_id", requestEvent.ImageID,
	)

	return nil
}

func (h *ImageHandlerService) processImage(
	ctx context.Context,
	requestEvent events.ImageProcessingRequestedEvent,
) error {
	inputPath := filepath.Join(h.cfg.MountPath.InputMountPath, requestEvent.OriginPath)

	file := &model.File{
		ID:       requestEvent.ImageID,
		Filename: filepath.Base(requestEvent.OriginPath),
		Path:     inputPath,
	}

	h.logger.Debug("Getting image info", "file_path", inputPath)
	if err := h.processor.GetImageInfo(ctx, file); err != nil {
		h.logger.Error("Failed to get image info", "error", err)
		return pkgErrors.WrapProcessingError(err, "failed to get image info")
	}

	if !model.SupportedFormats.IsSupported(*file.Format) {
		return pkgErrors.NewValidationError("unsupported image format").
			WithContext("format", *file.Format)
	}

	outputBaseName := filepath.Base(requestEvent.OriginPath)
	outputBaseName = outputBaseName[:len(outputBaseName)-len(filepath.Ext(outputBaseName))]
	outputDir := filepath.Join(h.cfg.MountPath.OutputMountPath, requestEvent.ImageID)
	outputPathBase := filepath.Join(outputDir, outputBaseName)

	file.ProcessedPath = &outputPathBase

	h.logger.Info("Starting DZI processing",
		"image_id", file.ID,
		"input_path", inputPath,
		"output_path", outputPathBase,
	)

	if err := h.processor.DZIProcessor(ctx, file); err != nil {
		h.logger.Error("Failed to process DZI", "error", err)
		return pkgErrors.WrapProcessingError(err, "failed to process DZI")
	}

	thumbnailPath := filepath.Join(outputDir, "thumbnail.jpg")
	h.logger.Info("Creating thumbnail",
		"image_id", file.ID,
		"thumbnail_path", thumbnailPath,
	)

	if err := h.processor.ExtractThumbnail(
		ctx,
		inputPath,
		thumbnailPath,
		h.cfg.ThumbnailConfig.Width,
		h.cfg.ThumbnailConfig.Height,
		h.cfg.ThumbnailConfig.Quality,
	); err != nil {
		h.logger.Warn("Failed to create thumbnail (continuing anyway)", "error", err)
	}

	relativeOutputPath := filepath.Join(requestEvent.ImageID, outputBaseName)

	completedEvent := events.NewImageProcessingCompletedEvent(
		requestEvent.ImageID,
		relativeOutputPath,
		*file.Width,
		*file.Height,
		*file.Size,
	)

	h.logger.Info("Publishing completion event",
		"image_id", requestEvent.ImageID,
		"processed_path", relativeOutputPath,
	)

	if err := h.eventService.PublishImageProcessingCompleted(ctx, completedEvent); err != nil {
		h.logger.Error("Failed to publish completion event", "error", err)
		return pkgErrors.WrapMessagingError(err, "failed to publish completion event")
	}

	h.logger.Info("Image processing completed successfully",
		"image_id", requestEvent.ImageID,
		"width", *file.Width,
		"height", *file.Height,
		"size", *file.Size,
	)

	return nil
}
