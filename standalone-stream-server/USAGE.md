# Standalone Video Streaming Server

## Quick Test Commands

### Basic Server Operations

```bash
# Build and run with defaults
./scripts/deploy.sh run

# Custom configuration
./scripts/deploy.sh run -p 8080 -d /path/to/videos -c 200

# Build only
./scripts/deploy.sh build

# Generate systemd service
./scripts/deploy.sh service

# Generate Docker files
./scripts/deploy.sh docker

# Clean build artifacts
./scripts/deploy.sh clean
```

### API Testing

```bash
# Health check
curl http://localhost:9000/health

# List videos
curl http://localhost:9000/api/videos

# Stream video (replace 'video-id' with actual ID)
curl http://localhost:9000/stream/video-id

# Range request (seek functionality)
curl -H "Range: bytes=1000000-2000000" http://localhost:9000/stream/video-id

# Upload video
curl -X POST -F "file=@video.mp4" http://localhost:9000/upload/my-video

# API information
curl http://localhost:9000/api/info
```

### HTML5 Video Integration

```html
<!-- Basic video player with seeking support -->
<video controls width="800">
  <source src="http://localhost:9000/stream/video-id" type="video/mp4">
</video>

<!-- JavaScript control -->
<script>
const video = document.querySelector('video');

// Jump to specific time (demonstrates range request)
function seekTo(seconds) {
  video.currentTime = seconds;
}

// Load video list
fetch('http://localhost:9000/api/videos')
  .then(r => r.json())
  .then(data => console.log(data.videos));
</script>
```

### Performance Testing

```bash
# Test concurrent connections
for i in {1..10}; do
  curl -s http://localhost:9000/stream/video-id > /dev/null &
done
wait

# Test range requests
curl -H "Range: bytes=0-1024" http://localhost:9000/stream/video-id
curl -H "Range: bytes=1024-2048" http://localhost:9000/stream/video-id
```

### Docker Deployment

#### 标准部署（x86_64）

```bash
# Generate Docker files
./scripts/deploy.sh docker

# Build and run with Docker Compose
docker-compose up --build

# Or with Docker directly
docker build -t streaming-server .
docker run -p 9000:9000 -v ./videos:/app/videos streaming-server
```

#### ARM64 架构部署

```bash
# 使用 ARM64 构建脚本（推荐）
./build-arm64.sh

# 启动服务
docker-compose up -d

# 验证架构
docker exec streaming-server uname -m
# 期望输出: aarch64

# 手动构建 ARM64 镜像
docker buildx build --platform linux/arm64 -t streaming-server:arm64 --load .
docker run -p 9000:9000 -v ./videos:/app/videos streaming-server:arm64
```

#### 多架构构建

```bash
# 构建支持多架构的镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t streaming-server:multi-arch \
  --push .

# 拉取并运行（自动选择架构）
docker pull streaming-server:multi-arch
docker run -p 9000:9000 streaming-server:multi-arch
```

### Production Deployment

```bash
# Build optimized binary
go build -ldflags="-s -w" -o streaming-server

# Generate systemd service
./scripts/deploy.sh service

# Install as system service
sudo cp standalone-stream-server.service /etc/systemd/system/
sudo systemctl enable standalone-stream-server
sudo systemctl start standalone-stream-server

# Check status
sudo systemctl status standalone-stream-server
sudo journalctl -u standalone-stream-server -f
```

### Configuration Examples

#### Basic config.json

```json
{
  "port": 9000,
  "video_dir": "./videos",
  "max_connections": 100,
  "allowed_origin": "*"
}
```

#### Production config.json

```json
{
  "port": 80,
  "video_dir": "/var/videos",
  "max_connections": 500,
  "allowed_origin": "https://yourdomain.com"
}
```

### Troubleshooting

#### Common Issues

1. **Port already in use**: Change port in config or use `-p` flag
2. **Permission denied**: Ensure video directory is readable
3. **File not found**: Check video ID matches filename (without extension)
4. **Range requests not working**: Ensure HTTP/1.1 client and proper headers

#### Logs

```bash
# View real-time logs
tail -f logs/streaming-server.log

# Check error logs
tail -f logs/streaming-server.error.log

# Systemd logs
sudo journalctl -u standalone-stream-server -f
```

#### Testing Range Requests

```bash
# Test if range requests work
curl -v -H "Range: bytes=0-1023" http://localhost:9000/stream/video-id

# Should return HTTP 206 Partial Content
# Content-Range: bytes 0-1023/total_size
```
