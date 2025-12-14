package model

import "fmt"

type JobInput struct {
	ImageID    string
	OriginPath string
}

func NewJobInputFromEnv(imageID, originPath string) (*JobInput, error) {
	if imageID == "" {
		return nil, fmt.Errorf("image ID is required")
	}
	if originPath == "" {
		return nil, fmt.Errorf("origin path is required")
	}
	return &JobInput{
		ImageID:    imageID,
		OriginPath: originPath,
	}, nil
}
