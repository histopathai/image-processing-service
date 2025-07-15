package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/histopathai/image-processing-service/config"
	"github.com/histopathai/image-processing-service/internal/models"
)

type File struct {
	FilePath string
	ext      string
}

func NewFile(filePath string) (*File, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext != "svs" && ext != "tif" && ext != "tiff" && ext != "jpg" && ext != "jpeg" && ext != "png" {
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	return &File{
		FilePath: filePath,
		ext:      ext,
	}, nil
}

func (f *File) Ext() string {
	return f.ext
}

func (f *File) FileInfo() (*models.ImageInfo, error) {

	if _, err := os.Stat(f.FilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", f.FilePath)
	}

	if f.ext == "svs" {
		return getSVSInfo(f.FilePath)
	} else {
		return getVIPSInfo(f.FilePath)
	}

}
func (f *File) ExportThumbnail(outputPath string, thumbSize int) error {
	if f.ext == "svs" {
		return exportSVSThumbnail(f.FilePath, outputPath, thumbSize)
	} else {
		return exportVIPSThumbnail(f.FilePath, outputPath, thumbSize)
	}
}

func (f *File) ExtractDZI(outputPath string, cfg *config.Config) error {
	params := cfg.Parameters

	tileSize := params.TileSize
	overlap := params.Overlap
	quality := params.Quality
	suffix := params.Suffix
	layout := params.Layout

	if tileSize <= 0 {
		return errors.New("tile_size must be a positive integer")
	}

	if overlap < 0 {
		return errors.New("overlap must be a non-negative integer")
	}

	if overlap >= tileSize {
		return errors.New("overlap must be less than tile_size")
	}

	if suffix == ".jpg" || suffix == ".jpeg" {
		if quality < 0 || quality > 100 {
			return errors.New("quality for JPEG must be between 0 and 100")
		}
		// suffixe kalite parametresini ekle
		suffix = fmt.Sprintf("%s[Q=%d]", suffix, quality)
	} else if quality != 0 {
		fmt.Printf("Warning: Quality parameter is ignored for non-JPEG formats, using default value.\n")
	}

	supportedSuffixes := map[string]bool{".jpg": true, ".jpeg": true, ".png": true}

	// kalite eklenmiş suffix için kontrolü sadeleştirmek adına
	suffixKey := suffix
	if strings.Contains(suffix, "[") {
		suffixKey = suffix[:strings.Index(suffix, "[")]
	}
	if !supportedSuffixes[strings.ToLower(suffixKey)] {
		return fmt.Errorf("unsupported suffix: %s. Supported formats are .jpg, .jpeg, .png", suffixKey)
	}

	switch strings.ToLower(layout) {
	case "dzi", "dz":
		layout = "dz"
	case "google", "zoomify", "iiif", "iiif3":
		// kabul
	default:
		return fmt.Errorf("unsupported layout: %s. Supported layouts are 'dz', 'google', 'zoomify', 'iiif', 'iiif3'", layout)
	}

	if strings.ToLower(layout) == "google" && suffixKey == ".png" {
		fmt.Printf("Warning: Google layout does not support PNG suffix, using dz layout instead.\n")
		layout = "dz"
	}

	args := []string{
		"dzsave",
		f.FilePath,
		outputPath,
		"--layout", layout,
		"--tile-size", fmt.Sprintf("%d", tileSize),
		"--overlap", fmt.Sprintf("%d", overlap),
		"--suffix", suffix,
	}

	cmd := exec.Command("vips", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create DZI: %w - VIPS Output: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func getSVSInfo(filepath string) (*models.ImageInfo, error) {
	cmd := exec.Command("openslide-show-properties", filepath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute openslide command: %w", err)
	}

	info := &models.ImageInfo{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		switch key {
		case "openslide.level[0].width":
			if _, err := fmt.Sscanf(value, "%d", &info.Width); err != nil {
				return nil, fmt.Errorf("invalid width value: %w", err)
			}
		case "openslide.level[0].height":
			if _, err := fmt.Sscanf(value, "%d", &info.Height); err != nil {
				return nil, fmt.Errorf("invalid height value: %w", err)
			}
		case "openslide.image-size":
			if _, err := fmt.Sscanf(value, "%d", &info.Size); err != nil {
				return nil, fmt.Errorf("invalid size value: %w", err)
			}
		case "openslide.format":
			info.Format = value
		default:
			continue
		}
	}

	return info, nil
}

func getVIPSInfo(filepath string) (*models.ImageInfo, error) {
	cmd := exec.Command("vipsheader", "-f", "json", filepath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute vipsheader: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(output, &header); err != nil {
		return nil, fmt.Errorf("failed to parse vipsheader output: %w", err)
	}

	info := &models.ImageInfo{}
	if width, ok := header["Xsize"].(float64); ok {
		info.Width = int(width)
	}
	if height, ok := header["Ysize"].(float64); ok {
		info.Height = int(height)
	}
	if size, ok := header["Size"].(float64); ok {
		info.Size = int64(size)
	}
	if format, ok := header["Format"].(string); ok {
		info.Format = format
	}

	return info, nil
}

func exportVIPSThumbnail(inputPath, outputPath string, thumbSize int) error {
	cmd := exec.Command("vips", "thumbnail", inputPath, outputPath,
		strconv.Itoa(thumbSize),
		"--size", "both")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create thumbnail with vips: %w", err)
	}

	return nil
}

func exportSVSThumbnail(inputPath, outputPath string, thumbSize int) error {
	levels, err := getSVSLevels(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get SVS levels: %w", err)
	}

	bestLevel := selectBestLevel(levels, thumbSize)

	dims := levels[bestLevel]
	width := dims[0]
	height := dims[1]

	tempPng, err := os.CreateTemp("", "thumb-*.png")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPng.Name())

	cmd := exec.Command("openslide-write-png",
		inputPath,
		"0", "0",
		fmt.Sprintf("%d", bestLevel),
		fmt.Sprintf("%d", width),
		fmt.Sprintf("%d", height),
		tempPng.Name(),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openslide-write-png failed: %w - output: %s", err, string(output))
	}

	cmd = exec.Command("vips", "thumbnail", tempPng.Name(), outputPath,
		strconv.Itoa(thumbSize),
		"--size", "both")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vips thumbnail failed: %w - output: %s", err, string(output))
	}

	return nil
}

func getSVSLevels(filepath string) (map[int][2]int, error) {
	cmd := exec.Command("openslide-show-properties", filepath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	levels := make(map[int][2]int)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "openslide.level[") {
			continue
		}

		var level int
		var key string

		n, err := fmt.Sscanf(line, "openslide.level[%d].%s", &level, &key)
		if n < 2 || err != nil {
			continue
		}

		key = strings.TrimSuffix(key, ":") // örn: width: => width

		// Değer tek tırnak içinde
		parts := strings.SplitN(line, "'", 3)
		if len(parts) < 3 {
			continue
		}
		valueStr := parts[1]

		value, err := strconv.Atoi(valueStr)
		if err != nil {
			continue
		}

		val := levels[level]
		if key == "width" {
			val[0] = value
		} else if key == "height" {
			val[1] = value
		}
		levels[level] = val
	}

	if len(levels) == 0 {
		return nil, fmt.Errorf("no levels found")
	}

	return levels, nil
}

func selectBestLevel(levels map[int][2]int, thumbSize int) int {
	bestLevel := -1
	bestDiff := int(^uint(0) >> 1) // max int

	for level, dim := range levels {
		w, h := dim[0], dim[1]
		maxDim := w
		if h > w {
			maxDim = h
		}

		// Thumbnail boyutuna eşit veya küçük en yakın max dimension seç
		if maxDim <= thumbSize {
			diff := thumbSize - maxDim
			if diff < bestDiff {
				bestDiff = diff
				bestLevel = level
			}
		}
	}

	// Eğer uygun level yoksa (hepsi thumbnail'dan büyükse), en yüksek level (en küçük çözünürlük) seçilir
	if bestLevel == -1 {
		bestLevel = 0
		for level := range levels {
			if level > bestLevel {
				bestLevel = level
			}
		}
	}

	return bestLevel
}
