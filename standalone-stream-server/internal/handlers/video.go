package handlers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"standalone-stream-server/internal/middleware"
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// VideoHandler 处理视频相关请求
type VideoHandler struct {
	config             *models.Config
	videoService       *services.VideoService
	streamingFlowController *middleware.StreamingFlowController
}

// NewVideoHandler 创建新的视频处理器
func NewVideoHandler(config *models.Config, videoService *services.VideoService) *VideoHandler {
	// Use configurable tokens per second, fallback to 1/4 of max connections if not set
	tokensPerSecond := config.Server.TokensPerSecond
	if tokensPerSecond == 0 {
		// Default: 1/4 of max connections (legacy behavior)
		tokensPerSecond = config.Server.MaxConns / 4
	}
	
	// Create streaming flow controller based on config
	streamingFlowController := middleware.NewStreamingFlowController(
		config.Server.MaxConns, // max connections
		tokensPerSecond,        // tokens per second
	)
	
	return &VideoHandler{
		config:                  config,
		videoService:            videoService,
		streamingFlowController: streamingFlowController,
	}
}

// ListAllVideos 返回所有启用目录中的所有视频
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

// ListVideosInDirectory 返回指定目录中的视频
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

// ListDirectories 返回所有视频目录的信息
func (vh *VideoHandler) ListDirectories(c *fiber.Ctx) error {
	directories := vh.videoService.GetDirectoriesInfo()

	// 添加查询参数以在响应中包含视频
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

// StreamVideo 流式传输视频文件，支持范围请求
func (vh *VideoHandler) StreamVideo(c *fiber.Ctx) error {
	videoID := c.Params("videoid")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	// 查找视频
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
	// Apply flow control for streaming requests
	allowed, reason := vh.streamingFlowController.CheckAccess()
	if !allowed {
		errorMsg := "Server busy"
		if reason == "rate_limited" {
			errorMsg = "Rate limit exceeded"
		} else if reason == "connection_limited" {
			errorMsg = "Too many concurrent connections"
		}
		
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":  errorMsg,
			"reason": reason,
		})
	}
	
	// Ensure connection is released when streaming completes
	defer vh.streamingFlowController.ReleaseConnection()
	
	// 首先获取文件信息
	stat, err := os.Stat(video.Path)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get file information",
			"details": err.Error(),
		})
	}

	// 设置头
	c.Set("Content-Type", video.ContentType)
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.Set("Cache-Control", vh.config.Video.StreamingSettings.CacheControl)
	c.Set("Last-Modified", stat.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	// 处理范围请求
	rangeHeader := c.Get("Range")
	if rangeHeader != "" && vh.config.Video.StreamingSettings.RangeSupport {
		// 对于范围请求，我们仍需要手动打开文件
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

	// 发送整个文件 - 使用 SendFile 以获得更好的兼容性
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

	// 设置头 for partial content
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

// GetVideoInfo 返回特定视频的详细信息
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

// SearchVideos 在所有目录中按名称搜索视频
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

// StreamVideoByDirectory 从指定目录流式传输视频文件（支持多层级路径）
func (vh *VideoHandler) StreamVideoByDirectory(c *fiber.Ctx) error {
	directory := c.Params("directory")
	videoPath := c.Params("*") // 使用通配符获取完整路径

	if directory == "" || videoPath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Both directory and video path parameters are required",
		})
	}

	// 构建完整的视频ID
	fullVideoID := directory + ":" + videoPath

	// 查找视频
	video, err := vh.videoService.FindVideoByID(fullVideoID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":      "Video not found",
			"directory":  directory,
			"video_path": videoPath,
			"full_id":    fullVideoID,
			"details":    err.Error(),
		})
	}

	return vh.streamVideoFile(c, video)
}

// ValidateVideo 验证视频文件是否可访问且格式正确
func (vh *VideoHandler) ValidateVideo(c *fiber.Ctx) error {
	videoID := c.Params("video-id")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}

	// 查找视频
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

// GetFlowControlStats returns flow control statistics
func (vh *VideoHandler) GetFlowControlStats(c *fiber.Ctx) error {
	stats := vh.streamingFlowController.GetDetailedStats()
	
	return c.JSON(fiber.Map{
		"flow_control": stats,
		"timestamp":    c.Context().Time().Unix(),
	})
}
