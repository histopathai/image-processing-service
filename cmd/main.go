package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/container"
	"github.com/histopathai/image-processing-service/pkg/logger"
)

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

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start subscriber in a goroutine
	errChan := make(chan error, 1)
	go func() {
		appLogger.Info("Starting subscriber",
			"subscription_id", cfg.PubSubConfig.ImageProcessingSubID,
		)

		err := cnt.Subscriber.Subscribe(
			ctx,
			cfg.PubSubConfig.ImageProcessingSubID,
			cnt.ImageHandlerService.HandleImageProcessingRequest,
		)
		if err != nil {
			appLogger.Error("Subscriber error", "error", err)
			errChan <- err
		}
	}()

	appLogger.Info("Image Processing Service is running",
		"subscription_id", cfg.PubSubConfig.ImageProcessingSubID,
		"topic_id", cfg.PubSubConfig.ImageProcessResultTopicID,
	)

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		appLogger.Info("Received shutdown signal", "signal", sig)
	case err := <-errChan:
		appLogger.Error("Service error", "error", err)
	}

	// Graceful shutdown
	appLogger.Info("Shutting down gracefully...")

	// Stop subscriber
	if err := cnt.Subscriber.Stop(); err != nil {
		appLogger.Error("Error stopping subscriber", "error", err)
	}

	// Give some time for in-flight messages to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()

	appLogger.Info("Image Processing Service stopped")
}
