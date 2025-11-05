package model

import "strings"

type Format map[string]bool

var SupportedFormats = Format{
	"jpeg": true,
	"png":  true,
	"tiff": true,
	"svs":  true,
	"jpg":  true,
	"tif":  true,
	"bmp":  true,
	"ndpi": true,
	"scn":  true,
	"bif":  true,
	"vms":  true,
	"vmu":  true,
}

func (f Format) IsSupported(format string) bool {
	standartizedFormat := strings.ToLower(strings.TrimPrefix(format, "."))
	_, ok := f[standartizedFormat]
	return ok
}

type File struct {
	ID            string
	Filename      string
	Path          string
	Width         *int
	Height        *int
	Size          *int64
	Format        *string
	ProcessedPath *string
}
