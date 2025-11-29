package processors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type VipsProcessor struct {
	*BaseProcessor
}

func NewVipsProcessor(logger *slog.Logger) *VipsProcessor {
	processor := &VipsProcessor{
		BaseProcessor: NewBaseProcessor(logger, "vips"),
	}

	// Verify binary at initialization
	if err := processor.VerifyBinary(); err != nil {
		logger.Error("vips binary verification failed", "error", err)
	}

	return processor
}

// CreateThumbnail generates a thumbnail image with specified dimensions and quality
func (p *VipsProcessor) CreateThumbnail(ctx context.Context, inputFilePath, outputFilePath string, width, height, quality int) (*CommandResult, error) {
	// Validate inputs
	if err := p.validateThumbnailInputs(inputFilePath, outputFilePath, width, height, quality); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	if err := p.ensureOutputDirectory(outputFilePath); err != nil {
		return nil, err
	}

	outputWithQuality := fmt.Sprintf("%s[Q=%d]", outputFilePath, quality)

	args := []string{
		"thumbnail",
		inputFilePath,
		outputWithQuality,
		fmt.Sprintf("%d", width),
		"--height", fmt.Sprintf("%d", height),
		"--size", "down",
		"--auto-rotate",
	}

	result, err := p.Execute(ctx, args, 10)

	if err != nil {
		return result, errors.WrapProcessingError(err, "failed to create thumbnail").
			WithContext("input_file", inputFilePath).
			WithContext("output_file", outputFilePath).
			WithContext("width", width).
			WithContext("height", height).
			WithContext("quality", quality)
	}

	// Verify output file was created
	if err := p.verifyOutputFile(outputFilePath); err != nil {
		return result, err
	}

	return result, nil
}

// CreateDZI generates Deep Zoom Image tiles for high-resolution image viewing
func (p *VipsProcessor) CreateDZI(ctx context.Context, inputFilePath, outputDir string, timeoutMinutes int, cfg config.DZIConfig) (*CommandResult, error) {
	// Validate inputs
	if err := p.validateDZIInputs(inputFilePath, outputDir, timeoutMinutes, cfg); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, errors.WrapStorageError(err, "failed to create output directory").
			WithContext("output_dir", outputDir)
	}

	suffixWithQuality := fmt.Sprintf("%s[Q=%d]", cfg.Suffix, cfg.Quality)

	args := []string{
		"dzsave",
		inputFilePath,
		outputDir,
		"--layout", cfg.Layout,
		"--suffix", suffixWithQuality,
		"--tile-size", fmt.Sprintf("%d", cfg.TileSize),
		"--overlap", fmt.Sprintf("%d", cfg.Overlap),
		"--background", "255",
		"--depth", "onetile",
	}

	result, err := p.Execute(ctx, args, timeoutMinutes)

	if err != nil {
		return result, errors.WrapProcessingError(err, "failed to create DZI tiles").
			WithContext("input_file", inputFilePath).
			WithContext("output_dir", outputDir).
			WithContext("tile_size", cfg.TileSize).
			WithContext("layout", cfg.Layout)
	}

	// Verify DZI output was created
	if err := p.verifyDZIOutput(outputDir); err != nil {
		return result, err
	}

	return result, nil
}

func (p *VipsProcessor) validateThumbnailInputs(inputFilePath, outputFilePath string, width, height, quality int) error {
	// Check input file exists
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		return errors.NewValidationError("input file does not exist").
			WithContext("input_file", inputFilePath)
	}

	// Validate dimensions
	if width <= 0 {
		return errors.NewValidationError("width must be positive").
			WithContext("width", width)
	}
	if height <= 0 {
		return errors.NewValidationError("height must be positive").
			WithContext("height", height)
	}

	// Validate quality (JPEG quality range: 1-100)
	if quality < 1 || quality > 100 {
		return errors.NewValidationError("quality must be between 1 and 100").
			WithContext("quality", quality)
	}

	// Check output file extension
	ext := strings.ToLower(filepath.Ext(outputFilePath))
	validExts := []string{".jpg", ".jpeg", ".png", ".webp"}
	isValidExt := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return errors.NewValidationError("output file must have valid image extension (.jpg, .jpeg, .png, .webp)").
			WithContext("output_file", outputFilePath).
			WithContext("extension", ext)
	}

	return nil
}

