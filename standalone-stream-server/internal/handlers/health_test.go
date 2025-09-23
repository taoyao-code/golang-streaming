package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"standalone-stream-server/internal/middleware"
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

func TestHealthHandler_Health(t *testing.T) {
	// 创建测试配置
	config := &models.Config{
		Server: models.ServerConfig{
			Port:     9000,
			MaxConns: 100,
		},
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        t.TempDir(),
					Description: "Test directory",
					Enabled:     true,
				},
			},
		},
		Security: models.SecurityConfig{
			CORS: models.CORSConfig{
				Enabled: true,
			},
			RateLimit: models.RateConfig{
				Enabled: true,
			},
			Auth: models.AuthConfig{
				Enabled: false,
			},
		},
	}

	// 创建服务和处理器
	videoService := services.NewVideoService(config)
	connLimiter := middleware.NewConnectionLimiter(config.Server.MaxConns)
	handler := NewHealthHandler(config, videoService, connLimiter)

	// 创建Fiber应用
	app := fiber.New()
	app.Get("/health", handler.Health)

	// 创建测试请求
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// 解析JSON响应
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatal(err)
	}

	// 验证响应内容
	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%v'", response["version"])
	}

	// 检查server信息
	server, ok := response["server"].(map[string]interface{})
	if !ok {
		t.Fatal("Server info should be present")
	}

	if server["port"] != float64(9000) {
		t.Errorf("Expected port 9000, got %v", server["port"])
	}

	if server["max_connections"] != float64(100) {
		t.Errorf("Expected max_connections 100, got %v", server["max_connections"])
	}

	// 检查安全配置
	security, ok := response["security"].(map[string]interface{})
	if !ok {
		t.Fatal("Security info should be present")
	}

	if security["cors_enabled"] != true {
		t.Errorf("Expected cors_enabled true, got %v", security["cors_enabled"])
	}
}

func TestHealthHandler_Info(t *testing.T) {
	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        t.TempDir(),
					Description: "Test directory",
					Enabled:     true,
				},
			},
			SupportedFormats: []string{".mp4", ".avi", ".mov"},
			MaxUploadSize:    104857600,
			StreamingSettings: models.StreamSettings{
				RangeSupport: true,
				CacheControl: "public, max-age=3600",
				BufferSize:   32768,
				ChunkSize:    1048576,
				ConnTimeout:  60000000000, // 60s in nanoseconds
			},
		},
	}

	videoService := services.NewVideoService(config)
	handler := NewHealthHandler(config, videoService, nil)

	app := fiber.New()
	app.Get("/api/info", handler.Info)

	req := httptest.NewRequest("GET", "/api/info", nil)
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

	// 验证服务信息
	if response["service"] != "Standalone Video Streaming Server" {
		t.Errorf("Unexpected service name: %v", response["service"])
	}

	if response["version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got %v", response["version"])
	}

	if response["framework"] != "GoFiber" {
		t.Errorf("Expected framework 'GoFiber', got %v", response["framework"])
	}

	// 检查端点信息
	endpoints, ok := response["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("Endpoints should be present")
	}

	expectedEndpoints := []string{
		"GET /health",
		"GET /api/info",
		"GET /api/videos",
		"GET /stream/:video-id",
	}

	for _, endpoint := range expectedEndpoints {
		if _, exists := endpoints[endpoint]; !exists {
			t.Errorf("Expected endpoint '%s' not found", endpoint)
		}
	}

	// 检查功能列表
	features, ok := response["features"].([]interface{})
	if !ok {
		t.Fatal("Features should be present")
	}

	if len(features) == 0 {
		t.Error("Features list should not be empty")
	}

	// 检查视频配置
	video, ok := response["video"].(map[string]interface{})
	if !ok {
		t.Fatal("Video config should be present")
	}

	supportedFormats, ok := video["supported_formats"].([]interface{})
	if !ok {
		t.Fatal("Supported formats should be present")
	}

	if len(supportedFormats) != 3 {
		t.Errorf("Expected 3 supported formats, got %d", len(supportedFormats))
	}
}

func TestHealthHandler_Ping(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	app := fiber.New()
	app.Get("/ping", handler.Ping)

	req := httptest.NewRequest("GET", "/ping", nil)
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

	if response["message"] != "pong" {
		t.Errorf("Expected message 'pong', got '%v'", response["message"])
	}

	if response["timestamp"] == nil {
		t.Error("Timestamp should be present")
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	// 测试有可用目录的情况
	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        t.TempDir(),
					Description: "Test directory",
					Enabled:     true,
				},
			},
		},
	}

	videoService := services.NewVideoService(config)
	handler := NewHealthHandler(config, videoService, nil)

	app := fiber.New()
	app.Get("/ready", handler.Ready)

	req := httptest.NewRequest("GET", "/ready", nil)
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

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%v'", response["status"])
	}

	// 测试没有可用目录的情况
	configNoDir := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        t.TempDir(),
					Description: "Test directory",
					Enabled:     false, // 禁用目录
				},
			},
		},
	}

	videoServiceNoDir := services.NewVideoService(configNoDir)
	handlerNoDir := NewHealthHandler(configNoDir, videoServiceNoDir, nil)

	app2 := fiber.New()
	app2.Get("/ready", handlerNoDir.Ready)

	req2 := httptest.NewRequest("GET", "/ready", nil)
	resp2, err := app2.Test(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 503 {
		t.Errorf("Expected status 503, got %d", resp2.StatusCode)
	}
}

func TestHealthHandler_Live(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	app := fiber.New()
	app.Get("/live", handler.Live)

	req := httptest.NewRequest("GET", "/live", nil)
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

	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%v'", response["status"])
	}
}

// 基准测试
func BenchmarkHealthHandler_Health(b *testing.B) {
	config := &models.Config{
		Server: models.ServerConfig{
			Port:     9000,
			MaxConns: 100,
		},
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{
					Name:        "test",
					Path:        b.TempDir(),
					Description: "Test directory",
					Enabled:     true,
				},
			},
		},
		Security: models.SecurityConfig{},
	}

	videoService := services.NewVideoService(config)
	connLimiter := middleware.NewConnectionLimiter(config.Server.MaxConns)
	handler := NewHealthHandler(config, videoService, connLimiter)

	app := fiber.New()
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
