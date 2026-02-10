package storage

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/histopathai/image-processing-service/internal/domain/port"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type BaseStorage struct {
	logger *slog.Logger
}

func NewBaseStorage(logger *slog.Logger) *BaseStorage {
	return &BaseStorage{
		logger: logger,
	}
}

func (bs *BaseStorage) collectFiles(sourceDir string) ([]port.FileInfo, error) {
	var files []port.FileInfo
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		files = append(files, port.FileInfo{
			SourcePath: path,
			DestKey:    strings.ReplaceAll(relPath, string(os.PathSeparator), "/"),
			Size:       info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, errors.WrapStorageError(err, "failed to collect files").
			WithContext("sourceDir", sourceDir)
	}
	return files, nil
}

func (bs *BaseStorage) detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	contentTypes := map[string]string{
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".dzi":  "application/xml",
		".xml":  "application/xml",
		".json": "application/json",
		".zip":  "application/zip",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}
