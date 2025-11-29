package port

import "context"

// internal/domain/events/interfaces.go
type Publisher interface {
	Publish(ctx context.Context, topic string, data []byte, attributes map[string]string) error
}
