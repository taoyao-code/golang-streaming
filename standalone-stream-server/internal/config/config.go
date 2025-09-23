package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"standalone-stream-server/internal/models"

	"github.com/spf13/viper"
)

// Load loads configuration from YAML file and environment variables
func Load(configPath string) (*models.Config, error) {
	// Set default values
	setDefaults()

	// Configure viper
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("STREAMING")

	// If config path is provided, use it
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		viper.SetConfigName("config")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("./")
		viper.AddConfigPath("/etc/streaming-server/")
	}

	// Read configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			fmt.Printf("Warning: Config file not found, using defaults\n")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	var config models.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create video directories if they don't exist
	if err := ensureVideoDirectories(&config); err != nil {
		return nil, fmt.Errorf("error creating video directories: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 9000)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.max_connections", 100)
	viper.SetDefault("server.graceful_timeout", "30s")

	// Video defaults
	viper.SetDefault("video.directories", []models.VideoDirectory{
		{
			Name:        "default",
			Path:        "./videos",
			Description: "Default video directory",
			Enabled:     true,
		},
	})
	viper.SetDefault("video.max_upload_size", 100*1024*1024) // 100MB
	viper.SetDefault("video.supported_formats", []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv", ".m4v", ".3gp"})
	viper.SetDefault("video.streaming.cache_control", "public, max-age=3600")
	viper.SetDefault("video.streaming.buffer_size", 32*1024) // 32KB
	viper.SetDefault("video.streaming.range_support", true)
	viper.SetDefault("video.streaming.chunk_size", 1024*1024) // 1MB
	viper.SetDefault("video.streaming.connection_timeout", "60s")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.access_log", true)
	viper.SetDefault("logging.error_log", true)

	// Security defaults
	viper.SetDefault("security.cors.enabled", true)
	viper.SetDefault("security.cors.allowed_origins", []string{"*"})
	viper.SetDefault("security.cors.allowed_methods", []string{"GET", "POST", "OPTIONS"})
	viper.SetDefault("security.cors.allowed_headers", []string{"Content-Type", "Range", "Authorization"})

	viper.SetDefault("security.rate_limit.enabled", true)
	viper.SetDefault("security.rate_limit.requests_per_minute", 60)
	viper.SetDefault("security.rate_limit.burst_size", 10)
	viper.SetDefault("security.rate_limit.cleanup_time", "5m")

	viper.SetDefault("security.auth.enabled", false)
	viper.SetDefault("security.auth.type", "none")
}

// validateConfig validates the loaded configuration
func validateConfig(config *models.Config) error {
	// Validate server config
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Server.Port)
	}

	if config.Server.MaxConns <= 0 {
		return fmt.Errorf("max_connections must be positive: %d", config.Server.MaxConns)
	}

	// Validate video config
	if len(config.Video.Directories) == 0 {
		return fmt.Errorf("at least one video directory must be configured")
	}

	enabledDirs := 0
	for _, dir := range config.Video.Directories {
		if dir.Enabled {
			enabledDirs++
			if dir.Path == "" {
				return fmt.Errorf("video directory path cannot be empty for directory: %s", dir.Name)
			}
		}
	}

	if enabledDirs == 0 {
		return fmt.Errorf("at least one video directory must be enabled")
	}

	if config.Video.MaxUploadSize <= 0 {
		return fmt.Errorf("max_upload_size must be positive: %d", config.Video.MaxUploadSize)
	}

	// Validate timeouts
	if config.Server.ReadTimeout <= 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout <= 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.GracefulTimeout <= 0 {
		config.Server.GracefulTimeout = 30 * time.Second
	}

	return nil
}

// ensureVideoDirectories creates video directories if they don't exist
func ensureVideoDirectories(config *models.Config) error {
	for _, dir := range config.Video.Directories {
		if !dir.Enabled {
			continue
		}

		absPath, err := filepath.Abs(dir.Path)
		if err != nil {
			return fmt.Errorf("error getting absolute path for %s: %w", dir.Path, err)
		}

		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", absPath, err)
		}
	}

	return nil
}

// GetConfigExample returns an example configuration for documentation
func GetConfigExample() string {
	return `# Standalone Video Streaming Server Configuration

server:
  port: 9000
  host: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"
  max_connections: 100
  graceful_timeout: "30s"

video:
  directories:
    - name: "movies"
      path: "./videos/movies"
      description: "Movie collection"
      enabled: true
    - name: "series"
      path: "./videos/series"
      description: "TV series collection"
      enabled: true
    - name: "documentaries"
      path: "./videos/docs"
      description: "Documentary collection"
      enabled: false
  max_upload_size: 104857600  # 100MB
  supported_formats: [".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv", ".m4v", ".3gp"]
  streaming:
    cache_control: "public, max-age=3600"
    buffer_size: 32768  # 32KB
    range_support: true
    chunk_size: 1048576  # 1MB
    connection_timeout: "60s"

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, stderr, file
  access_log: true
  error_log: true

security:
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Range", "Authorization"]
  
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    cleanup_time: "5m"
  
  auth:
    enabled: false
    type: "none"  # none, api_key, basic
    api_key: ""
    basic_auth:
      username: ""
      password: ""
`
}