package middleware

import (
	"log"
	"time"

	"standalone-stream-server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Setup 为 Fiber 应用配置所有中间件
func Setup(app *fiber.App, config *models.Config) {
	// 恢复中间件 - 应该放在第一位
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// 日志中间件
	if config.Logging.AccessLog {
		setupLogging(app, config)
	}

	// CORS 中间件
	if config.Security.CORS.Enabled {
		setupCORS(app, config)
	}

	// 速率限制中间件
	if config.Security.RateLimit.Enabled {
		setupRateLimit(app, config)
	}

	// 认证中间件(如果启用)
	if config.Security.Auth.Enabled {
		setupAuth(app, config)
	}

	// 自定义头和安全
	setupSecurity(app, config)
}

// setupLogging 配置日志中间件
func setupLogging(app *fiber.App, config *models.Config) {
	logConfig := logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} - ${ip} - ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}

	if config.Logging.Format == "json" {
		logConfig.Format = `{"time":"${time}","status":"${status}","method":"${method}","path":"${path}","ip":"${ip}","latency":"${latency}","user_agent":"${ua}","error":"${error}"}` + "\n"
	}

	app.Use(logger.New(logConfig))
}

// setupCORS 配置 CORS 中间件
func setupCORS(app *fiber.App, config *models.Config) {
	corsConfig := cors.Config{
		AllowOrigins:     joinStringSlice(config.Security.CORS.AllowedOrigins, ","),
		AllowMethods:     joinStringSlice(config.Security.CORS.AllowedMethods, ","),
		AllowHeaders:     joinStringSlice(config.Security.CORS.AllowedHeaders, ","),
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length,Content-Range,Accept-Ranges",
	}

	app.Use(cors.New(corsConfig))
}

// setupRateLimit 配置速率限制中间件
func setupRateLimit(app *fiber.App, config *models.Config) {
	rateLimitConfig := limiter.Config{
		Max:               config.Security.RateLimit.RequestsPerMin,
		Expiration:        time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Rate limit exceeded",
				"retry_after": "60 seconds",
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	}

	app.Use(limiter.New(rateLimitConfig))
}

// setupAuth 配置认证中间件
func setupAuth(app *fiber.App, config *models.Config) {
	switch config.Security.Auth.Type {
	case "api_key":
		app.Use(func(c *fiber.Ctx) error {
			// 跳过认证健康检查和 info 端点
			if c.Path() == "/health" || c.Path() == "/api/info" {
				return c.Next()
			}

			apiKey := c.Get("X-API-Key")
			if apiKey == "" {
				apiKey = c.Query("api_key")
			}

			if apiKey != config.Security.Auth.ApiKey {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid or missing API key",
				})
			}

			return c.Next()
		})

	case "basic":
		app.Use(func(c *fiber.Ctx) error {
			// 跳过健康检查和信息端点的认证
			if c.Path() == "/health" || c.Path() == "/api/info" {
				return c.Next()
			}

			// 获取 Authorization 头
			auth := c.Get("Authorization")
			if auth == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Authorization required",
				})
			}

			// Simple basic auth check (in a real implementation, parse the header properly)
			if auth != "Basic "+config.Security.Auth.BasicAuth.Username+":"+config.Security.Auth.BasicAuth.Password {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid credentials",
				})
			}

			return c.Next()
		})
	}
}

// setupSecurity 配置安全头和其他安全措施
func setupSecurity(app *fiber.App, config *models.Config) {
	app.Use(func(c *fiber.Ctx) error {
		// 安全头
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 服务器识别
		c.Set("Server", "Standalone-Video-Streaming-Server/1.0")

		return c.Next()
	})
}

// ConnectionLimiter 提供连接限制功能
type ConnectionLimiter struct {
	semaphore chan struct{}
	maxConns  int
}

// NewConnectionLimiter 创建新的连接限制器
func NewConnectionLimiter(maxConns int) *ConnectionLimiter {
	return &ConnectionLimiter{
		semaphore: make(chan struct{}, maxConns),
		maxConns:  maxConns,
	}
}

// Acquire 尝试获取连接槽位
func (cl *ConnectionLimiter) Acquire() bool {
	select {
	case cl.semaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release 释放连接槽位
func (cl *ConnectionLimiter) Release() {
	select {
	case <-cl.semaphore:
	default:
		log.Printf("Warning: Attempted to release more connections than acquired")
	}
}

// GetActiveConnections 返回活跃连接数
func (cl *ConnectionLimiter) GetActiveConnections() int {
	return len(cl.semaphore)
}

// GetMaxConnections 返回最大连接数
func (cl *ConnectionLimiter) GetMaxConnections() int {
	return cl.maxConns
}

// SetupConnectionLimiting 添加连接限制中间件
func SetupConnectionLimiting(app *fiber.App, config *models.Config) *ConnectionLimiter {
	limiter := NewConnectionLimiter(config.Server.MaxConns)

	app.Use(func(c *fiber.Ctx) error {
		if !limiter.Acquire() {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":              "Server is at maximum capacity",
				"max_connections":    limiter.GetMaxConnections(),
				"active_connections": limiter.GetActiveConnections(),
			})
		}

		// 确保连接在请求完成时释放
		defer limiter.Release()

		return c.Next()
	})

	return limiter
}

// 辅助函数

func joinStringSlice(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	if len(slice) == 1 {
		return slice[0]
	}

	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}

// RequestLogger 提供结构化请求日志记录
func RequestLogger(config *models.Config) fiber.Handler {
	return logger.New(logger.Config{
		Format:     createLogFormat(config.Logging.Format),
		TimeFormat: "2006-01-02T15:04:05.000Z",
		TimeZone:   "UTC",
		Output:     nil, // 将使用默认输出
	})
}

func createLogFormat(format string) string {
	if format == "json" {
		return `{"timestamp":"${time}","level":"info","method":"${method}","path":"${path}","status":${status},"latency":"${latency}","ip":"${ip}","user_agent":"${ua}","bytes_sent":${bytesSent},"bytes_received":${bytesReceived},"referer":"${referer}"}` + "\n"
	}

	// 默认文本格式
	return "[${time}] ${ip} - ${method} ${path} ${status} ${latency} \"${ua}\"\n"
}
