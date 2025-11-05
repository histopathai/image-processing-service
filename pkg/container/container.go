package container

import (
	"context"
	"log/slog"

	"cloud.google.com/go/pubsub"
	appEvents "github.com/histopathai/image-processing-service/internal/application/events"
	"github.com/histopathai/image-processing-service/internal/application/handlers"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	pubsubInfra "github.com/histopathai/image-processing-service/internal/infrastructure/events/pubsub"
	"github.com/histopathai/image-processing-service/internal/service"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type Container struct {
	Config              *config.Config
	Logger              *slog.Logger
	PubSubClient        *pubsub.Client
	Publisher           events.Publisher
	Subscriber          events.Subscriber
	EventSerializer     events.EventSerializer
	ImageProcessor      *service.ImageProcessor
	ImageEventService   *appEvents.ImageEventService
	ImageHandlerService *handlers.ImageHandlerService
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Container, error) {
	logger.Info("Initializing container")

	// Create Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, cfg.GCP.ProjectID)
	if err != nil {
		logger.Error("Failed to create Pub/Sub client", "error", err)
		return nil, errors.WrapInternalError(err, "failed to create pubsub client")
	}

	// Event serializer
	eventSerializer := events.NewJSONEventSerializer()

	// Publisher
	publisher := pubsubInfra.NewPublisher(pubsubClient, logger)

	// Subscriber
	subscriber := pubsubInfra.NewSubscriber(
		pubsubClient,
		cfg.PubSubConfig.ImageProcessingSubID,
		logger,
	)

	// Image processor service
	imageProcessor := service.NewImageProcessor(logger, cfg.DZIConfig)

	// Image event service
	imageEventService := appEvents.NewImageEventService(
		publisher,
		eventSerializer,
		logger,
		&cfg.PubSubConfig,
	)

	// Image handler service
	imageHandlerService := handlers.NewImageHandlerService(
		imageProcessor,
		imageEventService,
		eventSerializer,
		cfg,
		logger,
	)

	logger.Info("Container initialized successfully")

	return &Container{
		Config:              cfg,
		Logger:              logger,
		PubSubClient:        pubsubClient,
		Publisher:           publisher,
		Subscriber:          subscriber,
		EventSerializer:     eventSerializer,
		ImageProcessor:      imageProcessor,
		ImageEventService:   imageEventService,
		ImageHandlerService: imageHandlerService,
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
