package utils

import (
	"encoding/json"
	"os"
	"strings"
)

type Format map[string]bool

// Global runtime-loaded supported formats
var SupportedFormats = Format{}

// Load JSON file into SupportedFormats
func LoadSupportedFormats(jsonFilePath string) error {
	data, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return err
	}

	// Unmarshal into global variable
	err = json.Unmarshal(data, &SupportedFormats)
	if err != nil {
		return err
	}

	return nil
}

func (f Format) IsSupported(format string) bool {
	standardizedFormat := strings.ToLower(strings.TrimPrefix(format, "."))
	_, ok := f[standardizedFormat]
	return ok
}
