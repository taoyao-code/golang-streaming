package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"standalone-stream-server/internal/models"
)

// VideoService 处理视频相关操作
type VideoService struct {
	config            *models.Config
	metadataExtractor *VideoMetadataExtractor
}

// NewVideoService 创建新的视频服务
func NewVideoService(config *models.Config) *VideoService {
	return &VideoService{
		config:            config,
		metadataExtractor: NewVideoMetadataExtractor(),
	}
}

// VideoInfo 表示视频文件信息
type VideoInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Size        int64         `json:"size"`
	Modified    int64         `json:"modified"`
	ContentType string        `json:"content_type"`
	Directory   string        `json:"directory"`
	Path        string        `json:"path"`
	Extension   string        `json:"extension"`
	Metadata    VideoMetadata `json:"metadata,omitempty"`
	StreamURL   string        `json:"stream_url"`
	Available   bool          `json:"available"`
}

// VideoMetadata 保存额外的视频信息
type VideoMetadata struct {
	Duration     float64 `json:"duration_seconds,omitempty"` // Duration in seconds
	DurationStr  string  `json:"duration,omitempty"`         // Human readable duration
	Width        int     `json:"width,omitempty"`            // Video width
	Height       int     `json:"height,omitempty"`           // Video height
	Resolution   string  `json:"resolution,omitempty"`       // e.g., "1920x1080"
	Bitrate      int64   `json:"bitrate,omitempty"`          // Bitrate in bps
	FrameRate    float64 `json:"frame_rate,omitempty"`       // FPS
	VideoCodec   string  `json:"video_codec,omitempty"`      // Video codec
	AudioCodec   string  `json:"audio_codec,omitempty"`      // Audio codec
	Format       string  `json:"format,omitempty"`           // Container format
	AspectRatio  string  `json:"aspect_ratio,omitempty"`     // Aspect ratio
	HasAudio     bool    `json:"has_audio,omitempty"`        // Has audio track
	HasVideo     bool    `json:"has_video,omitempty"`        // Has video track
	ThumbnailURL string  `json:"thumbnail_url,omitempty"`    // Thumbnail URL if available
}

// DirectoryInfo 表示目录信息
type DirectoryInfo struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	Enabled     bool        `json:"enabled"`
	VideoCount  int         `json:"video_count"`
	TotalSize   int64       `json:"total_size"`
	Videos      []VideoInfo `json:"videos,omitempty"`
}

// ListAllVideos 返回所有启用的目录中的所有视频
func (vs *VideoService) ListAllVideos() ([]VideoInfo, error) {
	var allVideos []VideoInfo

	for _, dir := range vs.config.Video.Directories {
		if !dir.Enabled {
			continue
		}

		videos, err := vs.ListVideosInDirectory(dir.Name)
		if err != nil {
			// 记录错误但继续处理其他目录
			continue
		}

		allVideos = append(allVideos, videos...)
	}

	return allVideos, nil
}

// ListVideosInDirectory 返回特定目录中的视频（支持递归扫描子目录）
func (vs *VideoService) ListVideosInDirectory(directoryName string) ([]VideoInfo, error) {
	dir := vs.findDirectory(directoryName)
	if dir == nil {
		return nil, fmt.Errorf("directory not found: %s", directoryName)
	}

	if !dir.Enabled {
		return nil, fmt.Errorf("directory is disabled: %s", directoryName)
	}

	return vs.scanDirectoryRecursive(dir.Path, directoryName, "", 0)
}

