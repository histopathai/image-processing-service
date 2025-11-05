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

type GCPConfig struct {
	ProjectID        string
	InputBucketName  string
	OutputBucketName string
}

type MountPath struct {
	InputMountPath  string
	OutputMountPath string
}

type PubSubConfig struct {
	ImageProcessingSubID      string
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

type ThumbnailConfig struct {
	Width   int
	Height  int
	Quality int
}

type Config struct {
	Env             Environment
	GCP             GCPConfig
	MountPath       MountPath
	PubSubConfig    PubSubConfig
	Logging         LoggingConfig
	DZIConfig       DZIConfig
	ThumbnailConfig ThumbnailConfig
}

func LoadConfig(logger *slog.Logger) (*Config, error) {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using environment variables")
	}

	env := Environment(getEnv("APP_ENV", "LOCAL"))

	// DZI Config
	tileSize, _ := strconv.Atoi(getEnv("TILE_SIZE", "256"))
	overlap, _ := strconv.Atoi(getEnv("OVERLAP", "0"))
	quality, _ := strconv.Atoi(getEnv("QUALITY", "85"))
	layout := getEnv("DZI_LAYOUT", "dz")
	suffix := getEnv("DZI_SUFFIX", "jpeg")

	dziConfig := DZIConfig{
		TileSize: tileSize,
		Overlap:  overlap,
		Quality:  quality,
		Layout:   layout,
		Suffix:   suffix,
	}

	// Thumbnail Config
	thumbSize, _ := strconv.Atoi(getEnv("THUMBNAIL_SIZE", "256"))
	thumbQuality, _ := strconv.Atoi(getEnv("THUMBNAIL_QUALITY", "90"))

	thumbnailConfig := ThumbnailConfig{
		Width:   thumbSize,
		Height:  thumbSize,
		Quality: thumbQuality,
	}

	// GCP Config
	gcpConfig := GCPConfig{
		ProjectID:        getEnv("PROJECT_ID", ""),
		InputBucketName:  getEnv("ORIGINAL_BUCKET_NAME", ""),
		OutputBucketName: getEnv("PROCESSED_BUCKET_NAME", ""),
	}

	// Mount Path
	mountPath := MountPath{
		InputMountPath:  getEnv("INPUT_MOUNT_PATH", "/mnt/input"),
		OutputMountPath: getEnv("OUTPUT_MOUNT_PATH", "/mnt/output"),
	}

	// Pub/Sub Config
	pubsubConfig := PubSubConfig{
		ImageProcessingSubID:      getEnv("IMAGE_PROCESSING_SUB_ID", ""),
		ImageProcessResultTopicID: getEnv("IMAGE_PROCESS_RESULT_TOPIC_ID", ""),
	}

	// Logging Config
	loggingConfig := LoggingConfig{
		Level:  getEnv("LOG_LEVEL", "INFO"),
		Format: getEnv("LOG_FORMAT", "TEXT"),
	}

	config := &Config{
		Env:             env,
		GCP:             gcpConfig,
		MountPath:       mountPath,
		PubSubConfig:    pubsubConfig,
		Logging:         loggingConfig,
		DZIConfig:       dziConfig,
		ThumbnailConfig: thumbnailConfig,
	}

	logger.Info("Configuration loaded", "env", config.Env)
	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
