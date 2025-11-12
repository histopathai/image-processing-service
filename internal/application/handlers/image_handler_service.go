package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	localTempDir := filepath.Join("/tmp", requestEvent.ImageID)
	if err := os.MkdirAll(localTempDir, os.ModePerm); err != nil {
		h.logger.Error("Failed to create local temp directory", "error", err, "path", localTempDir)
		return pkgErrors.WrapProcessingError(err, "failed to create local temp directory")
	}
	defer os.RemoveAll(localTempDir)

	localOutputPathBase := filepath.Join(localTempDir, outputBaseName)
	file.ProcessedPath = &localOutputPathBase

	h.logger.Info("Starting DZI processing (local)",
		"image_id", file.ID,
		"output_path_base", localOutputPathBase,
		"format", *file.Format,
	)

	if err := h.processor.DZIProcessor(ctx, file); err != nil {
		h.logger.Error("Failed to process DZI", "error", err)
		return pkgErrors.WrapProcessingError(err, "failed to process DZI")
	}

	// For thumbnail, use converted TIFF if it's a DNG file
	thumbnailInputPath := inputPath
	var tempTiffForThumbnail string

	if file.Format != nil && strings.ToLower(*file.Format) == "dng" {
		h.logger.Info("Creating intermediate TIFF for thumbnail from DNG",
			"image_id", file.ID,
		)
		tempTiffForThumbnail = filepath.Join(localTempDir, "thumbnail_source.tiff")

		if err := h.processor.ConvertDNGToTIFF(ctx, inputPath, tempTiffForThumbnail); err != nil {
			h.logger.Warn("Failed to convert DNG for thumbnail (continuing with original)", "error", err)
		} else {
			thumbnailInputPath = tempTiffForThumbnail
			defer os.Remove(tempTiffForThumbnail)
		}
	}

	localThumbnailPath := filepath.Join(localTempDir, "thumbnail.jpg")
	h.logger.Info("Creating thumbnail (local)",
		"image_id", file.ID,
		"thumbnail_path", localThumbnailPath,
		"source", thumbnailInputPath,
	)

	if err := h.processor.ExtractThumbnail(
		ctx,
		thumbnailInputPath,
		localThumbnailPath,
		h.cfg.ThumbnailConfig.Width,
		h.cfg.ThumbnailConfig.Height,
		h.cfg.ThumbnailConfig.Quality,
	); err != nil {
		h.logger.Warn("Failed to create thumbnail (continuing anyway)", "error", err)
	}

	finalGCSOutputDir := filepath.Join(h.cfg.MountPath.OutputMountPath, requestEvent.ImageID)

	if err := os.MkdirAll(finalGCSOutputDir, os.ModePerm); err != nil {
		h.logger.Error("Failed to create final GCS output directory", "error", err, "path", finalGCSOutputDir)
		return pkgErrors.WrapProcessingError(err, "failed to create final GCS output directory")
	}

	localDziFile := localOutputPathBase + ".dzi"
	gcsDziFile := filepath.Join(finalGCSOutputDir, outputBaseName+".dzi")
	if err := copyFile(localDziFile, gcsDziFile); err != nil {
		h.logger.Error("Failed to copy .dzi file to GCS", "error", err, "src", localDziFile, "dst", gcsDziFile)
		return pkgErrors.WrapProcessingError(err, "failed to copy .dzi file")
	}

	localDziDir := localOutputPathBase + "_files"
	gcsDziDir := filepath.Join(finalGCSOutputDir, outputBaseName+"_files")
	if err := copyDirectory(localDziDir, gcsDziDir); err != nil {
		h.logger.Error("Failed to copy _files directory to GCS", "error", err, "src", localDziDir, "dst", gcsDziDir)
		return pkgErrors.WrapProcessingError(err, "failed to copy _files directory")
	}

	if _, err := os.Stat(localThumbnailPath); !os.IsNotExist(err) {
		gcsThumbnailPath := filepath.Join(finalGCSOutputDir, "thumbnail.jpg")
		if err := copyFile(localThumbnailPath, gcsThumbnailPath); err != nil {
			h.logger.Warn("Failed to copy thumbnail to GCS", "error", err, "src", localThumbnailPath, "dst", gcsThumbnailPath)
		}
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
		"final_destination", finalGCSOutputDir,
	)

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("could not copy from %s to %s: %w", src, dst, err)
	}

	srcInfo, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, srcInfo.Mode())
	}

	return nil
}

func copyDirectory(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
