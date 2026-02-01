package vobj

// =========================== ContentType =================================
type ContentType string

func (ct ContentType) GetCategory() string {
	switch ct {
	case ContentTypeImageSVS, ContentTypeImageTIFF, ContentTypeImageNDPI,
		ContentTypeImageVMS, ContentTypeImageVMU, ContentTypeImageSCN,
		ContentTypeImageMIRAX, ContentTypeImageBIF, ContentTypeImageDNG,
		ContentTypeImageBMP, ContentTypeImageJPEG, ContentTypeImagePNG,
		ContentTypeThumbnailJPEG, ContentTypeThumbnailPNG:
		return "image"
	case ContentTypeApplicationZip:
		return "archive"
	case ContentTypeApplicationJSON, ContentTypeApplicationDZI:
		return "document"
	default:
		return "other"
	}
}

func (ct ContentType) IsValid() bool {
	switch ct {
	case ContentTypeImageSVS, ContentTypeImageTIFF, ContentTypeImageNDPI,
		ContentTypeImageVMS, ContentTypeImageVMU, ContentTypeImageSCN,
		ContentTypeImageMIRAX, ContentTypeImageBIF, ContentTypeImageDNG,
		ContentTypeImageBMP, ContentTypeImageJPEG, ContentTypeImagePNG,
		ContentTypeThumbnailJPEG, ContentTypeThumbnailPNG,
		ContentTypeApplicationZip, ContentTypeApplicationJSON,
		ContentTypeApplicationDZI, ContentTypeApplicationOctetStream:
		return true
	default:
		return false
	}
}
func (ct ContentType) IsOriginImage() bool {
	if ct.GetCategory() == "image" && ct.IsThumbnail() == false {
		return true
	}
	return false
}

func (ct ContentType) IsThumbnail() bool {
	switch ct {
	case ContentTypeThumbnailJPEG, ContentTypeThumbnailPNG:
		return true
	default:
		return false
	}
}

func (ct ContentType) IsIndexMap() bool {
	if ContentTypeApplicationJSON == ct {
		return true
	}
	return false
}

func (ct ContentType) IsArchive() bool {
	if ContentTypeApplicationZip == ct {
		return true
	}
	return false
}

func (ct ContentType) IsDZI() bool {
	if ContentTypeApplicationDZI == ct {
		return true
	}
	return false
}

func (ct ContentType) IsTiles() bool {
	if ContentTypeApplicationOctetStream == ct {
		return true
	}
	return false
}

func (ct ContentType) ToStandardType() ContentType {
	switch ct {
	case ContentTypeThumbnailJPEG:
		return ContentTypeImageJPEG
	case ContentTypeThumbnailPNG:
		return ContentTypeImagePNG
	default:
		return ct
	}
}

func (ct ContentType) String() string {
	return string(ct)
}

// =========================== ContentProvider =================================

type ContentProvider string

func (cp ContentProvider) IsValid() bool {
	switch cp {
	case ContentProviderLocal, ContentProviderS3, ContentProviderGCS,
		ContentProviderAzure, ContentProviderMinIO, ContentProviderHTTP:
		return true
	default:
		return false
	}
}

func (cp ContentProvider) String() string {
	return string(cp)
}

func (cp ContentProvider) IsCloud() bool {
	switch cp {
	case ContentProviderS3, ContentProviderGCS, ContentProviderAzure, ContentProviderMinIO:
		return true
	default:
		return false
	}
}
