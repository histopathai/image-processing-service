package storage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/histopathai/image-processing-service/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type GCSStorage struct {
	*BaseStorage
	gcsClient   *storage.Client
	bucketName  string
	maxParallel int
}

func NewGCSStorage(logger *slog.Logger, gcsClient *storage.Client, bucketName string) *GCSStorage {
	return &GCSStorage{
		BaseStorage: NewBaseStorage(logger),
		gcsClient:   gcsClient,
		bucketName:  bucketName,
		maxParallel: 20,
	}
}

func (s *GCSStorage) UploadDirectory(ctx context.Context, sourceDir, destPath string) error {
	s.logger.Info("Starting parallel GCS upload",
		"source", sourceDir,
		"destination", destPath,
		"bucket", s.bucketName,
		"max_parallel", s.maxParallel)

	files, err := s.collectFiles(sourceDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.NewStorageError("source directory is empty").
			WithContext("sourceDir", sourceDir)
	}

	s.logger.Info("Found files to upload",
		"count", len(files),
		"source", sourceDir)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxParallel)

	var uploaded, failed int64
	var mu sync.Mutex

	for _, fileInfo := range files {
		fileInfo := fileInfo

		g.Go(func() error {
			sourcePath := fileInfo.SourcePath
			fullDestKey := filepath.Join(destPath, fileInfo.DestKey)
			fullDestKey = filepath.ToSlash(fullDestKey)
			destKey := fullDestKey

			if err := s.uploadFileToGCS(ctx, sourcePath, destKey); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				s.logger.Error("Failed to upload file",
					"source", sourcePath,
					"dest", destKey,
					"error", err)
				return err
			}

			mu.Lock()
			uploaded++
			if uploaded%1000 == 0 {
				s.logger.Info("Upload progress",
					"uploaded", uploaded,
					"total", len(files))
			}
			mu.Unlock()

			return nil
		})
	}

	// Wait for all uploads to complete
	if err := g.Wait(); err != nil {
		return errors.WrapStorageError(err, "failed to upload directory to GCS").
			WithContext("source", sourceDir).
			WithContext("uploaded", uploaded).
			WithContext("failed", failed)
	}

	s.logger.Info("Successfully uploaded directory to GCS",
		"source", sourceDir,
		"destination", destPath,
		"uploaded", uploaded,
		"failed", failed)

	return nil
}

func (s *GCSStorage) uploadFileToGCS(ctx context.Context, sourcePath, destKey string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to open source file").
			WithContext("source_path", sourcePath)
	}
	defer file.Close()

	// GCS object writer
	obj := s.gcsClient.Bucket(s.bucketName).Object(destKey)
	writer := obj.NewWriter(ctx)

	writer.ChunkSize = 16 * 1024 * 1024 // 16MB chunks
	writer.ContentType = s.detectContentType(sourcePath)

	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return errors.WrapStorageError(err, "failed to upload file content").
			WithContext("source_path", sourcePath).
			WithContext("dest_key", destKey)
	}

	if err := writer.Close(); err != nil {
		return errors.WrapStorageError(err, "failed to close writer").
			WithContext("source_path", sourcePath).
			WithContext("dest_key", destKey)
	}

	return nil
}
