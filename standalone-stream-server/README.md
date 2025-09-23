# Standalone Video Streaming Server

A high-performance, feature-rich video streaming server built with Go and GoFiber. This server provides efficient video streaming with advanced features like multi-directory management, range request support, configurable authentication, and comprehensive YAML configuration.

## ✨ Features

### 🎬 Video Management
- **Multi-directory video management**: Configure multiple video source directories
- **Range request support**: Enables video seeking without full download
- **Multiple video format support**: MP4, AVI, MOV, MKV, WebM, FLV, M4V, 3GP
- **Video upload functionality**: Upload videos to specific directories
- **Batch upload support**: Upload multiple videos at once
- **Video search**: Search videos across all directories

### 🚀 Performance & Scalability
- **GoFiber framework**: High-performance web framework
- **Connection limiting**: Prevents resource exhaustion
- **Rate limiting**: Configurable request rate limiting
- **Efficient streaming**: Optimized for video streaming with configurable chunk sizes
- **Graceful shutdown**: Proper cleanup on server shutdown

### 🔧 Configuration & Management
- **YAML configuration**: Comprehensive configuration using Viper
- **Environment variable override**: Override any config with env vars
- **Multiple configuration sources**: File, environment, defaults
- **Hot-reload ready**: Structure supports configuration hot-reloading

### 🔒 Security & Monitoring
- **CORS support**: Configurable cross-origin resource sharing
- **Authentication options**: None, API key, or basic authentication
- **Security headers**: Comprehensive security header configuration
- **Health monitoring**: Multiple health check endpoints
- **Structured logging**: JSON or text format logging

## 🏗️ Project Structure

```
standalone-stream-server/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Viper YAML configuration management
│   ├── handlers/
│   │   ├── health.go         # Health check handlers
│   │   ├── video.go          # Video streaming and listing handlers
│   │   └── upload.go         # Video upload handlers
│   ├── middleware/
│   │   └── middleware.go     # CORS, rate limiting, auth middleware
│   ├── services/
│   │   └── video.go          # Video management business logic
│   └── models/
│       └── config.go         # Configuration data structures
├── configs/
│   └── config.yaml           # Default YAML configuration
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
└── README.md                 # This file
```

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/taoyao-code/golang-streaming.git
cd golang-streaming/standalone-stream-server

# Build the server
go build -o streaming-server ./cmd/server

# Or install directly
go install ./cmd/server
```

### Basic Usage

```bash
# Run with default configuration
./streaming-server

# Run with custom config file
./streaming-server --config /path/to/config.yaml

# Show example configuration
./streaming-server --show-config

# Show version information
./streaming-server --version
```

### Configuration

Create a `config.yaml` file or modify `configs/config.yaml`:

```yaml
server:
  port: 9000
  host: "0.0.0.0"
  max_connections: 100

video:
  directories:
    - name: "movies"
      path: "./videos/movies"
      description: "Movie collection"
      enabled: true
    - name: "series"
      path: "./videos/series"  
      description: "TV series collection"
      enabled: true
  max_upload_size: 104857600  # 100MB
  
security:
  cors:
    enabled: true
    allowed_origins: ["*"]
  rate_limit:
    enabled: true
    requests_per_minute: 60
  auth:
    enabled: false
    type: "none"  # none, api_key, basic
```

## 📡 API Endpoints

### Health & Monitoring
- `GET /health` - Comprehensive health check with server status
- `GET /ping` - Simple ping endpoint
- `GET /ready` - Readiness probe
- `GET /live` - Liveness probe
- `GET /api/info` - API information and capabilities

### Video Management
- `GET /api/videos` - List all videos from all directories
- `GET /api/videos/:directory` - List videos from specific directory
- `GET /api/directories` - List all video directories with stats
- `GET /api/search?q=term` - Search videos by name
- `GET /api/video/:video-id` - Get detailed video information

### Video Streaming
- `GET /stream/:video-id` - Stream video (supports range requests)

### Video Upload
- `POST /upload/:directory/:video-id` - Upload single video
- `POST /upload/:directory/batch` - Upload multiple videos

## 🎥 Video Management

### Video ID Format
Videos are identified using the format: `directory:filename` (without extension)

Examples:
- `movies:avatar`
- `series:breaking-bad-s01e01`

### Multi-Directory Support
Configure multiple video directories for better organization:

```yaml
video:
  directories:
    - name: "movies"
      path: "/media/movies"
      description: "Movie collection"
      enabled: true
    - name: "tv-shows"
      path: "/media/tv"
      description: "TV series"
      enabled: true
    - name: "documentaries"
      path: "/media/docs"
      description: "Documentary films"
      enabled: false
