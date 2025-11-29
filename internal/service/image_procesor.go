package service

import (
	"context"
	"log/slog"

	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/infrastructure/processors"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageProcessingService struct {
	logger         *slog.Logger
	dcrawProcessor *processors.DcrawProcessor
	vipsProcessor  *processors.VipsProcessor
	config         *config.Config
}

func NewImageProcessingService(
	logger *slog.Logger,
	cfg *config.Config,
) *ImageProcessingService {
	return &ImageProcessingService{
		logger:         logger,
		dcrawProcessor: processors.NewDcrawProcessor(logger),
		vipsProcessor:  processors.NewVipsProcessor(logger),
		config:         cfg,
	}
}

func (s *ImageProcessingService) ProcessFile(ctx context.Context, file *model.File) (*model.Workspace, error) {

	workspace, err := model.NewWorkspace(file)
	if err != nil {
		return nil, errors.NewStorageError("failed to create workspace").
			WithContext("fileID", file.ID)
	}
	defer func() {
		if err := workspace.Remove(); err != nil {
			s.logger.Warn("Failed to clean up workspace",
				"fileID", file.ID,
				"error", err)
		}
	}()

	// Step 1: Get basic image info
	if err := s.GetImageInfo(ctx, file); err != nil {
		return nil, err
	}

	// Step 3: Handle DNG conversion if needed
	if s.isDNGFile(file) {

		err := s.ConvertDNGToTIFF(ctx, workspace)
		if err != nil {
			return nil, err
		}
	}

	// Step 4: Generate Thumbnail
	err = s.GenerateThumbnail(ctx, workspace)
	if err != nil {
		return nil, err
	}

	// Step 5: Generate DZI
	if err := s.GenerateDZI(ctx, workspace); err != nil {
		return nil, err
	}

	s.logger.Info("File processing workflow completed successfully",
		"fileID", file.ID)

	return workspace, nil

}

func (s *ImageProcessingService) GetImageInfo(ctx context.Context, file *model.File) error {
	s.logger.Info("Getting image info",
		"fileID", file.ID,
		"filename", file.Filename)

	inputFilePath := file.AbsolutePath()
	image_info, err := s.vipsProcessor.GetImageInfo(ctx, inputFilePath)

	if err != nil {
		return err
	}

	file.SetDimensions(image_info.Width, image_info.Height, image_info.Size)
	return nil
}

func (s *ImageProcessingService) isDNGFile(file *model.File) bool {
	ext := file.Extension()
	return ext == ".dng"
}

func (s *ImageProcessingService) ConvertDNGToTIFF(ctx context.Context, workspace *model.Workspace) error {
	s.logger.Info("Converting DNG to TIFF",
		"fileID", workspace.File().ID,
		"filename", workspace.File().Filename)

	inputFilePath := workspace.File().AbsolutePath()
	outputFilePath := workspace.Join(workspace.File().BaseName() + ".tiff")

	result, err := s.dcrawProcessor.DNGToTIFF(ctx, inputFilePath, outputFilePath, s.config.ImageProcessTimeoutMinute.FormatConversion)
	if err != nil {
		s.logger.Error("DNG to TIFF conversion failed",
			"fileID", workspace.File().ID,
			"stdout", result.Stdout,
			"stderr", result.Stderr,
			"error", err)
		return err
	}

	workspace.File().SetFilename(workspace.File().BaseName() + ".tiff")
	workspace.File().SetFormat("tiff")
	workspace.File().SetDir(workspace.Dir())

	s.logger.Info("DNG to TIFF conversion succeeded",
		"fileID", workspace.File().ID,
		"outputFile", outputFilePath)

	return nil
}

func (s *ImageProcessingService) GenerateThumbnail(ctx context.Context, workspace *model.Workspace) error {
	s.logger.Info("Generating thumbnail",
		"fileID", workspace.File().ID,
		"filename", workspace.File().Filename)

	inputFilePath := workspace.File().AbsolutePath()
	outputFilePath := workspace.Join(workspace.File().BaseName() + "_thumbnail.jpg")
	result, err := s.vipsProcessor.CreateThumbnail(ctx, inputFilePath, outputFilePath,
		s.config.ThumbnailConfig.Width,
		s.config.ThumbnailConfig.Height,
		s.config.ThumbnailConfig.Quality)

	if err != nil {
		s.logger.Error("Thumbnail generation failed",
			"fileID", workspace.File().ID,
			"stdout", result.Stdout,
			"stderr", result.Stderr,
			"error", err)
		return err
	}

	s.logger.Info("Thumbnail generation succeeded",
		"fileID", workspace.File().ID,
		"outputFile", outputFilePath)

	return nil
}

func (s *ImageProcessingService) GenerateDZI(ctx context.Context, workspace *model.Workspace) error {
	s.logger.Info("Generating DZI",
		"fileID", workspace.File().ID,
		"filename", workspace.File().Filename)
	inputFilePath := workspace.File().AbsolutePath()
	outputDir := workspace.Dir()

	result, err := s.vipsProcessor.CreateDZI(ctx,
		inputFilePath,
		outputDir,
		s.config.ImageProcessTimeoutMinute.DZIConversion,
		s.config.DZIConfig)
	if err != nil {
		s.logger.Error("DZI generation failed",
			"fileID", workspace.File().ID,
			"stdout", result.Stdout,
			"stderr", result.Stderr,
			"error", err)
		return err
	}

	s.logger.Info("DZI generation succeeded",
		"fileID", workspace.File().ID,
		"outputDir", outputDir)

	return nil
}
