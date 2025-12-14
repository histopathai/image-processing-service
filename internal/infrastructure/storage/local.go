package storage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

type LocalStorage struct {
	*BaseStorage
}

func NewLocalStorage(logger *slog.Logger) *LocalStorage {
	return &LocalStorage{
		BaseStorage: NewBaseStorage(logger),
	}
}
func (s *LocalStorage) UploadDirectory(ctx context.Context, sourceDir, destDir string) error {
	s.logger.Info("Moving directory locally",
		"source", sourceDir,
		"destination", destDir,
	)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return errors.WrapStorageError(err, "failed to create destination dir").
			WithContext("destDir", destDir)
	}

	return filepath.Walk(sourceDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(sourceDir, srcPath)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(destDir, rel)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if err := os.Rename(srcPath, dstPath); err == nil {
			return nil
		}

		if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
			return err
		}
		return nil
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
