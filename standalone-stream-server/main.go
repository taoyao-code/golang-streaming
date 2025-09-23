package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Config holds the server configuration
type Config struct {
	Port          int    `json:"port"`
	VideoDir      string `json:"video_dir"`
	MaxConns      int    `json:"max_connections"`
	AllowedOrigin string `json:"allowed_origin"`
}

// Default configuration
var defaultConfig = Config{
	Port:          9000,
	VideoDir:      "./videos",
	MaxConns:      100,
	AllowedOrigin: "*",
}

var config Config

// ConnLimiter limits concurrent connections
type ConnLimiter struct {
	concurrentConn int
	bucket         chan int
}

func NewConnLimiter(cc int) *ConnLimiter {
	return &ConnLimiter{
		concurrentConn: cc,
		bucket:         make(chan int, cc),
	}
}

func (cl *ConnLimiter) GetConn() bool {
	if len(cl.bucket) >= cl.concurrentConn {
		log.Printf("Rate limit reached - concurrent connections: %d", len(cl.bucket))
		return false
	}
	cl.bucket <- 1
	return true
}

func (cl *ConnLimiter) ReleaseConn() {
	<-cl.bucket
}

// MiddlewareHandler wraps router with connection limiting
type MiddlewareHandler struct {
	router  *httprouter.Router
	limiter *ConnLimiter
}

func NewMiddlewareHandler(router *httprouter.Router, maxConns int) *MiddlewareHandler {
	return &MiddlewareHandler{
		router:  router,
		limiter: NewConnLimiter(maxConns),
	}
}

func (m *MiddlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check connection limit
	if !m.limiter.GetConn() {
		sendErrorResponse(w, http.StatusTooManyRequests, "Too many concurrent requests")
		return
	}
	defer m.limiter.ReleaseConn()

	m.router.ServeHTTP(w, r)
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	io.WriteString(w, message)
}

func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"config": map[string]interface{}{
			"video_dir": config.VideoDir,
			"port":      config.Port,
			"max_conns": config.MaxConns,
		},
	}
	sendJSONResponse(w, http.StatusOK, response)
}

// List videos handler
func listVideosHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	files, err := os.ReadDir(config.VideoDir)
	if err != nil {
		log.Printf("Error reading video directory: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to read video directory")
		return
	}

	var videos []map[string]interface{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if it's a video file (basic check by extension)
		name := file.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !isVideoFile(ext) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		video := map[string]interface{}{
			"name":         name,
			"id":           strings.TrimSuffix(name, ext),
			"size":         info.Size(),
			"modified":     info.ModTime().Unix(),
			"content_type": getContentType(ext),
		}
		videos = append(videos, video)
	}

	response := map[string]interface{}{
		"videos": videos,
		"count":  len(videos),
	}
	sendJSONResponse(w, http.StatusOK, response)
}

// Video streaming handler with range support
func streamHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	videoID := p.ByName("video-id")
	if videoID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	// Find the video file (try common extensions)
	videoPath := findVideoFile(videoID)
	if videoPath == "" {
		sendErrorResponse(w, http.StatusNotFound, "Video not found")
		return
	}

	file, err := os.Open(videoPath)
	if err != nil {
		log.Printf("Error opening video file %s: %v", videoPath, err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to open video file")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Printf("Error getting file stats for %s: %v", videoPath, err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get file information")
		return
	}

	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(videoPath))
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// Enable caching for better performance
	w.Header().Set("Cache-Control", "public, max-age=3600")
	
	// Use http.ServeContent for efficient range request handling
	http.ServeContent(w, r, filepath.Base(videoPath), stat.ModTime(), file)
}

// Upload handler for adding videos
func uploadHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	videoID := p.ByName("video-id")
	if videoID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Video ID is required")
		return
	}

	// Limit upload size (100MB default)
	maxUploadSize := int64(100 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "File too large or invalid")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isVideoFile(ext) {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid video file format")
		return
	}

	// Create video directory if it doesn't exist
	if err := os.MkdirAll(config.VideoDir, 0755); err != nil {
		log.Printf("Error creating video directory: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to create video directory")
		return
	}

	// Save file
	filename := videoID + ext
	filepath := filepath.Join(config.VideoDir, filename)
	dst, err := os.Create(filepath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filepath, err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Error copying file data: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	response := map[string]interface{}{
		"message":  "Upload successful",
		"video_id": videoID,
		"filename": filename,
		"size":     header.Size,
	}
	sendJSONResponse(w, http.StatusCreated, response)
}

// API info handler
func apiInfoHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	info := map[string]interface{}{
		"service": "Standalone Video Streaming Server",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"GET /health":           "Health check",
			"GET /api/info":         "API information", 
			"GET /api/videos":       "List all videos",
			"GET /stream/:video-id": "Stream video (supports range requests)",
			"POST /upload/:video-id": "Upload video file",
		},
		"supported_formats": []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv"},
		"features": []string{
			"Range request support for seeking",
			"Connection limiting",
			"CORS support",
			"No authentication required",
		},
	}
	sendJSONResponse(w, http.StatusOK, info)
}

// Helper functions
func findVideoFile(videoID string) string {
	extensions := []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv"}
	
	for _, ext := range extensions {
		path := filepath.Join(config.VideoDir, videoID+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func isVideoFile(ext string) bool {
	videoExts := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mov":  true,
		".mkv":  true,
		".webm": true,
		".flv":  true,
		".m4v":  true,
		".3gp":  true,
	}
	return videoExts[ext]
}

func getContentType(ext string) string {
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

func loadConfig(configFile string) error {
	if configFile == "" {
		config = defaultConfig
		return nil
	}

	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

func setupRoutes() *httprouter.Router {
	router := httprouter.New()

	// Health and info endpoints
	router.GET("/health", healthHandler)
	router.GET("/api/info", apiInfoHandler)
	
	// Video management endpoints
	router.GET("/api/videos", listVideosHandler)
	router.GET("/stream/:video-id", streamHandler)
	router.POST("/upload/:video-id", uploadHandler)

	return router
}

func main() {
	var (
		port       = flag.Int("port", 0, "Port to listen on")
		videoDir   = flag.String("video-dir", "", "Directory containing video files")
		maxConns   = flag.Int("max-conns", 0, "Maximum concurrent connections")
		configFile = flag.String("config", "", "Configuration file path")
	)
	flag.Parse()

	// Load configuration
	if err := loadConfig(*configFile); err != nil {
		log.Printf("Warning: Could not load config file: %v", err)
		config = defaultConfig
	}

	// Override config with command line flags
	if *port != 0 {
		config.Port = *port
	}
	if *videoDir != "" {
		config.VideoDir = *videoDir
	}
	if *maxConns != 0 {
		config.MaxConns = *maxConns
	}

	// Create video directory if it doesn't exist
	if err := os.MkdirAll(config.VideoDir, 0755); err != nil {
		log.Fatalf("Failed to create video directory: %v", err)
	}

	// Setup router and middleware
	router := setupRoutes()
	handler := NewMiddlewareHandler(router, config.MaxConns)

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Starting Standalone Video Streaming Server")
	log.Printf("Server listening on %s", addr)
	log.Printf("Video directory: %s", config.VideoDir)
	log.Printf("Max concurrent connections: %d", config.MaxConns)
	log.Printf("API endpoints:")
	log.Printf("  GET  /health - Health check")
	log.Printf("  GET  /api/info - API information")
	log.Printf("  GET  /api/videos - List videos")
	log.Printf("  GET  /stream/:video-id - Stream video")
	log.Printf("  POST /upload/:video-id - Upload video")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}