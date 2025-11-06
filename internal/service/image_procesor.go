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

	ip.logger.Info("Starting DZI processing",
		"file_path", file.Path,
		"file_id", file.ID,
		"output_base", outputPathBase,
	)

	err := ip.vipsDZIProcessor(ctx, file.Path, outputPathBase)
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

func (ip *ImageProcessor) vipsDZIProcessor(ctx context.Context, inputPath string, outputPathBase string) error {
	args := []string{
		"dzsave",
		inputPath,
		outputPathBase,
		"--layout", ip.cfg.Layout,
		"--suffix", ip.cfg.Suffix,
		"--tile-size", fmt.Sprintf("%d", ip.cfg.TileSize),
		"--overlap", fmt.Sprintf("%d", ip.cfg.Overlap),
		"--Q", fmt.Sprintf("%d", ip.cfg.Quality),
		"--background", "255",
		"--depth", "onetile",
	}

	cmd := exec.CommandContext(ctx, "vips", args...)

	ip.logger.Debug("Executing command", "command", "vips "+strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.NewInternalError("vips dzisave failed").WithContext("error", fmt.Sprintf("%s | Output: %s", err.Error(), string(output)))
	}

	return nil
}
