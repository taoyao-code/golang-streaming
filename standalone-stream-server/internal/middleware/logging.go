package middleware

import (
	"encoding/json"
	"log"
	"time"

	"standalone-stream-server/internal/models"

	"github.com/gofiber/fiber/v2"
)

// StructuredLogger provides structured logging capabilities
type StructuredLogger struct {
	config *models.Config
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *models.Config) *StructuredLogger {
	return &StructuredLogger{
		config: config,
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    string                 `json:"timestamp"`
	Level        string                 `json:"level"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	ResponseTime int64                  `json:"response_time_ms,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Size         int                    `json:"response_size,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// AccessLogger returns a middleware for access logging
func (sl *StructuredLogger) AccessLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate response time
		elapsed := time.Since(start)

		// Create log entry
		entry := LogEntry{
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			Level:        "info",
			Method:       c.Method(),
			Path:         c.Path(),
			StatusCode:   c.Response().StatusCode(),
			ResponseTime: elapsed.Milliseconds(),
			IP:           c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Size:         len(c.Response().Body()),
		}

		// Add error if present
		if err != nil {
			entry.Level = "error"
			entry.Error = err.Error()
		}

		// Log based on format preference
		if sl.config.Logging.Format == "json" {
			sl.logJSON(entry)
		} else {
			sl.logText(entry)
		}

		return err
	}
}

// ErrorLogger logs errors with additional context
func (sl *StructuredLogger) ErrorLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		
		if err != nil {
			entry := LogEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Level:     "error",
				Method:    c.Method(),
				Path:      c.Path(),
				IP:        c.IP(),
				Error:     err.Error(),
				Extra: map[string]interface{}{
					"headers": map[string]interface{}{
						"content-type": c.Get("Content-Type"),
						"accept":       c.Get("Accept"),
					},
					"query": c.Queries(),
				},
			}

			if sl.config.Logging.Format == "json" {
				sl.logJSON(entry)
			} else {
				sl.logText(entry)
			}
		}

		return err
	}
}

// LogInfo logs an info message
func (sl *StructuredLogger) LogInfo(message string, extra map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "info",
		Message:   message,
		Extra:     extra,
	}

	if sl.config.Logging.Format == "json" {
		sl.logJSON(entry)
	} else {
		sl.logText(entry)
	}
}

// LogError logs an error message
func (sl *StructuredLogger) LogError(message string, err error, extra map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "error",
		Message:   message,
		Extra:     extra,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if sl.config.Logging.Format == "json" {
		sl.logJSON(entry)
	} else {
		sl.logText(entry)
	}
}

// LogWarning logs a warning message
func (sl *StructuredLogger) LogWarning(message string, extra map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "warning",
		Message:   message,
		Extra:     extra,
	}

	if sl.config.Logging.Format == "json" {
		sl.logJSON(entry)
	} else {
		sl.logText(entry)
	}
}

// logJSON outputs the log entry as JSON
func (sl *StructuredLogger) logJSON(entry LogEntry) {
	if jsonData, err := json.Marshal(entry); err == nil {
		log.Println(string(jsonData))
	} else {
		log.Printf("Failed to marshal log entry: %v", err)
	}
}

// logText outputs the log entry in human-readable format
func (sl *StructuredLogger) logText(entry LogEntry) {
	if entry.Method != "" && entry.Path != "" {
		// Access log format
		log.Printf("[%s] %s %s %d %dms %s %s",
			entry.Level,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.ResponseTime,
			entry.IP,
			entry.UserAgent)
	} else {
		// General log format
		if entry.Error != "" {
			log.Printf("[%s] %s - Error: %s", entry.Level, entry.Message, entry.Error)
		} else {
			log.Printf("[%s] %s", entry.Level, entry.Message)
		}
	}
}

// MetricsCollector collects and stores application metrics
type MetricsCollector struct {
	requestCount     int64
	errorCount       int64
	totalResponseTime int64
	startTime        time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// MetricsMiddleware returns a middleware that collects metrics
func (mc *MetricsCollector) MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		err := c.Next()
		
		// Update metrics
		mc.requestCount++
		mc.totalResponseTime += time.Since(start).Milliseconds()
		
		if err != nil || c.Response().StatusCode() >= 400 {
			mc.errorCount++
		}
		
		return err
	}
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	uptime := time.Since(mc.startTime)
	avgResponseTime := int64(0)
	if mc.requestCount > 0 {
		avgResponseTime = mc.totalResponseTime / mc.requestCount
	}
	
	return map[string]interface{}{
		"uptime_seconds":       uptime.Seconds(),
		"uptime_human":         uptime.String(),
		"total_requests":       mc.requestCount,
		"error_count":          mc.errorCount,
		"success_rate":         float64(mc.requestCount-mc.errorCount) / float64(mc.requestCount) * 100,
		"avg_response_time_ms": avgResponseTime,
		"start_time":           mc.startTime.Format(time.RFC3339),
	}
}