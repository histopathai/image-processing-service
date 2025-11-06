package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/container"
	"github.com/histopathai/image-processing-service/pkg/errors"
	"github.com/histopathai/image-processing-service/pkg/logger"
)

type PubSubMessage struct {
	Message struct {
		Data       string            `json:"data"`
		Attributes map[string]string `json:"attributes"`
		MessageID  string            `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

func main() {
	// Initialize basic logger for startup
	startupLogger := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
	})

	startupLogger.Info("Starting Image Processing Service")

	// Load configuration
	cfg, err := config.LoadConfig(startupLogger)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize configured logger
	appLogger := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	appLogger.Info("Configuration loaded",
		"env", cfg.Env,
		"project_id", cfg.GCP.ProjectID,
	)

	// Create context
	ctx := context.Background()

	// Initialize container
	cnt, err := container.New(ctx, cfg, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize container", "error", err)
		log.Fatalf("Failed to initialize container: %v", err)
	}
	defer func() {
		if err := cnt.Close(); err != nil {
			appLogger.Error("Error closing container", "error", err)
		}
	}()

	appLogger.Info("Container initialized successfully")

	// Setup HTTP handlers
	http.HandleFunc("/health", healthHandler(appLogger))
	http.HandleFunc("/", pubsubHandler(cnt, appLogger))

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info("Starting HTTP server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed", "error", err)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Server exited")
}

// healthHandler returns a simple health check endpoint
func healthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}

// pubsubHandler handles incoming Pub/Sub push messages
func pubsubHandler(cnt *container.Container, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		// Parse the Pub/Sub message
		var msg PubSubMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			logger.Error("Failed to decode Pub/Sub message", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		logger.Info("Received Pub/Sub message",
			"message_id", msg.Message.MessageID,
			"subscription", msg.Subscription,
		)

		// Decode the base64 encoded data
		decodedData, err := base64.StdEncoding.DecodeString(msg.Message.Data)
		if err != nil {
			logger.Error("Failed to decode message data", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		attributes := msg.Message.Attributes
		if attributes == nil {
			attributes = make(map[string]string)
		}

		// Process the image
		logger.Info("Processing image request",
			"event_type", attributes["event_type"],
			"image_id", attributes["image_id"],
		)

		err = cnt.ImageHandlerService.HandleImageProcessingRequest(ctx, decodedData, attributes)
		if err != nil {
			logger.Error("Failed to process image",
				"message_id", msg.Message.MessageID,
				"image_id", attributes["image_id"],
				"error", err,
			)

			// Check if the error is non-retryable
			if errors.IsNonRetryable(err) {
				// ACK the message by returning 200 OK.
				// The handler service has already published a failure event.
				logger.Warn("Acknowledging non-retryable error to stop Pub/Sub retry loop",
					"message_id", msg.Message.MessageID,
					"image_id", attributes["image_id"],
				)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK: Acknowledged non-retryable error"))
			} else {
				// NACK the message by returning 500. Pub/Sub will retry.
				http.Error(w, fmt.Sprintf("Processing failed, will retry: %v", err), http.StatusInternalServerError)
			}
			return
		}

		logger.Info("Successfully processed image",
			"message_id", msg.Message.MessageID,
			"image_id", attributes["image_id"],
		)

		// Return 200 to acknowledge the message
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
