package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"standalone-stream-server/internal/handlers"
	"standalone-stream-server/internal/middleware"
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// setupTestServer 创建测试服务器
func setupTestServer(t *testing.T) (*fiber.App, *models.Config, string) {
	// 创建临时目录
	tmpDir := t.TempDir()
	videosDir := filepath.Join(tmpDir, "videos")
	moviesDir := filepath.Join(videosDir, "movies")
	seriesDir := filepath.Join(videosDir, "series")

	err := os.MkdirAll(moviesDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(seriesDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// 创建测试视频文件
	createTestVideo(t, moviesDir, "test.mp4", "fake movie content")
	createTestVideo(t, seriesDir, "test.avi", "fake series content")

	// 创建测试配置
	cfg := &models.Config{
		Server: models.ServerConfig{
			Port:     9000,
			Host:     "0.0.0.0",
			MaxConns: 100,
		},
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "movies",
					Path:        moviesDir,
					Description: "电影收藏",
					Enabled:     true,
				},
				{
					Name:        "series",
					Path:        seriesDir,
					Description: "电视剧收藏",
					Enabled:     true,
				},
			},
			MaxUploadSize:    104857600,
			SupportedFormats: []string{".mp4", ".avi", ".mov", ".mkv"},
			StreamingSettings: models.StreamSettings{
				RangeSupport: true,
				CacheControl: "public, max-age=3600",
				BufferSize:   32768,
				ChunkSize:    1048576,
				ConnTimeout:  60 * time.Second,
			},
		},
		Security: models.SecurityConfig{
			CORS: models.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Range", "Authorization"},
			},
			RateLimit: models.RateConfig{
				Enabled:        false, // Disable for testing
				RequestsPerMin: 60,
				BurstSize:      10,
			},
			Auth: models.AuthConfig{
				Enabled: false,
				Type:    "none",
			},
		},
		Logging: models.LoggingConfig{
			Level:     "info",
			Format:    "json",
			Output:    "stdout",
			AccessLog: false, // Disable for testing
			ErrorLog:  true,
		},
	}

	// 创建服务
	videoService := services.NewVideoService(cfg)
	connLimiter := middleware.NewConnectionLimiter(cfg.Server.MaxConns)

	// 创建处理器
	metricsCollector := middleware.NewMetricsCollector()
	structuredLogger := middleware.NewStructuredLogger(cfg)
	healthHandler := handlers.NewHealthHandler(cfg, videoService, connLimiter, metricsCollector, structuredLogger)
	videoHandler := handlers.NewVideoHandler(cfg, videoService)
	uploadHandler := handlers.NewUploadHandler(cfg, videoService)

	// 创建Fiber应用
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// 设置中间件
	if cfg.Security.CORS.Enabled {
		app.Use(cors.New(cors.Config{
			AllowOrigins: "*",
			AllowHeaders: "Content-Type, Range, Authorization",
		}))
	}

	// 连接限制中间件
	middleware.SetupConnectionLimiting(app, cfg)

	// 健康检查路由
	app.Get("/health", healthHandler.Health)
	app.Get("/ping", healthHandler.Ping)
	app.Get("/ready", healthHandler.Ready)
	app.Get("/live", healthHandler.Live)

	// API路由
	api := app.Group("/api")
	api.Get("/info", healthHandler.Info)
	api.Get("/videos", videoHandler.ListAllVideos)
	api.Get("/videos/:directory", videoHandler.ListVideosInDirectory)
	api.Get("/directories", videoHandler.ListDirectories)
	api.Get("/search", videoHandler.SearchVideos)
	api.Get("/video/:video-id", videoHandler.GetVideoInfo)

	// 流媒体路由 (更具体的路由应该在前面)
	app.Get("/stream/:directory/*", videoHandler.StreamVideoByDirectory)
	app.Get("/stream/:videoid", videoHandler.StreamVideo)

	// 上传路由
	app.Post("/upload/:directory/:videoid", uploadHandler.UploadVideo)
	app.Post("/upload/:directory/batch", uploadHandler.UploadMultipleVideos)

	return app, cfg, tmpDir
}

// createTestVideo 创建测试视频文件
func createTestVideo(t *testing.T, dir, filename, content string) {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHealthEndpoints(t *testing.T) {
	app, _, _ := setupTestServer(t)

	tests := []struct {
		endpoint       string
		expectedStatus int
		expectedFields []string
	}{
		{"/health", 200, []string{"status", "timestamp", "version", "server", "video", "security"}},
		{"/ping", 200, []string{"message", "timestamp"}},
		{"/ready", 200, []string{"status", "enabled_directories"}},
		{"/live", 200, []string{"status", "timestamp"}},
		{"/api/info", 200, []string{"service", "version", "framework", "endpoints", "features", "video"}},
	}

	for _, test := range tests {
		t.Run(test.endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", test.endpoint, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatal(err)
			}

			for _, field := range test.expectedFields {
				if _, exists := response[field]; !exists {
					t.Errorf("Expected field '%s' not found in response", field)
				}
			}
		})
	}
}

