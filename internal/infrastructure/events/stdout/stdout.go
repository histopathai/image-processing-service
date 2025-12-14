package stdout

import (
	"context"
	"log/slog"

	"github.com/histopathai/image-processing-service/internal/domain/port"
)

type Publisher struct {
	logger *slog.Logger
}

func NewPublisher(logger *slog.Logger) *Publisher {
	return &Publisher{
		logger: logger,
	}
}

func (p *Publisher) Publish(ctx context.Context, topicID string, data []byte, attributes map[string]string) error {
	msgContent := string(data)

	p.logger.InfoContext(ctx, "Event published to STDOUT (Local Dev)",
		slog.String("topic", topicID),
		slog.String("data", msgContent),
		slog.Any("attributes", attributes),
	)

	return nil
}

func (p *Publisher) Close() error {
	return nil
}

var _ port.EventPublisher = (*Publisher)(nil)
