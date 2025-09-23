package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// UploadHandler handles video upload requests
type UploadHandler struct {
	config       *models.Config
	videoService *services.VideoService
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(config *models.Config, videoService *services.VideoService) *UploadHandler {
	return &UploadHandler{
		config:       config,
		videoService: videoService,
	}
}

// UploadVideo handles video file uploads to a specific directory
func (uh *UploadHandler) UploadVideo(c *fiber.Ctx) error {
	directory := c.Params("directory")
	videoID := c.Params("videoid")

	if directory == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Directory parameter is required",
		})
	}

	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID parameter is required",
		})
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to parse multipart form",
			"details": err.Error(),
		})
	}

	// Get the uploaded file
	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file provided",
			"hint":  "Use 'file' as the form field name",
		})
	}

	file := files[0]

	// Validate file size
	if file.Size > uh.config.Video.MaxUploadSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error":     "File size exceeds limit",
			"max_size":  uh.config.Video.MaxUploadSize,
			"file_size": file.Size,
		})
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !uh.isVideoFile(ext) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "Unsupported file format",
			"extension":         ext,
			"supported_formats": uh.config.Video.SupportedFormats,
		})
	}

	// Validate directory
	err = uh.videoService.SaveUploadedVideo(directory, file.Filename, file.Size)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Find the target directory
	var targetDir *models.VideoDirectory
	for _, dir := range uh.config.Video.Directories {
		if dir.Name == directory && dir.Enabled {
			targetDir = &dir
			break
		}
	}

	if targetDir == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":     "Directory not found or disabled",
			"directory": directory,
		})
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to open uploaded file",
			"details": err.Error(),
		})
	}
	defer src.Close()

	// Create target file path
	filename := videoID + ext
	targetPath := filepath.Join(targetDir.Path, filename)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir.Path, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create target directory",
			"details": err.Error(),
		})
	}

	// Check if file already exists
	if _, err := os.Stat(targetPath); err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":     "File already exists",
			"video_id":  videoID,
			"directory": directory,
			"filename":  filename,
		})
	}

	// Create target file
	dst, err := os.Create(targetPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create target file",
			"details": err.Error(),
		})
	}
	defer dst.Close()

	// Copy file data
	bytesWritten, err := io.Copy(dst, src)
	if err != nil {
		// Clean up partially written file
		os.Remove(targetPath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to save file",
			"details": err.Error(),
		})
	}

	// Verify file size
	if bytesWritten != file.Size {
		os.Remove(targetPath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":    "File size mismatch during upload",
			"expected": file.Size,
			"written":  bytesWritten,
		})
	}

	// Get file info for response
	stat, err := dst.Stat()
	if err != nil {
		stat = nil // Continue without detailed file info
	}

	response := fiber.Map{
		"message":           "Upload successful",
		"video_id":          videoID,
		"directory":         directory,
		"filename":          filename,
		"original_filename": file.Filename,
		"size":              file.Size,
		"bytes_written":     bytesWritten,
		"content_type":      uh.getContentType(ext),
		"path":              targetPath,
	}

	if stat != nil {
		response["modified"] = stat.ModTime().Unix()
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// UploadMultipleVideos handles multiple video uploads to a specific directory
func (uh *UploadHandler) UploadMultipleVideos(c *fiber.Ctx) error {
	directory := c.Params("directory")

	if directory == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Directory parameter is required",
		})
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to parse multipart form",
			"details": err.Error(),
		})
	}

	// Get uploaded files
	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No files provided",
			"hint":  "Use 'files' as the form field name for multiple uploads",
		})
	}

	var results []fiber.Map
	var errors []fiber.Map
	successCount := 0

	for _, file := range files {
		// Generate video ID from filename (without extension)
		videoID := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))

		// Validate and process each file
		result, err := uh.processUploadedFile(file, directory, videoID)
		if err != nil {
			errors = append(errors, fiber.Map{
				"filename": file.Filename,
				"error":    err.Error(),
			})
		} else {
			results = append(results, result)
			successCount++
		}
	}

	response := fiber.Map{
		"message":     fmt.Sprintf("Processed %d files, %d successful, %d failed", len(files), successCount, len(errors)),
		"directory":   directory,
		"total_files": len(files),
		"successful":  successCount,
		"failed":      len(errors),
		"results":     results,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := fiber.StatusCreated
	if len(errors) > 0 && successCount == 0 {
		statusCode = fiber.StatusBadRequest
	} else if len(errors) > 0 {
		statusCode = fiber.StatusPartialContent
	}

	return c.Status(statusCode).JSON(response)
}

// processUploadedFile processes a single uploaded file
func (uh *UploadHandler) processUploadedFile(file *multipart.FileHeader, directory, videoID string) (fiber.Map, error) {
	// Validate file size
	if file.Size > uh.config.Video.MaxUploadSize {
		return nil, fmt.Errorf("file size exceeds limit: %d > %d", file.Size, uh.config.Video.MaxUploadSize)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !uh.isVideoFile(ext) {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	// Validate directory
	err := uh.videoService.SaveUploadedVideo(directory, file.Filename, file.Size)
	if err != nil {
		return nil, err
	}

	// Find target directory
	var targetDir *models.VideoDirectory
	for _, dir := range uh.config.Video.Directories {
		if dir.Name == directory && dir.Enabled {
			targetDir = &dir
			break
		}
	}

	if targetDir == nil {
		return nil, fmt.Errorf("directory not found or disabled: %s", directory)
	}

	// Process file upload
	filename := videoID + ext
	targetPath := filepath.Join(targetDir.Path, filename)

	// Check if file exists
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", filename)
	}

	// Open source file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create target file
	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create target file: %w", err)
	}
	defer dst.Close()

	// Copy data
	bytesWritten, err := io.Copy(dst, src)
	if err != nil {
		os.Remove(targetPath)
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	if bytesWritten != file.Size {
		os.Remove(targetPath)
		return nil, fmt.Errorf("file size mismatch: expected %d, got %d", file.Size, bytesWritten)
	}

	return fiber.Map{
		"video_id":          videoID,
		"filename":          filename,
		"original_filename": file.Filename,
		"size":              file.Size,
		"path":              targetPath,
	}, nil
}

// Helper methods

func (uh *UploadHandler) isVideoFile(ext string) bool {
	for _, supportedExt := range uh.config.Video.SupportedFormats {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

func (uh *UploadHandler) getContentType(ext string) string {
	contentTypes := map[string]string{
		".mp4":  "video/mp4",
		".avi":  "video/avi",
		".mov":  "video/quicktime",
		".mkv":  "video/x-matroska",
		".webm": "video/webm",
		".flv":  "video/x-flv",
		".m4v":  "video/mp4",
		".3gp":  "video/3gpp",
	}
	if ct, exists := contentTypes[ext]; exists {
		return ct
	}
	return "video/mp4" // default
}
