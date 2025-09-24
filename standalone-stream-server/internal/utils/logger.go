package utils

import (
"go.uber.org/zap"
"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogger initializes the structured logger with zap
func InitLogger(level string, format string) error {
var config zap.Config

// Set log level
var logLevel zapcore.Level
switch level {
case "debug":
logLevel = zapcore.DebugLevel
case "info":
logLevel = zapcore.InfoLevel
case "warn":
logLevel = zapcore.WarnLevel
case "error":
logLevel = zapcore.ErrorLevel
default:
logLevel = zapcore.InfoLevel
}

if format == "json" {
config = zap.NewProductionConfig()
} else {
config = zap.NewDevelopmentConfig()
config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
}

config.Level = zap.NewAtomicLevelAt(logLevel)
config.OutputPaths = []string{"stdout"}
config.ErrorOutputPaths = []string{"stderr"}

// Build logger
logger, err := config.Build()
if err != nil {
return err
}

Logger = logger

// Replace standard library's log
zap.ReplaceGlobals(logger)

return nil
}

// NewRequestLogger creates a structured logger for HTTP requests
func NewRequestLogger() *zap.Logger {
if Logger == nil {
// Fallback to a basic logger if not initialized
logger, _ := zap.NewProduction()
return logger
}
return Logger
}

// LogServerStart logs server startup information
func LogServerStart(port int, host string) {
Logger.Info("Server started",
zap.String("host", host),
zap.Int("port", port),
zap.String("service", "standalone-stream-server"),
)
}

// LogServerStop logs server shutdown
func LogServerStop() {
Logger.Info("Server shutdown completed")
}

// LogVideoStream logs video streaming requests
func LogVideoStream(videoID string, clientIP string, success bool) {
if success {
Logger.Info("Video stream served",
zap.String("video_id", videoID),
zap.String("client_ip", clientIP),
zap.String("operation", "video_stream"),
)
} else {
Logger.Warn("Video stream failed",
zap.String("video_id", videoID),
zap.String("client_ip", clientIP),
zap.String("operation", "video_stream"),
)
}
}

// LogError logs errors with context
func LogError(operation string, err error, fields ...zap.Field) {
allFields := append([]zap.Field{
zap.String("operation", operation),
zap.Error(err),
}, fields...)
Logger.Error("Operation failed", allFields...)
}

// Sync flushes any buffered log entries
func Sync() {
if Logger != nil {
Logger.Sync()
}
}
