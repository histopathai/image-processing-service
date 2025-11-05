package events

import "context"

// internal/domain/events/interfaces.go
type Publisher interface {
	Publish(ctx context.Context, topic string, data []byte, attributes map[string]string) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, subscription string, handler EventHandler) error
	Stop() error
}

type EventHandler func(ctx context.Context, data []byte, attributes map[string]string) error
