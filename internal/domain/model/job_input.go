package model

import "fmt"

type JobInput struct {
	ImageID           string
	OriginPath        string
	ProcessingVersion string
	bucketName        string
}

func NewJobInputFromEnv(imageID, originPath, processingVersion, bucketName string) (*JobInput, error) {
	if imageID == "" {
		return nil, fmt.Errorf("image ID is required")
	}
	if originPath == "" {
		return nil, fmt.Errorf("origin path is required")
	}
	if processingVersion == "" {
		return nil, fmt.Errorf("processing version is required")
	}
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	return &JobInput{
		ImageID:           imageID,
		OriginPath:        originPath,
		ProcessingVersion: processingVersion,
		bucketName:        bucketName,
	}, nil
}
