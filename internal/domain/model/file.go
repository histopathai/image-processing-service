package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	ID       string
	Filename string
	Dir      string

	Width  *int
	Height *int
	Size   *int64
	Format *string
}

func NewFile(id, filename, dir string, width, height *int, size *int64, format *string) (*File, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("directory cannot be empty")
	}

	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", dir)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}

	return &File{
		ID:       id,
		Filename: filename,
		Dir:      dir,
		Width:    width,
		Height:   height,
		Size:     size,
		Format:   format,
	}, nil
}

func (f *File) BaseName() string {
	return strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename))
}

func (f *File) Exists() bool {
	_, err := os.Stat(f.AbsolutePath())
	return err == nil
}

func (f *File) Extension() string {
	return strings.ToLower(filepath.Ext(f.Filename))
}

func (f *File) WidthValue() int {
	if f.Width != nil {
		return *f.Width
	}
	return 0
}

func (f *File) HeightValue() int {
	if f.Height != nil {
		return *f.Height
	}
	return 0
}

func (f *File) SizeValue() int64 {
	if f.Size != nil {
		return *f.Size
	}
	return 0
}

func (f *File) FormatValue() string {
	if f.Format != nil {
		return *f.Format
	}
	return ""
}

func (f *File) AbsolutePath() string {
	return filepath.Join(f.Dir, f.Filename)
}

func (f *File) SetDimensions(width, height int, size int64) {
	f.Width = &width
	f.Height = &height
	f.Size = &size
}

func (f *File) SetFormat(format string) {
	f.Format = &format
}

func (f *File) SetFilename(filename string) {
	f.Filename = filename
}

func (f *File) SetDir(dir string) {
	f.Dir = dir
}

func (f *File) Clone() *File {
	clone := &File{
		ID:       f.ID,
		Filename: f.Filename,
		Dir:      f.Dir,
	}

	if f.Width != nil {
		width := *f.Width
		clone.Width = &width
	}
	if f.Height != nil {
		height := *f.Height
		clone.Height = &height
	}
	if f.Size != nil {
		size := *f.Size
		clone.Size = &size
	}
	if f.Format != nil {
		format := *f.Format
		clone.Format = &format
	}

	return clone
}
