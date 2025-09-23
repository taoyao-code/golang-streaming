package models

import "time"

// Config holds the complete server configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server" yaml:"server"`
	Video    VideoConfig    `mapstructure:"video" yaml:"video"`
	Logging  LoggingConfig  `mapstructure:"logging" yaml:"logging"`
	Security SecurityConfig `mapstructure:"security" yaml:"security"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port            int           `mapstructure:"port" yaml:"port"`
	Host            string        `mapstructure:"host" yaml:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
	MaxConns        int           `mapstructure:"max_connections" yaml:"max_connections"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout" yaml:"graceful_timeout"`
}

// VideoConfig holds video-related configuration
type VideoConfig struct {
	Directories       []VideoDirectory `mapstructure:"directories" yaml:"directories"`
	MaxUploadSize     int64            `mapstructure:"max_upload_size" yaml:"max_upload_size"`
	SupportedFormats  []string         `mapstructure:"supported_formats" yaml:"supported_formats"`
	StreamingSettings StreamSettings   `mapstructure:"streaming" yaml:"streaming"`
}

// VideoDirectory represents a video source directory
type VideoDirectory struct {
	Name        string `mapstructure:"name" yaml:"name"`
	Path        string `mapstructure:"path" yaml:"path"`
	Description string `mapstructure:"description" yaml:"description"`
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled"`
}

// StreamSettings holds streaming-specific settings
type StreamSettings struct {
	CacheControl string        `mapstructure:"cache_control" yaml:"cache_control"`
	BufferSize   int           `mapstructure:"buffer_size" yaml:"buffer_size"`
	RangeSupport bool          `mapstructure:"range_support" yaml:"range_support"`
	ChunkSize    int           `mapstructure:"chunk_size" yaml:"chunk_size"`
	ConnTimeout  time.Duration `mapstructure:"connection_timeout" yaml:"connection_timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level     string `mapstructure:"level" yaml:"level"`
	Format    string `mapstructure:"format" yaml:"format"`
	Output    string `mapstructure:"output" yaml:"output"`
	AccessLog bool   `mapstructure:"access_log" yaml:"access_log"`
	ErrorLog  bool   `mapstructure:"error_log" yaml:"error_log"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	CORS      CORSConfig `mapstructure:"cors" yaml:"cors"`
	RateLimit RateConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
	Auth      AuthConfig `mapstructure:"auth" yaml:"auth"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers" yaml:"allowed_headers"`
	Enabled        bool     `mapstructure:"enabled" yaml:"enabled"`
}

// RateConfig holds rate limiting configuration
type RateConfig struct {
	Enabled        bool          `mapstructure:"enabled" yaml:"enabled"`
	RequestsPerMin int           `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
	BurstSize      int           `mapstructure:"burst_size" yaml:"burst_size"`
	CleanupTime    time.Duration `mapstructure:"cleanup_time" yaml:"cleanup_time"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Type      string `mapstructure:"type" yaml:"type"`
	ApiKey    string `mapstructure:"api_key" yaml:"api_key"`
	BasicAuth struct {
		Username string `mapstructure:"username" yaml:"username"`
		Password string `mapstructure:"password" yaml:"password"`
	} `mapstructure:"basic_auth" yaml:"basic_auth"`
}
