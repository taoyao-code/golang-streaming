# Quick Start Guide

## Test the Server

1. **Build and run the server:**
```bash
go build -o streaming-server ./cmd/server
./streaming-server --config configs/config.yaml
```

2. **Test the API endpoints:**
```bash
# Health check
curl http://localhost:9000/health | jq

# API information
curl http://localhost:9000/api/info | jq

# List directories
curl http://localhost:9000/api/directories | jq

# List all videos
curl http://localhost:9000/api/videos | jq

# Search videos
curl http://localhost:9000/api/search?q=movie | jq
```

3. **Upload a test video:**
```bash
# Create a test video directory
mkdir -p videos/movies

# Upload a video (replace with actual video file)
curl -X POST -F "file=@test.mp4" http://localhost:9000/upload/movies/test-video

# List videos in movies directory
curl http://localhost:9000/api/videos/movies | jq
```

4. **Stream a video:**
```bash
# Stream full video
curl http://localhost:9000/stream/movies:test-video

# Stream with range request (first 1MB)
curl -H "Range: bytes=0-1048576" http://localhost:9000/stream/movies:test-video
```

## Configuration Examples

### Basic Configuration
```yaml
server:
  port: 9000
  host: "0.0.0.0"

video:
  directories:
    - name: "movies"
      path: "./videos/movies"
      enabled: true
```

### Production Configuration
```yaml
server:
  port: 80
  host: "0.0.0.0"
  max_connections: 500

video:
  directories:
    - name: "movies"
      path: "/media/movies"
      enabled: true
    - name: "series"
      path: "/media/tv-shows"
      enabled: true
  max_upload_size: 1073741824  # 1GB

security:
  cors:
    allowed_origins: ["https://yourdomain.com"]
  auth:
    enabled: true
    type: "api_key"
    api_key: "your-secret-key"
  rate_limit:
    requests_per_minute: 120
```

## Environment Variables

```bash
export STREAMING_SERVER_PORT=8080
export STREAMING_VIDEO_MAX_UPLOAD_SIZE=209715200
export STREAMING_SECURITY_AUTH_ENABLED=true
export STREAMING_SECURITY_AUTH_API_KEY=my-secret-key
```