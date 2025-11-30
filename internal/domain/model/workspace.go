package model

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Workspace represents a temporary working directory for file operations.
// It manages a temporary filesystem location associated with a specific file.
type Workspace struct {
	file *File
	dir  string
}

// NewWorkspace creates a new temporary workspace directory for the given file.
// The workspace is created in the system's temp directory with a prefix based on the file ID.
// Returns an error if the temporary directory cannot be created.
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

// Join constructs a path by joining the workspace path with the provided elements.
func (w *Workspace) Join(elem ...string) string {
	elements := append([]string{w.dir}, elem...)
	return filepath.Join(elements...)
}

// Remove deletes the workspace directory and all its contents.
// Returns an error if the removal fails.
func (w *Workspace) Remove() error {
	if err := os.RemoveAll(w.dir); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}
	return nil
}

// File returns the file associated with this workspace.
func (w *Workspace) File() *File {
	return w.file
}

// List returns all directory entries in the workspace directory.
// Returns an error if the directory cannot be read.
func (w *Workspace) List() ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace contents: %w", err)
	}
	return entries, nil
}

// Path returns the absolute path to the workspace directory.
func (w *Workspace) Dir() string {
	return w.dir
}

// Exists checks if the workspace directory still exists on the filesystem.
func (w *Workspace) Exists() bool {
	info, err := os.Stat(w.dir)
	return err == nil && info.IsDir()
}
