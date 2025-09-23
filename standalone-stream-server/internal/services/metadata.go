package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// VideoMetadataExtractor provides video metadata extraction capabilities
type VideoMetadataExtractor struct {
	ffprobeAvailable bool
}

// NewVideoMetadataExtractor creates a new metadata extractor
func NewVideoMetadataExtractor() *VideoMetadataExtractor {
	extractor := &VideoMetadataExtractor{}
	extractor.checkFFProbeAvailability()
	return extractor
}

// checkFFProbeAvailability checks if ffprobe is available on the system
func (vme *VideoMetadataExtractor) checkFFProbeAvailability() {
	_, err := exec.LookPath("ffprobe")
	vme.ffprobeAvailable = err == nil
}

// ExtractedMetadata contains detailed video metadata
type ExtractedMetadata struct {
	Duration     float64 `json:"duration_seconds"`
	DurationStr  string  `json:"duration"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Resolution   string  `json:"resolution"`
	Bitrate      int64   `json:"bitrate"`
	FrameRate    float64 `json:"frame_rate"`
	VideoCodec   string  `json:"video_codec"`
	AudioCodec   string  `json:"audio_codec"`
	Format       string  `json:"format"`
	FileSize     int64   `json:"file_size"`
	AspectRatio  string  `json:"aspect_ratio"`
	HasAudio     bool    `json:"has_audio"`
	HasVideo     bool    `json:"has_video"`
	CreationTime string  `json:"creation_time,omitempty"`
}

// ExtractMetadata extracts detailed metadata from a video file
func (vme *VideoMetadataExtractor) ExtractMetadata(filePath string) (*ExtractedMetadata, error) {
	// Get basic file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	metadata := &ExtractedMetadata{
		FileSize: fileInfo.Size(),
		Format:   strings.TrimPrefix(filepath.Ext(filePath), "."),
	}

	// Try to extract metadata using ffprobe if available
	if vme.ffprobeAvailable {
		if err := vme.extractWithFFProbe(filePath, metadata); err == nil {
			return metadata, nil
		}
	}

	// Fallback to basic analysis
	vme.extractBasicMetadata(filePath, metadata)
	return metadata, nil
}

// extractWithFFProbe uses ffprobe to extract detailed metadata
func (vme *VideoMetadataExtractor) extractWithFFProbe(filePath string, metadata *ExtractedMetadata) error {
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",
		"-show_format",
		"-show_streams",
		"-print_format", "json",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return err
	}

	return vme.parseFFProbeOutput(string(output), metadata)
}

// parseFFProbeOutput parses ffprobe JSON output
func (vme *VideoMetadataExtractor) parseFFProbeOutput(output string, metadata *ExtractedMetadata) error {
	// Basic regex-based parsing since we want to avoid adding JSON dependencies
	// Extract duration
	if durationRegex := regexp.MustCompile(`"duration":\s*"([^"]+)"`); durationRegex.MatchString(output) {
		matches := durationRegex.FindStringSubmatch(output)
		if len(matches) > 1 {
			if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
				metadata.Duration = duration
				metadata.DurationStr = formatDuration(duration)
			}
		}
	}

	// Extract bitrate
	if bitrateRegex := regexp.MustCompile(`"bit_rate":\s*"([^"]+)"`); bitrateRegex.MatchString(output) {
		matches := bitrateRegex.FindStringSubmatch(output)
		if len(matches) > 1 {
			if bitrate, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				metadata.Bitrate = bitrate
			}
		}
	}

	// Extract video stream info
	videoStreamRegex := regexp.MustCompile(`"codec_type":\s*"video".*?"width":\s*(\d+).*?"height":\s*(\d+).*?"codec_name":\s*"([^"]+)"`)
	if matches := videoStreamRegex.FindStringSubmatch(output); len(matches) > 3 {
		metadata.HasVideo = true
		if width, err := strconv.Atoi(matches[1]); err == nil {
			metadata.Width = width
		}
		if height, err := strconv.Atoi(matches[2]); err == nil {
			metadata.Height = height
		}
		metadata.VideoCodec = matches[3]
		metadata.Resolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		
		if metadata.Width > 0 && metadata.Height > 0 {
			aspectRatio := float64(metadata.Width) / float64(metadata.Height)
			metadata.AspectRatio = fmt.Sprintf("%.2f:1", aspectRatio)
		}
	}

	// Extract audio stream info
	audioStreamRegex := regexp.MustCompile(`"codec_type":\s*"audio".*?"codec_name":\s*"([^"]+)"`)
	if matches := audioStreamRegex.FindStringSubmatch(output); len(matches) > 1 {
		metadata.HasAudio = true
		metadata.AudioCodec = matches[1]
	}

	// Extract frame rate
	frameRateRegex := regexp.MustCompile(`"r_frame_rate":\s*"([^"]+)"`)
	if matches := frameRateRegex.FindStringSubmatch(output); len(matches) > 1 {
		if frameRateStr := matches[1]; strings.Contains(frameRateStr, "/") {
			parts := strings.Split(frameRateStr, "/")
			if len(parts) == 2 {
				if num, err1 := strconv.ParseFloat(parts[0], 64); err1 == nil {
					if den, err2 := strconv.ParseFloat(parts[1], 64); err2 == nil && den != 0 {
						metadata.FrameRate = num / den
					}
				}
			}
		}
	}

	return nil
}

