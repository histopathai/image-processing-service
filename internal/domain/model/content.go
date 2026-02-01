package model

import "github.com/histopathai/image-processing-service/internal/domain/vobj"

type Content struct {
	vobj.Entity
	Provider      vobj.ContentProvider
	Path          string
	ContentType   vobj.ContentType
	Size          int64
	UploadPending bool
}
