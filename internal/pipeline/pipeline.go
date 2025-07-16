package pipeline

import (
	"context"
	"fmt"

	"github.com/histopathai/image-processing-service/internal/adapter"
	"github.com/histopathai/image-processing-service/internal/models"
	"github.com/histopathai/image-processing-service/internal/service"
	"github.com/histopathai/image-processing-service/internal/utils"
)

type JobRequest struct {
	ImagePath   string             `json:"image_path"`
	DatasetInfo models.DatasetInfo `json:"dataset_info"`
}

type JobResult struct {
	Image    *models.Image
	TmpDir   string
	Error    error
	Success  bool
	ErrorMsg string
}

type Pipeline struct {
	ProcessCh  chan JobRequest
	RegisterCh chan JobResult
	DoneCh     chan struct{}

	ImgService *service.ImgProcService
	FsAdapter  *adapter.FirestoreAdapter
}

func NewPipeline(imgService *service.ImgProcService, fsAdapter *adapter.FirestoreAdapter) *Pipeline {
	p := &Pipeline{
		ProcessCh:  make(chan JobRequest, 100),
		RegisterCh: make(chan JobResult, 100),
		DoneCh:     make(chan struct{}),
		ImgService: imgService,
		FsAdapter:  fsAdapter,
	}

	go p.startProcessWorker()
	go p.startRegisterWorker()

	return p
}

func (p *Pipeline) startProcessWorker() {
	for job := range p.ProcessCh {
		_ = utils.LogInfo(map[string]interface{}{
			"module":    "pipeline",
			"event":     "process-start",
			"imagePath": job.ImagePath,
		})

		// --- Duplicate kontrolü buraya taşındı (işleme başlamadan önce) ---
		isDup, err := p.isDuplicate(context.Background(), job.DatasetInfo)
		if err != nil {
			_ = utils.LogError(map[string]interface{}{
				"module":      "pipeline",
				"event":       "duplicate-check-error-pre-process",
				"imagePath":   job.ImagePath,
				"datasetName": job.DatasetInfo.DatasetName,
				"fileName":    job.DatasetInfo.FileName,
				"organType":   job.DatasetInfo.OrganType,
				"error":       err.Error(),
				"success":     false,
			})
			p.RegisterCh <- JobResult{
				Image:    nil,
				TmpDir:   "",
				Error:    err,
				Success:  false,
				ErrorMsg: err.Error(),
			}
			continue
		}

		if isDup {
			_ = utils.LogWarning(map[string]interface{}{
				"module":      "pipeline",
				"event":       "image-duplicate-skipped-pre-process",
				"imagePath":   job.ImagePath,
				"datasetName": job.DatasetInfo.DatasetName,
				"fileName":    job.DatasetInfo.FileName,
				"organType":   job.DatasetInfo.OrganType,
				"message":     "Duplicate entry found based on DatasetInfo, skipping processing.",
				"success":     false,
			})
			p.RegisterCh <- JobResult{
				Image:    nil,
				TmpDir:   "",
				Error:    fmt.Errorf("duplicate image found, processing skipped"),
				Success:  false,
				ErrorMsg: "Duplicate image found, processing skipped",
			}
			continue
		}

		image, tmpDir, err := p.ImgService.ProcessImage(context.Background(), job.ImagePath)

		if image != nil {
			image.DatasetInfo = job.DatasetInfo
		}

		result := JobResult{
			Image:   image,
			TmpDir:  tmpDir,
			Error:   err,
			Success: err == nil,
		}

		if err != nil {
			result.ErrorMsg = err.Error()
			_ = utils.LogError(map[string]interface{}{
				"module":  "pipeline",
				"event":   "process-error",
				"error":   result.ErrorMsg,
				"path":    tmpDir,
				"success": false,
			})
		} else {
			_ = utils.LogSuccess(map[string]interface{}{
				"module":  "pipeline",
				"event":   "process-success",
				"imageID": image.ID,
				"success": true,
			})
		}

		p.RegisterCh <- result
	}
}

func (p *Pipeline) startRegisterWorker() {
	for result := range p.RegisterCh {
		ctx := context.Background()

		imageID := "N/A"
		if result.Image != nil {
			imageID = result.Image.ID
		}

		_ = utils.LogInfo(map[string]interface{}{
			"module":  "pipeline",
			"event":   "register-start",
			"imageID": imageID,
			"tmpDir":  result.TmpDir,
			"success": result.Success,
			"error":   result.ErrorMsg,
		})

		if !result.Success {
			_ = utils.LogWarning(map[string]interface{}{
				"module":  "pipeline",
				"event":   "register-failed-because-process-failed",
				"error":   result.ErrorMsg,
				"path":    result.TmpDir,
				"success": false,
			})
			continue
		}

		err := p.ImgService.RegisterImage(ctx, result.Image, result.TmpDir)
		if err != nil {
			_ = utils.LogError(map[string]interface{}{
				"module":  "pipeline",
				"event":   "register-error",
				"imageID": result.Image.ID,
				"error":   err.Error(),
				"success": false,
			})
			continue
		}

		if _, err := p.FsAdapter.Create(ctx, result.Image.ToDbMap()); err != nil {
			_ = utils.LogError(map[string]interface{}{
				"module":  "pipeline",
				"event":   "firestore-write-error",
				"imageID": result.Image.ID,
				"error":   err.Error(),
				"success": false,
			})
			continue
		}

		_ = p.ImgService.Cleanup(result.TmpDir)

		_ = utils.LogSuccess(map[string]interface{}{
			"module":  "pipeline",
			"event":   "register-success",
			"imageID": result.Image.ID,
			"success": true,
		})
	}
}

func (p *Pipeline) isDuplicate(ctx context.Context, datasetInfo models.DatasetInfo) (bool, error) {
	filter := map[string]interface{}{
		"dataset_name": datasetInfo.DatasetName,
		"file_name":    datasetInfo.FileName,
		"organ_type":   datasetInfo.OrganType,
	}

	existingDocs, err := p.FsAdapter.List(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check for duplicate in Firestore: %w", err)
	}

	return len(existingDocs) > 0, nil
}
