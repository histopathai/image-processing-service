package storage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

// MountStorage implements storage interfaces for mount-based access (GCS FUSE, local filesystem)
// It simply copies files between the mount point and local /tmp
type MountStorage struct {
	basePath string
	logger   *slog.Logger
}

// NewMountStorage creates a new mount-based storage
// basePath is the mount point (e.g., "/input", "/gcs/bucket-name", "./test-data/input")
func NewMountStorage(basePath string, logger *slog.Logger) *MountStorage {
	return &MountStorage{
		basePath: basePath,
		logger:   logger,
	}
}

// GetReader implements InputStorage.GetReader
func (m *MountStorage) GetReader(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(m.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError("file not found").
				WithContext("path", path).
				WithContext("full_path", fullPath)
		}
		return nil, errors.WrapStorageError(err, "failed to open file").
			WithContext("path", path).
			WithContext("full_path", fullPath)
	}

	return file, nil
}

// CopyToLocal implements InputStorage.CopyToLocal
func (m *MountStorage) CopyToLocal(ctx context.Context, remotePath, localPath string) error {
	// Handle absolute paths by using them directly as the source
	// This is common in local development where INPUT_ORIGIN_PATH is an absolute path
	var fullRemotePath string
	if filepath.IsAbs(remotePath) {
		// Use the absolute path directly
		fullRemotePath = remotePath
		m.logger.Debug("Using absolute path directly",
			"remote_path", remotePath,
			"full_remote_path", fullRemotePath)
	} else {
		// Join with basePath for relative paths
		fullRemotePath = filepath.Join(m.basePath, remotePath)
		m.logger.Debug("Joining with basePath",
			"remote_path", remotePath,
			"basePath", m.basePath,
			"full_remote_path", fullRemotePath)
	}

	m.logger.Debug("Copying file from mount to local",
		"remote_path", remotePath,
		"full_remote_path", fullRemotePath,
		"local_path", localPath)

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create local directory").
			WithContext("dir", localDir)
	}

	// Open source file
	src, err := os.Open(fullRemotePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError("source file not found").
				WithContext("remote_path", remotePath).
				WithContext("full_path", fullRemotePath)
		}
		return errors.WrapStorageError(err, "failed to open source file").
			WithContext("remote_path", remotePath).
			WithContext("full_path", fullRemotePath)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(localPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to create destination file").
			WithContext("local_path", localPath)
	}
	defer dst.Close()

	// Copy data
	copied, err := io.Copy(dst, src)
	if err != nil {
		return errors.WrapStorageError(err, "failed to copy file data").
			WithContext("remote_path", remotePath).
			WithContext("local_path", localPath)
	}

	m.logger.Debug("File copied successfully",
		"remote_path", remotePath,
		"local_path", localPath,
		"bytes", copied)

	return nil
}

// Exists implements InputStorage.Exists
func (m *MountStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(m.basePath, path)

	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.WrapStorageError(err, "failed to check file existence").
		WithContext("path", path).
		WithContext("full_path", fullPath)
}

// PutFile implements OutputStorage.PutFile
func (m *MountStorage) PutFile(ctx context.Context, localPath, remotePath string) error {
	fullRemotePath := filepath.Join(m.basePath, remotePath)

	m.logger.Debug("Copying file from local to mount",
		"local_path", localPath,
		"remote_path", remotePath,
		"full_remote_path", fullRemotePath)

	// Ensure remote directory exists
	remoteDir := filepath.Dir(fullRemotePath)
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create remote directory").
			WithContext("dir", remoteDir)
	}

	// Open source file
	src, err := os.Open(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError("local file not found").
				WithContext("local_path", localPath)
		}
		return errors.WrapStorageError(err, "failed to open local file").
			WithContext("local_path", localPath)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullRemotePath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to create remote file").
			WithContext("remote_path", remotePath).
			WithContext("full_path", fullRemotePath)
	}
	defer dst.Close()

	// Copy data
	copied, err := io.Copy(dst, src)
	if err != nil {
		return errors.WrapStorageError(err, "failed to copy file data").
			WithContext("local_path", localPath).
			WithContext("remote_path", remotePath)
	}

	m.logger.Debug("File uploaded successfully",
		"local_path", localPath,
		"remote_path", remotePath,
		"bytes", copied)

	return nil
}

// PutDirectory implements OutputStorage.PutDirectory
func (m *MountStorage) PutDirectory(ctx context.Context, localDir, remoteDir string) error {
	fullRemoteDir := filepath.Join(m.basePath, remoteDir)

	m.logger.Debug("Copying directory from local to mount",
		"local_dir", localDir,
		"remote_dir", remoteDir,
		"full_remote_dir", fullRemoteDir)

	// Walk the local directory
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return errors.WrapStorageError(err, "failed to calculate relative path").
				WithContext("local_path", localPath).
				WithContext("local_dir", localDir)
		}

		remotePath := filepath.Join(fullRemoteDir, relPath)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(remotePath, 0755); err != nil {
				return errors.WrapStorageError(err, "failed to create remote directory").
					WithContext("remote_path", remotePath)
			}
			return nil
		}

		// Copy file
		src, err := os.Open(localPath)
		if err != nil {
			return errors.WrapStorageError(err, "failed to open local file").
				WithContext("local_path", localPath)
		}
		defer src.Close()

		dst, err := os.Create(remotePath)
		if err != nil {
			return errors.WrapStorageError(err, "failed to create remote file").
				WithContext("remote_path", remotePath)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return errors.WrapStorageError(err, "failed to copy file").
				WithContext("local_path", localPath).
				WithContext("remote_path", remotePath)
		}

		return nil
	})
}

// Delete implements OutputStorage.Delete
func (m *MountStorage) Delete(ctx context.Context, remotePath string) error {
	fullPath := filepath.Join(m.basePath, remotePath)

	m.logger.Debug("Deleting file/directory",
		"remote_path", remotePath,
		"full_path", fullPath)

	err := os.RemoveAll(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.WrapStorageError(err, "failed to delete").
			WithContext("remote_path", remotePath).
			WithContext("full_path", fullPath)
	}

	return nil
}

// Verify interfaces are implemented
var _ InputStorage = (*MountStorage)(nil)
var _ OutputStorage = (*MountStorage)(nil)
