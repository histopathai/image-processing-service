package pubsub

import (
	"context"
	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type Publisher struct {
	client *pubsub.Client
	logger *slog.Logger
}

func NewPublisher(client *pubsub.Client, logger *slog.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger,
	}
}

func (p *Publisher) Publish(ctx context.Context, topicID string, data []byte, attributes map[string]string) error {

	topic := p.client.Topic(topicID)
	defer topic.Stop()

	msg := &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}

	result := topic.Publish(ctx, msg)

	_, err := result.Get(ctx)
	if err != nil {
		p.logger.Error("Failed to publish message", "topic", topicID, "error", err)
		return errors.NewInternalError("could not publish message").WithContext("topic", topicID)
	}

	p.logger.Info("Message published successfully", "topic", topicID)
	return nil
}

// Ensure Publisher implements the EventPublisher interface
var _ events.Publisher = (*Publisher)(nil)
