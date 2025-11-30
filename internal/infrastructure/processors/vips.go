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

func (p *VipsProcessor) CreateDZI(ctx context.Context, inputFilePath, outputBase string, timeoutMinutes int, cfg config.DZIConfig) (*CommandResult, error) {
	// Validate inputs
	if err := p.validateDZIInputs(inputFilePath, outputBase, timeoutMinutes, cfg); err != nil {
		return nil, err
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputBase)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, errors.WrapStorageError(err, "failed to create output directory").
			WithContext("output_dir", outputDir)
	}

	suffixWithQuality := fmt.Sprintf(".%s[Q=%d]", cfg.Suffix, cfg.Quality)

	args := []string{
		"dzsave",
		inputFilePath,
		outputBase, // vips dzsave uses base name without extension
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
			WithContext("output_base", outputBase).
			WithContext("tile_size", cfg.TileSize).
			WithContext("layout", cfg.Layout)
	}

	// Verify DZI output
	dziFilesDir := outputBase + "_files"
	if err := p.verifyDZIOutput(dziFilesDir); err != nil {
		return result, err
	}

	return result, nil
}

func (p *VipsProcessor) verifyDZIOutput(dziFilesDir string) error {
	// Check if _files directory exists
	info, err := os.Stat(dziFilesDir)
	if os.IsNotExist(err) {
		return errors.NewProcessingError("DZI files directory was not created").
			WithContext("dzi_files_dir", dziFilesDir)
	}
	if err != nil {
		return errors.WrapStorageError(err, "failed to verify DZI files directory").
			WithContext("dzi_files_dir", dziFilesDir)
	}
	if !info.IsDir() {
		return errors.NewProcessingError("DZI files path is not a directory").
			WithContext("dzi_files_dir", dziFilesDir)
	}

	entries, err := os.ReadDir(dziFilesDir)
	if err != nil {
		return errors.WrapStorageError(err, "failed to read DZI files directory").
			WithContext("dzi_files_dir", dziFilesDir)
	}
	if len(entries) == 0 {
		return errors.NewProcessingError("DZI files directory is empty, no tiles were created").
			WithContext("dzi_files_dir", dziFilesDir)
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