func (p *VipsProcessor) validateDZIInputs(inputFilePath, outputDir string, timeoutMinutes int, cfg config.DZIConfig) error {
	// Check input file exists
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		return errors.NewValidationError("input file does not exist").
			WithContext("input_file", inputFilePath)
	}

	// Validate timeout
	if timeoutMinutes <= 0 {
		return errors.NewValidationError("timeout must be positive").
			WithContext("timeout_minutes", timeoutMinutes)
	}

	// Validate DZI config
	if cfg.TileSize <= 0 {
		return errors.NewValidationError("tile size must be positive").
			WithContext("tile_size", cfg.TileSize)
	}

	if cfg.Overlap < 0 {
		return errors.NewValidationError("overlap cannot be negative").
			WithContext("overlap", cfg.Overlap)
	}

	if cfg.Quality < 1 || cfg.Quality > 100 {
		return errors.NewValidationError("quality must be between 1 and 100").
			WithContext("quality", cfg.Quality)
	}

	validLayouts := []string{"dz", "google", "zoomify", "iiif"}
	isValidLayout := false
	for _, validLayout := range validLayouts {
		if cfg.Layout == validLayout {
			isValidLayout = true
			break
		}
	}
	if !isValidLayout {
		return errors.NewValidationError("invalid layout, must be one of: dz, google, zoomify, iiif").
			WithContext("layout", cfg.Layout)
	}

	return nil
}

func (p *VipsProcessor) ensureOutputDirectory(outputFilePath string) error {
	outputDir := filepath.Dir(outputFilePath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create output directory").
			WithContext("output_dir", outputDir)
	}
	return nil
}

func (p *VipsProcessor) verifyOutputFile(outputFilePath string) error {
	info, err := os.Stat(outputFilePath)
	if os.IsNotExist(err) {
		return errors.NewProcessingError("output file was not created").
			WithContext("output_file", outputFilePath)
	}
	if err != nil {
		return errors.WrapStorageError(err, "failed to verify output file").
			WithContext("output_file", outputFilePath)
	}
	if info.Size() == 0 {
		return errors.NewProcessingError("output file is empty").
			WithContext("output_file", outputFilePath)
	}
	return nil
}

func (p *VipsProcessor) verifyDZIOutput(outputDir string) error {
	// Check if output directory exists
	info, err := os.Stat(outputDir)
	if os.IsNotExist(err) {
		return errors.NewProcessingError("output directory was not created").
			WithContext("output_dir", outputDir)
	}
	if err != nil {
		return errors.WrapStorageError(err, "failed to verify output directory").
			WithContext("output_dir", outputDir)
	}
	if !info.IsDir() {
		return errors.NewProcessingError("output path is not a directory").
			WithContext("output_dir", outputDir)
	}

	// Check if directory contains any files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return errors.WrapStorageError(err, "failed to read output directory").
			WithContext("output_dir", outputDir)
	}
	if len(entries) == 0 {
		return errors.NewProcessingError("output directory is empty, no tiles were created").
			WithContext("output_dir", outputDir)
	}

	return nil
}

type ImageInfo struct {
	Width  int
	Height int
	Size   int64
}

func (p *VipsProcessor) GetImageInfo(ctx context.Context, inputFilePath string) (*ImageInfo, error) {
	fileInfo, err := os.Stat(inputFilePath)
	if err != nil {
		return nil, err
	}

	widthResult, err := p.Execute(ctx, []string{"-f", "width", inputFilePath}, 1)
	if err != nil {
		return nil, err
	}

	heightResult, err := p.Execute(ctx, []string{"-f", "height", inputFilePath}, 1)
	if err != nil {
		return nil, err
	}

	var width, height int
	fmt.Sscanf(strings.TrimSpace(widthResult.Stdout), "%d", &width)
	fmt.Sscanf(strings.TrimSpace(heightResult.Stdout), "%d", &height)

	return &ImageInfo{
		Width:  width,
		Height: height,
		Size:   fileInfo.Size(),
	}, nil
}
