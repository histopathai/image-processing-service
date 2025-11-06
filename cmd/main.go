package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/container"
	"github.com/histopathai/image-processing-service/pkg/logger"
)

type CloudEventPayload struct {
	Message struct {
		Data       string            `json:"data"`
		Attributes map[string]string `json:"attributes"`
	} `json:"message"`
}

func main() {
	// Initialize basic logger for startup
	startupLogger := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
	})

	startupLogger.Info("Starting Image Processing Service")

	// Load configuration
	cfg, err := config.LoadConfig(startupLogger)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize configured logger
	appLogger := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	appLogger.Info("Configuration loaded",
		"env", cfg.Env,
		"project_id", cfg.GCP.ProjectID,
		"subscription_id", cfg.PubSubConfig.ImageProcessingSubID,
	)

	// Create context
	ctx := context.Background()

	// Initialize container
	cnt, err := container.New(ctx, cfg, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize container", "error", err)
		log.Fatalf("Failed to initialize container: %v", err)
	}
	defer func() {
		if err := cnt.Close(); err != nil {
			appLogger.Error("Error closing container", "error", err)
		}
	}()

	appLogger.Info("Container initialized successfully")

	ceDataEnv := os.Getenv("CE_DATA")
	if ceDataEnv == "" {
		appLogger.Error("CE_DATA environment variable not found. Job must be triggered by Eventarc.")
		log.Fatal("CE_DATA environment variable not found.")
	}

	decodedCloudEvent, err := base64.StdEncoding.DecodeString(ceDataEnv)
	if err != nil {
		appLogger.Error("Failed to decode CE_DATA (Base64)", "error", err)
		log.Fatalf("Failed to decode CE_DATA: %v", err)
	}

	var payload CloudEventPayload
	if err := json.Unmarshal(decodedCloudEvent, &payload); err != nil {
		appLogger.Error("Failed to unmarshal CloudEvent JSON", "error", err, "data", string(decodedCloudEvent))
		log.Fatalf("Failed to unmarshal CloudEvent JSON: %v", err)
	}

	actualEventData, err := base64.StdEncoding.DecodeString(payload.Message.Data)
	if err != nil {
		appLogger.Error("Failed to decode inner Pub/Sub message data (Base64)", "error", err)
		log.Fatalf("Failed to decode inner Pub/Sub message data: %v", err)
	}

	attributes := payload.Message.Attributes
	if attributes == nil {
		attributes = make(map[string]string)
	}

	appLogger.Info("Calling image processing handler",
		"event_type", attributes["event_type"],
		"image_id", attributes["image_id"],
	)
	err = cnt.ImageHandlerService.HandleImageProcessingRequest(ctx, actualEventData, attributes)
	if err != nil {
		appLogger.Error("Image processing handler failed", "error", err)
		log.Fatalf("Image processing handler failed: %v", err)
	}

	appLogger.Info("Image Processing Job completed successfully")

}
