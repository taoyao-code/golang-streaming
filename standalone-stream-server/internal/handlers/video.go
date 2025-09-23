package handlers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// VideoHandler handles video-related requests
type VideoHandler struct {
	config       *models.Config
	videoService *services.VideoService
}

// NewVideoHandler creates a new video handler
func NewVideoHandler(config *models.Config, videoService *services.VideoService) *VideoHandler {
	return &VideoHandler{
		config:       config,
		videoService: videoService,
	}
}

// ListAllVideos returns all videos from all enabled directories
func (vh *VideoHandler) ListAllVideos(c *fiber.Ctx) error {
	videos, err := vh.videoService.ListAllVideos()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to list videos",
			"details": err.Error(),
		})
	}

	response := fiber.Map{
		"videos": videos,
		"count":  len(videos),
		"directories": func() []string {
			var dirs []string
			for _, dir := range vh.config.Video.Directories {
				if dir.Enabled {
					dirs = append(dirs, dir.Name)
				}
			}
			return dirs
		}(),
	}

	return c.JSON(response)
}

// ListVideosInDirectory returns videos from a specific directory
func (vh *VideoHandler) ListVideosInDirectory(c *fiber.Ctx) error {
	directory := c.Params("directory")
	if directory == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Directory parameter is required",
		})
	}

	videos, err := vh.videoService.ListVideosInDirectory(directory)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   fmt.Sprintf("Failed to list videos in directory: %s", directory),
			"details": err.Error(),
		})
	}

	response := fiber.Map{
		"directory": directory,
		"videos":    videos,
		"count":     len(videos),
	}

	return c.JSON(response)
}

// ListDirectories returns information about all video directories
func (vh *VideoHandler) ListDirectories(c *fiber.Ctx) error {
	directories := vh.videoService.GetDirectoriesInfo()

	// Add query parameter to include videos in response
	includeVideos := c.Query("include_videos", "false")
	if includeVideos == "true" {
		for i := range directories {
			if directories[i].Enabled {
				videos, err := vh.videoService.ListVideosInDirectory(directories[i].Name)
				if err == nil {
					directories[i].Videos = videos
				}
			}
		}
	}

	response := fiber.Map{
		"directories": directories,
		"count":       len(directories),
		"enabled_count": func() int {
			count := 0
			for _, dir := range directories {
				if dir.Enabled {
					count++
				}
			}
			return count
		}(),
	}

	return c.JSON(response)
}

// StreamVideo streams a video file with range support
func (vh *VideoHandler) StreamVideo(c *fiber.Ctx) error {
	videoID := c.Params("videoid")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	// Find the video
	video, err := vh.videoService.FindVideoByID(videoID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":    "Video not found",
			"video_id": videoID,
			"details":  err.Error(),
		})
	}

	return vh.streamVideoFile(c, video)
}

// streamVideoFile handles the actual streaming logic for both streaming methods
func (vh *VideoHandler) streamVideoFile(c *fiber.Ctx, video *services.VideoInfo) error {
	// Get file info first
	stat, err := os.Stat(video.Path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get file information",
			"details": err.Error(),
		})
	}

	// Set headers
	c.Set("Content-Type", video.ContentType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.Set("Cache-Control", vh.config.Video.StreamingSettings.CacheControl)
	c.Set("Last-Modified", stat.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// Handle range requests
	rangeHeader := c.Get("Range")
	if rangeHeader != "" && vh.config.Video.StreamingSettings.RangeSupport {
		// For range requests, we still need to open the file manually
		file, err := os.Open(video.Path)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to open video file",
				"details": err.Error(),
			})
		}
		defer file.Close()
		return vh.handleRangeRequest(c, file, stat.Size(), rangeHeader)
	}

	// Send entire file - use SendFile for better compatibility
	return c.SendFile(video.Path)
}

