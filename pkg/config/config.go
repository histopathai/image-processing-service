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

type GCPConfig struct {
	ProjectID        string
	Region           string
	InputBucketName  string
	OutputBucketName string
}

type MountPath struct {
	InputMountPath  string
	OutputMountPath string
}

type PubSubConfig struct {
	ImageProcessResultTopicID string
}

type LoggingConfig struct {
	Level  string
	Format string
}

type DZIConfig struct {
	TileSize int
	Overlap  int
	Quality  int
	Layout   string
	Suffix   string
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
	UseGCSUpload       bool // true = GCS SDK, false = mount
	MaxParallelUploads int
	UploadChunkSizeMB  int
}

type Config struct {
	Env                       Environment
	WorkerType                WorkerType
	GCP                       GCPConfig
	MountPath                 MountPath
	PubSubConfig              PubSubConfig
	Logging                   LoggingConfig
	DZIConfig                 DZIConfig
	ThumbnailConfig           ThumbnailConfig
	ImageProcessTimeoutMinute ImageProcessTimeoutMinute
	Storage                   StorageConfig
}

func LoadConfig(logger *slog.Logger) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using environment variables")
	}

	env := Environment(getEnv("APP_ENV", "LOCAL"))
	workerType := WorkerType(getEnv("WORKER_TYPE", "medium"))

	tileSize, _ := strconv.Atoi(getEnv("TILE_SIZE", "256"))
	overlap, _ := strconv.Atoi(getEnv("OVERLAP", "0"))
	quality, _ := strconv.Atoi(getEnv("QUALITY", "85"))
	layout := getEnv("DZI_LAYOUT", "dz")
	suffix := getEnv("DZI_SUFFIX", "jpg")
	dziConfig := DZIConfig{
		TileSize: tileSize,
		Overlap:  overlap,
		Quality:  quality,
		Layout:   layout,
		Suffix:   suffix,
	}

	thumbSize, _ := strconv.Atoi(getEnv("THUMBNAIL_SIZE", "256"))
	thumbQuality, _ := strconv.Atoi(getEnv("THUMBNAIL_QUALITY", "90"))

	thumbnailConfig := ThumbnailConfig{
		Width:   thumbSize,
		Height:  thumbSize,
		Quality: thumbQuality,
	}

	gcpConfig := GCPConfig{
		ProjectID:        getEnv("PROJECT_ID", ""),
		Region:           getEnv("REGION", ""),
		InputBucketName:  getEnv("ORIGINAL_BUCKET_NAME", ""),
		OutputBucketName: getEnv("PROCESSED_BUCKET_NAME", ""),
	}

	mountPath := MountPath{
		InputMountPath:  getEnv("INPUT_MOUNT_PATH", "/mnt/input"),
		OutputMountPath: getEnv("OUTPUT_MOUNT_PATH", "/mnt/output"),
	}

	pubsubConfig := PubSubConfig{
		ImageProcessResultTopicID: getEnv("IMAGE_PROCESS_RESULT_TOPIC_ID", ""),
	}

	loggingConfig := LoggingConfig{
		Level:  getEnv("LOG_LEVEL", "INFO"),
		Format: getEnv("LOG_FORMAT", "json"),
	}

	timeoutFormatConversion, _ := strconv.Atoi(getEnv("FORMAT_CONVERSION_TIMEOUT_MINUTE", "20"))
	timeoutDZIConversion, _ := strconv.Atoi(getEnv("DZI_CONVERSION_TIMEOUT_MINUTE", "120"))
	timeoutThumbnail, _ := strconv.Atoi(getEnv("THUMBNAIL_TIMEOUT_MINUTE", "10"))
	timeoutGeneral, _ := strconv.Atoi(getEnv("GENERAL_IMAGE_PROCESS_TIMEOUT_MINUTE", "10"))

	imageProcessTimeout := ImageProcessTimeoutMinute{
		FormatConversion: timeoutFormatConversion,
		DZIConversion:    timeoutDZIConversion,
		Thumbnail:        timeoutThumbnail,
		General:          timeoutGeneral,
	}

	// Storage configuration
	useGCSUpload := getEnv("USE_GCS_UPLOAD", "true") == "true"
	maxParallelUploads, _ := strconv.Atoi(getEnv("MAX_PARALLEL_UPLOADS", "20"))
	uploadChunkSizeMB, _ := strconv.Atoi(getEnv("UPLOAD_CHUNK_SIZE_MB", "16"))

	storageConfig := StorageConfig{
		UseGCSUpload:       useGCSUpload,
		MaxParallelUploads: maxParallelUploads,
		UploadChunkSizeMB:  uploadChunkSizeMB,
	}

	config := &Config{
		Env:                       env,
		WorkerType:                workerType,
		GCP:                       gcpConfig,
		MountPath:                 mountPath,
		PubSubConfig:              pubsubConfig,
		Logging:                   loggingConfig,
		DZIConfig:                 dziConfig,
		ThumbnailConfig:           thumbnailConfig,
		ImageProcessTimeoutMinute: imageProcessTimeout,
		Storage:                   storageConfig,
	}

	logger.Info("Configuration loaded",
		"env", config.Env,
		"worker_type", config.WorkerType,
		"use_gcs_upload", config.Storage.UseGCSUpload)
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
