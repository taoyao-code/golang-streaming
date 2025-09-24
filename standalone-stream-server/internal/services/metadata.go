package services

import (
"encoding/json"
"fmt"
"os/exec"
"path/filepath"
"strconv"
"strings"
"time"

"standalone-stream-server/internal/models"
"standalone-stream-server/internal/utils"

"go.uber.org/zap"
)

// MetadataService handles video metadata extraction
type MetadataService struct {
config *models.Config
}

// NewMetadataService creates a new metadata service
func NewMetadataService(config *models.Config) *MetadataService {
return &MetadataService{
config: config,
}
}

// FFProbeOutput represents the output structure from ffprobe
type FFProbeOutput struct {
Streams []struct {
Index          int    `json:"index"`
CodecName      string `json:"codec_name"`
CodecType      string `json:"codec_type"`
Width          int    `json:"width,omitempty"`
Height         int    `json:"height,omitempty"`
PixelFormat    string `json:"pix_fmt,omitempty"`
RFrameRate     string `json:"r_frame_rate,omitempty"`
AvgFrameRate   string `json:"avg_frame_rate,omitempty"`
Duration       string `json:"duration,omitempty"`
BitRate        string `json:"bit_rate,omitempty"`
SampleRate     string `json:"sample_rate,omitempty"`
Channels       int    `json:"channels,omitempty"`
} `json:"streams"`
Format struct {
Filename   string `json:"filename"`
FormatName string `json:"format_name"`
Duration   string `json:"duration"`
Size       string `json:"size"`
BitRate    string `json:"bit_rate"`
} `json:"format"`
}

// ExtractMetadata extracts video metadata using FFprobe
func (ms *MetadataService) ExtractMetadata(videoPath string) (VideoMetadata, error) {
// First try FFprobe for detailed metadata
if ffprobeMetadata, err := ms.extractWithFFprobe(videoPath); err == nil {
return ffprobeMetadata, nil
} else {
if utils.Logger != nil {
utils.Logger.Warn("FFprobe extraction failed, using fallback",
zap.String("video_path", videoPath),
zap.Error(err),
)
}
}

// Fallback to basic metadata based on file extension
return ms.extractFallbackMetadata(videoPath), nil
}

