package processors

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/pkg/errors"
)

type ZipIndexProcessor struct {
	*BaseProcessor
}

func NewZipIndexProcessor() *ZipIndexProcessor {
	return &ZipIndexProcessor{
		BaseProcessor: NewBaseProcessor(nil, "zip-index-internal"),
	}
}

type ZipEntryIndex struct {
	Name             string `json:"name"`
	Offset           int64  `json:"offset"`
	CompressedSize   int64  `json:"compressed_size"`
	UncompressedSize int64  `json:"uncompressed_size"`
	Method           uint16 `json:"method"`
}

type ZipIndexMap struct {
	Version int             `json:"version"`
	ZipFile string          `json:"zip_file"`
	Entries []ZipEntryIndex `json:"entries"`
}

func (z *ZipIndexProcessor) BuildIndexMap(
	ctx context.Context,
	zipPath string,
	destDir string,
) error {

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to open zip").
			WithContext("zip", zipPath)
	}
	defer r.Close()

	index := ZipIndexMap{
		Version: 1,
		ZipFile: filepath.Base(zipPath),
	}

	for _, f := range r.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		offset, err := f.DataOffset()
		if err != nil {
			return errors.WrapProcessingError(err, "failed to get data offset").
				WithContext("file", f.Name)
		}
		index.Entries = append(index.Entries, ZipEntryIndex{
			Name:             f.Name,
			Offset:           offset,
			CompressedSize:   int64(f.CompressedSize64),
			UncompressedSize: int64(f.UncompressedSize64),
			Method:           f.Method,
		})
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.WrapStorageError(err, "failed to create dest dir").
			WithContext("dir", destDir)
	}

	outPath := filepath.Join(destDir, "IndexMap.json")
	out, err := os.Create(outPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to create index file").
			WithContext("file", outPath)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(index); err != nil {
		return errors.WrapProcessingError(err, "failed to write index map")
	}

	return nil
}

func (z *ZipIndexProcessor) ExtractDesiredFile(
	ctx context.Context,
	zipPath string,
	targetFile string,
	destPath string,
) error {

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to open zip").
			WithContext("zip", zipPath)
	}
	defer r.Close()

	var file *zip.File
	for _, f := range r.File {
		if f.Name == targetFile {
			file = f
			break
		}
	}

	if file == nil {
		return errors.NewNotFoundError("file not found in zip").
			WithContext("file", targetFile).
			WithContext("zip", zipPath)
	}

	rc, err := file.Open()
	if err != nil {
		return errors.WrapStorageError(err, "failed to open target file in zip").
			WithContext("file", targetFile)
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return errors.WrapStorageError(err, "failed to create dest file").
			WithContext("file", destPath)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	if err != nil {
		return errors.WrapProcessingError(err, "failed to copy file content")
	}

	return nil
}
