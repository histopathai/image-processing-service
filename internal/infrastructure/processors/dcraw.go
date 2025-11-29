package processors

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

type DcrawProcessor struct {
	*BaseProcessor
}

func NewDcrawProcessor(logger *slog.Logger) *DcrawProcessor {
	processor := &DcrawProcessor{
		BaseProcessor: NewBaseProcessor(logger, "dcraw"),
	}

	// Verify binary at initialization
	if err := processor.VerifyBinary(); err != nil {
		logger.Error("dcraw binary verification failed", "error", err)
	}

	return processor
}

// DNGToTIFF converts a DNG file to TIFF format
func (p *DcrawProcessor) DNGToTIFF(ctx context.Context, inputFilePath, outputFilePath string, timeoutMinutes int) (*CommandResult, error) {
	// Validate inputs
	if err := p.validateDNGToTIFFInputs(inputFilePath, outputFilePath, timeoutMinutes); err != nil {
		return nil, err
	}

	// Ensure output directory exists
	if err := p.ensureOutputDirectory(outputFilePath); err != nil {
		return nil, err
	}

	// Build command arguments
	args := []string{
		"-c",      // Write to stdout
		"-T",      // Output TIFF
		"-4",      // Linear 16-bit
		"-q", "3", // AHD interpolation (high-quality)
		"-w",      // Camera white balance
		"-H", "0", // No highlight clipping
		"-o", "1", // sRGB color space
		inputFilePath,
	}

	result, err := p.ExecuteToFile(ctx, args, outputFilePath, timeoutMinutes)

	// Add specific context for DNG conversion errors
	if err != nil {
		return result, errors.WrapProcessingError(err, "failed to convert DNG to TIFF").
			WithContext("input_file", inputFilePath).
			WithContext("output_file", outputFilePath)
	}

	// Verify output file was created and has content
	if err := p.verifyOutputFile(outputFilePath); err != nil {
		return result, err
	}

	return result, nil
}

func (p *DcrawProcessor) validateDNGToTIFFInputs(inputFilePath, outputFilePath string, timeoutMinutes int) error {
	// Check input file exists
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		return errors.NewValidationError("input file does not exist").
			WithContext("input_file", inputFilePath)
	}

	// Check input file extension
	ext := filepath.Ext(inputFilePath)
	if ext != ".dng" && ext != ".DNG" {
		return errors.NewValidationError("input file must be a DNG file").
			WithContext("input_file", inputFilePath).
			WithContext("extension", ext)
	}

	// Check output file extension
	outputExt := filepath.Ext(outputFilePath)
	if outputExt != ".tif" && outputExt != ".tiff" && outputExt != ".TIF" && outputExt != ".TIFF" {
		return errors.NewValidationError("output file must have .tif or .tiff extension").
			WithContext("output_file", outputFilePath).
			WithContext("extension", outputExt)
	}

	// Validate timeout
	if timeoutMinutes <= 0 {
		return errors.NewValidationError("timeout must be positive").
			WithContext("timeout_minutes", timeoutMinutes)
	}

	return nil
}

func (p *DcrawProcessor) ensureOutputDirectory(outputFilePath string) error {
	outputDir := filepath.Dir(outputFilePath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create output directory").
			WithContext("output_dir", outputDir)
	}
	return nil
}

func (p *DcrawProcessor) verifyOutputFile(outputFilePath string) error {
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
