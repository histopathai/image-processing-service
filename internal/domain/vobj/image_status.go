package vobj

type ImageStatus string

func (is ImageStatus) String() string {
	return string(is)
}

func (is ImageStatus) IsValid() bool {
	switch is {
	case StatusPending, StatusProcessing, StatusProcessed, StatusFailed, StatusFailedPermanent, StatusDeleting:
		return true
	default:
		return false
	}
}
