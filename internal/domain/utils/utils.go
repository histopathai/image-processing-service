package utils

import (
	_ "embed"
	"encoding/json"
	"strings"
)

type Format map[string]bool

//go:embed supported_formats.json
var supportedFormatsBytes []byte

// Global runtime-loaded supported formats
var SupportedFormats = Format{}

// Load JSON file into SupportedFormats via go:embed
func LoadSupportedFormats() error {
	// Unmarshal into global variable
	err := json.Unmarshal(supportedFormatsBytes, &SupportedFormats)
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
