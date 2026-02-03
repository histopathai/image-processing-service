package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

// validateOutputs checks that all expected output files exist based on container type
func (s *ImageProcessingService) validateOutputs(workspace *model.Workspace, container string) error {
	s.logger.Info("Validating outputs", "container", container)

	// Common outputs for both container types
	requiredFiles := []string{
		"thumbnail.jpg",
		"image.dzi",
	}

	if container == "zip" {
		// V2 outputs (zip container)
		requiredFiles = append(requiredFiles,
			"image.zip",
			"IndexMap.json",
		)
	} else {
		// V1 outputs (fs container)
		// Check tiles directory exists
		tilesDir := workspace.Join("tiles")
		info, err := os.Stat(tilesDir)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.NewProcessingError("tiles directory was not created").
					WithContext("tiles_dir", tilesDir)
			}
			return errors.WrapStorageError(err, "failed to check tiles directory").
				WithContext("tiles_dir", tilesDir)
		}
		if !info.IsDir() {
			return errors.NewProcessingError("tiles path is not a directory").
				WithContext("tiles_dir", tilesDir)
		}

		// Check tiles directory is not empty
		entries, err := os.ReadDir(tilesDir)
		if err != nil {
			return errors.WrapStorageError(err, "failed to read tiles directory").
				WithContext("tiles_dir", tilesDir)
		}
		if len(entries) == 0 {
			return errors.NewProcessingError("tiles directory is empty").
				WithContext("tiles_dir", tilesDir)
		}
	}

	// Validate all required files exist and are not empty
	for _, filename := range requiredFiles {
		filePath := workspace.Join(filename)
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.NewProcessingError(fmt.Sprintf("required output file not found: %s", filename)).
					WithContext("file", filename).
					WithContext("path", filePath)
			}
			return errors.WrapStorageError(err, fmt.Sprintf("failed to check output file: %s", filename)).
				WithContext("file", filename).
				WithContext("path", filePath)
		}
		if info.Size() == 0 {
			return errors.NewProcessingError(fmt.Sprintf("output file is empty: %s", filename)).
				WithContext("file", filename).
				WithContext("path", filePath).
				WithContext("size", info.Size())
		}

		s.logger.Debug("Output file validated",
			"file", filename,
			"size", info.Size())
	}

	s.logger.Info("All outputs validated successfully", "container", container)
	return nil
}

// copyOutputsToStorage copies all output files from /tmp workspace to destination storage
func (s *ImageProcessingService) copyOutputsToStorage(ctx context.Context, workspace *model.Workspace, imageID string, container string) error {
	s.logger.Info("Copying outputs to storage", "imageID", imageID, "container", container)

	// Output files to copy
	outputFiles := []string{
		"thumbnail.jpg",
		"image.dzi",
	}

	if container == "zip" {
		// V2 outputs
		outputFiles = append(outputFiles,
			"image.zip",
			"IndexMap.json",
		)
	}

	// Copy individual files
	for _, filename := range outputFiles {
		localPath := workspace.Join(filename)
		remotePath := filepath.Join(imageID, filename)

		s.logger.Debug("Copying output file",
			"file", filename,
			"local_path", localPath,
			"remote_path", remotePath)

		if err := s.outputStorage.PutFile(ctx, localPath, remotePath); err != nil {
			return errors.WrapStorageError(err, "failed to copy output file to storage").
				WithContext("file", filename).
				WithContext("local_path", localPath).
				WithContext("remote_path", remotePath)
		}
	}

	// Copy tiles directory for fs container
	if container == "fs" {
		localTilesDir := workspace.Join("tiles")
		remoteTilesDir := filepath.Join(imageID, "tiles")

		s.logger.Debug("Copying tiles directory",
			"local_dir", localTilesDir,
			"remote_dir", remoteTilesDir)

		if err := s.outputStorage.PutDirectory(ctx, localTilesDir, remoteTilesDir); err != nil {
			return errors.WrapStorageError(err, "failed to copy tiles directory to storage").
				WithContext("local_dir", localTilesDir).
				WithContext("remote_dir", remoteTilesDir)
		}
	}

	s.logger.Info("All outputs copied to storage successfully", "imageID", imageID)
	return nil
}
