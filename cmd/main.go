package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	// Parse CLI flags
	inputPath := flag.String("input", "", "Path to input image file (required)")
	flag.StringVar(inputPath, "i", "", "Path to input image file (shorthand)")

	outputDir := flag.String("output", "./output", "Output directory for processed files")
	flag.StringVar(outputDir, "o", "./output", "Output directory (shorthand)")

	imageID := flag.String("image-id", "", "Image ID (optional, derived from filename if omitted)")
	version := flag.String("version", "v2", "Processing version (v1 or v2)")
	logLevel := flag.String("log-level", "", "Log level (DEBUG, INFO, WARN, ERROR)")
	logFormat := flag.String("log-format", "", "Log format (text or json)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: himgproc [options]\n\n")
		fmt.Fprintf(os.Stderr, "Process medical whole slide images locally.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  himgproc -i ./image.svs -o ./output\n")
		fmt.Fprintf(os.Stderr, "  himgproc --input ./image.png --image-id my-img-001 --version v2\n")
	}

	flag.Parse()

	// Determine if running in CLI mode (flags provided) or env var mode (legacy)
	cliMode := *inputPath != ""

	if cliMode {
		return runCLI(ctx, *inputPath, *outputDir, *imageID, *version, *logLevel, *logFormat)
	}

	// Legacy env var mode (for Cloud Run Jobs compatibility)
	return runLegacy(ctx, *logLevel, *logFormat)
}

func runCLI(ctx context.Context, inputPath, outputDir, imageID, version, logLevel, logFormat string) error {
	// Resolve absolute paths
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve input path: %w", err)
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Validate input file exists
	if _, err := os.Stat(absInput); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", absInput)
	}

	// Derive image ID from filename if not provided
	if imageID == "" {
		base := filepath.Base(absInput)
		imageID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Set env vars for config loader (CLI overrides)
	os.Setenv("APP_ENV", "LOCAL")
	os.Setenv("INPUT_MOUNT_PATH", filepath.Dir(absInput))
	os.Setenv("OUTPUT_MOUNT_PATH", absOutput)

	if logLevel == "" {
		logLevel = "INFO"
	}
	if logFormat == "" {
		logFormat = "text"
	}
	os.Setenv("LOG_LEVEL", logLevel)
	os.Setenv("LOG_FORMAT", logFormat)

	// DZI defaults for local
	setEnvDefault("DZI_CONTAINER", "zip")
	setEnvDefault("DZI_COMPRESSION", "0")
	setEnvDefault("TILE_SIZE", "256")
	setEnvDefault("OVERLAP", "0")
	setEnvDefault("QUALITY", "85")
	setEnvDefault("DZI_LAYOUT", "dz")
	setEnvDefault("DZI_SUFFIX", "jpg")
	setEnvDefault("THUMBNAIL_SIZE", "256")
	setEnvDefault("THUMBNAIL_QUALITY", "90")

	log := logger.New(logger.Config{
		Level:  logLevel,
		Format: logFormat,
	})

	log.Info("Starting himgproc",
		"input", absInput,
		"output", absOutput,
		"image_id", imageID,
		"version", version,
	)

	cfg, err := config.LoadConfig(log)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := utils.LoadSupportedFormats("./supported_formats.json"); err != nil {
		// Try from binary's directory
		execPath, _ := os.Executable()
		altPath := filepath.Join(filepath.Dir(execPath), "supported_formats.json")
		if err2 := utils.LoadSupportedFormats(altPath); err2 != nil {
			return fmt.Errorf("failed to load supported formats: %w (also tried %s: %v)", err, altPath, err2)
		}
	}

	input, err := model.NewJobInput(imageID, filepath.Base(absInput), version)
	if err != nil {
		return fmt.Errorf("failed to create job input: %w", err)
	}

	cnt, err := container.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	defer func() {
		if err := cnt.Close(); err != nil {
			log.Error("Failed to close container", "error", err)
		}
	}()

	if err := cnt.JobOrchestrator.ProcessJob(ctx, input); err != nil {
		return fmt.Errorf("image processing failed: %w", err)
	}

	log.Info("Job completed successfully", "image_id", imageID)
	return nil
}

func runLegacy(ctx context.Context, logLevel, logFormat string) error {
	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}
	if logLevel == "" {
		logLevel = "INFO"
	}
	if logFormat == "" {
		logFormat = os.Getenv("LOG_FORMAT")
	}
	if logFormat == "" {
		logFormat = "json"
	}

	log := logger.New(logger.Config{
		Level:  logLevel,
		Format: logFormat,
	})
	log.Info("Starting image processing job (legacy env var mode)")

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

	cnt, err := container.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize container: %w", err)
	}
	defer func() {
		if err := cnt.Close(); err != nil {
			log.Error("Failed to close container", "error", err)
		}
	}()

	if err := cnt.JobOrchestrator.ProcessJob(ctx, input); err != nil {
		return fmt.Errorf("image processing failed: %w", err)
	}

	log.Info("Job completed successfully", "image_id", input.ImageID)
	return nil
}

func getJobInput() (*model.JobInput, error) {
	imageID := os.Getenv("INPUT_IMAGE_ID")
	originPath := os.Getenv("INPUT_ORIGIN_PATH")
	processingVersion := os.Getenv("INPUT_PROCESSING_VERSION")
	bucketName := os.Getenv("INPUT_BUCKET_NAME")

	return model.NewJobInputFromEnv(imageID, originPath, processingVersion, bucketName)
}

func setEnvDefault(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}
