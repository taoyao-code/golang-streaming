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

// Setup configures all middleware for the Fiber app
func Setup(app *fiber.App, config *models.Config) {
	// Recovery middleware - should be first
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Logging middleware
	if config.Logging.AccessLog {
		setupLogging(app, config)
	}

	// CORS middleware
	if config.Security.CORS.Enabled {
		setupCORS(app, config)
	}

	// Rate limiting middleware
	if config.Security.RateLimit.Enabled {
		setupRateLimit(app, config)
	}

	// Authentication middleware (if enabled)
	if config.Security.Auth.Enabled {
		setupAuth(app, config)
	}

	// Custom headers and security
	setupSecurity(app, config)
}

// setupLogging configures logging middleware
func setupLogging(app *fiber.App, config *models.Config) {
	logConfig := logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${ip} - ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	}

	if config.Logging.Format == "json" {
		logConfig.Format = `{"time":"${time}","status":"${status}","method":"${method}","path":"${path}","ip":"${ip}","latency":"${latency}","user_agent":"${ua}","error":"${error}"}` + "\n"
	}

	app.Use(logger.New(logConfig))
}

// setupCORS configures CORS middleware
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

// setupRateLimit configures rate limiting middleware
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
				"error": "Rate limit exceeded",
				"retry_after": "60 seconds",
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
	}

	app.Use(limiter.New(rateLimitConfig))
}

// setupAuth configures authentication middleware
func setupAuth(app *fiber.App, config *models.Config) {
	switch config.Security.Auth.Type {
	case "api_key":
		app.Use(func(c *fiber.Ctx) error {
			// Skip auth for health check and info endpoints
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
			// Skip auth for health check and info endpoints
			if c.Path() == "/health" || c.Path() == "/api/info" {
				return c.Next()
			}

			// Get Authorization header
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

// setupSecurity configures security headers and other security measures
func setupSecurity(app *fiber.App, config *models.Config) {
	app.Use(func(c *fiber.Ctx) error {
		// Security headers
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Server identification
		c.Set("Server", "Standalone-Video-Streaming-Server/1.0")

		return c.Next()
	})
}

// ConnectionLimiter provides connection limiting functionality
type ConnectionLimiter struct {
	semaphore chan struct{}
	maxConns  int
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxConns int) *ConnectionLimiter {
	return &ConnectionLimiter{
		semaphore: make(chan struct{}, maxConns),
		maxConns:  maxConns,
	}
}

// Acquire attempts to acquire a connection slot
func (cl *ConnectionLimiter) Acquire() bool {
	select {
	case cl.semaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a connection slot
func (cl *ConnectionLimiter) Release() {
	select {
	case <-cl.semaphore:
	default:
		log.Printf("Warning: Attempted to release more connections than acquired")
	}
}

// GetActiveConnections returns the number of active connections
func (cl *ConnectionLimiter) GetActiveConnections() int {
	return len(cl.semaphore)
}

// GetMaxConnections returns the maximum number of connections
func (cl *ConnectionLimiter) GetMaxConnections() int {
	return cl.maxConns
}

// SetupConnectionLimiting adds connection limiting middleware
func SetupConnectionLimiting(app *fiber.App, config *models.Config) *ConnectionLimiter {
	limiter := NewConnectionLimiter(config.Server.MaxConns)

	app.Use(func(c *fiber.Ctx) error {
		if !limiter.Acquire() {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Server is at maximum capacity",
				"max_connections": limiter.GetMaxConnections(),
				"active_connections": limiter.GetActiveConnections(),
			})
		}

		// Ensure connection is released when request completes
		defer limiter.Release()

		return c.Next()
	})

	return limiter
}

// Helper functions

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

// RequestLogger provides structured request logging
func RequestLogger(config *models.Config) fiber.Handler {
	return logger.New(logger.Config{
		Format: createLogFormat(config.Logging.Format),
		TimeFormat: "2006-01-02T15:04:05.000Z",
		TimeZone:   "UTC",
		Output:     nil, // Will use default output
	})
}

func createLogFormat(format string) string {
	if format == "json" {
		return `{"timestamp":"${time}","level":"info","method":"${method}","path":"${path}","status":${status},"latency":"${latency}","ip":"${ip}","user_agent":"${ua}","bytes_sent":${bytesSent},"bytes_received":${bytesReceived},"referer":"${referer}"}` + "\n"
	}
	
	// Default text format
	return "[${time}] ${ip} - ${method} ${path} ${status} ${latency} \"${ua}\"\n"
}