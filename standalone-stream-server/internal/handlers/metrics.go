package handlers

import (
	"standalone-stream-server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gofiber/adaptor/v2"
)

// MetricsHandler handles Prometheus metrics endpoint
type MetricsHandler struct {
	config *models.Config
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(config *models.Config) *MetricsHandler {
	return &MetricsHandler{
		config: config,
	}
}

// GetMetrics serves Prometheus metrics
func (mh *MetricsHandler) GetMetrics(c *fiber.Ctx) error {
	// Convert Prometheus HTTP handler to Fiber handler
	handler := adaptor.HTTPHandler(promhttp.Handler())
	return handler(c)
}

// GetSystemStats returns custom system statistics
func (mh *MetricsHandler) GetSystemStats(c *fiber.Ctx) error {
	stats := map[string]interface{}{
		"timestamp": c.Context().Time().Unix(),
		"service":   "standalone-stream-server",
		"version":   "2.0.0",
		"uptime":    c.Context().Time().Unix(), // Placeholder for actual uptime
		"config": map[string]interface{}{
			"max_connections":   mh.config.Server.MaxConns,
			"tokens_per_second": mh.config.Server.TokensPerSecond,
			"port":              mh.config.Server.Port,
			"host":              mh.config.Server.Host,
		},
		"directories": len(mh.config.Video.Directories),
		"formats":     len(mh.config.Video.SupportedFormats),
	}

	return c.JSON(stats)
}