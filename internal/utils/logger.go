package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var logFilePath = "logs/job-log.jsonl"
var mu sync.Mutex

func InitLogger(path string) {
	logFilePath = path
}

// Genel log fonksiyonu
func LogJob(level string, data map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(logFilePath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	data["level"] = level

	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write log to file: %w", err)
	}

	return nil
}

// Helper fonksiyonlar

func LogError(data map[string]interface{}) error {
	return LogJob("error", data)
}

func LogInfo(data map[string]interface{}) error {
	return LogJob("info", data)
}

func LogSuccess(data map[string]interface{}) error {
	return LogJob("success", data)
}

func LogWarning(data map[string]interface{}) error {
	return LogJob("warning", data)
}
