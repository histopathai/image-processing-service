package container

import (
	"context"
	"log/slog"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/internal/domain/port"
	InfraPubsub "github.com/histopathai/image-processing-service/internal/infrastructure/events/pubsub"
	"github.com/histopathai/image-processing-service/internal/infrastructure/events/stdout"
	InfraStorage "github.com/histopathai/image-processing-service/internal/infrastructure/storage"
	"github.com/histopathai/image-processing-service/internal/service"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type Container struct {
	Config                 *config.Config
	Logger                 *slog.Logger
	EventPublisher         port.EventPublisher
	OutputStorage          port.Storage
	EventSerializer        events.EventSerializer
	ImageProcessingService *service.ImageProcessingService
	JobOrchestrator        *service.JobOrchestrator
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Container, error) {

	if cfg.Env == "" {
		logger.Error("Environment not set in configuration")
		return nil, errors.NewInternalError("environment not set in configuration")
	}
	var publisher port.EventPublisher
	var outputStorage port.Storage
	var eventSerializer events.EventSerializer
	var imageProcessor *service.ImageProcessingService
	var jobOrchestrator *service.JobOrchestrator

	if cfg.Env == config.EnvLocal {
		logger.Info("Running in local environment")
		publisher = stdout.NewPublisher(logger)

		outputStorage = InfraStorage.NewLocalStorage(logger)
		logger.Info("Using local storage service")

	} else {
		logger.Info("Running in cloud environment")

		pubsubClient, err := pubsub.NewClient(ctx, cfg.GCP.ProjectID)
		if err != nil {
			logger.Error("Failed to create Pub/Sub client", "error", err)
			return nil, errors.WrapInternalError(err, "failed to create pubsub client")
		}
		publisher = InfraPubsub.NewPublisher(pubsubClient, logger)
		logger.Info("Using Pub/Sub publisher")

		storageClient, err := storage.NewClient(ctx)
		if err != nil {
			logger.Error("Failed to create GCS client", "error", err)
			return nil, errors.WrapInternalError(err, "failed to create GCS client")
		}
		outputStorage = InfraStorage.NewGCSStorage(logger, storageClient, cfg.GCP.OutputBucketName)
		logger.Info("Using GCS storage service")
	}

	eventSerializer = events.NewJSONEventSerializer()

	// Create storage instances based on configuration
	inputStorage := InfraStorage.NewMountStorage(cfg.Storage.InputMountPath, logger)
	outputMountStorage := InfraStorage.NewMountStorage(cfg.Storage.OutputMountPath, logger)

	imageProcessor = service.NewImageProcessingService(logger, cfg, inputStorage, outputMountStorage)

	jobOrchestrator = service.NewJobOrchestrator(
		logger,
		cfg,
		imageProcessor,
		outputStorage,
		publisher,
		eventSerializer,
	)

	logger.Info("Container initialized successfully")

	return &Container{
		Config:                 cfg,
		Logger:                 logger,
		EventPublisher:         publisher,
		OutputStorage:          outputStorage,
		EventSerializer:        eventSerializer,
		ImageProcessingService: imageProcessor,
		JobOrchestrator:        jobOrchestrator,
	}, nil
}

func (c *Container) Close() error {
	c.Logger.Info("Closing container resources")

	if err := c.EventPublisher.Close(); err != nil {
		c.Logger.Error("Failed to close event publisher", "error", err)
		return errors.WrapInternalError(err, "failed to close event publisher")
	}

	c.Logger.Info("Container resources closed successfully")
	return nil
}
