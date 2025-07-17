package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func GenerateUniqueID() string {
	for {
		b := make([]byte, 15) // 15 byte -> ~20 karakter base64
		if _, err := rand.Read(b); err == nil {
			id := base64.URLEncoding.EncodeToString(b)
			id = strings.TrimRight(id, "=")
			if len(id) > 20 {
				id = id[:20]
			}
			return id
		}
	}
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
