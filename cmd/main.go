package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/domain/utils"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/container"
	"github.com/histopathai/image-processing-service/pkg/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v. Shutting down gracefully...\n", sig)
		cancel()
	}()

	// Run the job
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Job failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	loggerCfg := logger.Config{
		Level:  os.Getenv("LOG_LEVEL"),
		Format: os.Getenv("LOG_FORMAT"),
	}
	if loggerCfg.Level == "" {
		loggerCfg.Level = "INFO"
	}
	if loggerCfg.Format == "" {
		loggerCfg.Format = "json"
	}

	log := logger.New(loggerCfg)
	log.Info("Starting image processing job")

	cfg, err := config.LoadConfig(log)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := utils.LoadSupportedFormats("./supported_formats.json"); err != nil {
		return fmt.Errorf("failed to load supported formats: %w", err)
	}

	input, err := getJobInput()
	if err != nil {
		return fmt.Errorf("failed to get job input: %w", err)
	}

	log.Info("Job input loaded",
		"image_id", input.ImageID,
		"origin_path", input.OriginPath,
	)

	// Initialize container
	cnt, err := container.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	defer func() {
		if err := cnt.Close(); err != nil {
			log.Error("Failed to close container", "error", err)
		}
	}()

	// Process the image
	if err := cnt.JobOrchestrator.ProcessJob(ctx, input); err != nil {
		return fmt.Errorf("image processing failed: %w", err)
	}

	log.Info("Job completed successfully", "image_id", input.ImageID)
	return nil
}

func getJobInput() (*model.JobInput, error) {
	imageID := os.Getenv("INPUT_IMAGE_ID")
	originPath := os.Getenv("INPUT_ORIGIN_PATH")

	return model.NewJobInputFromEnv(imageID, originPath)
}
