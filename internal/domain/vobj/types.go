package vobj

const (
	EntityTypeImage          EntityType = "image"
	EntityTypeAnnotation     EntityType = "annotation"
	EntityTypePatient        EntityType = "patient"
	EntityTypeWorkspace      EntityType = "workspace"
	EntityTypeAnnotationType EntityType = "annotation_type"
	EntityTypeContent        EntityType = "content"
)

const (
	ParentTypeNone           ParentType = "None"
	ParentTypeWorkspace      ParentType = "workspace"
	ParentTypePatient        ParentType = "patient"
	ParentTypeImage          ParentType = "image"
	ParentTypeAnnotationType ParentType = "annotation_type"
	ParentTypeAnnotation     ParentType = "annotation"
	ParentTypeContent        ParentType = "content"
)

const (
	StatusPending         ImageStatus = "pending"          // Initial state, waiting for processing
	StatusProcessing      ImageStatus = "processing"       // Currently being processed
	StatusProcessed       ImageStatus = "processed"        // Successfully processed
	StatusFailed          ImageStatus = "failed"           // Processing failed (retrying)
	StatusFailedPermanent ImageStatus = "failed_permanent" // Permanent failure (DLQ)
	StatusDeleting        ImageStatus = "deleting"         // Marked for deletion
	StatusUploaded        ImageStatus = "uploaded"         // Successfully uploaded
)

const (
	// Image types (standard MIME types)
	ContentTypeImageSVS   ContentType = "image/x-aperio-svs"
	ContentTypeImageTIFF  ContentType = "image/tiff"
	ContentTypeImageNDPI  ContentType = "image/x-ndpi"
	ContentTypeImageVMS   ContentType = "image/x-vms"
	ContentTypeImageVMU   ContentType = "image/x-vmu"
	ContentTypeImageSCN   ContentType = "image/x-scn"
	ContentTypeImageMIRAX ContentType = "image/x-mirax"
	ContentTypeImageBIF   ContentType = "image/x-bif"
	ContentTypeImageDNG   ContentType = "image/x-adobe-dng"
	ContentTypeImageBMP   ContentType = "image/bmp"
	ContentTypeImageJPEG  ContentType = "image/jpeg"
	ContentTypeImagePNG   ContentType = "image/png"

	// Custom Image types
	ContentTypeThumbnailJPEG ContentType = "image/x-thumb-jpeg"
	ContentTypeThumbnailPNG  ContentType = "image/x-thumb-png"

	// Archive types
	ContentTypeApplicationZip ContentType = "application/zip"

	// Document types
	ContentTypeApplicationJSON ContentType = "application/json"

	// DZI (Deep Zoom Image) - XML based format
	ContentTypeApplicationDZI ContentType = "application/xml"

	// Generic fallback
	ContentTypeApplicationOctetStream ContentType = "application/octet-stream"
)

const (
	ContentProviderLocal ContentProvider = "local"
	ContentProviderS3    ContentProvider = "s3"
	ContentProviderGCS   ContentProvider = "gcs"
	ContentProviderAzure ContentProvider = "azure"
	ContentProviderMinIO ContentProvider = "minio"
	ContentProviderHTTP  ContentProvider = "http"
)
