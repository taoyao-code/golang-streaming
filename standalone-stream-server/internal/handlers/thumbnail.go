package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"
	"standalone-stream-server/internal/utils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// ThumbnailHandler handles thumbnail generation and serving
type ThumbnailHandler struct {
	config          *models.Config
	videoService    *services.VideoService
	metadataService *services.MetadataService
}

// NewThumbnailHandler creates a new thumbnail handler
func NewThumbnailHandler(config *models.Config, videoService *services.VideoService, metadataService *services.MetadataService) *ThumbnailHandler {
	return &ThumbnailHandler{
		config:          config,
		videoService:    videoService,
		metadataService: metadataService,
	}
}

// GetThumbnail generates and serves a thumbnail for a video
func (th *ThumbnailHandler) GetThumbnail(c *fiber.Ctx) error {
	videoID := c.Params("videoid")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	start := time.Now()
	defer func() {
		utils.RecordHTTPRequest(c.Method(), "/api/thumbnail/:videoid", fmt.Sprintf("%d", c.Response().StatusCode()), time.Since(start))
	}()

	// Parse video ID to get directory and filename
	parts := strings.SplitN(videoID, ":", 2)
	if len(parts) != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid video ID format",
			"details": "Video ID should be in format 'directory:filename'",
		})
	}

	directory := parts[0]
	filename := parts[1]

	// Find the video file
	videoInfo, err := th.videoService.FindVideoByID(videoID)
	if err != nil {
		utils.LogError("thumbnail_find_video", err,
			zap.String("video_id", videoID),
			zap.String("directory", directory),
			zap.String("filename", filename),
		)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Video not found",
			"details": err.Error(),
		})
	}

	videoPath := videoInfo.Path

	// Check if video file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Video file not found",
			"details": fmt.Sprintf("File does not exist: %s", videoPath),
		})
	}

	// Generate thumbnail path
	thumbnailDir := "./thumbnails"
	thumbnailFilename := fmt.Sprintf("%s_%s.jpg", directory, filename)
	thumbnailPath := filepath.Join(thumbnailDir, thumbnailFilename)

	// Check if thumbnail already exists
	if _, err := os.Stat(thumbnailPath); err == nil {
		// Serve existing thumbnail
		return c.SendFile(thumbnailPath)
	}

	// Extract video metadata to get optimal thumbnail timestamp
	metadata, err := th.metadataService.ExtractMetadata(videoPath)
	if err != nil {
		utils.LogError("thumbnail_extract_metadata", err,
			zap.String("video_path", videoPath),
		)
		// Continue with default timestamp
	}

	// Determine thumbnail timestamp
	timestamp := th.metadataService.GetOptimalThumbnailTimestamp(metadata.Duration)

	// Generate thumbnail
	if err := th.metadataService.GenerateThumbnail(videoPath, thumbnailPath, timestamp); err != nil {
		utils.LogError("thumbnail_generation", err,
			zap.String("video_path", videoPath),
			zap.String("thumbnail_path", thumbnailPath),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to generate thumbnail",
			"details": err.Error(),
		})
	}

	utils.Logger.Info("Thumbnail generated and served",
		zap.String("video_id", videoID),
		zap.String("thumbnail_path", thumbnailPath),
		zap.Duration("generation_time", time.Since(start)),
	)

	// Serve the generated thumbnail
	return c.SendFile(thumbnailPath)
}

// ListThumbnails returns a list of available thumbnails
func (th *ThumbnailHandler) ListThumbnails(c *fiber.Ctx) error {
	start := time.Now()
	defer func() {
		utils.RecordHTTPRequest(c.Method(), "/api/thumbnails", fmt.Sprintf("%d", c.Response().StatusCode()), time.Since(start))
	}()

	thumbnailDir := "./thumbnails"
	
	// Create thumbnails directory if it doesn't exist
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to access thumbnails directory",
		})
	}

	files, err := os.ReadDir(thumbnailDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read thumbnails directory",
		})
	}

	var thumbnails []map[string]interface{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		thumbnails = append(thumbnails, map[string]interface{}{
			"filename":  file.Name(),
			"size":      info.Size(),
			"modified":  info.ModTime().Unix(),
			"url":       fmt.Sprintf("/api/thumbnail/file/%s", file.Name()),
		})
	}

	return c.JSON(fiber.Map{
		"thumbnails": thumbnails,
		"count":      len(thumbnails),
	})
}

// ServeThumbnailFile serves a thumbnail file by filename
func (th *ThumbnailHandler) ServeThumbnailFile(c *fiber.Ctx) error {
	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	start := time.Now()
	defer func() {
		utils.RecordHTTPRequest(c.Method(), "/api/thumbnail/file/:filename", fmt.Sprintf("%d", c.Response().StatusCode()), time.Since(start))
	}()

	// Security check: ensure filename doesn't contain path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid filename",
		})
	}

	thumbnailPath := filepath.Join("./thumbnails", filename)
	
	// Check if file exists
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Thumbnail not found",
		})
	}

	return c.SendFile(thumbnailPath)
}