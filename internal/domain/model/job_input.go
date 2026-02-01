package model

import "fmt"

type JobInput struct {
	ImageID           string
	OriginPath        string
	ProcessingVersion string
}

func NewJobInputFromEnv(imageID, originPath, processingVersion string) (*JobInput, error) {
	if imageID == "" {
		return nil, fmt.Errorf("image ID is required")
	}
	if originPath == "" {
		return nil, fmt.Errorf("origin path is required")
	}
	if processingVersion == "" {
		return nil, fmt.Errorf("processing version is required")
	}
	return &JobInput{
		ImageID:           imageID,
		OriginPath:        originPath,
		ProcessingVersion: processingVersion,
	}, nil
}
