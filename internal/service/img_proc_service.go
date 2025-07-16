package service

import (
	"context"
	"fmt"
	"time"

	"github.com/histopathai/image-processing-service/config"
	"github.com/histopathai/image-processing-service/internal/adapter"
	"github.com/histopathai/image-processing-service/internal/models"
	"github.com/histopathai/image-processing-service/internal/utils"
)

type ImgProcService struct {
	cfg *config.Config
	gcs *adapter.GCSAdapter
}

func NewImgProcService(cfg *config.Config, gcs *adapter.GCSAdapter) *ImgProcService {
	return &ImgProcService{
		cfg: cfg,
		gcs: gcs,
	}
}

func (s *ImgProcService) ProcessImage(ctx context.Context, filePath string) (*models.Image, string, error) {
	file, err := utils.NewFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create file object: %w", err)
	}

	if !utils.Contains(s.cfg.ServerConfig.SupportedFormats, file.Ext()) {
		return nil, "", fmt.Errorf("unsupported file format: %s", file.Ext())
	}

	imageInfo, err := file.FileInfo()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file info: %w", err)
	}

	uid := utils.GenerateUniqueID()

	tmpdir := fmt.Sprintf("/tmp/%s", uid)
	if err := utils.CreateDir(tmpdir); err != nil {
		return nil, "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	thumbnailPath := fmt.Sprintf("%s/thumbnail.jpg", tmpdir)
	if err := file.ExportThumbnail(thumbnailPath, s.cfg.Parameters.ThumbnailSize); err != nil {
		utils.RemoveDir(tmpdir)
		return nil, "", fmt.Errorf("failed to extract thumbnail: %w", err)
	}

	dziPath := fmt.Sprintf("%s/image", tmpdir)
	if err := file.ExtractDZI(dziPath, s.cfg); err != nil {
		utils.RemoveDir(tmpdir)
		return nil, "", fmt.Errorf("failed to extract DZI: %w", err)
	}

	image := &models.Image{
		ID:               uid,
		ImageInfo:        *imageInfo,
		TilesGCSPath:     fmt.Sprintf("%s/image_files", uid),
		DZIGCSPath:       fmt.Sprintf("%s/image.dzi", uid),
		ThumbnailGCSPath: fmt.Sprintf("%s/thumbnail.jpg", uid),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return image, tmpdir, nil
}

func (s *ImgProcService) RegisterImage(ctx context.Context, image *models.Image, tmpDir string) error {
	defer s.Cleanup(tmpDir)
	if err := s.gcs.UploadFile(ctx, fmt.Sprintf("%s/thumbnail.jpg", tmpDir), image.ThumbnailGCSPath); err != nil {
		return fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	if err := s.gcs.UploadFile(ctx, fmt.Sprintf("%s/image.dzi", tmpDir), image.DZIGCSPath); err != nil {
		s.gcs.DeleteFile(ctx, image.ThumbnailGCSPath) // Clean up thumbnail if DZI upload fails
		return fmt.Errorf("failed to upload DZI: %w", err)
	}

	if err := s.gcs.UploadDir(ctx, fmt.Sprintf("%s/image_files", tmpDir), image.TilesGCSPath); err != nil {
		s.gcs.DeleteFile(ctx, image.DZIGCSPath)
		s.gcs.DeleteFile(ctx, image.ThumbnailGCSPath)
		return fmt.Errorf("failed to upload tiles: %w", err)
	}

	image.CreatedAt = time.Now()
	image.UpdatedAt = time.Now()

	return nil
}

func (s *ImgProcService) Cleanup(tmpDir string) error {
	if err := utils.RemoveDir(tmpDir); err != nil {
		return fmt.Errorf("failed to remove temporary directory: %w", err)
	}
	return nil
}
