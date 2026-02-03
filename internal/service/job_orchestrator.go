package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/domain/port"
	"github.com/histopathai/image-processing-service/internal/domain/vobj"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type JobOrchestrator struct {
	logger                 *slog.Logger
	config                 *config.Config
	imageProcessingService *ImageProcessingService
	storage                port.Storage
	publisher              port.EventPublisher
	eventSerializer        events.EventSerializer
}

func NewJobOrchestrator(
	logger *slog.Logger,
	config *config.Config,
	imageProcessingService *ImageProcessingService,
	storage port.Storage,
	publisher port.EventPublisher,
	eventSerializer events.EventSerializer,
) *JobOrchestrator {
	return &JobOrchestrator{
		logger:                 logger,
		config:                 config,
		imageProcessingService: imageProcessingService,
		storage:                storage,
		publisher:              publisher,
		eventSerializer:        eventSerializer,
	}
}

func (o *JobOrchestrator) ProcessJob(ctx context.Context, input *model.JobInput) error {
	o.logger.Info("Starting job processing",
		"imageID", input.ImageID,
		"originPath", input.OriginPath,
	)

	// OriginPath is relative to the input storage mount point
	// e.g., "image-id/file.png" or just "file.png"
	// The storage layer handles the actual mount point (/input, /gcs/bucket, etc.)

	file, err := model.NewFile(
		input.ImageID,
		input.OriginPath, // Use OriginPath directly as filename (relative path in storage)
		"",               // Dir will be set by ImageProcessingService after copying to /tmp
		nil, nil, nil, nil,
	)
	if err != nil {
		o.publishEvent(ctx, &events.ImageProcessCompleteEvent{
			ImageID:           input.ImageID,
			ProcessingVersion: input.ProcessingVersion,
			Success:           false,
			FailureReason:     err.Error(),
			Retryable:         !errors.IsNonRetryable(err),
		})
		return err
	}

	var container string
	if input.ProcessingVersion == "v1" {
		container = "fs"
	} else {
		container = "zip"
	}

	outputWorkspace, err := o.imageProcessingService.ProcessFile(ctx, file, container)
	if err != nil {
		o.publishEvent(ctx, &events.ImageProcessCompleteEvent{
			ImageID:           input.ImageID,
			ProcessingVersion: input.ProcessingVersion,
			Success:           false,
			FailureReason:     err.Error(),
			Retryable:         !errors.IsNonRetryable(err),
		})
		return err
	}

	finalOutputPath := o.constructOutputPath(input.ImageID)

	o.logger.Info("Preparing contents", "imageID", input.ImageID)

	var contentProvider vobj.ContentProvider
	if o.config.Env == config.EnvLocal {
		contentProvider = vobj.ContentProviderLocal
	} else {
		contentProvider = vobj.ContentProviderGCS
	}

	contents, err := o.prepareContents(input, outputWorkspace.Dir(), finalOutputPath, contentProvider)
	if err != nil {
		o.publishEvent(ctx, &events.ImageProcessCompleteEvent{
			ImageID:           input.ImageID,
			ProcessingVersion: input.ProcessingVersion,
			Success:           false,
			FailureReason:     fmt.Sprintf("failed to prepare contents: %v", err),
			Retryable:         false,
		})
		return err
	}

	o.logger.Info("Starting upload",
		"imageID", input.ImageID,
		"source", outputWorkspace.Dir(),
		"destination", finalOutputPath,
	)

	if err := o.storage.UploadDirectory(ctx, outputWorkspace.Dir(), finalOutputPath); err != nil {
		o.publishEvent(ctx, &events.ImageProcessCompleteEvent{
			ImageID:           input.ImageID,
			ProcessingVersion: input.ProcessingVersion,
			Success:           false,
			FailureReason:     err.Error(),
			Retryable:         !errors.IsNonRetryable(err),
		})
		return err
	}

	o.logger.Info("Upload completed successfully",
		"imageID", input.ImageID,
		"destination", finalOutputPath,
	)

	var eventContents []model.Content
	for _, c := range contents {
		eventContents = append(eventContents, *c)
	}

	o.publishEvent(ctx, &events.ImageProcessCompleteEvent{
		ImageID:           input.ImageID,
		ProcessingVersion: input.ProcessingVersion,
		Success:           true,
		Contents:          eventContents,
		Result: &events.ProcessResult{
			Width:  file.WidthValue(),
			Height: file.HeightValue(),
			Size:   file.SizeValue(),
		},
	})

	if o.config.Env != config.EnvProduction {
		if err := outputWorkspace.Remove(); err != nil {
			o.logger.Warn("Failed to clean up output workspace",
				"imageID", input.ImageID,
				"error", err,
			)
		}
	}

	o.logger.Info("Image processing job completed successfully",
		"imageID", input.ImageID,
	)

	return nil
}

