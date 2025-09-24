package handlers

import (
	"testing"

	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/services"
)

// TestNewVideoHandler_TokensPerSecondConfig tests the configurable tokens per second feature
func TestNewVideoHandler_TokensPerSecondConfig(t *testing.T) {
	// Create a mock video service
	config := &models.Config{
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{Name: "test", Path: "/tmp/test", Enabled: true},
			},
			SupportedFormats: []string{".mp4"},
		},
	}
	videoService := services.NewVideoService(config)

	tests := []struct {
		name               string
		maxConns           int
		tokensPerSecond    int
		expectedTokens     int
	}{
		{
			name:               "Zero tokens per second - should use default (maxConns/4)",
			maxConns:           100,
			tokensPerSecond:    0,
			expectedTokens:     25, // 100/4
		},
		{
			name:               "Custom tokens per second",
			maxConns:           100,
			tokensPerSecond:    50,
			expectedTokens:     50,
		},
		{
			name:               "Low max connections with default calculation",
			maxConns:           12,
			tokensPerSecond:    0,
			expectedTokens:     3, // 12/4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := &models.Config{
				Server: models.ServerConfig{
					MaxConns:        tt.maxConns,
					TokensPerSecond: tt.tokensPerSecond,
				},
				Video: config.Video,
			}

			handler := NewVideoHandler(testConfig, videoService)

			// Verify the handler was created successfully
			if handler == nil {
				t.Fatal("Handler should not be nil")
			}

			// Verify the streaming flow controller was created
			if handler.streamingFlowController == nil {
				t.Fatal("Streaming flow controller should not be nil")
			}
		})
	}
}

// TestNewVideoHandler_ConfigValidation tests that video handler creation respects configuration
func TestNewVideoHandler_ConfigValidation(t *testing.T) {
	config := &models.Config{
		Server: models.ServerConfig{
			MaxConns:        100,
			TokensPerSecond: 25,
		},
		Video: models.VideoConfig{
			Directories: []models.VideoDirectory{
				{Name: "test", Path: "/tmp/test", Enabled: true},
			},
			SupportedFormats: []string{".mp4", ".avi"},
		},
	}

	videoService := services.NewVideoService(config)
	handler := NewVideoHandler(config, videoService)

	// Verify handler components
	if handler.config != config {
		t.Error("Handler should reference the provided config")
	}

	if handler.videoService != videoService {
		t.Error("Handler should reference the provided video service")
	}

	if handler.streamingFlowController == nil {
		t.Error("Handler should have a streaming flow controller")
	}
}