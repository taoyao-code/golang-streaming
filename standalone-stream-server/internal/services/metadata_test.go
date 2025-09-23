package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVideoMetadataExtractor(t *testing.T) {
	extractor := NewVideoMetadataExtractor()

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mp4")
	
	// Create a test file with some content
	testContent := []byte("fake video content for testing")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("ExtractBasicMetadata", func(t *testing.T) {
		metadata, err := extractor.ExtractMetadata(testFile)
		if err != nil {
			t.Fatalf("Failed to extract metadata: %v", err)
		}

		// Verify basic properties
		if metadata.FileSize != int64(len(testContent)) {
			t.Errorf("Expected file size %d, got %d", len(testContent), metadata.FileSize)
		}

		if metadata.Format != "mp4" {
			t.Errorf("Expected format 'mp4', got '%s'", metadata.Format)
		}

		if !metadata.HasVideo {
			t.Error("Expected HasVideo to be true")
		}

		if !metadata.HasAudio {
			t.Error("Expected HasAudio to be true")
		}

		if metadata.VideoCodec == "" {
			t.Error("Expected VideoCodec to be set")
		}

		if metadata.AudioCodec == "" {
			t.Error("Expected AudioCodec to be set")
		}
	})

	t.Run("ExtractMetadataFromDifferentFormats", func(t *testing.T) {
		formats := []string{".mp4", ".avi", ".mov", ".mkv", ".webm"}
		
		for _, format := range formats {
			testFile := filepath.Join(tmpDir, "test"+format)
			if err := os.WriteFile(testFile, testContent, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			metadata, err := extractor.ExtractMetadata(testFile)
			if err != nil {
				t.Errorf("Failed to extract metadata for %s: %v", format, err)
				continue
			}

			expectedFormat := format[1:] // Remove the dot
			if metadata.Format != expectedFormat {
				t.Errorf("Expected format '%s', got '%s'", expectedFormat, metadata.Format)
			}
		}
	})
}

func TestMetadataIntegration(t *testing.T) {
	// Create a temporary video service configuration
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mp4")
	testContent := []byte("fake video content for testing metadata integration")
	
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	extractor := NewVideoMetadataExtractor()
	
	t.Run("MetadataExtractionIntegration", func(t *testing.T) {
		metadata, err := extractor.ExtractMetadata(testFile)
		if err != nil {
			t.Fatalf("Failed to extract metadata: %v", err)
		}

		// Test that all expected fields are populated
		if metadata.Duration <= 0 {
			t.Error("Expected Duration to be greater than 0")
		}

		if metadata.DurationStr == "" {
			t.Error("Expected DurationStr to be set")
		}

		if metadata.Width <= 0 || metadata.Height <= 0 {
			t.Error("Expected Width and Height to be greater than 0")
		}

		if metadata.Resolution == "" {
			t.Error("Expected Resolution to be set")
		}

		if metadata.AspectRatio == "" {
			t.Error("Expected AspectRatio to be set")
		}

		if metadata.FrameRate <= 0 {
			t.Error("Expected FrameRate to be greater than 0")
		}
	})
}