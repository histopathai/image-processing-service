package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/histopathai/image-processing-service/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type StorageService struct {
	logger       *slog.Logger
	gcsClient    *storage.Client
	bucketName   string
	maxParallel  int
	useGCSUpload bool // true = GCS SDK, false = mount copy
}

func NewStorageService(logger *slog.Logger, gcsClient *storage.Client, bucketName string, useGCSUpload bool) *StorageService {
	return &StorageService{
		logger:       logger,
		gcsClient:    gcsClient,
		bucketName:   bucketName,
		maxParallel:  20,
		useGCSUpload: useGCSUpload,
	}
}

func (s *StorageService) UploadDirectory(ctx context.Context, sourceDir, destPath string) error {
	if s.useGCSUpload {
		return s.uploadDirectoryToGCS(ctx, sourceDir, destPath)
	}
	return s.uploadDirectoryToMount(ctx, sourceDir, destPath)
}

func (s *StorageService) uploadDirectoryToGCS(ctx context.Context, sourceDir, destPath string) error {
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
			destKey := fileInfo.DestKey

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
			if uploaded%100 == 0 {
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

// uploadFileToGCS uploads a single file to GCS
func (s *StorageService) uploadFileToGCS(ctx context.Context, sourcePath, destKey string) error {
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

func (s *StorageService) uploadDirectoryToMount(ctx context.Context, sourceDir, destDir string) error {
	s.logger.Info("Uploading directory to mount", "source", sourceDir, "destination", destDir)

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
			if err := s.uploadDirectoryToMount(ctx, sourcePath, destPath); err != nil {
				return err
			}
		} else {
			if err := s.uploadFileToMount(ctx, sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	s.logger.Info("Successfully uploaded directory to mount", "source", sourceDir, "destination", destDir)
	return nil
}

// uploadFileToMount uploads a single file to mount path
func (s *StorageService) uploadFileToMount(ctx context.Context, sourcePath, destPath string) error {
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

	return nil
}

// collectFiles recursively collects all files in a directory
func (s *StorageService) collectFiles(rootDir string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Relative path hesapla
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		files = append(files, FileInfo{
			SourcePath: path,
			DestKey:    filepath.ToSlash(relPath),
			Size:       info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to collect files").
			WithContext("rootDir", rootDir)
	}

	return files, nil
}

// detectContentType detects MIME type from file extension
func (s *StorageService) detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".tif":  "image/tiff",
		".tiff": "image/tiff",
		".dzi":  "application/xml",
		".xml":  "application/xml",
		".json": "application/json",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}

// FileInfo holds file information for upload
type FileInfo struct {
	SourcePath string
	DestKey    string
	Size       int64
}

// FileExists checks if a file exists
func (s *StorageService) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileInfo gets file information
func (s *StorageService) GetFileInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to get file info").
			WithContext("path", path)
	}
	return info, nil
}
