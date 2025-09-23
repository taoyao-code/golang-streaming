package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"standalone-stream-server/internal/config"
	"standalone-stream-server/internal/handlers"
	"standalone-stream-server/internal/middleware"
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

var (
	configPath = flag.String("config", "", "Path to configuration file")
	showConfig = flag.Bool("show-config", false, "Show example configuration and exit")
	version    = flag.Bool("version", false, "Show version information")
)

const (
	AppName    = "Standalone Video Streaming Server"
	AppVersion = "2.0.0"
	Framework  = "GoFiber"
)

func main() {
	flag.Parse()

	// Show version information
	if *version {
		fmt.Printf("%s v%s (Framework: %s)\n", AppName, AppVersion, Framework)
		os.Exit(0)
	}

	// Show example configuration
	if *showConfig {
		fmt.Println(config.GetConfigExample())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	videoService := services.NewVideoService(cfg)

	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		ServerHeader: fmt.Sprintf("%s/%s", AppName, AppVersion),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":     err.Error(),
				"timestamp": time.Now().Unix(),
			})
		},
	})

	// Setup middleware
	middleware.Setup(app, cfg)
	connLimiter := middleware.SetupConnectionLimiting(app, cfg)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(cfg, videoService, connLimiter)
	videoHandler := handlers.NewVideoHandler(cfg, videoService)
	uploadHandler := handlers.NewUploadHandler(cfg, videoService)

	// Setup routes
	setupRoutes(app, healthHandler, videoHandler, uploadHandler)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Log startup information
	logStartupInfo(cfg, addr)

	// Graceful shutdown
	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Gracefully shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// setupRoutes configures all application routes
func setupRoutes(app *fiber.App, health *handlers.HealthHandler, video *handlers.VideoHandler, upload *handlers.UploadHandler) {
	// Health and monitoring endpoints
	app.Get("/health", health.Health)
	app.Get("/ping", health.Ping)
	app.Get("/ready", health.Ready)
	app.Get("/live", health.Live)

	// API information
	app.Get("/api/info", health.Info)

	// Video management endpoints
	api := app.Group("/api")
	{
		// Directory management
		api.Get("/directories", video.ListDirectories)

		// Video listing
		api.Get("/videos", video.ListAllVideos)
		api.Get("/videos/:directory", video.ListVideosInDirectory)

		// Video search
		api.Get("/search", video.SearchVideos)

		// Video information
		api.Get("/video/:video-id", video.GetVideoInfo)
		api.Get("/video/:video-id/validate", video.ValidateVideo)
	}

	// Video streaming endpoints (order matters - more specific routes first)
	app.Get("/stream/:directory/:videoid", video.StreamVideoByDirectory)
	app.Get("/stream/:videoid", video.StreamVideo)

	// Upload endpoints
	upload_group := app.Group("/upload")
	{
		upload_group.Post("/:directory/:videoid", upload.UploadVideo)
		upload_group.Post("/:directory/batch", upload.UploadMultipleVideos)
	}

	// Root endpoint - redirect to API info
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/api/info")
	})

	// Serve video test player
	app.Get("/player", func(c *fiber.Ctx) error {
		return c.SendFile("./web/player.html")
	})

	// Catch-all for undefined routes
	app.All("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "Endpoint not found",
			"path":   c.Path(),
			"method": c.Method(),
			"available_endpoints": []string{
				"GET /health",
				"GET /ping",
				"GET /ready",
				"GET /live",
				"GET /api/info",
				"GET /api/videos",
				"GET /api/videos/:directory",
				"GET /api/directories",
				"GET /api/search?q=term",
				"GET /api/video/:video-id",
				"GET /api/video/:video-id/validate",
				"GET /stream/:video-id",
				"GET /stream/:directory/:video-id",
				"POST /upload/:directory/:video-id",
				"POST /upload/:directory/batch",
				"GET /player",
			},
		})
	})
}

// logStartupInfo logs server startup information
func logStartupInfo(cfg *models.Config, addr string) {
	log.Printf("🚀 Starting %s v%s", AppName, AppVersion)
	log.Printf("📡 Server listening on %s", addr)
	log.Printf("🎬 Video directories:")

	for _, dir := range cfg.Video.Directories {
		status := "✅ enabled"
		if !dir.Enabled {
			status = "❌ disabled"
		}
		log.Printf("   - %s: %s (%s)", dir.Name, dir.Path, status)
	}

	log.Printf("⚙️  Configuration:")
	log.Printf("   - Max connections: %d", cfg.Server.MaxConns)
	log.Printf("   - Max upload size: %d MB", cfg.Video.MaxUploadSize/(1024*1024))
	log.Printf("   - CORS enabled: %t", cfg.Security.CORS.Enabled)
	log.Printf("   - Rate limiting: %t", cfg.Security.RateLimit.Enabled)
	log.Printf("   - Authentication: %t (%s)", cfg.Security.Auth.Enabled, cfg.Security.Auth.Type)

	log.Printf("📋 API Endpoints:")
	log.Printf("   - GET  /health                      - Health check and server status")
	log.Printf("   - GET  /api/info                    - API information")
	log.Printf("   - GET  /api/videos                  - List all videos")
	log.Printf("   - GET  /api/videos/:directory       - List videos in directory")
	log.Printf("   - GET  /api/directories             - List video directories")
	log.Printf("   - GET  /api/search?q=term           - Search videos")
	log.Printf("   - GET  /stream/:video-id            - Stream video (range requests supported)")
	log.Printf("   - POST /upload/:directory/:video-id - Upload video")
	log.Printf("   - POST /upload/:directory/batch     - Upload multiple videos")

	log.Printf("🎥 Supported formats: %v", cfg.Video.SupportedFormats)
	log.Printf("✨ Ready to serve video streams!")
}
