package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"standalone-stream-server/internal/models"
)

// VideoService handles video-related operations
type VideoService struct {
	config *models.Config
}

// NewVideoService creates a new video service
func NewVideoService(config *models.Config) *VideoService {
	return &VideoService{
		config: config,
	}
}

// VideoInfo represents video file information
type VideoInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Size        int64                  `json:"size"`
	Modified    int64                  `json:"modified"`
	ContentType string                 `json:"content_type"`
	Directory   string                 `json:"directory"`
	Path        string                 `json:"path"`
	Extension   string                 `json:"extension"`
	Metadata    VideoMetadata          `json:"metadata,omitempty"`
	StreamURL   string                 `json:"stream_url"`
	Available   bool                   `json:"available"`
}

// VideoMetadata holds additional video information
type VideoMetadata struct {
	Duration    float64 `json:"duration,omitempty"`    // Duration in seconds
	Bitrate     int64   `json:"bitrate,omitempty"`     // Bitrate in bps
	Resolution  string  `json:"resolution,omitempty"`  // e.g., "1920x1080"
	Codec       string  `json:"codec,omitempty"`       // Video codec
	AudioCodec  string  `json:"audio_codec,omitempty"` // Audio codec
	FrameRate   float64 `json:"frame_rate,omitempty"`  // FPS
	Format      string  `json:"format,omitempty"`      // Container format
}

// DirectoryInfo represents directory information
type DirectoryInfo struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	Enabled     bool        `json:"enabled"`
	VideoCount  int         `json:"video_count"`
	TotalSize   int64       `json:"total_size"`
	Videos      []VideoInfo `json:"videos,omitempty"`
}

// ListAllVideos returns all videos from all enabled directories
func (vs *VideoService) ListAllVideos() ([]VideoInfo, error) {
	var allVideos []VideoInfo

	for _, dir := range vs.config.Video.Directories {
		if !dir.Enabled {
			continue
		}

		videos, err := vs.ListVideosInDirectory(dir.Name)
		if err != nil {
			// Log error but continue with other directories
			continue
		}

		allVideos = append(allVideos, videos...)
	}

	return allVideos, nil
}

