package model

import "github.com/histopathai/image-processing-service/internal/domain/vobj"

type Content struct {
	vobj.Entity
	Provider      vobj.ContentProvider `json:"provider"`
	Path          string               `json:"path"`
	ContentType   vobj.ContentType     `json:"content_type"`
	Size          int64                `json:"size"`
	UploadPending bool                 `json:"upload_pending"`
}