// scanDirectoryRecursive 递归扫描目录以查找视频文件
func (vs *VideoService) scanDirectoryRecursive(basePath, dirName, currentPath string, depth int) ([]VideoInfo, error) {
	// 限制递归深度，防止无限递归或性能问题
	const maxDepth = 10
	if depth > maxDepth {
		return nil, fmt.Errorf("directory depth exceeds maximum allowed depth (%d)", maxDepth)
	}

	fullPath := filepath.Join(basePath, currentPath)

	// 检查是否为符号链接，避免循环引用
	if info, err := os.Lstat(fullPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		// 对于符号链接，我们跳过以避免潜在的循环引用
		return []VideoInfo{}, nil
	}

	files, err := os.ReadDir(fullPath)
	if err != nil {
		// 如果无法读取目录（权限问题等），返回空列表而不是错误
		// 但记录调试信息以便排查问题
		// fmt.Printf("Warning: Unable to read directory %s: %v\n", fullPath, err)
		return []VideoInfo{}, nil
	}

	var videos []VideoInfo
	for _, file := range files {
		fileName := file.Name()
		// 跳过隐藏文件和特殊目录
		if strings.HasPrefix(fileName, ".") {
			continue
		}

		filePath := filepath.Join(currentPath, fileName)
		fullFilePath := filepath.Join(basePath, filePath)

		if file.IsDir() {
			// 递归处理子目录
			subVideos, err := vs.scanDirectoryRecursive(basePath, dirName, filePath, depth+1)
			if err == nil {
				videos = append(videos, subVideos...)
			}
			continue
		}

		// 处理文件
		ext := strings.ToLower(filepath.Ext(fileName))
		if !vs.isVideoFile(ext) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// 生成相对路径（用于ID和URL）
		relativeVideoPath := strings.TrimSuffix(filePath, ext)
		if currentPath == "" {
			// 根目录下的文件，保持原有格式兼容性
			relativeVideoPath = strings.TrimSuffix(fileName, ext)
		}

		video := VideoInfo{
			ID:          vs.generateVideoID(dirName, relativeVideoPath),
			Name:        fileName,
			Size:        info.Size(),
			Modified:    info.ModTime().Unix(),
			ContentType: vs.getContentType(ext),
			Directory:   dirName,
			Path:        fullFilePath,
			Extension:   ext,
			StreamURL:   vs.generateStreamURL(dirName, relativeVideoPath),
			Available:   true,
			Metadata:    vs.extractVideoMetadata(fullFilePath, ext),
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetDirectoriesInfo 返回所有目录的信息
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
				// 可选地包含视频在响应中
				// dirInfo.Videos = videos
			}
		}

		directories = append(directories, dirInfo)
	}

	return directories
}

// FindVideoByID 通过 ID 查找视频（支持多层级路径）
func (vs *VideoService) FindVideoByID(videoID string) (*VideoInfo, error) {
	// Parse video ID to extract directory and relative path
	parts := strings.SplitN(videoID, ":", 2)
	if len(parts) != 2 {
		// 回退: 在所有目录中搜索
		return vs.findVideoInAllDirectories(videoID)
	}

	directoryName := parts[0]
	relativePath := parts[1]

	dir := vs.findDirectory(directoryName)
	if dir == nil || !dir.Enabled {
		return nil, fmt.Errorf("directory not found or disabled: %s", directoryName)
	}

	// 尝试直接查找文件（支持多层级路径）
	videoPath := vs.findVideoFileByRelativePath(dir.Path, relativePath)
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
		StreamURL:   vs.generateStreamURL(directoryName, relativePath),
		Available:   true,
		Metadata:    vs.extractVideoMetadata(videoPath, ext),
	}

	return video, nil
}

// SaveUploadedVideo 保存上传的视频到指定目录
func (vs *VideoService) SaveUploadedVideo(directoryName, filename string, size int64) error {
	dir := vs.findDirectory(directoryName)
	if dir == nil || !dir.Enabled {
		return fmt.Errorf("directory not found or disabled: %s", directoryName)
	}

	// 验证文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	if !vs.isVideoFile(ext) {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	// 检查上传大小限制
	if size > vs.config.Video.MaxUploadSize {
		return fmt.Errorf("file size exceeds limit: %d > %d", size, vs.config.Video.MaxUploadSize)
	}

	return nil
}

