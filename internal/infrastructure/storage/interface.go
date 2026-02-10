package storage

import (
	"context"
	"io"
)

// InputStorage abstracts reading files from various sources (GCS FUSE mount, local filesystem, etc.)
type InputStorage interface {
	// GetReader returns a reader for the file at the given path
	GetReader(ctx context.Context, path string) (io.ReadCloser, error)

	// CopyToLocal copies a file from storage to a local path
	CopyToLocal(ctx context.Context, remotePath, localPath string) error

	// Exists checks if a file exists at the given path
	Exists(ctx context.Context, path string) (bool, error)
}

// OutputStorage abstracts writing files to various destinations (GCS upload, GCS FUSE mount, local filesystem, etc.)
type OutputStorage interface {
	// PutFile uploads a single file from local path to remote path
	PutFile(ctx context.Context, localPath, remotePath string) error

	// PutDirectory uploads an entire directory recursively
	PutDirectory(ctx context.Context, localDir, remoteDir string) error

	// Delete removes a file or directory
	Delete(ctx context.Context, remotePath string) error
}
