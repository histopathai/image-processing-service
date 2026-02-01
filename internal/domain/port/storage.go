package port

import (
	"context"
)

type FileInfo struct {
	SourcePath string
	DestKey    string
	Size       int64
}
type Storage interface {
	UploadDirectory(ctx context.Context, sourceDir, destPath string) error
}
