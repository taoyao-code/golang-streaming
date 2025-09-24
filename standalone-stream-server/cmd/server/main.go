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
	"standalone-stream-server/internal/scheduler"
	"standalone-stream-server/internal/services"
	"standalone-stream-server/internal/utils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "", "配置文件路径")
	showConfig = flag.Bool("show-config", false, "显示示例配置并退出")
	version    = flag.Bool("version", false, "显示版本信息")
)

const (
	AppName    = "Standalone Video Streaming Server"
	AppVersion = "2.0.0"
	Framework  = "GoFiber"
)

func main() {
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Printf("%s v%s (Framework: %s)\n", AppName, AppVersion, Framework)
		os.Exit(0)
	}

	// 显示示例配置
	if *showConfig {
		fmt.Println(config.GetConfigExample())
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 初始化结构化日志
	if err := utils.InitLogger(cfg.Logging.Level, cfg.Logging.Format); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer utils.Sync()

	utils.Logger.Info("Starting server",
		zap.String("version", AppVersion),
	)

	// 初始化服务
	videoService := services.NewVideoService(cfg)
	metadataService := services.NewMetadataService(cfg)
	schedulerService := scheduler.NewSchedulerService(cfg)

	// 创建 Fiber 应用并配置
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

	// 设置中间件
	middleware.Setup(app, cfg)
	connLimiter := middleware.SetupConnectionLimiting(app, cfg)

	// 初始化处理器
	healthHandler := handlers.NewHealthHandler(cfg, videoService, connLimiter)
	videoHandler := handlers.NewVideoHandler(cfg, videoService)
	uploadHandler := handlers.NewUploadHandler(cfg, videoService)
	schedulerHandler := handlers.NewSchedulerHandler(cfg, schedulerService)
	thumbnailHandler := handlers.NewThumbnailHandler(cfg, videoService, metadataService)
	metricsHandler := handlers.NewMetricsHandler(cfg)

	// 设置路由
	setupRoutes(app, healthHandler, videoHandler, uploadHandler, schedulerHandler, thumbnailHandler, metricsHandler)

	// 启动调度器服务
	if err := schedulerService.Start(); err != nil {
		utils.LogError("scheduler_start", err)
		utils.Logger.Warn("The server will continue running, but background cleanup tasks will be unavailable")
	} else {
		utils.Logger.Info("Scheduler service started successfully")
	}

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// 记录启动信息
	logStartupInfo(cfg, addr)
	utils.LogServerStart(cfg.Server.Port, cfg.Server.Host)

	// 优雅关闭
	go func() {
		if err := app.Listen(addr); err != nil {
			utils.LogError("server_listen", err)
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	utils.Logger.Info("Graceful shutdown initiated")

	// 停止调度器服务
	if err := schedulerService.Stop(); err != nil {
		utils.LogError("scheduler_stop", err)
	} else {
		utils.Logger.Info("Scheduler service stopped successfully")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		utils.LogError("server_shutdown", err)
	}

	utils.LogServerStop()
}

// setupRoutes 配置所有应用路由
func setupRoutes(app *fiber.App, health *handlers.HealthHandler, video *handlers.VideoHandler, upload *handlers.UploadHandler, scheduler *handlers.SchedulerHandler, thumbnail *handlers.ThumbnailHandler, metrics *handlers.MetricsHandler) {
	// 健康检查和监控端点
	app.Get("/health", health.Health)
	app.Get("/ping", health.Ping)
	app.Get("/ready", health.Ready)
	app.Get("/live", health.Live)

	// API 信息
	app.Get("/api/info", health.Info)

	// 现代化管理界面
	app.Static("/dashboard", "./web/dashboard.html")
	app.Static("/player", "./web/player.html")

	// Prometheus 指标端点
	app.Get("/metrics", metrics.GetMetrics)
	
	// 视频管理端点
	api := app.Group("/api")
	{
		// 目录管理
		api.Get("/directories", video.ListDirectories)

		// 视频列表
		api.Get("/videos", video.ListAllVideos)
		api.Get("/videos/:directory", video.ListVideosInDirectory)

		// 视频搜索
		api.Get("/search", video.SearchVideos)

		// 视频信息
		api.Get("/video/:video-id", video.GetVideoInfo)
		api.Get("/video/:video-id/validate", video.ValidateVideo)
		
		// 缩略图端点
		api.Get("/thumbnail/:videoid", thumbnail.GetThumbnail)
		api.Get("/thumbnails", thumbnail.ListThumbnails)
		api.Get("/thumbnail/file/:filename", thumbnail.ServeThumbnailFile)
		
		// 系统统计和监控
		api.Get("/system/stats", metrics.GetSystemStats)
		api.Get("/streaming/stats", video.GetFlowControlStats)
		
		// 调度器管理
		api.Get("/scheduler/stats", scheduler.GetStats)
		api.Get("/scheduler/status", scheduler.Status)
		api.Post("/scheduler/start", scheduler.Start)
		api.Post("/scheduler/stop", scheduler.Stop)
		api.Post("/scheduler/video-delete/:videoid", scheduler.AddVideoDeletionTask)
	}

	// 视频流媒体端点（顺序很重要 - 更具体的路由在前）
	app.Get("/stream/:directory/*", video.StreamVideoByDirectory)
	app.Get("/stream/:videoid", video.StreamVideo)

	// 上传端点
	upload_group := app.Group("/upload")
	{
		upload_group.Post("/:directory/:videoid", upload.UploadVideo)
		upload_group.Post("/:directory/batch", upload.UploadMultipleVideos)
	}

	// Root endpoint - redirect to dashboard
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/dashboard")
	})

	// Serve video test player
	app.Get("/player", func(c *fiber.Ctx) error {
		return c.SendFile("./web/player.html")
	})

	// Debug endpoint to list all routes
	app.Get("/debug/routes", func(c *fiber.Ctx) error {
		routes := app.GetRoutes()
		var routeInfo []map[string]string
		for _, route := range routes {
			routeInfo = append(routeInfo, map[string]string{
				"method": route.Method,
				"path":   route.Path,
			})
		}
		return c.JSON(fiber.Map{
			"total_routes": len(routes),
			"routes":       routeInfo,
		})
	})

	// Catch-all for undefined routes
	// TODO: Re-implement catch-all that doesn't interfere with API routes
	/*
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
				"GET /stream/:directory/* (supports multi-level paths)",
				"POST /upload/:directory/:video-id",
				"POST /upload/:directory/batch",
				"GET /player",
			},
		})
	})
	*/
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
	log.Printf("   - GET  /stream/:directory/*         - Stream video from directory (supports multi-level paths)")
	log.Printf("   - GET  /stream/:video-id            - Stream video (range requests supported)")
	log.Printf("   - POST /upload/:directory/:video-id - Upload video")
	log.Printf("   - POST /upload/:directory/batch     - Upload multiple videos")

	log.Printf("🎥 Supported formats: %v", cfg.Video.SupportedFormats)
	log.Printf("✨ Ready to serve video streams!")
}
