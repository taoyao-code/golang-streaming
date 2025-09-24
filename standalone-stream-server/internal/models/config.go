package models

import "time"

// Config 保存完整的服务器配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server" yaml:"server"`
	Video    VideoConfig    `mapstructure:"video" yaml:"video"`
	Logging  LoggingConfig  `mapstructure:"logging" yaml:"logging"`
	Security SecurityConfig `mapstructure:"security" yaml:"security"`
}

// ServerConfig 保存服务器特定的配置
type ServerConfig struct {
	Port            int           `mapstructure:"port" yaml:"port"`
	Host            string        `mapstructure:"host" yaml:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
	MaxConns        int           `mapstructure:"max_connections" yaml:"max_connections"`
	TokensPerSecond int           `mapstructure:"tokens_per_second" yaml:"tokens_per_second"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout" yaml:"graceful_timeout"`
}

// VideoConfig 保存视频相关的配置
type VideoConfig struct {
	Directories       []VideoDirectory `mapstructure:"directories" yaml:"directories"`
	MaxUploadSize     int64            `mapstructure:"max_upload_size" yaml:"max_upload_size"`
	SupportedFormats  []string         `mapstructure:"supported_formats" yaml:"supported_formats"`
	StreamingSettings StreamSettings   `mapstructure:"streaming" yaml:"streaming"`
}

// VideoDirectory 表示视频源目录
type VideoDirectory struct {
	Name        string `mapstructure:"name" yaml:"name"`
	Path        string `mapstructure:"path" yaml:"path"`
	Description string `mapstructure:"description" yaml:"description"`
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled"`
}

// StreamSettings 保存流媒体特定的设置
type StreamSettings struct {
	CacheControl string        `mapstructure:"cache_control" yaml:"cache_control"`
	BufferSize   int           `mapstructure:"buffer_size" yaml:"buffer_size"`
	RangeSupport bool          `mapstructure:"range_support" yaml:"range_support"`
	ChunkSize    int           `mapstructure:"chunk_size" yaml:"chunk_size"`
	ConnTimeout  time.Duration `mapstructure:"connection_timeout" yaml:"connection_timeout"`
}

// LoggingConfig 保存日志配置
type LoggingConfig struct {
	Level     string `mapstructure:"level" yaml:"level"`
	Format    string `mapstructure:"format" yaml:"format"`
	Output    string `mapstructure:"output" yaml:"output"`
	AccessLog bool   `mapstructure:"access_log" yaml:"access_log"`
	ErrorLog  bool   `mapstructure:"error_log" yaml:"error_log"`
}

// SecurityConfig 保存安全相关的配置
type SecurityConfig struct {
	CORS      CORSConfig `mapstructure:"cors" yaml:"cors"`
	RateLimit RateConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
	Auth      AuthConfig `mapstructure:"auth" yaml:"auth"`
}

// CORSConfig 保存 CORS 配置
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers" yaml:"allowed_headers"`
	Enabled        bool     `mapstructure:"enabled" yaml:"enabled"`
}

// RateConfig 保存速率限制配置
type RateConfig struct {
	Enabled        bool          `mapstructure:"enabled" yaml:"enabled"`
	RequestsPerMin int           `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
	BurstSize      int           `mapstructure:"burst_size" yaml:"burst_size"`
	CleanupTime    time.Duration `mapstructure:"cleanup_time" yaml:"cleanup_time"`
}

// AuthConfig 保存认证配置
type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	Type      string `mapstructure:"type" yaml:"type"`
	ApiKey    string `mapstructure:"api_key" yaml:"api_key"`
	BasicAuth struct {
		Username string `mapstructure:"username" yaml:"username"`
		Password string `mapstructure:"password" yaml:"password"`
	} `mapstructure:"basic_auth" yaml:"basic_auth"`
}
