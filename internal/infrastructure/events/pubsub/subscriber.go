package pubsub

import (
	"context"
	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/histopathai/image-processing-service/internal/domain/events"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type Subscriber struct {
	client       *pubsub.Client
	subscription *pubsub.Subscription
	logger       *slog.Logger
	cancel       context.CancelFunc
}

func NewSubscriber(client *pubsub.Client, subID string, logger *slog.Logger) *Subscriber {
	sub := client.Subscription(subID)
	return &Subscriber{
		client:       client,
		subscription: sub,
		logger:       logger,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, subscription string, handler events.EventHandler) error {
	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.logger.Info("Starting Pub/Sub subscriber", "subscription", subscription)

	err := s.subscription.Receive(subCtx, func(ctx context.Context, msg *pubsub.Message) {
		s.logger.Debug("Received message", "msg_id", msg.ID)

		err := handler(ctx, msg.Data, msg.Attributes)
		if err != nil {
			s.logger.Error("Error processing message, sending NACK", "msg_id", msg.ID, "error", err)
			msg.Nack()
		} else {
			s.logger.Info("Successfully processed message, sending ACK", "msg_id", msg.ID)
			msg.Ack()
		}
	})

	if err != nil && err != context.Canceled {
		s.logger.Error("Subscriber Receive returned error", "error", err)
		return errors.NewInternalError("pubsub.Receive failed").WithContext("error", err)
	}

	s.logger.Info("Subscriber stopped.")
	return nil
}

func (s *Subscriber) Stop() error {
	s.logger.Info("Stopping subscriber...")
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

var _ events.Subscriber = (*Subscriber)(nil)
