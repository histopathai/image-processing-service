package models

import (
	"time"
)

type ImageInfo struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
	Format string `json:"format"`
}

type DatasetInfo struct {
	FileName       string `json:"file_name"`
	FileUID        string `json:"file_uid"`
	DatasetName    string `json:"dataset_name"`
	OrganType      string `json:"organ_type"`
	DiseaseType    string `json:"disease_type"`
	Classification string `json:"classification"`
	SubType        string `json:"sub_type"`
	Grade          string `json:"grade"`
}

type Image struct {
	ID          string      `json:"id" firestore:"id"`
	DatasetInfo DatasetInfo `json:"dataset_info" firestore:"dataset_info"`
	ImageInfo   ImageInfo   `json:"image_info" firestore:"image_info"`

	DZIGCSPath       string `json:"dzi_gcs_path" firestore:"dzi_gcs_path"`
	TilesGCSPath     string `json:"tiles_gcs_path" firestore:"tiles_gcs_path"`
	ThumbnailGCSPath string `json:"thumbnail_gcs_path" firestore:"thumbnail_gcs_path"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}

func (img *Image) ToDbMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             img.ID,
		"file_name":      img.DatasetInfo.FileName,
		"file_uid":       img.DatasetInfo.FileUID,
		"dataset_name":   img.DatasetInfo.DatasetName,
		"organ_type":     img.DatasetInfo.OrganType,
		"disease_type":   img.DatasetInfo.DiseaseType,
		"classification": img.DatasetInfo.Classification,
		"sub_type":       img.DatasetInfo.SubType,
		"grade":          img.DatasetInfo.Grade,

		"width":  img.ImageInfo.Width,
		"height": img.ImageInfo.Height,
		"size":   img.ImageInfo.Size,
		"format": img.ImageInfo.Format,

		"dzi_gcs_path":       img.DZIGCSPath,
		"tiles_gcs_path":     img.TilesGCSPath,
		"thumbnail_gcs_path": img.ThumbnailGCSPath,

		"created_at": img.CreatedAt,
		"updated_at": img.UpdatedAt,
	}
}

func Now() time.Time {
	return time.Now().UTC()
}
