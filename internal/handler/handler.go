package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/histopathai/image-processing-service/config"
	"github.com/histopathai/image-processing-service/internal/pipeline"
)

type Handler struct {
	cfg      *config.Config
	pipeline *pipeline.Pipeline
}

func NewHandler(cfg *config.Config, p *pipeline.Pipeline) *Handler {
	return &Handler{
		cfg:      cfg,
		pipeline: p,
	}
}

func (h *Handler) UploadImages(c *gin.Context) {
	rawData, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	// İlk önce ham json objesini map[string]interface{} veya []interface{} olarak decode et
	var raw interface{}
	if err := json.Unmarshal(rawData, &raw); err != nil {
		c.JSON(400, gin.H{"error": "invalid JSON"})
		return
	}

	switch raw.(type) {
	case map[string]interface{}:
		// Tekli istek, yeniden marshal edip struct'a decode et
		var req pipeline.JobRequest
		if err := json.Unmarshal(rawData, &req); err != nil {
			c.JSON(400, gin.H{"error": "invalid single request"})
			return
		}
		h.pipeline.ProcessCh <- req

	case []interface{}:
		// Çoklu istek
		var reqs []pipeline.JobRequest
		if err := json.Unmarshal(rawData, &reqs); err != nil {
			c.JSON(400, gin.H{"error": "invalid batch request"})
			return
		}
		for _, job := range reqs {
			h.pipeline.ProcessCh <- job
		}

	default:
		c.JSON(400, gin.H{"error": "invalid JSON or not an array"})
		return
	}

	c.JSON(200, gin.H{"status": "processing started"})
}