func TestVideoListingEndpoints(t *testing.T) {
	app, _, _ := setupTestServer(t)

	t.Run("ListAllVideos", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/videos", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatal(err)
		}

		videos, ok := response["videos"].([]interface{})
		if !ok {
			t.Fatal("Videos field should be an array")
		}

		if len(videos) < 2 {
			t.Errorf("Expected at least 2 videos, got %d", len(videos))
		}
	})

	t.Run("ListDirectoryVideos", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/videos/movies", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatal(err)
		}

		videos, ok := response["videos"].([]interface{})
		if !ok {
			t.Fatal("Videos field should be an array")
		}

		if len(videos) != 1 {
			t.Errorf("Expected 1 video in movies directory, got %d", len(videos))
		}
	})

	t.Run("ListDirectories", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/directories", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatal(err)
		}

		directories, ok := response["directories"].([]interface{})
		if !ok {
			t.Fatal("Directories field should be an array")
		}

		if len(directories) != 2 {
			t.Errorf("Expected 2 directories, got %d", len(directories))
		}
	})
}

func TestDebugListVideos(t *testing.T) {
	app, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/videos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	t.Logf("All videos response: %s", string(body))
}

func TestVideoStreaming(t *testing.T) {
	app, _, _ := setupTestServer(t)

	t.Run("StreamExistingVideo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stream/movies/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			// Debug the actual response
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Actual response: %s", string(body))
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// 检查内容类型
		contentType := resp.Header.Get("Content-Type")
		if contentType != "video/mp4" {
			t.Errorf("Expected content type 'video/mp4', got '%s'", contentType)
		}

		// 检查是否支持范围请求
		acceptRanges := resp.Header.Get("Accept-Ranges")
		if acceptRanges != "bytes" {
			t.Errorf("Expected 'Accept-Ranges: bytes', got '%s'", acceptRanges)
		}

		// 不检查响应体内容，只确保响应体存在且可以关闭
		resp.Body.Close()
	})

	t.Run("StreamNonexistentVideo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stream/movies/nonexistent", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("RangeRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stream/movies/test", nil)
		req.Header.Set("Range", "bytes=0-10")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 206 {
			t.Errorf("Expected status 206 for range request, got %d", resp.StatusCode)
		}

		// 检查内容范围头
		contentRange := resp.Header.Get("Content-Range")
		if contentRange == "" {
			t.Error("Expected Content-Range header for range request")
		}
	})
}

func TestVideoSearch(t *testing.T) {
	app, _, _ := setupTestServer(t)

	t.Run("SearchVideos", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=movie", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatal(err)
		}

		videos, ok := response["videos"].([]interface{})
		if !ok {
			t.Fatal("Videos field should be an array")
		}

		if len(videos) != 1 {
			t.Errorf("Expected 1 video for 'movie' search, got %d", len(videos))
		}
	})

	t.Run("SearchNoResults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=nonexistent", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatal(err)
		}

		videos, ok := response["videos"].([]interface{})
		if !ok {
			t.Fatal("Videos field should be an array")
		}

		if len(videos) != 0 {
			t.Errorf("Expected 0 videos for 'nonexistent' search, got %d", len(videos))
		}
	})
}

func TestVideoUpload(t *testing.T) {
	app, _, tmpDir := setupTestServer(t)

	t.Run("UploadVideo", func(t *testing.T) {
		// 创建multipart表单
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// 添加文件
		fileWriter, err := writer.CreateFormFile("file", "upload_test.mp4")
		if err != nil {
			t.Fatal(err)
		}
		_, err = fileWriter.Write([]byte("uploaded video content"))
		if err != nil {
			t.Fatal(err)
		}

		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("POST", "/upload/movies/upload_test", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req, 10000) // 增加超时时间
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
		}

		// 验证文件是否被创建
		uploadedFile := filepath.Join(tmpDir, "videos", "movies", "upload_test.mp4")
		if _, err := os.Stat(uploadedFile); os.IsNotExist(err) {
			t.Error("Uploaded file should exist")
		}

		// 验证文件内容
		content, err := os.ReadFile(uploadedFile)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "uploaded video content" {
			t.Errorf("Expected content 'uploaded video content', got '%s'", string(content))
		}
	})

	t.Run("UploadToNonexistentDirectory", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		fileWriter, err := writer.CreateFormFile("file", "test.mp4")
		if err != nil {
			t.Fatal(err)
		}
		_, err = fileWriter.Write([]byte("test content"))
		if err != nil {
			t.Fatal(err)
		}

		err = writer.Close()
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("POST", "/upload/nonexistent/test", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("Expected status 400 for nonexistent directory, got %d", resp.StatusCode)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	app, _, _ := setupTestServer(t)

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/videos", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// 检查CORS头
		if resp.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Error("Access-Control-Allow-Origin header should be present")
		}
	})
}

// 并发测试
func TestConcurrentRequests(t *testing.T) {
	app, _, _ := setupTestServer(t)

	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/api/videos", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("Request %d failed: %v", id, err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != 200 {
					t.Errorf("Request %d returned status %d", id, resp.StatusCode)
				}
			}
			done <- true
		}(i)
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// 性能测试
func BenchmarkListVideos(b *testing.B) {
	app, _, _ := setupTestServer(&testing.T{})

	req := httptest.NewRequest("GET", "/api/videos", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkStreamVideo(b *testing.B) {
	app, _, _ := setupTestServer(&testing.T{})

	req := httptest.NewRequest("GET", "/stream/movies/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}
