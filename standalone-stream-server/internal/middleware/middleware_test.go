package middleware

import (
	"testing"
	"time"
	"standalone-stream-server/internal/models"
)

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	t.Run("InitialMetrics", func(t *testing.T) {
		metrics := collector.GetMetrics()
		
		if metrics["total_requests"] != int64(0) {
			t.Errorf("Expected total_requests to be 0, got %v", metrics["total_requests"])
		}
		
		if metrics["error_count"] != int64(0) {
			t.Errorf("Expected error_count to be 0, got %v", metrics["error_count"])
		}
		
		if metrics["avg_response_time_ms"] != int64(0) {
			t.Errorf("Expected avg_response_time_ms to be 0, got %v", metrics["avg_response_time_ms"])
		}
	})

	t.Run("UptimeCalculation", func(t *testing.T) {
		// Wait a bit to ensure uptime is measured
		time.Sleep(10 * time.Millisecond)
		
		metrics := collector.GetMetrics()
		
		uptimeSeconds, ok := metrics["uptime_seconds"].(float64)
		if !ok {
			t.Fatal("uptime_seconds should be a float64")
		}
		
		if uptimeSeconds <= 0 {
			t.Error("Expected uptime_seconds to be greater than 0")
		}
		
		if metrics["uptime_human"] == "" {
			t.Error("Expected uptime_human to be set")
		}
		
		if metrics["start_time"] == "" {
			t.Error("Expected start_time to be set")
		}
	})
}

func TestStructuredLogger(t *testing.T) {
	config := &models.Config{
		Logging: models.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
	
	logger := NewStructuredLogger(config)
	
	t.Run("LoggerCreation", func(t *testing.T) {
		if logger == nil {
			t.Fatal("Logger should not be nil")
		}
		
		if logger.config != config {
			t.Error("Logger config should match provided config")
		}
	})
	
	t.Run("LogMethods", func(t *testing.T) {
		// These are mainly to ensure methods don't panic
		// In a real test environment, you'd capture log output
		
		logger.LogInfo("Test info message", map[string]interface{}{
			"test": "value",
		})
		
		logger.LogWarning("Test warning message", map[string]interface{}{
			"warning": "test",
		})
		
		logger.LogError("Test error message", nil, map[string]interface{}{
			"error": "test",
		})
	})
}

func TestTokenBucket(t *testing.T) {
	bucket := NewTokenBucket(10, 5, time.Second)
	
	t.Run("InitialTokens", func(t *testing.T) {
		available := bucket.AvailableTokens()
		if available != 10 {
			t.Errorf("Expected 10 initial tokens, got %d", available)
		}
	})
	
	t.Run("TokenConsumption", func(t *testing.T) {
		// Take one token
		if !bucket.TakeToken() {
			t.Error("Should be able to take one token")
		}
		
		available := bucket.AvailableTokens()
		if available != 9 {
			t.Errorf("Expected 9 tokens after taking one, got %d", available)
		}
	})
	
	t.Run("TokenExhaustion", func(t *testing.T) {
		// Take all remaining tokens
		for i := 0; i < 9; i++ {
			if !bucket.TakeToken() {
				t.Errorf("Should be able to take token %d", i)
			}
		}
		
		// Now bucket should be empty
		if bucket.TakeToken() {
			t.Error("Should not be able to take token from empty bucket")
		}
		
		available := bucket.AvailableTokens()
		if available != 0 {
			t.Errorf("Expected 0 tokens in empty bucket, got %d", available)
		}
	})
	
	t.Run("TokenRefill", func(t *testing.T) {
		// Wait for refill (need to wait longer than refill interval)
		time.Sleep(1100 * time.Millisecond)
		
		available := bucket.AvailableTokens()
		if available == 0 {
			t.Error("Tokens should have been refilled after waiting")
		}
		
		// Should be able to take tokens again
		if !bucket.TakeToken() {
			t.Error("Should be able to take token after refill")
		}
	})
}

func TestConnectionLimiter(t *testing.T) {
	limiter := NewConnectionLimiter(5)
	
	t.Run("InitialState", func(t *testing.T) {
		if limiter.GetActiveConnections() != 0 {
			t.Errorf("Expected 0 active connections, got %d", limiter.GetActiveConnections())
		}
		
		if limiter.GetMaxConnections() != 5 {
			t.Errorf("Expected max connections 5, got %d", limiter.GetMaxConnections())
		}
	})
	
	t.Run("AcquireConnections", func(t *testing.T) {
		// Acquire connections up to limit
		for i := 0; i < 5; i++ {
			if !limiter.Acquire() {
				t.Errorf("Should be able to acquire connection %d", i)
			}
		}
		
		if limiter.GetActiveConnections() != 5 {
			t.Errorf("Expected 5 active connections, got %d", limiter.GetActiveConnections())
		}
		
		// Should not be able to acquire more
		if limiter.Acquire() {
			t.Error("Should not be able to acquire connection beyond limit")
		}
	})
	
	t.Run("ReleaseConnections", func(t *testing.T) {
		// Release all connections
		for i := 0; i < 5; i++ {
			limiter.Release()
		}
		
		if limiter.GetActiveConnections() != 0 {
			t.Errorf("Expected 0 active connections after releasing all, got %d", limiter.GetActiveConnections())
		}
		
		// Should be able to acquire again
		if !limiter.Acquire() {
			t.Error("Should be able to acquire connection after releasing")
		}
		
		// Clean up
		limiter.Release()
	})
}