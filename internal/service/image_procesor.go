package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageProcessor struct {
	logger *slog.Logger
	cfg    config.DZIConfig
}

func NewImageProcessor(logger *slog.Logger, cfg config.DZIConfig) *ImageProcessor {
	return &ImageProcessor{
		logger: logger,
		cfg:    cfg,
	}
}

func (ip *ImageProcessor) GetImageInfo(ctx context.Context, file *model.File) error {
	ip.logger.Debug("Getting image info", "file_path", file.Path)

	info, err := os.Stat(file.Path)
	if err != nil {
		ip.logger.Error("Failed to stat file", "file_path", file.Path, "error", err)
		return errors.NewNotFoundError("file not found").WithContext("file_path", file.Path)
	}
	size := info.Size()
	file.Size = &size

	ext := filepath.Ext(file.Path)
	standardFormat := strings.ToLower(strings.TrimPrefix(ext, "."))
	if !model.SupportedFormats.IsSupported(standardFormat) {
		ip.logger.Warn("File format may not be supported", "format", standardFormat)
	}
	file.Format = &standardFormat

	// For DNG files, we need to get dimensions from the raw file
	if standardFormat == "dng" {
		return ip.getDNGInfo(ctx, file)
	}

	widthStr, err := ip.runVipsHeaderField(ctx, file.Path, "width")
	if err != nil {
		ip.logger.Error("Failed to get image width", "file_path", file.Path, "error", err)
		return err
	}
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		return errors.NewValidationError("invalid image width").WithContext("width", widthStr)
	}
	file.Width = &width

	heightStr, err := ip.runVipsHeaderField(ctx, file.Path, "height")
	if err != nil {
		ip.logger.Error("Failed to get image height", "file_path", file.Path, "error", err)
		return err
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		return errors.NewValidationError("invalid image height").WithContext("height", heightStr)
	}
	file.Height = &height

	ip.logger.Info("Successfully retrieved image info",
		"file_path", file.Path,
		"width", *file.Width,
		"height", *file.Height,
		"size", *file.Size,
		"format", *file.Format,
	)

	return nil
}

func (ip *ImageProcessor) getDNGInfo(ctx context.Context, file *model.File) error {
	ip.logger.Debug("Getting DNG file info using dcraw", "file_path", file.Path)

	// Use dcraw -i -v to get image info
	cmd := exec.CommandContext(ctx, "dcraw", "-i", "-v", file.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		ip.logger.Error("Failed to get DNG info", "error", err, "output", string(output))
		return errors.NewProcessingError("failed to read DNG metadata")
	}

	// Parse dcraw output to extract dimensions
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Image size:") {
			// Expected format: "Image size:  4032 x 3024"
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				width, err := strconv.Atoi(parts[2])
				if err == nil {
					file.Width = &width
				}
				height, err := strconv.Atoi(parts[4])
				if err == nil {
					file.Height = &height
				}
			}
		} else if strings.Contains(line, "Full size:") {
			// Alternative: "Full size:  4032 x 3024"
			parts := strings.Fields(line)
			if len(parts) >= 5 && file.Width == nil {
				width, err := strconv.Atoi(parts[2])
				if err == nil {
					file.Width = &width
				}
				height, err := strconv.Atoi(parts[4])
				if err == nil {
					file.Height = &height
				}
			}
		}
	}

	if file.Width == nil || file.Height == nil {
		ip.logger.Error("Could not extract dimensions from DNG file")
		return errors.NewProcessingError("failed to extract DNG dimensions")
	}

	ip.logger.Info("Successfully retrieved DNG info",
		"file_path", file.Path,
		"width", *file.Width,
		"height", *file.Height,
	)

	return nil
}

func (ip *ImageProcessor) ConvertDNGToTIFF(ctx context.Context, dngPath string, outputTiffPath string) error {
	ip.logger.Info("Converting DNG to TIFF",
		"input", dngPath,
		"output", outputTiffPath,
	)

	// dcraw options for high-quality lossless conversion:
	// -T: Write TIFF instead of PPM
	// -4: Linear 16-bit (more depth)
	// -q 3: Use high-quality interpolation (AHD)
	// -w: Use camera white balance
	// -H 0: No highlight recovery (preserve all data)
	// -o 1: Output in sRGB color space
	// -c: Write to stdout (we'll redirect to file)
	args := []string{
		"-T",      // Output TIFF
		"-4",      // 16-bit linear
		"-q", "3", // AHD interpolation (high quality)
		"-w",      // Camera white balance
		"-H", "0", // No highlight clipping
		"-o", "1", // sRGB color space
		"-c", // Write to stdout
		dngPath,
	}

	cmd := exec.CommandContext(ctx, "dcraw", args...)

	// Create output file
	outFile, err := os.Create(outputTiffPath)
	if err != nil {
		return errors.NewProcessingError("failed to create output TIFF file").
			WithContext("path", outputTiffPath)
	}
	defer outFile.Close()

	cmd.Stdout = outFile

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.NewProcessingError("failed to create stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.NewProcessingError("failed to start dcraw conversion").
			WithContext("error", err.Error())
	}

	// Read stderr for error messages
	stderrOutput, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		ip.logger.Error("dcraw conversion failed",
			"error", err,
			"stderr", string(stderrOutput),
		)
		return errors.NewProcessingError("dcraw conversion failed").
			WithContext("error", err.Error()).
			WithContext("stderr", string(stderrOutput))
	}

	// Verify the output file was created
	if info, err := os.Stat(outputTiffPath); err != nil || info.Size() == 0 {
		return errors.NewProcessingError("dcraw produced empty or invalid output")
	}

	ip.logger.Info("Successfully converted DNG to TIFF",
		"input", dngPath,
		"output", outputTiffPath,
	)

	return nil
}

