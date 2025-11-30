package model

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Workspace struct {
	file *File
	dir  string
}

func NewWorkspace(file *File) (*Workspace, error) {
	if file == nil {
		return nil, fmt.Errorf("file cannot be nil")
	}

	tempDir, err := os.MkdirTemp("/tmp", fmt.Sprintf("workspace-%s", file.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return &Workspace{
		file: file,
		dir:  tempDir,
	}, nil
}

func (w *Workspace) Join(elem ...string) string {
	elements := append([]string{w.dir}, elem...)
	return filepath.Join(elements...)
}

func (w *Workspace) Remove() error {
	if err := os.RemoveAll(w.dir); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}
	return nil
}

func (w *Workspace) RemoveFile(filePath string) error {
	var absPath string

	if filepath.IsAbs(filePath) {
		absPath = filePath
	} else {
		absPath = w.Join(filePath)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", absPath)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", absPath, err)
	}

	return nil
}

func (w *Workspace) File() *File {
	return w.file
}

func (w *Workspace) List() ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace contents: %w", err)
	}
	return entries, nil
}

func (w *Workspace) Dir() string {
	return w.dir
}

func (w *Workspace) Exists() bool {
	info, err := os.Stat(w.dir)
	return err == nil && info.IsDir()
}
