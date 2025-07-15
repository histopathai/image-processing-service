package utils

import (
	"fmt"
	"os"
	"time"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func GenerateUID() string {
	// Generate a unique identifier for the file
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func CreateDir(path string) error {
	// Create a directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func RemoveDir(path string) error {
	// Remove a directory and its contents
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", path, err)
	}
	return nil
}