// handleRangeRequest handles HTTP range requests for video seeking
func (vh *VideoHandler) handleRangeRequest(c *fiber.Ctx, file *os.File, fileSize int64, rangeHeader string) error {
	// Parse range header (format: "bytes=start-end")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).JSON(fiber.Map{
			"error": "Invalid range format",
		})
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	rangeParts := strings.Split(rangeSpec, "-")

	if len(rangeParts) != 2 {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).JSON(fiber.Map{
			"error": "Invalid range specification",
		})
	}

	var start, end int64
	var err error

	// Parse start
	if rangeParts[0] != "" {
		start, err = strconv.ParseInt(rangeParts[0], 10, 64)
		if err != nil || start < 0 {
			return c.Status(fiber.StatusRequestedRangeNotSatisfiable).JSON(fiber.Map{
				"error": "Invalid start range",
			})
		}
	}

	// Parse end
	if rangeParts[1] != "" {
		end, err = strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil || end >= fileSize {
			end = fileSize - 1
		}
	} else {
		end = fileSize - 1
	}

	// Validate range
	if start > end || start >= fileSize {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).JSON(fiber.Map{
			"error": "Invalid range values",
		})
	}

	// Calculate content length
	contentLength := end - start + 1

	// Set headers for partial content
	c.Status(fiber.StatusPartialContent)
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Set("Content-Length", strconv.FormatInt(contentLength, 10))

	// Seek to start position
	if _, err := file.Seek(start, 0); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to seek in file",
			"details": err.Error(),
		})
	}

	// Send the requested range
	buffer := make([]byte, vh.config.Video.StreamingSettings.ChunkSize)
	remaining := contentLength

	for remaining > 0 {
		chunkSize := vh.config.Video.StreamingSettings.ChunkSize
		if remaining < int64(chunkSize) {
			chunkSize = int(remaining)
		}

		n, err := file.Read(buffer[:chunkSize])
		if err != nil {
			break
		}

		if _, err := c.Response().BodyWriter().Write(buffer[:n]); err != nil {
			break
		}

		remaining -= int64(n)
	}

	return nil
}

// GetVideoInfo returns detailed information about a specific video
func (vh *VideoHandler) GetVideoInfo(c *fiber.Ctx) error {
	videoID := c.Params("video-id")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	video, err := vh.videoService.FindVideoByID(videoID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":    "Video not found",
			"video_id": videoID,
			"details":  err.Error(),
		})
	}

	return c.JSON(video)
}

// SearchVideos searches for videos by name across all directories
func (vh *VideoHandler) SearchVideos(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	allVideos, err := vh.videoService.ListAllVideos()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to search videos",
			"details": err.Error(),
		})
	}

	// Simple text search in video names
	var matchedVideos []services.VideoInfo
	searchTerm := strings.ToLower(query)

	for _, video := range allVideos {
		if strings.Contains(strings.ToLower(video.Name), searchTerm) ||
			strings.Contains(strings.ToLower(video.ID), searchTerm) {
			matchedVideos = append(matchedVideos, video)
		}
	}

	// Ensure videos is always an array, even if empty
	if matchedVideos == nil {
		matchedVideos = []services.VideoInfo{}
	}

	response := fiber.Map{
		"query":  query,
		"videos": matchedVideos,
		"count":  len(matchedVideos),
		"total":  len(allVideos),
	}

	return c.JSON(response)
}

// StreamVideoByDirectory streams a video file from a specific directory
func (vh *VideoHandler) StreamVideoByDirectory(c *fiber.Ctx) error {
	directory := c.Params("directory")
	videoID := c.Params("videoid")

	if directory == "" || videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Both directory and video ID parameters are required",
		})
	}

	// Construct the full video ID
	fullVideoID := directory + ":" + videoID

	// Find the video
	video, err := vh.videoService.FindVideoByID(fullVideoID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":     "Video not found",
			"directory": directory,
			"video_id":  videoID,
			"details":   err.Error(),
		})
	}

	return vh.streamVideoFile(c, video)
}

// ValidateVideo validates if a video file is accessible and properly formatted
func (vh *VideoHandler) ValidateVideo(c *fiber.Ctx) error {
	videoID := c.Params("video-id")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	// Find the video
	video, err := vh.videoService.FindVideoByID(videoID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":    "Video not found",
			"video_id": videoID,
			"details":  err.Error(),
		})
	}

	// Validate the video file
	if err := vh.videoService.ValidateVideoFile(video.Path); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":    "Video validation failed",
			"video_id": videoID,
			"details":  err.Error(),
			"valid":    false,
		})
	}

	return c.JSON(fiber.Map{
		"video_id": videoID,
		"valid":    true,
		"message":  "Video file is valid and ready for streaming",
		"video":    video,
	})
}
