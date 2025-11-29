package container

import (
	"context"
	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/internal/domain/port"
	pubsubInfra "github.com/histopathai/image-processing-service/internal/infrastructure/events/pubsub"
	"github.com/histopathai/image-processing-service/internal/infrastructure/events/stdout"
	"github.com/histopathai/image-processing-service/internal/service"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type Container struct {
	Config                 *config.Config
	Logger                 *slog.Logger
	PubSubClient           *pubsub.Client
	Publisher              port.Publisher
	EventSerializer        events.EventSerializer
	ImageProcessingService *service.ImageProcessingService
	StorageService         *service.StorageService
	JobOrchestrator        *service.JobOrchestrator
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Container, error) {
	logger.Info("Initializing container",
		"environment", cfg.Env,
		"worker_type", cfg.WorkerType)

	var publisher port.Publisher
	var pubsubClient *pubsub.Client
	var err error

	// Initialize publisher based on environment
	if cfg.Env == config.EnvLocal {
		// Use stdout publisher for local development
		logger.Info("Using STDOUT publisher for local development")
		publisher = stdout.NewPublisher(logger)
	} else {
		// Use Pub/Sub publisher for cloud environment
		pubsubClient, err = pubsub.NewClient(ctx, cfg.GCP.ProjectID)
		if err != nil {
			logger.Error("Failed to create Pub/Sub client", "error", err)
			return nil, errors.WrapInternalError(err, "failed to create pubsub client")
		}
		publisher = pubsubInfra.NewPublisher(pubsubClient, logger)
		logger.Info("Using Pub/Sub publisher")
	}

	// Event serializer
	eventSerializer := events.NewJSONEventSerializer()

	// Image processor service
	imageProcessor := service.NewImageProcessingService(logger, cfg)

	// Storage service
	storageService := service.NewStorageService(logger)

	// Job orchestrator
	jobOrchestrator := service.NewJobOrchestrator(
		logger,
		cfg,
		imageProcessor,
		storageService,
		publisher,
		eventSerializer,
	)

	logger.Info("Container initialized successfully")

	return &Container{
		Config:                 cfg,
		Logger:                 logger,
		PubSubClient:           pubsubClient,
		Publisher:              publisher,
		EventSerializer:        eventSerializer,
		ImageProcessingService: imageProcessor,
		StorageService:         storageService,
		JobOrchestrator:        jobOrchestrator,
	}, nil
}

func (c *Container) Close() error {
	c.Logger.Info("Closing container resources")

	if c.PubSubClient != nil {
		if err := c.PubSubClient.Close(); err != nil {
			c.Logger.Error("Failed to close Pub/Sub client", "error", err)
			return err
		}
	}

	c.Logger.Info("Container resources closed successfully")
	return nil
}
