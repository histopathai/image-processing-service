package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerConfig ServerConfig
	GCPConfig    GCPConfig
	Parameters   ParameterConfig
}

type ServerConfig struct {
	Port             int
	Host             string
	SupportedFormats []string
}

type GCPConfig struct {
	ProjectID           string
	Location            string
	Bucket              string
	FirestoreCollection string
}

type ParameterConfig struct {
	TileSize      int64
	Overlap       int64
	Layout        string
	Quality       int64
	Suffix        string
	ThumbnailSize int
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, fmt.Errorf("error loading .env file: %v", err)
	}
	env := os.Getenv("ENV")
	if env == "LOCAL" {
		gacPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if gacPath == "" {
			return Config{}, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
		}
		if _, err := os.Stat(gacPath); os.IsNotExist(err) {
			return Config{}, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS file does not exist at path: %s", gacPath)
		}
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", gacPath)
		fmt.Printf("Using local Google Application Credentials: %s\n", gacPath)
	}

	supported_formats := strings.Split(strings.TrimSpace(os.Getenv("SUPPORTED_FORMATS")), ",")

	if len(supported_formats) == 0 || (len(supported_formats) == 1 && supported_formats[0] == "") {
		supported_formats = []string{"svs", "tif", "tiff", "jpg", "jpeg", "png"}
	}

	return Config{
		ServerConfig: ServerConfig{
			Port:             int(getEnvAsInt("SERVER_PORT", 8080)),
			Host:             getEnvOrDefault("SERVER_HOST", "localhost"),
			SupportedFormats: supported_formats,
		},

		GCPConfig: GCPConfig{
			ProjectID:           os.Getenv("GCP_PROJECT_ID"),
			Location:            os.Getenv("GCP_LOCATION"),
			Bucket:              os.Getenv("GCP_BUCKET"),
			FirestoreCollection: os.Getenv("GCP_FIRESTORE_COLLECTION"),
		},
		Parameters: ParameterConfig{
			TileSize:      getEnvAsInt("TILE_SIZE", 256),
			Overlap:       getEnvAsInt("OVERLAP", 0),
			Layout:        getEnvOrDefault("LAYOUT", "dzi"),
			Quality:       getEnvAsInt("QUALITY", 75),
			Suffix:        getEnvOrDefault("SUFFIX", ".jpg"),
			ThumbnailSize: int(getEnvAsInt("THUMBNAIL_SIZE", 256)),
		},
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing environment variable %s: %v. Using default value: %d\n", key, err, defaultValue)
		return defaultValue
	}
	return value
}
