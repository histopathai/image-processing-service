package main

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"

	"github.com/histopathai/image-processing-service/config"
	"github.com/histopathai/image-processing-service/internal/adapter"
	"github.com/histopathai/image-processing-service/internal/handler"
	"github.com/histopathai/image-processing-service/internal/pipeline"
	"github.com/histopathai/image-processing-service/internal/server"
	"github.com/histopathai/image-processing-service/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Config load error: %v", err)
	}

	gcsAdapter, err := adapter.NewGCSAdapter(cfg.GCPConfig.ProjectID, cfg.GCPConfig.Bucket)
	if err != nil {
		log.Fatalf("GCS init error: %v", err)
	}

	// Firestore client'ı gerçek olarak oluştur
	fsClient, err := firestore.NewClient(ctx, cfg.GCPConfig.ProjectID)
	if err != nil {
		log.Fatalf("Firestore client creation failed: %v", err)
	}
	defer fsClient.Close()

	fsAdapter := adapter.NewFirestoreAdapter(fsClient, cfg.GCPConfig.FirestoreCollection)

	imgService := service.NewImgProcService(&cfg, gcsAdapter)
	pipe := pipeline.NewPipeline(imgService, fsAdapter)
	h := handler.NewHandler(&cfg, pipe)

	server.Start(&cfg, h)
}
