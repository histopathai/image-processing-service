package processors

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageInfo struct {
	Width  int
	Height int
	Size   int64
}

type ImageInfoProcessor struct {
	logger *slog.Logger
}

func NewImageInfoProcessor(logger *slog.Logger) *ImageInfoProcessor {
	return &ImageInfoProcessor{
		logger: logger,
	}
}

func (p *ImageInfoProcessor) GetImageInfo(ctx context.Context, inputFilePath string) (*ImageInfo, error) {
	fileInfo, err := os.Stat(inputFilePath)
	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to stat file").
			WithContext("file", inputFilePath)
	}

	ext := strings.ToLower(filepath.Ext(inputFilePath))

	switch ext {
	case ".dng":
		p.logger.Info("Detected RAW format, using ExifTool for dimensions",
			"file", inputFilePath,
			"format", ext)
		return p.getDimensionsWithExifTool(ctx, inputFilePath, fileInfo.Size())

	case ".ndpi", ".svs", ".scn", ".bif", ".vms", ".vmu":
		p.logger.Info("Detected whole slide image format, using OpenSlide for dimensions",
			"file", inputFilePath,
			"format", ext)
		return p.getDimensionsWithOpenSlide(ctx, inputFilePath, fileInfo.Size())

	default:
		p.logger.Info("Using vipsheader for dimensions",
			"file", inputFilePath,
			"format", ext)
		return p.getDimensionsWithVips(ctx, inputFilePath, fileInfo.Size())
	}
}

func (p *ImageInfoProcessor) getDimensionsWithOpenSlide(ctx context.Context, inputFilePath string, size int64) (*ImageInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "openslide-show-properties", inputFilePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		p.logger.Error("openslide-show-properties failed",
			"file", inputFilePath,
			"stderr", stderr.String(),
			"error", err)
		return nil, errors.WrapProcessingError(err, "failed to get dimensions with OpenSlide").
			WithContext("file", inputFilePath).
			WithContext("stderr", stderr.String())
	}

	output := stdout.String()

	// OpenSlide properties format:
	// openslide.level[0].width: 46000
	// openslide.level[0].height: 32914

	var width, height int

	// Extract width
	widthRegex := regexp.MustCompile(`openslide\.level\[0\]\.width:\s*(\d+)`)
	if matches := widthRegex.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &width)
	}

	// Extract height
	heightRegex := regexp.MustCompile(`openslide\.level\[0\]\.height:\s*(\d+)`)
	if matches := heightRegex.FindStringSubmatch(output); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &height)
	}

	if width == 0 || height == 0 {
		p.logger.Error("Failed to parse dimensions from OpenSlide output",
			"file", inputFilePath,
			"width", width,
			"height", height)
		return nil, errors.NewProcessingError("invalid dimensions detected from OpenSlide").
			WithContext("file", inputFilePath).
			WithContext("width", width).
			WithContext("height", height)
	}

	p.logger.Info("Successfully extracted dimensions with OpenSlide",
		"file", inputFilePath,
		"width", width,
		"height", height,
		"size", size)

	return &ImageInfo{
		Width:  width,
		Height: height,
		Size:   size,
	}, nil
}

func (p *ImageInfoProcessor) getDimensionsWithExifTool(ctx context.Context, inputFilePath string, size int64) (*ImageInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []string{"-ImageWidth", "-ImageHeight", "-s3", "-n", inputFilePath}
	cmd := exec.CommandContext(ctx, "exiftool", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		p.logger.Error("exiftool failed",
			"file", inputFilePath,
			"stderr", stderr.String(),
			"error", err)
		return nil, errors.WrapProcessingError(err, "failed to get dimensions with ExifTool").
			WithContext("file", inputFilePath).
			WithContext("stderr", stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	lines := strings.Split(output, "\n")

	if len(lines) < 2 {
		p.logger.Error("exiftool returned incomplete data",
			"file", inputFilePath,
			"output", output)
		return nil, errors.NewProcessingError("unexpected output from exiftool").
			WithContext("file", inputFilePath).
			WithContext("output", output)
	}

	var width, height int
	fmt.Sscanf(strings.TrimSpace(lines[0]), "%d", &width)
	fmt.Sscanf(strings.TrimSpace(lines[1]), "%d", &height)

	if width == 0 || height == 0 {
		p.logger.Error("Failed to parse dimensions from exiftool",
			"file", inputFilePath,
			"width", width,
			"height", height)
		return nil, errors.NewProcessingError("invalid dimensions detected from exiftool").
			WithContext("file", inputFilePath).
			WithContext("width", width).
			WithContext("height", height)
	}

	p.logger.Info("Successfully extracted dimensions with ExifTool",
		"file", inputFilePath,
		"width", width,
		"height", height,
		"size", size)

	return &ImageInfo{
		Width:  width,
		Height: height,
		Size:   size,
	}, nil
}

func (p *ImageInfoProcessor) getDimensionsWithVips(ctx context.Context, inputFilePath string, size int64) (*ImageInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get width
	widthCmd := exec.CommandContext(ctx, "vipsheader", "-f", "width", inputFilePath)
	var widthOut, widthErr bytes.Buffer
	widthCmd.Stdout = &widthOut
	widthCmd.Stderr = &widthErr

	if err := widthCmd.Run(); err != nil {
		p.logger.Error("vipsheader width lookup failed",
			"file", inputFilePath,
			"stderr", widthErr.String(),
			"error", err)
		return nil, errors.WrapProcessingError(err, "failed to get width with vipsheader").
			WithContext("file", inputFilePath).
			WithContext("stderr", widthErr.String())
	}

	// Get height
	heightCmd := exec.CommandContext(ctx, "vipsheader", "-f", "height", inputFilePath)
	var heightOut, heightErr bytes.Buffer
	heightCmd.Stdout = &heightOut
	heightCmd.Stderr = &heightErr

	if err := heightCmd.Run(); err != nil {
		p.logger.Error("vipsheader height lookup failed",
			"file", inputFilePath,
			"stderr", heightErr.String(),
			"error", err)
		return nil, errors.WrapProcessingError(err, "failed to get height with vipsheader").
			WithContext("file", inputFilePath).
			WithContext("stderr", heightErr.String())
	}

	var width, height int
	fmt.Sscanf(strings.TrimSpace(widthOut.String()), "%d", &width)
	fmt.Sscanf(strings.TrimSpace(heightOut.String()), "%d", &height)

	if width == 0 || height == 0 {
		p.logger.Error("Failed to parse dimensions from vipsheader",
			"file", inputFilePath,
			"width", width,
			"height", height)
		return nil, errors.NewProcessingError("invalid dimensions detected from vipsheader").
			WithContext("file", inputFilePath).
			WithContext("width", width).
			WithContext("height", height)
	}

	p.logger.Info("Successfully extracted dimensions with vipsheader",
		"file", inputFilePath,
		"width", width,
		"height", height,
		"size", size)

	return &ImageInfo{
		Width:  width,
		Height: height,
		Size:   size,
	}, nil
}
