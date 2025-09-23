package handlers

import (
	"time"

	"standalone-stream-server/internal/middleware"
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// HealthHandler 处理健康检查请求
type HealthHandler struct {
	config             *models.Config
	videoService       *services.VideoService
	connectionLimiter  *middleware.ConnectionLimiter
	metricsCollector   *middleware.MetricsCollector
	structuredLogger   *middleware.StructuredLogger
}

// NewHealthHandler 创建新的健康检查处理器
func NewHealthHandler(config *models.Config, videoService *services.VideoService, connLimiter *middleware.ConnectionLimiter, metricsCollector *middleware.MetricsCollector, structuredLogger *middleware.StructuredLogger) *HealthHandler {
	return &HealthHandler{
		config:            config,
		videoService:      videoService,
		connectionLimiter: connLimiter,
		metricsCollector:  metricsCollector,
		structuredLogger:  structuredLogger,
	}
}

// Health 返回服务器健康状态
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	stats := h.videoService.GetStats()

	response := fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "2.0.0",
		"uptime":    time.Since(time.Now()).String(), // This would need to be tracked properly
		"server": fiber.Map{
			"port":            h.config.Server.Port,
			"max_connections": h.config.Server.MaxConns,
			"active_connections": func() int {
				if h.connectionLimiter != nil {
					return h.connectionLimiter.GetActiveConnections()
				}
				return 0
			}(),
		},
		"video": stats,
		"security": fiber.Map{
			"cors_enabled":       h.config.Security.CORS.Enabled,
			"rate_limit_enabled": h.config.Security.RateLimit.Enabled,
			"auth_enabled":       h.config.Security.Auth.Enabled,
		},
	}

	return c.JSON(response)
}

// Info 返回 API 信息
func (h *HealthHandler) Info(c *fiber.Ctx) error {
	directories := h.videoService.GetDirectoriesInfo()

	response := fiber.Map{
		"service":   "Standalone Video Streaming Server",
		"version":   "2.0.0",
		"framework": "GoFiber",
		"endpoints": fiber.Map{
			"GET /health":                       "Health check and server status",
			"GET /api/info":                     "API information and capabilities",
			"GET /api/videos":                   "List all videos from all directories",
			"GET /api/videos/:directory":        "List videos from specific directory",
			"GET /api/directories":              "List all video directories",
			"GET /stream/:video-id":             "Stream video (supports range requests)",
			"POST /upload/:directory/:video-id": "Upload video to specific directory",
		},
		"features": []string{
			"Multi-directory video management",
			"Range request support for seeking",
			"Connection limiting",
			"Rate limiting",
			"CORS support",
			"Configurable authentication",
			"YAML configuration with Viper",
			"Graceful shutdown",
			"Structured logging",
		},
		"video": fiber.Map{
			"supported_formats": h.config.Video.SupportedFormats,
			"max_upload_size":   h.config.Video.MaxUploadSize,
			"directories":       directories,
			"streaming": fiber.Map{
				"range_support":      h.config.Video.StreamingSettings.RangeSupport,
				"cache_control":      h.config.Video.StreamingSettings.CacheControl,
				"buffer_size":        h.config.Video.StreamingSettings.BufferSize,
				"chunk_size":         h.config.Video.StreamingSettings.ChunkSize,
				"connection_timeout": h.config.Video.StreamingSettings.ConnTimeout.String(),
			},
		},
		"configuration": fiber.Map{
			"config_format": "YAML",
			"env_override":  true,
			"hot_reload":    false, // Future feature
		},
	}

	return c.JSON(response)
}

// Ping 提供简单的 ping 端点
func (h *HealthHandler) Ping(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message":   "pong",
		"timestamp": time.Now().Unix(),
	})
}

// Ready 检查服务器是否准备好处理请求
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	// Check if video directories are accessible
	directories := h.videoService.GetDirectoriesInfo()
	readyDirs := 0

	for _, dir := range directories {
		if dir.Enabled {
			readyDirs++
		}
	}

	if readyDirs == 0 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"reason": "no enabled video directories",
		})
	}

	return c.JSON(fiber.Map{
		"status":              "ready",
		"enabled_directories": readyDirs,
	})
}

// Live 提供存活探针端点
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
	})
}

// Metrics returns application metrics and performance data
func (h *HealthHandler) Metrics(c *fiber.Ctx) error {
	metrics := h.metricsCollector.GetMetrics()
	
	// Add connection limiter stats
	connectionStats := map[string]interface{}{
		"active":     h.connectionLimiter.GetActiveConnections(),
		"max":        h.connectionLimiter.GetMaxConnections(),
		"usage_pct":  float64(h.connectionLimiter.GetActiveConnections()) / float64(h.connectionLimiter.GetMaxConnections()) * 100,
	}

	// Add video service stats
	videoStats := h.videoService.GetStats()

	return c.JSON(fiber.Map{
		"metrics":      metrics,
		"connections":  connectionStats,
		"video_stats":  videoStats,
		"timestamp":    time.Now().Unix(),
		"server_info": fiber.Map{
			"version":    "2.0.0",
			"framework":  "GoFiber",
			"go_version": "1.25",
		},
	})
}
