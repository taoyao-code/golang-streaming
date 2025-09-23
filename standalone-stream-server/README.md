# Standalone Video Streaming Server

A lightweight, standalone video streaming server designed for internal network usage. This server provides efficient video streaming with range request support (seek functionality) and requires no authentication.

## Features

- **Range Request Support**: Enables "click to play at position" functionality
- **Multiple Video Formats**: Supports MP4, AVI, MOV, MKV, WebM, FLV
- **Connection Limiting**: Prevents server overload with configurable limits
- **CORS Support**: Ready for web applications
- **No Dependencies**: No database, user management, or cloud services required
- **REST API**: Simple HTTP API for third-party integration
- **Health Monitoring**: Built-in health check endpoint

## Quick Start

### Build and Run

```bash
# Build the server
go build -o streaming-server

# Run with default settings
./streaming-server

# Run with custom settings
./streaming-server -port 8080 -video-dir /path/to/videos -max-conns 200

# Run with config file
./streaming-server -config config.json
```

### Configuration

Create a `config.json` file:

```json
{
  "port": 9000,
  "video_dir": "./videos",
  "max_connections": 100,
  "allowed_origin": "*"
}
```

## API Endpoints

### Health Check
```
GET /health
```
Returns server status and configuration.

### API Information
```
GET /api/info
```
Returns API documentation and supported features.

### List Videos
```
GET /api/videos
```
Returns a list of available videos with metadata.

### Stream Video
```
GET /stream/:video-id
```
Stream a video file. Supports HTTP range requests for seeking.

**Example:**
```bash
# Stream video with ID "movie1" (will look for movie1.mp4, movie1.avi, etc.)
curl http://localhost:9000/stream/movie1

# Range request for seeking
curl -H "Range: bytes=1000000-2000000" http://localhost:9000/stream/movie1
```

### Upload Video
```
POST /upload/:video-id
```
Upload a video file with multipart form data.

**Example:**
```bash
curl -X POST -F "file=@video.mp4" http://localhost:9000/upload/my-video
```

## Usage Examples

### HTML Video Player
```html
<video controls width="800">
  <source src="http://localhost:9000/stream/movie1" type="video/mp4">
  Your browser does not support the video tag.
</video>
```

### JavaScript Integration
```javascript
// List available videos
fetch('http://localhost:9000/api/videos')
  .then(response => response.json())
  .then(data => console.log(data.videos));

// Health check
fetch('http://localhost:9000/health')
  .then(response => response.json())
  .then(data => console.log('Server status:', data.status));
```

### Third-party API Integration
```bash
# Check server health
curl http://localhost:9000/health

# Get list of videos
curl http://localhost:9000/api/videos

# Stream video (automatic range support)
curl http://localhost:9000/stream/video-id

# Upload new video
curl -X POST -F "file=@newvideo.mp4" http://localhost:9000/upload/newvideo
```

## Command Line Options

- `-port`: Server port (default: 9000)
- `-video-dir`: Video files directory (default: ./videos)
- `-max-conns`: Maximum concurrent connections (default: 100)
- `-config`: Configuration file path

## Supported Video Formats

- MP4 (.mp4)
- AVI (.avi)
- MOV (.mov)
- MKV (.mkv)
- WebM (.webm)
- FLV (.flv)
- M4V (.m4v)
- 3GP (.3gp)

## Network Configuration

### For Internal Network Usage

1. **No Authentication**: Ready to use without user management
2. **CORS Enabled**: Accessible from web applications
3. **Range Requests**: Supports video seeking without full download
4. **Connection Limiting**: Prevents resource exhaustion

### Performance Considerations

- Use SSD storage for better streaming performance
- Adjust `max_connections` based on your network bandwidth
- Consider video file encoding for optimal streaming (H.264 recommended)

## Docker Usage (Optional)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o streaming-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/streaming-server .
COPY --from=builder /app/config.json .
EXPOSE 9000
CMD ["./streaming-server", "-config", "config.json"]
```

```bash
# Build and run
docker build -t streaming-server .
docker run -p 9000:9000 -v /path/to/videos:/root/videos streaming-server
```

## License

This is a standalone extraction from the golang-streaming project, designed for internal network usage without complex dependencies.