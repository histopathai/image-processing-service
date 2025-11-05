package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/histopathai/image-processing-service/internal/domain/model"
)

type DZIProcessorConfig struct {
	TileSize int64
	Overlap  int64
	Layout   string
	Quality  int64
	Suffix   string
}

type ImageProcessor struct {
	logger *slog.Logger
}

func NewImageProcessor(logger *slog.Logger) *ImageProcessor {
	return &ImageProcessor{
		logger: logger,
	}
}

func (ip *ImageProcessor) GetImageInfo(ctx context.Context, file *model.File) error {
	ip.logger.Debug("Getting image info", "file_path", file.Path)

	info, err := os.Stat(file.Path)
	if err != nil {
		ip.logger.Error("Failed to stat file", "file_path", file.Path, "error", err)
		return fmt.Errorf("could not stat file %s: %w", file.Path, err)
	}
	size := info.Size()
	file.Size = &size

	ext := filepath.Ext(file.Path)
	standardFormat := strings.ToLower(strings.TrimPrefix(ext, "."))
	if !model.SupportedFormats.IsSupported(standardFormat) {
		ip.logger.Warn("File format may not be supported", "format", standardFormat)
	}
	file.Format = &standardFormat

	widthStr, err := ip.runVipsHeaderField(ctx, file.Path, "width")
	if err != nil {
		ip.logger.Error("Failed to get image width", "file_path", file.Path, "error", err)
		return err
	}
	width, err := strconv.Atoi(widthStr)
	if err != nil {
		return fmt.Errorf("could not parse width '%s': %w", widthStr, err)
	}
	file.Width = &width

	heightStr, err := ip.runVipsHeaderField(ctx, file.Path, "height")
	if err != nil {
		ip.logger.Error("Failed to get image height", "file_path", file.Path, "error", err)
		return err
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		return fmt.Errorf("could not parse height '%s': %w", heightStr, err)
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

func (ip *ImageProcessor) runVipsHeaderField(ctx context.Context, inputPath string, fieldName string) (string, error) {
	args := []string{
		"vipsheader",
		"-f",
		fieldName,
		inputPath,
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vipsheader failed (field: %s): %w | Output: %s", fieldName, err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func (ip *ImageProcessor) ExtractThumbnail(ctx context.Context, inputPath, outputPath string, width, height int, quality int) error {
	sizeStr := fmt.Sprintf("%d", width)
	heightStr := fmt.Sprintf("%d", height)
	qualityStr := fmt.Sprintf("%d", quality)

	args := []string{
		"thumbnail",
		inputPath,
		outputPath,
		sizeStr,
		"--height", heightStr,
		"--size", "down",
		"--auto-rotate",
		"--Q", qualityStr,
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	ip.logger.Debug("Executing command", "command", "vips "+strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vips thumbnail failed: %w | Output: %s", err, string(output))
	}

	ip.logger.Info("Successfully created thumbnail",
		"input", inputPath,
		"output", outputPath,
		"size", fmt.Sprintf("%dx%d", width, height),
	)

	return nil
}

func (ip *ImageProcessor) DZIProcessor(ctx context.Context, file *model.File, cfg DZIProcessorConfig) error {
	if file.ProcessedPath == nil {
		err := fmt.Errorf("ProcessedPath is nil for file ID: %s", file.ID)
		ip.logger.Error("Cannot start DZI processing", "error", err)
		return err
	}

	outputPathBase := *file.ProcessedPath

	ip.logger.Info("Starting DZI processing",
		"file_path", file.Path,
		"file_id", file.ID,
		"output_base", outputPathBase,
	)

	err := ip.vipsDZIProcessor(ctx, file.Path, outputPathBase, cfg)
	if err != nil {
		ip.logger.Error("FAILED DZI processing",
			"file_path", file.Path,
			"file_id", file.ID,
			"error", err,
		)
		return err
	}

	ip.logger.Info("Successfully processed DZI",
		"file_path", file.Path,
		"output_base", outputPathBase,
		"dzi_file", outputPathBase+".dzi",
	)
	return nil
}

func (ip *ImageProcessor) vipsDZIProcessor(ctx context.Context, inputPath string, outputPathBase string, cfg DZIProcessorConfig) error {
	args := []string{
		"dzsave",
		inputPath,
		outputPathBase,
		"--layout", cfg.Layout,
		"--suffix", cfg.Suffix,
		"--tile-size", fmt.Sprintf("%d", cfg.TileSize),
		"--overlap", fmt.Sprintf("%d", cfg.Overlap),
		"--Q", fmt.Sprintf("%d", cfg.Quality),
		"--background", "255",
		"--depth", "onetile",
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	ip.logger.Debug("Executing command", "command", "vips "+strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vips dzsave failed: %w | Output: %s", err, string(output))
	}

	return nil
}
