package stdout

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/internal/domain/port"
)

type Publisher struct {
	logger    *slog.Logger
	outputDir string
}

func NewPublisher(logger *slog.Logger, outputDir string) *Publisher {
	return &Publisher{
		logger:    logger,
		outputDir: outputDir,
	}
}

func (p *Publisher) Publish(ctx context.Context, topicID string, data []byte, attributes map[string]string) error {
	// Pretty-print the JSON for stdout
	var prettyJSON json.RawMessage
	if err := json.Unmarshal(data, &prettyJSON); err != nil {
		// If not valid JSON, write raw data
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stdout, string(data))
		} else {
			fmt.Fprintln(os.Stdout, string(formatted))
		}
	}

	// Write result.json to the output directory
	if p.outputDir != "" {
		imageID, ok := attributes["image_id"]
		if !ok || imageID == "" {
			imageID = ""
		}

		var resultDir string
		if imageID != "" {
			resultDir = filepath.Join(p.outputDir, imageID)
		} else {
			resultDir = p.outputDir
		}

		if err := os.MkdirAll(resultDir, 0755); err != nil {
			p.logger.Error("Failed to create result directory", "path", resultDir, "error", err)
			return fmt.Errorf("failed to create result directory: %w", err)
		}

		resultPath := filepath.Join(resultDir, "result.json")

		// Write pretty JSON to file
		formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
		if err != nil {
			formatted = data
		}

		if err := os.WriteFile(resultPath, formatted, 0644); err != nil {
			p.logger.Error("Failed to write result.json", "path", resultPath, "error", err)
			return fmt.Errorf("failed to write result.json: %w", err)
		}

		p.logger.Info("Result written to file", "path", resultPath)
	}

	return nil
}

func (p *Publisher) Close() error {
	return nil
}

var _ port.EventPublisher = (*Publisher)(nil)