func (o *JobOrchestrator) constructInputPath(input *model.JobInput) string {

	if o.config.Env == config.EnvLocal {
		return input.OriginPath
	}
	return filepath.Join("/gcs/"+o.config.GCP.InputBucketName, input.OriginPath)
}

func (o *JobOrchestrator) constructOutputPath(imageID string) string {
	// if GCS upload is used and not local env, return imageID as is
	if o.config.Env != config.EnvLocal {
		return imageID
	}
	// otherwise, construct full path
	if o.config.Env == config.EnvLocal {
		return filepath.Join(o.config.OutputRootPath, imageID)
	}
	return filepath.Join(o.config.OutputRootPath, imageID)
}

func (o *JobOrchestrator) publishEvent(ctx context.Context, event *events.ImageProcessCompleteEvent) error {
	data, err := o.eventSerializer.Serialize(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	attributes := map[string]string{
		"event_type": string(event.EventType),
		"image_id":   event.ImageID,
	}

	return o.publisher.Publish(ctx, o.config.ImageProcessingTopicID, data, attributes)
}

func (o *JobOrchestrator) prepareContents(input *model.JobInput, sourceDir string, finalOutputPath string, contentProvider vobj.ContentProvider) ([]*model.Content, error) {
	contents := make([]*model.Content, 0)
	parent := vobj.ParentRef{
		ID:   input.ImageID,
		Type: vobj.ParentTypeImage,
	}

	// Helper to create content
	addContent := func(filename string, contentType vobj.ContentType) error {
		sourcePath := filepath.Join(sourceDir, filename)
		info, err := os.Stat(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				o.logger.Warn("Content file not found", "path", sourcePath)
				return fmt.Errorf("content file not found: %s", filename)
			}
			return fmt.Errorf("failed to stat file %s: %w", sourcePath, err)
		}

		content := &model.Content{
			Entity: vobj.Entity{
				ID:         uuid.New().String(),
				Name:       filename,
				EntityType: vobj.EntityTypeContent,
				Parent:     parent,
				CreatedAt:  info.ModTime(),
				UpdatedAt:  info.ModTime(),
			},
			Provider:      contentProvider,
			Path:          filepath.Join(finalOutputPath, filename),
			ContentType:   contentType,
			Size:          info.Size(),
			UploadPending: false,
		}
		contents = append(contents, content)
		return nil
	}

	// Add Thumbnail
	if err := addContent("thumbnail.jpg", vobj.ContentTypeThumbnailJPEG); err != nil {
		return nil, err
	}

	// Add DZI
	if err := addContent("image.dzi", vobj.ContentTypeApplicationDZI); err != nil {
		return nil, err
	}

	if input.ProcessingVersion == "v1" {
		// Add Tiles
		// For v1, "tiles" might be a directory or a specific file structure.
		// Assuming "tiles" is a directory or file that represents the tiles data.
		// Existing code referenced filepath.Join(finalOutputPath, "tiles").
		// If it's a directory, os.Stat works.
		if err := addContent("tiles", vobj.ContentTypeApplicationOctetStream); err != nil {
			return nil, err
		}
	} else {
		// v2: Zip and IndexMap
		if err := addContent("image.zip", vobj.ContentTypeApplicationZip); err != nil {
			return nil, err
		}
		if err := addContent("IndexMap.json", vobj.ContentTypeApplicationJSON); err != nil {
			return nil, err
		}
	}

	return contents, nil
}
