package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/histopathai/image-processing-service/internal/domain/model"
	"github.com/histopathai/image-processing-service/internal/infrastructure/processors"
	"github.com/histopathai/image-processing-service/internal/infrastructure/storage"
	"github.com/histopathai/image-processing-service/pkg/config"
	"github.com/histopathai/image-processing-service/pkg/errors"
)

type ImageProcessingService struct {
	logger            *slog.Logger
	dcrawProcessor    *processors.DcrawProcessor
	vipsProcessor     *processors.VipsProcessor
	fileInfoProcessor *processors.ImageInfoProcessor
	zipProcessor      *processors.ZipProcessor
	inputStorage      storage.InputStorage
	outputStorage     storage.OutputStorage
	config            *config.Config
}

func NewImageProcessingService(
	logger *slog.Logger,
	cfg *config.Config,
	inputStorage storage.InputStorage,
	outputStorage storage.OutputStorage,
) *ImageProcessingService {
	return &ImageProcessingService{
		logger:            logger,
		dcrawProcessor:    processors.NewDcrawProcessor(logger),
		vipsProcessor:     processors.NewVipsProcessor(logger),
		fileInfoProcessor: processors.NewImageInfoProcessor(logger),
		zipProcessor:      processors.NewZipProcessor(logger),
		inputStorage:      inputStorage,
		outputStorage:     outputStorage,
		config:            cfg,
	}
}

func (s *ImageProcessingService) ProcessFile(ctx context.Context, file *model.File, container string) (*model.Workspace, error) {
	// Create workspace in /tmp (ephemeral, instance-local storage)
	workspace, err := model.NewWorkspace(file)
	if err != nil {
		return nil, errors.NewStorageError("failed to create workspace").
			WithContext("fileID", file.ID)
	}

	s.logger.Info("Created workspace in /tmp",
		"fileID", file.ID,
		"workspace", workspace.Dir())

	// Step 1: Use original file directly from its location (no copying needed)
	// For local development, inputPath is already an absolute path
	// For cloud, it would be a path within the mounted GCS bucket
	originalFilePath := file.Filename

	s.logger.Info("Using original file directly",
		"fileID", file.ID,
		"original_path", originalFilePath)

	// Update file to point to the original file location
	// Extract directory and filename from the original path
	originalDir := filepath.Dir(originalFilePath)
	originalFilename := filepath.Base(originalFilePath)

	file.SetDir(originalDir)
	file.SetFilename(originalFilename)

	// Step 2: Process file in /tmp workspace
	wasDNGFile := s.isDNGFile(file)
	tiffFilename := ""

	if err := s.GetImageInfo(ctx, file); err != nil {
		return nil, err
	}

	if wasDNGFile {
		tiffFilename, err = s.ConvertDNGToTIFF(ctx, file, workspace)
		if err != nil {
			return nil, err
		}
	}

	if err := s.GenerateThumbnail(ctx, file, workspace); err != nil {
		return nil, err
	}

	if err := s.GenerateDZI(ctx, file, workspace, container); err != nil {
		return nil, err
	}

	// Step 3: Post-process based on container type
	if container == "zip" {
		// Build index map for zip container
		if err := s.zipProcessor.BuildIndexMap(ctx, workspace.Join("image.zip"), workspace.Dir()); err != nil {
			return nil, err
		}

		// Extract image.dzi from zip so it can be uploaded as a separate file
		if err := s.zipProcessor.ExtractDesiredFile(ctx, workspace.Join("image.zip"), "image.dzi", workspace.Join("image.dzi")); err != nil {
			return nil, err
		}
	} else {
		// container == "fs"
		// vips generates "image_files", rename it to "tiles" as expected by output validation
		oldPath := workspace.Join("image_files")
		newPath := workspace.Join("tiles")
		if err := os.Rename(oldPath, newPath); err != nil {
			return nil, errors.WrapStorageError(err, "failed to rename tiles directory").
				WithContext("old", oldPath).
				WithContext("new", newPath)
		}
	}

	// Step 4: Validate outputs before copying to storage
	if err := s.validateOutputs(workspace, container); err != nil {
		return nil, err
	}

	s.logger.Info("File processing workflow completed successfully",
		"fileID", file.ID)

	// Step 5: Copy outputs to destination storage
	if err := s.copyOutputsToStorage(ctx, workspace, file.ID, container); err != nil {
		return nil, err
	}

	// Cleanup: Remove converted TIFF file if it was created
	if wasDNGFile && tiffFilename != "" {
		tiffPath := workspace.Join(tiffFilename)
		if err := workspace.RemoveFile(tiffPath); err != nil {
			s.logger.Warn("Failed to remove converted TIFF file from workspace",
				"fileID", file.ID,
				"tiffPath", tiffPath,
				"error", err)
		} else {
			s.logger.Info("Removed converted TIFF file from workspace",
				"fileID", file.ID,
				"tiffPath", tiffPath)
		}
	}

	return workspace, nil
}