```

## 🔒 Security Configuration

### CORS Configuration
```yaml
security:
  cors:
    enabled: true
    allowed_origins: ["https://yourdomain.com", "http://localhost:3000"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Range", "Authorization"]
```

### Authentication Options

#### API Key Authentication
```yaml
security:
  auth:
    enabled: true
    type: "api_key"
    api_key: "your-secret-api-key"
```

Use with header: `X-API-Key: your-secret-api-key`

#### Basic Authentication
```yaml
security:
  auth:
    enabled: true
    type: "basic"
    basic_auth:
      username: "admin"
      password: "secret"
```

### Rate Limiting
```yaml
security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    cleanup_time: "5m"
```

## 🌍 Environment Variables

Override any configuration using environment variables with the `STREAMING_` prefix:

```bash
export STREAMING_SERVER_PORT=8080
export STREAMING_VIDEO_MAX_UPLOAD_SIZE=209715200  # 200MB
export STREAMING_SECURITY_AUTH_ENABLED=true
export STREAMING_SECURITY_AUTH_API_KEY=my-secret-key
```

## 📊 Monitoring & Logging

### Health Checks
```bash
# Basic health check
curl http://localhost:9000/health

# Readiness probe (for Kubernetes)
curl http://localhost:9000/ready

# Liveness probe (for Kubernetes)
curl http://localhost:9000/live
```

### Logging Configuration
```yaml
logging:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, text
  output: "stdout"   # stdout, stderr, file
  access_log: true
  error_log: true
```

## 🔧 Advanced Configuration

### Streaming Settings
```yaml
video:
  streaming:
    cache_control: "public, max-age=3600"
    buffer_size: 32768     # 32KB
    range_support: true
    chunk_size: 1048576    # 1MB
    connection_timeout: "60s"
```

### Server Timeouts
```yaml
server:
  read_timeout: "30s"
  write_timeout: "30s"
  graceful_timeout: "30s"
```

## 🐳 Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o streaming-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/streaming-server .
COPY --from=builder /app/configs/config.yaml ./configs/
EXPOSE 9000
CMD ["./streaming-server", "--config", "configs/config.yaml"]
```

## 🚀 Production Deployment

### Systemd Service
```ini
[Unit]
Description=Standalone Video Streaming Server
After=network.target

[Service]
Type=simple
User=streaming
WorkingDirectory=/opt/streaming-server
ExecStart=/opt/streaming-server/streaming-server --config /etc/streaming-server/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Performance Tuning
1. **Use SSD storage** for video files
2. **Adjust max_connections** based on your bandwidth
3. **Configure chunk_size** for optimal streaming
4. **Enable caching** with appropriate cache headers
5. **Use a reverse proxy** (nginx, traefik) for SSL termination

## 📈 Performance Considerations

- **Connection Limiting**: Prevents server overload
- **Range Request Support**: Efficient video seeking
- **Streaming Optimization**: Configurable chunk sizes
- **Memory Management**: Efficient file streaming without loading entire files
- **Concurrent Handling**: GoFiber's high-performance request handling

## 🆚 Migration from v1.x

This v2.0 represents a complete rewrite with significant improvements:

### Key Changes
- **Framework**: Migrated from httprouter to GoFiber
- **Configuration**: JSON → YAML with Viper
- **Structure**: Monolithic → Modular architecture
- **Features**: Added multi-directory support, advanced auth, rate limiting

### Migration Steps
1. **Update configuration**: Convert JSON config to YAML format
2. **Update API calls**: Some endpoint paths have changed
3. **Review video organization**: Take advantage of multi-directory support
4. **Configure security**: Set up authentication and rate limiting as needed

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is part of the golang-streaming repository and follows the same license terms.

## 🔗 Related Projects

- [golang-streaming](https://github.com/taoyao-code/golang-streaming) - Main repository
- [video_server](../video_server) - Alternative video server implementation
- [webserver](../webserver) - Web interface for video management