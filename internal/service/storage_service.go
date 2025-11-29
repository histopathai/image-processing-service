package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

type StorageService struct {
	logger *slog.Logger
}

func NewStorageService(logger *slog.Logger) *StorageService {
	return &StorageService{
		logger: logger,
	}
}

func (s *StorageService) UploadDirectory(ctx context.Context, sourceDir, destDir string) error {
	s.logger.Info("Uploading directory", "source", sourceDir, "destination", destDir)

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return errors.WrapStorageError(err, "failed to read source directory").
			WithContext("sourceDir", sourceDir)
	}

	if len(entries) == 0 {
		return errors.NewStorageError("source directory is empty").
			WithContext("sourceDir", sourceDir)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			if err := s.UploadDirectory(ctx, sourcePath, destPath); err != nil {
				return err
			}
		} else {
			// Upload file
			if err := s.uploadFile(ctx, sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	s.logger.Info("Successfully uploaded directory", "source", sourceDir, "destination", destDir)
	return nil
}

func (s *StorageService) uploadFile(ctx context.Context, sourcePath, destPath string) error {
	destDir := filepath.Dir(destPath)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create destination directory").
			WithContext("dest_dir", destDir)
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to open source file").
			WithContext("source_path", sourcePath)
	}

	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to create destination file").
			WithContext("dest_path", destPath)
	}

	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return errors.WrapStorageError(err, "failed to copy file content").
			WithContext("source_path", sourcePath).
			WithContext("dest_path", destPath)
	}

	s.logger.Info("Uploaded file", "source", sourcePath, "destination", destPath)
	return nil
}

func (s *StorageService) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *StorageService) GetFileInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to get file info").
			WithContext("path", path)
	}
	return info, nil
}
