package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"standalone-stream-server/internal/models"
)

func TestNewVideoService(t *testing.T) {
	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        "./test_videos",
					Description: "Test directory",
					Enabled:     true,
				},
			},
		},
	}

	service := NewVideoService(config)
	if service == nil {
		t.Fatal("VideoService should not be nil")
	}

	if len(service.config.Video.Directories) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(service.config.Video.Directories))
	}
}

func TestVideoService_GetDirectoriesInfo(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
		},
	}

	service := NewVideoService(config)
	directories := service.GetDirectoriesInfo()

	if len(directories) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(directories))
	}

	dir := directories[0]
	if dir.Name != "test" {
		t.Errorf("Expected directory name 'test', got '%s'", dir.Name)
	}

	if !dir.Enabled {
		t.Error("Directory should be enabled")
	}
}

func TestVideoService_GetStats(t *testing.T) {
	// 创建临时测试目录和文件
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 创建测试视频文件
	testFile := filepath.Join(testDir, "test.mp4")
	err = os.WriteFile(testFile, []byte("fake video content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4", ".avi", ".mov"},
		},
	}

	service := NewVideoService(config)
	stats := service.GetStats()

	totalVideos, ok := stats["total_videos"].(int)
	if !ok || totalVideos == 0 {
		t.Error("Expected at least 1 video")
	}

	totalSize, ok := stats["total_size"].(int64)
	if !ok || totalSize == 0 {
		t.Error("Expected total size > 0")
	}

	enabledDirs, ok := stats["enabled_directories"].(int)
	if !ok || enabledDirs == 0 {
		t.Error("Expected enabled directories")
	}
}

func TestVideoService_ListVideos(t *testing.T) {
	// 创建临时测试目录和文件
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 创建多个测试视频文件
	testFiles := []string{"video1.mp4", "video2.avi", "video3.mov", "not_video.txt"}
	for _, file := range testFiles {
		testFile := filepath.Join(testDir, file)
		err = os.WriteFile(testFile, []byte("fake content"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4", ".avi", ".mov"},
		},
	}

	service := NewVideoService(config)

	// 测试列出所有视频
	videos, err := service.ListAllVideos()
	if err != nil {
		t.Fatal(err)
	}

	// 应该只有3个视频文件（排除.txt文件）
	if len(videos) != 3 {
		t.Errorf("Expected 3 videos, got %d", len(videos))
	}

	// 测试列出特定目录的视频
	videos, err = service.ListVideosInDirectory("test")
	if err != nil {
		t.Fatal(err)
	}

	if len(videos) != 3 {
		t.Errorf("Expected 3 videos in test directory, got %d", len(videos))
	}

	// 测试视频ID格式
	for _, video := range videos {
		if video.Directory != "test" {
			t.Errorf("Expected directory 'test', got '%s'", video.Directory)
		}
		if video.ID == "" {
			t.Error("Video ID should not be empty")
		}
	}
}

func TestVideoService_GetVideoPath(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(testDir, "test.mp4")
	err = os.WriteFile(testFile, []byte("fake video content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4"},
		},
	}

	service := NewVideoService(config)

	// 测试存在的视频
	video, err := service.FindVideoByID("test:test")
	if err != nil {
		t.Fatal(err)
	}

	if video.Path != testFile {
		t.Errorf("Expected path '%s', got '%s'", testFile, video.Path)
	}

	if video.ContentType != "video/mp4" {
		t.Errorf("Expected content type 'video/mp4', got '%s'", video.ContentType)
	}

	// 测试不存在的视频
	_, err = service.FindVideoByID("test:nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent video")
	}

	// 测试无效的video ID格式（这里应该会尝试在所有目录中搜索）
	_, err = service.FindVideoByID("invalid-id")
	if err == nil {
		t.Error("Expected error for invalid video ID format")
	}
}

func TestVideoService_SearchVideos(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 创建测试视频文件
	testFiles := []string{"action_movie.mp4", "comedy_show.avi", "drama_series.mov"}
	for _, file := range testFiles {
		testFile := filepath.Join(testDir, file)
		err = os.WriteFile(testFile, []byte("fake content"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4", ".avi", ".mov"},
		},
	}

	service := NewVideoService(config)

	// 测试搜索
	videos, err := service.SearchVideos("action")
	if err != nil {
		t.Fatal(err)
	}

	if len(videos) != 1 {
		t.Errorf("Expected 1 video for 'action' search, got %d", len(videos))
	}

	expectedName := "action_movie.mp4"
	if videos[0].Name != expectedName {
		t.Errorf("Expected video name '%s', got '%s'", expectedName, videos[0].Name)
	}

	// 测试大小写不敏感搜索
	videos, err = service.SearchVideos("COMEDY")
	if err != nil {
		t.Fatal(err)
	}

	if len(videos) != 1 {
		t.Errorf("Expected 1 video for 'COMEDY' search, got %d", len(videos))
	}

	// 测试无结果搜索
	videos, err = service.SearchVideos("nonexistent")
	if err != nil {
		t.Fatal(err)
	}

	if len(videos) != 0 {
		t.Errorf("Expected 0 videos for 'nonexistent' search, got %d", len(videos))
	}
}

func TestVideoService_isVideoFile(t *testing.T) {
	config := &models.Config{
		Video: models.VideoConfig{
			SupportedFormats: []string{".mp4", ".avi", ".mov", ".mkv"},
		},
	}

	service := NewVideoService(config)

	testCases := []struct {
		extension string
		expected  bool
	}{
		{".mp4", true},
		{".MP4", false}, // 大小写敏感
		{".avi", true},
		{".mov", true},
		{".mkv", true},
		{".txt", false},
		{".jpg", false},
		{"", false}, // 空字符串
	}

	for _, tc := range testCases {
		result := service.isVideoFile(tc.extension)
		if result != tc.expected {
			t.Errorf("isVideoFile('%s') = %v, expected %v", tc.extension, result, tc.expected)
		}
	}
}

// 基准测试
func BenchmarkVideoService_ListVideos(b *testing.B) {
	tmpDir := b.TempDir()
	testDir := filepath.Join(tmpDir, "test_videos")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		b.Fatal(err)
	}

	// 创建大量测试文件
	for i := 0; i < 1000; i++ {
		testFile := filepath.Join(testDir, fmt.Sprintf("video%d.mp4", i))
		err = os.WriteFile(testFile, []byte("fake content"), 0o644)
		if err != nil {
			b.Fatal(err)
		}
	}

	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        testDir,
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4"},
		},
	}

	service := NewVideoService(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ListAllVideos()
		if err != nil {
			b.Fatal(err)
		}
	}
}