func (s *ImageProcessingService) GetImageInfo(ctx context.Context, file *model.File) error {
	s.logger.Info("Getting image info",
		"fileID", file.ID,
		"filename", file.Filename)

	inputFilePath := file.AbsolutePath()
	imageInfo, err := s.fileInfoProcessor.GetImageInfo(ctx, inputFilePath)

	if err != nil {
		return err
	}

	file.SetDimensions(imageInfo.Width, imageInfo.Height, imageInfo.Size)
	return nil
}

func (s *ImageProcessingService) isDNGFile(file *model.File) bool {
	ext := file.Extension()
	return ext == ".dng"
}

func (s *ImageProcessingService) ConvertDNGToTIFF(ctx context.Context, file *model.File, workspace *model.Workspace) (string, error) {
	s.logger.Info("Converting DNG to TIFF",
		"fileID", file.ID,
		"filename", file.Filename)

	inputFilePath := file.AbsolutePath()
	tiffFilename := file.BaseName() + ".tiff"
	outputFilePath := workspace.Join(tiffFilename)

	result, err := s.dcrawProcessor.DNGToTIFF(ctx, inputFilePath, outputFilePath, s.config.ImageProcessTimeoutMinute.FormatConversion)
	if err != nil {
		stdout := ""
		stderr := ""
		if result != nil {
			stdout = result.Stdout
			stderr = result.Stderr
		}
		s.logger.Error("DNG to TIFF conversion failed",
			"fileID", file.ID,
			"stdout", stdout,
			"stderr", stderr,
			"error", err)
		return "", err
	}

	s.logger.Info("DNG to TIFF conversion succeeded",
		"fileID", file.ID,
		"outputFile", outputFilePath)

	return tiffFilename, nil
}

func (s *ImageProcessingService) GenerateThumbnail(ctx context.Context, file *model.File, workspace *model.Workspace) error {
	s.logger.Info("Generating thumbnail",
		"fileID", file.ID,
		"filename", file.Filename)

	var inputFilePath string

	// DNG ise workspace'teki TIFF'i kullan, değilse orijinal dosyayı kullan
	if s.isDNGFile(file) {
		tiffFilename := file.BaseName() + ".tiff"
		inputFilePath = workspace.Join(tiffFilename)
	} else {
		inputFilePath = file.AbsolutePath()
	}

	outputFilePath := workspace.Join("thumbnail.jpg")

	result, err := s.vipsProcessor.CreateThumbnail(ctx, inputFilePath, outputFilePath,
		s.config.ThumbnailConfig.Width,
		s.config.ThumbnailConfig.Height,
		s.config.ThumbnailConfig.Quality)

	if err != nil {
		stdout := ""
		stderr := ""
		if result != nil {
			stdout = result.Stdout
			stderr = result.Stderr
		}
		s.logger.Error("Thumbnail generation failed",
			"fileID", file.ID,
			"stdout", stdout,
			"stderr", stderr,
			"error", err)
		return err
	}

	s.logger.Info("Thumbnail generation succeeded",
		"fileID", file.ID,
		"outputFile", outputFilePath)

	return nil
}

func (s *ImageProcessingService) GenerateDZI(ctx context.Context, file *model.File, workspace *model.Workspace, container string) error {
	s.logger.Info("Generating DZI",
		"fileID", file.ID,
		"filename", file.Filename)

	var inputFilePath string

	if s.isDNGFile(file) {
		tiffFilename := file.BaseName() + ".tiff"
		inputFilePath = workspace.Join(tiffFilename)
	} else {
		inputFilePath = file.AbsolutePath()
	}

	outputBase := workspace.Join("image")

	dziConfig := s.config.DZIConfig
	if container == "zip" && dziConfig.Compression > 9 {
		s.logger.Warn("DZI compression level out of range for zip container, clamping to 0",
			"compression", dziConfig.Compression)
		dziConfig.Compression = 0
	}

	result, err := s.vipsProcessor.CreateDZI(ctx,
		inputFilePath,
		outputBase,
		s.config.ImageProcessTimeoutMinute.DZIConversion,
		dziConfig, container)

	if err != nil {
		stdout := ""
		stderr := ""
		if result != nil {
			stdout = result.Stdout
			stderr = result.Stderr
		}
		s.logger.Error("DZI generation failed",
			"fileID", file.ID,
			"stdout", stdout,
			"stderr", stderr,
			"error", err)
		return err
	}

	s.logger.Info("DZI generation succeeded",
		"fileID", file.ID,
		"outputBase", outputBase)

	return nil

}
