package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/domain/port"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type JobOrchestrator struct {
	logger                 *slog.Logger
	config                 *config.Config
	imageProcessingService *ImageProcessingService
	storageService         *StorageService
	publisher              port.Publisher
	eventSerializer        events.EventSerializer
}

func NewJobOrchestrator(
	logger *slog.Logger,
	config *config.Config,
	imageProcessingService *ImageProcessingService,
	storageService *StorageService,
	publisher port.Publisher,
	eventSerializer events.EventSerializer,
) *JobOrchestrator {
	return &JobOrchestrator{
		logger:                 logger,
		config:                 config,
		imageProcessingService: imageProcessingService,
		storageService:         storageService,
		publisher:              publisher,
		eventSerializer:        eventSerializer,
	}
}

func (o *JobOrchestrator) ProcessJob(ctx context.Context, input *model.JobInput) error {
	o.logger.Info("Starting job processing",
		"imageID", input.ImageID,
		"originPath", input.OriginPath,
		"bucketName", input.BucketName,
		"use_gcs_upload", o.config.Storage.UseGCSUpload,
	)

	inputPath := o.constructInputPath(input)

	if !o.storageService.FileExists(inputPath) {
		err := errors.NewNotFoundError("input file").
			WithContext("path", inputPath).
			WithContext("imageID", input.ImageID)
		o.publishFailureEvent(ctx, input.ImageID, err, false)
		return err
	}

	file, err := model.NewFile(
		input.ImageID,
		filepath.Base(inputPath),
		filepath.Dir(inputPath),
		nil, nil, nil, nil,
	)
	if err != nil {
		retryable := !errors.IsNonRetryable(err)
		o.publishFailureEvent(ctx, input.ImageID, err, retryable)
		return err
	}

	outputWorkspace, err := o.imageProcessingService.ProcessFile(ctx, file)
	if err != nil {
		retryable := !errors.IsNonRetryable(err)
		o.publishFailureEvent(ctx, input.ImageID, err, retryable)
		return err
	}

	finalOutputPath := o.constructOutputPath(input.ImageID)

	o.logger.Info("Starting upload",
		"imageID", input.ImageID,
		"source", outputWorkspace.Dir(),
		"destination", finalOutputPath,
		"method", o.getUploadMethod(),
	)

	if err := o.storageService.UploadDirectory(ctx, outputWorkspace.Dir(), finalOutputPath); err != nil {
		retryable := !errors.IsNonRetryable(err)
		o.publishFailureEvent(ctx, input.ImageID, err, retryable)
		return err
	}

	o.logger.Info("Upload completed successfully",
		"imageID", input.ImageID,
		"method", o.getUploadMethod(),
	)

	if err := outputWorkspace.Remove(); err != nil {
		o.logger.Warn("Failed to clean up output workspace",
			"imageID", input.ImageID,
			"error", err,
		)
	}

	o.publishSuccessEvent(ctx, input.ImageID, file, input.ImageID)

	o.logger.Info("Image processing job completed successfully",
		"imageID", input.ImageID,
	)

	return nil
}

func (o *JobOrchestrator) getUploadMethod() string {
	if o.config.Storage.UseGCSUpload {
		return "gcs_sdk_parallel"
	}
	return "mount_copy"
}

func (o *JobOrchestrator) constructInputPath(input *model.JobInput) string {

	if o.config.Env == config.EnvLocal {
		return input.OriginPath
	}
	return filepath.Join(o.config.MountPath.InputMountPath, input.OriginPath)
}

func (o *JobOrchestrator) constructOutputPath(imageID string) string {
	// if GCS upload is used and not local env, return imageID as is
	if o.config.Storage.UseGCSUpload && o.config.Env != config.EnvLocal {
		return imageID
	}
	// otherwise, construct full path
	if o.config.Env == config.EnvLocal {
		return filepath.Join(o.config.MountPath.OutputMountPath, imageID)
	}
	return filepath.Join(o.config.MountPath.OutputMountPath, imageID)
}

func (o *JobOrchestrator) publishSuccessEvent(ctx context.Context, imageID string, file *model.File, outputPath string) {
	event := events.NewImageProcessingResultEvent(imageID, true, string(o.config.WorkerType)).
		WithSuccess(
			outputPath,
			file.WidthValue(),
			file.HeightValue(),
			file.SizeValue(),
			file.FormatValue(),
		)

	if err := o.publishEvent(ctx, event); err != nil {
		o.logger.Error("Failed to publish success event",
			"imageID", imageID,
			"error", err,
		)
	}
}

func (o *JobOrchestrator) publishFailureEvent(ctx context.Context, imageID string, processingErr error, retryable bool) {
	reason := processingErr.Error()
	event := events.NewImageProcessingResultEvent(imageID, false, string(o.config.WorkerType)).
		WithFailure(reason, retryable)

	if err := o.publishEvent(ctx, event); err != nil {
		o.logger.Error("Failed to publish failure event",
			"image_id", imageID,
			"error", err)
	}
}

func (o *JobOrchestrator) publishEvent(ctx context.Context, event *events.ImageProcessingResultEvent) error {
	data, err := o.eventSerializer.Serialize(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	attributes := map[string]string{
		"event_type": string(event.EventType),
		"image_id":   event.ImageID,
	}

	return o.publisher.Publish(ctx, o.config.PubSubConfig.ImageProcessResultTopicID, data, attributes)
}