// ListVideosInDirectory returns videos from a specific directory
func (vs *VideoService) ListVideosInDirectory(directoryName string) ([]VideoInfo, error) {
	dir := vs.findDirectory(directoryName)
	if dir == nil {
		return nil, fmt.Errorf("directory not found: %s", directoryName)
	}

	if !dir.Enabled {
		return nil, fmt.Errorf("directory is disabled: %s", directoryName)
	}

	files, err := os.ReadDir(dir.Path)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", dir.Path, err)
	}

	var videos []VideoInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		ext := strings.ToLower(filepath.Ext(name))
		
		if !vs.isVideoFile(ext) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		video := VideoInfo{
			ID:          vs.generateVideoID(directoryName, strings.TrimSuffix(name, ext)),
			Name:        name,
			Size:        info.Size(),
			Modified:    info.ModTime().Unix(),
			ContentType: vs.getContentType(ext),
			Directory:   directoryName,
			Path:        filepath.Join(dir.Path, name),
			Extension:   ext,
			StreamURL:   fmt.Sprintf("/stream/%s/%s", directoryName, strings.TrimSuffix(name, ext)),
			Available:   true, // File exists and is readable
			Metadata:    vs.extractVideoMetadata(filepath.Join(dir.Path, name), ext),
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetDirectoriesInfo returns information about all directories
func (vs *VideoService) GetDirectoriesInfo() []DirectoryInfo {
	var directories []DirectoryInfo

	for _, dir := range vs.config.Video.Directories {
		dirInfo := DirectoryInfo{
			Name:        dir.Name,
			Path:        dir.Path,
			Description: dir.Description,
			Enabled:     dir.Enabled,
		}

		if dir.Enabled {
			videos, err := vs.ListVideosInDirectory(dir.Name)
			if err == nil {
				dirInfo.VideoCount = len(videos)
				for _, video := range videos {
					dirInfo.TotalSize += video.Size
				}
				// Optionally include videos in response
				// dirInfo.Videos = videos
			}
		}

		directories = append(directories, dirInfo)
	}

	return directories
}

// FindVideoByID finds a video by its ID across all directories
func (vs *VideoService) FindVideoByID(videoID string) (*VideoInfo, error) {
	// Parse video ID to extract directory and filename
	parts := strings.SplitN(videoID, ":", 2)
	if len(parts) != 2 {
		// Fallback: search in all directories
		return vs.findVideoInAllDirectories(videoID)
	}

	directoryName := parts[0]
	filename := parts[1]

	dir := vs.findDirectory(directoryName)
	if dir == nil || !dir.Enabled {
		return nil, fmt.Errorf("directory not found or disabled: %s", directoryName)
	}

	videoPath := vs.findVideoFile(dir.Path, filename)
	if videoPath == "" {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	stat, err := os.Stat(videoPath)
	if err != nil {
		return nil, fmt.Errorf("error getting video info: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(videoPath))
	video := &VideoInfo{
		ID:          videoID,
		Name:        filepath.Base(videoPath),
		Size:        stat.Size(),
		Modified:    stat.ModTime().Unix(),
		ContentType: vs.getContentType(ext),
		Directory:   directoryName,
		Path:        videoPath,
		Extension:   ext,
		StreamURL:   fmt.Sprintf("/stream/%s/%s", directoryName, filename),
		Available:   true,
		Metadata:    vs.extractVideoMetadata(videoPath, ext),
	}

	return video, nil
}

// SaveUploadedVideo saves an uploaded video to the specified directory
func (vs *VideoService) SaveUploadedVideo(directoryName, filename string, size int64) error {
	dir := vs.findDirectory(directoryName)
	if dir == nil || !dir.Enabled {
		return fmt.Errorf("directory not found or disabled: %s", directoryName)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if !vs.isVideoFile(ext) {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	// Check upload size limit
	if size > vs.config.Video.MaxUploadSize {
		return fmt.Errorf("file size exceeds limit: %d > %d", size, vs.config.Video.MaxUploadSize)
	}

	return nil
}

// Helper methods

func (vs *VideoService) findDirectory(name string) *models.VideoDirectory {
	for _, dir := range vs.config.Video.Directories {
		if dir.Name == name {
			return &dir
		}
	}
	return nil
}

func (vs *VideoService) findVideoInAllDirectories(videoID string) (*VideoInfo, error) {
	for _, dir := range vs.config.Video.Directories {
		if !dir.Enabled {
			continue
		}

		videoPath := vs.findVideoFile(dir.Path, videoID)
		if videoPath != "" {
			stat, err := os.Stat(videoPath)
			if err != nil {
				continue
			}

			ext := strings.ToLower(filepath.Ext(videoPath))
			video := &VideoInfo{
				ID:          vs.generateVideoID(dir.Name, videoID),
				Name:        filepath.Base(videoPath),
				Size:        stat.Size(),
				Modified:    stat.ModTime().Unix(),
				ContentType: vs.getContentType(ext),
				Directory:   dir.Name,
				Path:        videoPath,
				Extension:   ext,
			}

			return video, nil
		}
	}

	return nil, fmt.Errorf("video not found: %s", videoID)
}

func (vs *VideoService) findVideoFile(dirPath, videoID string) string {
	for _, ext := range vs.config.Video.SupportedFormats {
		path := filepath.Join(dirPath, videoID+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (vs *VideoService) isVideoFile(ext string) bool {
	for _, supportedExt := range vs.config.Video.SupportedFormats {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

func (vs *VideoService) getContentType(ext string) string {
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

func (vs *VideoService) generateVideoID(directory, filename string) string {
	return fmt.Sprintf("%s:%s", directory, filename)
}

// GetStats returns overall video statistics
func (vs *VideoService) GetStats() map[string]interface{} {
	totalVideos := 0
	totalSize := int64(0)
	enabledDirs := 0

	for _, dir := range vs.config.Video.Directories {
		if !dir.Enabled {
			continue
		}

		enabledDirs++
		videos, err := vs.ListVideosInDirectory(dir.Name)
		if err != nil {
			continue
		}

		totalVideos += len(videos)
		for _, video := range videos {
			totalSize += video.Size
		}
	}

	return map[string]interface{}{
		"total_videos":      totalVideos,
		"total_size":        totalSize,
		"enabled_directories": enabledDirs,
		"total_directories": len(vs.config.Video.Directories),
		"supported_formats": vs.config.Video.SupportedFormats,
		"max_upload_size":   vs.config.Video.MaxUploadSize,
		"last_updated":      time.Now().Unix(),
	}
}

// extractVideoMetadata extracts basic metadata from video files
func (vs *VideoService) extractVideoMetadata(filePath, ext string) VideoMetadata {
	metadata := VideoMetadata{
		Format: strings.TrimPrefix(ext, "."),
	}

	// For now, we'll extract basic file-based metadata
	// In a production system, you'd want to use ffprobe or similar
	if stat, err := os.Stat(filePath); err == nil {
		// Estimate duration based on file size and codec (very rough estimate)
		switch ext {
		case ".mp4", ".mov", ".m4v":
			metadata.Codec = "H.264"
			metadata.AudioCodec = "AAC"
			// Rough estimate: ~1MB per minute for standard quality
			if stat.Size() > 0 {
				metadata.Duration = float64(stat.Size()) / (1024 * 1024) * 60 // Very rough estimate
			}
		case ".webm":
			metadata.Codec = "VP8/VP9"
			metadata.AudioCodec = "Vorbis/Opus"
		case ".mkv":
			metadata.Codec = "Various"
			metadata.AudioCodec = "Various"
		case ".avi":
			metadata.Codec = "Various"
			metadata.AudioCodec = "Various"
		}

		// Set common defaults
		if metadata.Duration > 0 && metadata.Duration < 1 {
			metadata.Duration = 1 // Minimum 1 second
		}
		if metadata.Bitrate == 0 && metadata.Duration > 0 {
			metadata.Bitrate = int64(float64(stat.Size()) * 8 / metadata.Duration) // bits per second
		}
	}

	return metadata
}

// ValidateVideoFile checks if a video file is properly accessible and valid
func (vs *VideoService) ValidateVideoFile(filePath string) error {
	// Check if file exists and is readable
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !vs.isVideoFile(ext) {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	// Check file size
	if stat, err := os.Stat(filePath); err == nil {
		if stat.Size() == 0 {
			return fmt.Errorf("video file is empty")
		}
		if stat.Size() > vs.config.Video.MaxUploadSize {
			return fmt.Errorf("video file exceeds size limit: %d > %d", stat.Size(), vs.config.Video.MaxUploadSize)
		}
	}

	return nil
}