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

	// DZI overrides
	tileSize := flag.Int("tile-size", 0, "DZI Tile Size (default 256 or env TILE_SIZE)")
	overlap := flag.Int("overlap", -1, "DZI Overlap (default 0 or env OVERLAP)")
	quality := flag.Int("quality", 0, "DZI Quality (default 85 or env QUALITY)")
	dziContainer := flag.String("dzi-container", "", "DZI Container format, zip or fs (default zip or env DZI_CONTAINER)")
	dziLayout := flag.String("dzi-layout", "", "DZI Layout (default dz or env DZI_LAYOUT)")
	dziSuffix := flag.String("dzi-suffix", "", "DZI Suffix (default jpg or env DZI_SUFFIX)")
	dziCompression := flag.Int("dzi-compression", -1, "DZI Zip Compression Level 0-9 (default 0 or env DZI_COMPRESSION)")

	// Thumbnail overrides
	thumbnailSize := flag.Int("thumbnail-size", 0, "Thumbnail size (default 256 or env THUMBNAIL_SIZE)")
	thumbnailQuality := flag.Int("thumbnail-quality", 0, "Thumbnail quality (default 90 or env THUMBNAIL_QUALITY)")

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
		opts := CLIOptions{
			InputPath:        *inputPath,
			OutputDir:        *outputDir,
			ImageID:          *imageID,
			Version:          *version,
			LogLevel:         *logLevel,
			LogFormat:        *logFormat,
			TileSize:         *tileSize,
			Overlap:          *overlap,
			Quality:          *quality,
			DZIContainer:     *dziContainer,
			DZILayout:        *dziLayout,
			DZISuffix:        *dziSuffix,
			DZICompression:   *dziCompression,
			ThumbnailSize:    *thumbnailSize,
			ThumbnailQuality: *thumbnailQuality,
		}
		return runCLI(ctx, opts)
	}

	// Legacy env var mode (for Cloud Run Jobs compatibility)
	return runLegacy(ctx, *logLevel, *logFormat)
}

// CLIOptions encapsulates all CLI flag parameters.
type CLIOptions struct {
	InputPath        string
	OutputDir        string
	ImageID          string
	Version          string
	LogLevel         string
	LogFormat        string
	TileSize         int
	Overlap          int
	Quality          int
	DZIContainer     string
	DZILayout        string
	DZISuffix        string
	DZICompression   int
	ThumbnailSize    int
	ThumbnailQuality int
}

func runCLI(ctx context.Context, opts CLIOptions) error {
	// Resolve absolute paths
	absInput, err := filepath.Abs(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve input path: %w", err)
	}

	absOutput, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Validate input file exists
	if _, err := os.Stat(absInput); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", absInput)
	}

	// Derive image ID from filename if not provided
	if opts.ImageID == "" {
		base := filepath.Base(absInput)
		opts.ImageID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Set env vars for config loader (CLI overrides)
	os.Setenv("APP_ENV", "LOCAL")
	os.Setenv("INPUT_MOUNT_PATH", filepath.Dir(absInput))
	os.Setenv("OUTPUT_MOUNT_PATH", absOutput)

	if opts.LogLevel == "" {
		opts.LogLevel = "INFO"
	}
	if opts.LogFormat == "" {
		opts.LogFormat = "text"
	}
	os.Setenv("LOG_LEVEL", opts.LogLevel)
	os.Setenv("LOG_FORMAT", opts.LogFormat)

	// Apply CLI overrides to environment variables ONLY if passed
	if opts.TileSize > 0 {
		os.Setenv("TILE_SIZE", fmt.Sprintf("%d", opts.TileSize))
	}
	if opts.Overlap >= 0 {
		os.Setenv("OVERLAP", fmt.Sprintf("%d", opts.Overlap))
	}
	if opts.Quality > 0 {
		os.Setenv("QUALITY", fmt.Sprintf("%d", opts.Quality))
	}
	if opts.DZIContainer != "" {
		os.Setenv("DZI_CONTAINER", opts.DZIContainer)
	}
	if opts.DZILayout != "" {
		os.Setenv("DZI_LAYOUT", opts.DZILayout)
	}
	if opts.DZISuffix != "" {
		os.Setenv("DZI_SUFFIX", opts.DZISuffix)
	}
	if opts.DZICompression >= 0 {
		os.Setenv("DZI_COMPRESSION", fmt.Sprintf("%d", opts.DZICompression))
	}
	if opts.ThumbnailSize > 0 {
		os.Setenv("THUMBNAIL_SIZE", fmt.Sprintf("%d", opts.ThumbnailSize))
	}
	if opts.ThumbnailQuality > 0 {
		os.Setenv("THUMBNAIL_QUALITY", fmt.Sprintf("%d", opts.ThumbnailQuality))
	}

	log := logger.New(logger.Config{
		Level:  opts.LogLevel,
		Format: opts.LogFormat,
	})

	log.Info("Starting himgproc",
		"input", absInput,
		"output", absOutput,
		"image_id", opts.ImageID,
		"version", opts.Version,
	)

	cfg, err := config.LoadConfig(log)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := utils.LoadSupportedFormats(); err != nil {
		return fmt.Errorf("failed to load supported formats from embed: %w", err)
	}

	input, err := model.NewJobInput(opts.ImageID, filepath.Base(absInput), opts.Version)
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

	log.Info("Job completed successfully", "image_id", opts.ImageID)
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

	if err := utils.LoadSupportedFormats(); err != nil {
		return fmt.Errorf("failed to load supported formats from embed: %w", err)
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