// 辅助方法

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

// findVideoFileByRelativePath 根据相对路径查找视频文件（支持多层级）
func (vs *VideoService) findVideoFileByRelativePath(basePath, relativePath string) string {
	for _, ext := range vs.config.Video.SupportedFormats {
		fullPath := filepath.Join(basePath, relativePath+ext)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
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

func (vs *VideoService) generateVideoID(directory, relativePath string) string {
	return fmt.Sprintf("%s:%s", directory, relativePath)
}

// generateStreamURL 生成流媒体URL，支持多层级路径
func (vs *VideoService) generateStreamURL(directory, relativePath string) string {
	// 现在直接使用原始路径，不需要编码，因为路由支持通配符
	return fmt.Sprintf("/stream/%s/%s", directory, relativePath)
}

// GetStats 返回整体视频统计信息
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
		"total_videos":        totalVideos,
		"total_size":          totalSize,
		"enabled_directories": enabledDirs,
		"total_directories":   len(vs.config.Video.Directories),
		"supported_formats":   vs.config.Video.SupportedFormats,
		"max_upload_size":     vs.config.Video.MaxUploadSize,
		"last_updated":        time.Now().Unix(),
	}
}

// extractVideoMetadata 提取视频文件的详细元数据
func (vs *VideoService) extractVideoMetadata(filePath, ext string) VideoMetadata {
	// Use the new metadata extractor
	extracted, err := vs.metadataExtractor.ExtractMetadata(filePath)
	if err != nil {
		// Fallback to basic metadata on error
		return VideoMetadata{
			Format:      strings.TrimPrefix(ext, "."),
			VideoCodec:  "Unknown",
			AudioCodec:  "Unknown",
			Duration:    1,
			DurationStr: "00:00:01",
			HasVideo:    true,
			HasAudio:    true,
		}
	}

	// Convert ExtractedMetadata to VideoMetadata
	metadata := VideoMetadata{
		Duration:     extracted.Duration,
		DurationStr:  extracted.DurationStr,
		Width:        extracted.Width,
		Height:       extracted.Height,
		Resolution:   extracted.Resolution,
		Bitrate:      extracted.Bitrate,
		FrameRate:    extracted.FrameRate,
		VideoCodec:   extracted.VideoCodec,
		AudioCodec:   extracted.AudioCodec,
		Format:       extracted.Format,
		AspectRatio:  extracted.AspectRatio,
		HasAudio:     extracted.HasAudio,
		HasVideo:     extracted.HasVideo,
	}

	// Generate thumbnail URL path (will be implemented later)
	if metadata.HasVideo {
		videoID := vs.generateVideoID(filepath.Dir(filePath), filepath.Base(filePath))
		metadata.ThumbnailURL = fmt.Sprintf("/thumbnails/%s.jpg", videoID)
	}

	return metadata
}

// GetMetadataExtractor returns the metadata extractor instance
func (vs *VideoService) GetMetadataExtractor() *VideoMetadataExtractor {
	return vs.metadataExtractor
}

// ValidateVideoFile 检查视频文件是否可以正确访问且有效
func (vs *VideoService) ValidateVideoFile(filePath string) error {
	// Check if file exists and is readable
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if !vs.isVideoFile(ext) {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	// 检查文件大小
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

// SearchVideos 通过名称搜索所有目录中的视频
func (vs *VideoService) SearchVideos(query string) ([]VideoInfo, error) {
	if query == "" {
		return []VideoInfo{}, nil
	}

	query = strings.ToLower(query)
	var results []VideoInfo

	allVideos, err := vs.ListAllVideos()
	if err != nil {
		return nil, err
	}

	for _, video := range allVideos {
		// 在视频名称中搜索(不包括扩展名)
		videoName := strings.ToLower(strings.TrimSuffix(video.Name, video.Extension))
		if strings.Contains(videoName, query) {
			results = append(results, video)
		}
	}

	return results, nil
}
