package stdout

import (
	"context"
	"log/slog"

	"github.com/histopathai/image-processing-service/internal/domain/events"
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

var _ events.Publisher = (*Publisher)(nil)
