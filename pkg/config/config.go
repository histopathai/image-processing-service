package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Environment string

const (
	EnvLocal      Environment = "LOCAL"
	EnvDev        Environment = "DEV"
	EnvProduction Environment = "PROD"
)

type WorkerType string

const (
	WorkerTypeSmall  WorkerType = "small"
	WorkerTypeMedium WorkerType = "medium"
	WorkerTypeLarge  WorkerType = "large"
)

// GCPConfig holds Google Cloud Platform related configuration.
type GCPConfig struct {
	ProjectID          string
	Region             string
	InputBucketName    string
	OutputBucketName   string
	MaxParallelUploads int
	UploadChunkSizeMB  int
}

type LoggingConfig struct {
	Level  string
	Format string
}

type DZIConfig struct {
	TileSize    int
	Overlap     int
	Quality     int
	Layout      string
	Suffix      string
	Container   string
	Compression int
}

type ImageProcessTimeoutMinute struct {
	FormatConversion int
	DZIConversion    int
	Thumbnail        int
	General          int
}

type ThumbnailConfig struct {
	Width   int
	Height  int
	Quality int
}

type StorageConfig struct {
	InputMountPath  string // Mount path for input files (e.g., /input, /gcs/bucket-original, ./test-data/input)
	OutputMountPath string // Mount path for output files (e.g., /output, /gcs/bucket-processed, ./test-data/output)
}

type Config struct {
	Env                       Environment
	WorkerType                WorkerType
	GCP                       GCPConfig
	Storage                   StorageConfig
	OutputRootPath            string // Deprecated: use Storage.OutputMountPath
	Logging                   LoggingConfig
	DZIConfig                 DZIConfig
	ThumbnailConfig           ThumbnailConfig
	ImageProcessTimeoutMinute ImageProcessTimeoutMinute
	ImageProcessingTopicID    string
}

func LoadGCPConfig() GCPConfig {
	return GCPConfig{
		ProjectID:        os.Getenv("PROJECT_ID"),
		Region:           os.Getenv("REGION"),
		InputBucketName:  os.Getenv("ORIGINAL_BUCKET_NAME"),
		OutputBucketName: os.Getenv("PROCESSED_BUCKET_NAME"),
	}
}

func LoadDZIConfig() DZIConfig {
	tileSize, err := strconv.Atoi(os.Getenv("TILE_SIZE"))
	if err != nil {
		tileSize = 256
	}
	overlap, err := strconv.Atoi(os.Getenv("OVERLAP"))
	if err != nil {
		overlap = 0
	}
	quality, err := strconv.Atoi(os.Getenv("QUALITY"))
	if err != nil {
		quality = 85
	}
	layout := os.Getenv("DZI_LAYOUT")
	if layout == "" {
		layout = "dz"
	}
	suffix := os.Getenv("DZI_SUFFIX")
	if suffix == "" {
		suffix = "jpg"
	}

	container := os.Getenv("DZI_CONTAINER")
	if container != "zip" {
		container = "fs"
	}

	compression, err := strconv.Atoi(os.Getenv("DZI_COMPRESSION"))
	if err != nil {
		compression = 0
	}
	if compression < 0 || compression > 9 {
		compression = 0
	}
	return DZIConfig{
		TileSize:    tileSize,
		Overlap:     overlap,
		Quality:     quality,
		Layout:      layout,
		Suffix:      suffix,
		Container:   container,
		Compression: compression,
	}
}

func LoadThumbnailConfig() ThumbnailConfig {
	width, err := strconv.Atoi(os.Getenv("THUMBNAIL_SIZE"))
	if err != nil {
		width = 256
	}
	height, err := strconv.Atoi(os.Getenv("THUMBNAIL_SIZE"))
	if err != nil {
		height = 256
	}
	quality, err := strconv.Atoi(os.Getenv("THUMBNAIL_QUALITY"))
	if err != nil {
		quality = 90
	}
	return ThumbnailConfig{
		Width:   width,
		Height:  height,
		Quality: quality,
	}
}

func LoadTimeoutConfig() ImageProcessTimeoutMinute {
	formatConversion, err := strconv.Atoi(os.Getenv("FORMAT_CONVERSION_TIMEOUT_MINUTE"))
	if err != nil {
		formatConversion = 20
	}
	dziConversion, err := strconv.Atoi(os.Getenv("DZI_CONVERSION_TIMEOUT_MINUTE"))
	if err != nil {
		dziConversion = 120
	}
	thumbnail, err := strconv.Atoi(os.Getenv("THUMBNAIL_TIMEOUT_MINUTE"))
	if err != nil {
		thumbnail = 10
	}
	general, err := strconv.Atoi(os.Getenv("GENERAL_IMAGE_PROCESS_TIMEOUT_MINUTE"))
	if err != nil {
		general = 10
	}
	return ImageProcessTimeoutMinute{
		FormatConversion: formatConversion,
		DZIConversion:    dziConversion,
		Thumbnail:        thumbnail,
		General:          general,
	}
}

func LoadLoggingConfig() LoggingConfig {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "INFO"
	}
	format := os.Getenv("LOG_FORMAT")
	if format == "" {
		format = "json"
	}
	return LoggingConfig{
		Level:  level,
		Format: format,
	}
}
func LoadConfig(logger *slog.Logger) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using environment variables")
	}

	env := Environment(getEnv("APP_ENV", "LOCAL"))
	workerType := WorkerType(getEnv("WORKER_TYPE", "medium"))
	imageProcessingTopicID := getEnv("IMAGE_PROCESSING_TOPIC_ID", "image-processing-tasks")

	dziConfig := LoadDZIConfig()
	thumbnailConfig := LoadThumbnailConfig()
	timeoutConfig := LoadTimeoutConfig()
	loggingConfig := LoadLoggingConfig()
	var outputRootPath string
	var gcpConfig GCPConfig
	var storageConfig StorageConfig

	if env == EnvLocal {
		outputRootPath = getEnv("OUTPUT_ROOT_PATH", "./output")
		storageConfig = StorageConfig{
			InputMountPath:  getEnv("INPUT_MOUNT_PATH", "./test-data/input"),
			OutputMountPath: getEnv("OUTPUT_MOUNT_PATH", "./test-data/output"),
		}
		gcpConfig = GCPConfig{}
	} else {
		outputRootPath = ""
		// In cloud, use /input and /output mount points (GCS FUSE)
		storageConfig = StorageConfig{
			InputMountPath:  getEnv("INPUT_MOUNT_PATH", "/input"),
			OutputMountPath: getEnv("OUTPUT_MOUNT_PATH", "/output"),
		}
		gcpConfig = LoadGCPConfig()
	}

	config := &Config{
		Env:                       env,
		WorkerType:                workerType,
		Storage:                   storageConfig,
		OutputRootPath:            outputRootPath,
		GCP:                       gcpConfig,
		Logging:                   loggingConfig,
		DZIConfig:                 dziConfig,
		ThumbnailConfig:           thumbnailConfig,
		ImageProcessTimeoutMinute: timeoutConfig,
		ImageProcessingTopicID:    imageProcessingTopicID,
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