// extractBasicMetadata provides fallback metadata extraction
func (vme *VideoMetadataExtractor) extractBasicMetadata(filePath string, metadata *ExtractedMetadata) {
	// Basic metadata based on file extension and analysis
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".mp4":
		metadata.VideoCodec = "H.264"
		metadata.AudioCodec = "AAC"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	case ".avi":
		metadata.VideoCodec = "Various"
		metadata.AudioCodec = "Various"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	case ".mov":
		metadata.VideoCodec = "H.264"
		metadata.AudioCodec = "AAC"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	case ".mkv":
		metadata.VideoCodec = "H.264"
		metadata.AudioCodec = "AC3"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	case ".webm":
		metadata.VideoCodec = "VP8/VP9"
		metadata.AudioCodec = "Vorbis/Opus"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	default:
		metadata.VideoCodec = "Unknown"
		metadata.AudioCodec = "Unknown"
		metadata.HasVideo = true
		metadata.HasAudio = true
		vme.estimateBasicProperties(metadata)
	}
}

// estimateBasicProperties provides estimated properties when detailed analysis isn't available
func (vme *VideoMetadataExtractor) estimateBasicProperties(metadata *ExtractedMetadata) {
	// Estimate duration based on file size (very rough approximation)
	if metadata.FileSize > 0 {
		// Assume average bitrate of 1 Mbps for estimation
		estimatedDuration := float64(metadata.FileSize) / (1024 * 1024 / 8)
		metadata.Duration = estimatedDuration
		metadata.DurationStr = formatDuration(estimatedDuration)
	}
	
	// Default resolution assumptions
	if metadata.Width == 0 || metadata.Height == 0 {
		if metadata.FileSize > 100*1024*1024 { // > 100MB, assume HD
			metadata.Width = 1920
			metadata.Height = 1080
			metadata.Resolution = "1920x1080"
			metadata.AspectRatio = "16:9"
		} else if metadata.FileSize > 50*1024*1024 { // > 50MB, assume 720p
			metadata.Width = 1280
			metadata.Height = 720
			metadata.Resolution = "1280x720"
			metadata.AspectRatio = "16:9"
		} else {
			metadata.Width = 640
			metadata.Height = 480
			metadata.Resolution = "640x480"
			metadata.AspectRatio = "4:3"
		}
	}
	
	// Estimate bitrate
	if metadata.Bitrate == 0 && metadata.Duration > 0 {
		metadata.Bitrate = int64(float64(metadata.FileSize*8) / metadata.Duration)
	}
	
	// Default frame rate
	if metadata.FrameRate == 0 {
		metadata.FrameRate = 25.0
	}
}

// formatDuration converts seconds to human-readable duration string
func formatDuration(seconds float64) string {
	if seconds <= 0 {
		return "00:00:00"
	}
	
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// GenerateThumbnail generates a thumbnail for the video (if ffmpeg is available)
func (vme *VideoMetadataExtractor) GenerateThumbnail(filePath, outputPath string, timeOffset float64) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not available for thumbnail generation")
	}
	
	cmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-ss", fmt.Sprintf("%.0f", timeOffset),
		"-vframes", "1",
		"-f", "image2",
		"-y", // Overwrite output file
		outputPath)
	
	return cmd.Run()
}

// IsFFProbeAvailable returns whether ffprobe is available for metadata extraction
func (vme *VideoMetadataExtractor) IsFFProbeAvailable() bool {
	return vme.ffprobeAvailable
}