// extractWithFFprobe uses FFprobe to extract detailed metadata
func (ms *MetadataService) extractWithFFprobe(videoPath string) (VideoMetadata, error) {
cmd := exec.Command("ffprobe", 
"-v", "quiet",
"-print_format", "json",
"-show_format",
"-show_streams",
videoPath,
)

output, err := cmd.Output()
if err != nil {
return VideoMetadata{}, fmt.Errorf("ffprobe command failed: %w", err)
}

var ffprobeOutput FFProbeOutput
if err := json.Unmarshal(output, &ffprobeOutput); err != nil {
return VideoMetadata{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
}

return ms.parseFFprobeOutput(ffprobeOutput), nil
}

// parseFFprobeOutput converts FFprobe output to VideoMetadata
func (ms *MetadataService) parseFFprobeOutput(output FFProbeOutput) VideoMetadata {
metadata := VideoMetadata{}

// Extract format-level information
if duration, err := strconv.ParseFloat(output.Format.Duration, 64); err == nil {
metadata.Duration = duration
}

if bitrate, err := strconv.ParseInt(output.Format.BitRate, 10, 64); err == nil {
metadata.Bitrate = bitrate
}

metadata.Format = output.Format.FormatName

// Extract stream information
var videoStream, audioStream *struct {
Index          int    `json:"index"`
CodecName      string `json:"codec_name"`
CodecType      string `json:"codec_type"`
Width          int    `json:"width,omitempty"`
Height         int    `json:"height,omitempty"`
PixelFormat    string `json:"pix_fmt,omitempty"`
RFrameRate     string `json:"r_frame_rate,omitempty"`
AvgFrameRate   string `json:"avg_frame_rate,omitempty"`
Duration       string `json:"duration,omitempty"`
BitRate        string `json:"bit_rate,omitempty"`
SampleRate     string `json:"sample_rate,omitempty"`
Channels       int    `json:"channels,omitempty"`
}

for i := range output.Streams {
stream := &output.Streams[i]
if stream.CodecType == "video" && videoStream == nil {
videoStream = stream
} else if stream.CodecType == "audio" && audioStream == nil {
audioStream = stream
}
}

// Video stream information
if videoStream != nil {
metadata.Codec = strings.ToUpper(videoStream.CodecName)
if videoStream.Width > 0 && videoStream.Height > 0 {
metadata.Resolution = fmt.Sprintf("%dx%d", videoStream.Width, videoStream.Height)
}

// Parse frame rate
if videoStream.RFrameRate != "" {
if fps := ms.parseFraction(videoStream.RFrameRate); fps > 0 {
metadata.FrameRate = fps
}
}
}

// Audio stream information
if audioStream != nil {
metadata.AudioCodec = strings.ToUpper(audioStream.CodecName)
}

return metadata
}

// parseFraction parses a fraction string like "30/1" and returns the decimal value
func (ms *MetadataService) parseFraction(fraction string) float64 {
parts := strings.Split(fraction, "/")
if len(parts) != 2 {
return 0
}

numerator, err1 := strconv.ParseFloat(parts[0], 64)
denominator, err2 := strconv.ParseFloat(parts[1], 64)

if err1 != nil || err2 != nil || denominator == 0 {
return 0
}

return numerator / denominator
}

// extractFallbackMetadata provides basic metadata when FFprobe is not available
func (ms *MetadataService) extractFallbackMetadata(videoPath string) VideoMetadata {
ext := strings.ToLower(filepath.Ext(videoPath))

metadata := VideoMetadata{}

// Set basic codec information based on file extension
switch ext {
case ".mp4":
metadata.Codec = "H.264"
metadata.AudioCodec = "AAC"
metadata.Format = "mp4"
metadata.Duration = 1 // Placeholder
metadata.Bitrate = 1000 * 144 // 144 kbps placeholder
case ".avi":
metadata.Codec = "Various"
metadata.AudioCodec = "Various"
metadata.Format = "avi"
case ".mov":
metadata.Codec = "H.264"
metadata.AudioCodec = "AAC"
metadata.Format = "quicktime"
case ".mkv":
metadata.Codec = "H.264"
metadata.AudioCodec = "AAC"
metadata.Format = "matroska"
case ".webm":
metadata.Codec = "VP8"
metadata.AudioCodec = "Vorbis"
metadata.Format = "webm"
case ".flv":
metadata.Codec = "H.264"
metadata.AudioCodec = "AAC"
metadata.Format = "flv"
default:
metadata.Codec = "Unknown"
metadata.AudioCodec = "Unknown"
metadata.Format = "unknown"
}

return metadata
}

// GenerateThumbnail generates a thumbnail for a video file
func (ms *MetadataService) GenerateThumbnail(videoPath string, outputPath string, timestamp time.Duration) error {
// Create output directory if it doesn't exist
outputDir := filepath.Dir(outputPath)
if err := exec.Command("mkdir", "-p", outputDir).Run(); err != nil {
return fmt.Errorf("failed to create thumbnail directory: %w", err)
}

// Use FFmpeg to generate thumbnail
timestampStr := fmt.Sprintf("%.2f", timestamp.Seconds())
cmd := exec.Command("ffmpeg",
"-i", videoPath,
"-ss", timestampStr,
"-vframes", "1",
"-q:v", "2",
"-y", // Overwrite output file
outputPath,
)

if err := cmd.Run(); err != nil {
return fmt.Errorf("ffmpeg thumbnail generation failed: %w", err)
}

if utils.Logger != nil {
utils.Logger.Info("Thumbnail generated",
zap.String("video_path", videoPath),
zap.String("thumbnail_path", outputPath),
zap.Duration("timestamp", timestamp),
)
}

return nil
}

// GetOptimalThumbnailTimestamp returns an optimal timestamp for thumbnail generation
func (ms *MetadataService) GetOptimalThumbnailTimestamp(duration float64) time.Duration {
if duration <= 0 {
return 10 * time.Second // Default to 10 seconds
}

// Use 10% of video duration, but at least 5 seconds and at most 30 seconds
timestamp := duration * 0.1
if timestamp < 5 {
timestamp = 5
}
if timestamp > 30 {
timestamp = 30
}

// Don't exceed video duration
if timestamp >= duration {
timestamp = duration / 2
}

return time.Duration(timestamp * float64(time.Second))
}
