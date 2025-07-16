package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/histopathai/image-processing-service/config"
	"github.com/histopathai/image-processing-service/internal/handler"
)

func Start(cfg *config.Config, h *handler.Handler) {

	gin.SetMode(cfg.ServerConfig.GinMode)
	router := gin.Default()

	router.POST("/upload", h.UploadImages)

	router.Run(fmt.Sprintf(":%d", cfg.ServerConfig.Port))
}