func (ip *ImageProcessor) runVipsHeaderField(ctx context.Context, inputPath string, fieldName string) (string, error) {

	args := []string{
		"-f",
		fieldName,
		inputPath,
	}

	ip.logger.Debug("Running vipsheader", "command", "vipsheader "+strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "vipsheader", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		vipsErrOutput := string(output)
		ip.logger.Error("vipsheader command failed",
			"file_path", inputPath,
			"field", fieldName,
			"error", err,
			"vips_output", vipsErrOutput,
		)
		return "", errors.NewInternalError("vips header failed").WithContext("error", fmt.Sprintf("%s | Output: %s", err.Error(), vipsErrOutput))
	}

	return strings.TrimSpace(string(output)), nil
}

func (ip *ImageProcessor) ExtractThumbnail(ctx context.Context, inputPath, outputPath string, width, height int, quality int) error {
	sizeStr := fmt.Sprintf("%d", width)
	heightStr := fmt.Sprintf("%d", height)

	outputWithQuality := fmt.Sprintf("%s[Q=%d]", outputPath, quality)

	args := []string{
		"thumbnail",
		inputPath,
		outputWithQuality,
		sizeStr,
		"--height", heightStr,
		"--size", "down",
		"--auto-rotate",
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	ip.logger.Debug("Executing command", "command", "vips "+strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewInternalError("vips thumbnail failed").WithContext("error", fmt.Sprintf("%s | Output: %s", err.Error(), string(output)))
	}

	ip.logger.Info("Successfully created thumbnail",
		"input", inputPath,
		"output", outputPath,
		"size", fmt.Sprintf("%dx%d", width, height),
	)

	return nil
}

func (ip *ImageProcessor) DZIProcessor(ctx context.Context, file *model.File) error {
	if file.ProcessedPath == nil {
		ip.logger.Error("Cannot start DZI processing")
		return errors.NewValidationError("processed path is not set").WithContext("file_id", file.ID)
	}

	outputPathBase := *file.ProcessedPath
	inputPath := file.Path

	// Check if this is a DNG file
	if file.Format != nil && strings.ToLower(*file.Format) == "dng" {
		ip.logger.Info("Detected DNG file, converting to TIFF first",
			"file_path", file.Path,
			"file_id", file.ID,
		)

		// Create temporary TIFF file path
		tempTiffPath := filepath.Join(filepath.Dir(outputPathBase), file.ID+"_converted.tiff")

		// Convert DNG to TIFF
		if err := ip.ConvertDNGToTIFF(ctx, file.Path, tempTiffPath); err != nil {
			ip.logger.Error("Failed to convert DNG to TIFF", "error", err)
			return err
		}

		// Update input path to use the converted TIFF
		inputPath = tempTiffPath
		defer os.Remove(tempTiffPath) // Clean up after processing

		ip.logger.Info("DNG converted to TIFF, proceeding with DZI processing",
			"tiff_path", tempTiffPath,
		)
	}

	ip.logger.Info("Starting DZI processing",
		"file_path", inputPath,
		"file_id", file.ID,
		"output_base", outputPathBase,
	)

	err := ip.vipsDZIProcessor(ctx, inputPath, outputPathBase)
	if err != nil {
		ip.logger.Error("FAILED DZI processing",
			"file_path", inputPath,
			"file_id", file.ID,
			"error", err,
		)
		return err
	}

	ip.logger.Info("Successfully processed DZI",
		"file_path", inputPath,
		"output_base", outputPathBase,
		"dzi_file", outputPathBase+".dzi",
	)
	return nil
}

func (ip *ImageProcessor) vipsDZIProcessor(ctx context.Context, inputPath string, outputPathBase string) error {
	suffixWithQuality := fmt.Sprintf("%s[Q=%d]", ip.cfg.Suffix, ip.cfg.Quality)

	args := []string{
		"dzsave",
		inputPath,
		outputPathBase,
		"--layout", ip.cfg.Layout,
		"--suffix", suffixWithQuality,
		"--tile-size", fmt.Sprintf("%d", ip.cfg.TileSize),
		"--overlap", fmt.Sprintf("%d", ip.cfg.Overlap),
		"--background", "255",
		"--depth", "onetile",
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	ip.logger.Debug("Executing command", "command", "vips "+strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		vipsErrOutput := string(output)
		ip.logger.Error("vips dzsave command failed",
			"file_path", inputPath,
			"output_base", outputPathBase,
			"error", err,
			"vips_output", vipsErrOutput,
		)
		return errors.NewInternalError("vips dzisave failed").WithContext("error", fmt.Sprintf("%s | Output: %s", err.Error(), vipsErrOutput))
	}

	return nil
}
