# ARM64 架构部署指南

本指南详细介绍如何在 ARM64 架构（如 Apple Silicon M1/M2、树莓派等）上部署独立视频流媒体服务器。

## 🏗️ 架构支持

### 支持的平台

- **Apple Silicon**: M1, M1 Pro, M1 Max, M2 系列
- **树莓派**: Raspberry Pi 4/5 (64位系统)
- **AWS Graviton**: Graviton2/3 处理器
- **ARM 服务器**: 基于 ARM64 的云服务器

### 系统要求

- ARM64 架构处理器
- Linux/macOS 操作系统
- Docker 20.10+ (支持 buildx)
- 至少 1GB 内存
- 充足的存储空间用于视频文件

## 🚀 快速部署

### 方法一：一键构建脚本（推荐）

```bash
# 进入项目目录
cd standalone-stream-server

# 运行 ARM64 构建脚本
./build-arm64.sh

# 启动服务
docker-compose up -d
```

### 方法二：手动构建

```bash
# 确保启用 Docker buildx
docker buildx create --use

# 构建 ARM64 镜像
docker buildx build \
  --platform linux/arm64 \
  --tag streaming-server:arm64 \
  --load \
  .

# 启动容器
docker run -d \
  --name streaming-server \
  -p 9000:9000 \
  -v $(pwd)/videos:/app/videos \
  -v $(pwd)/configs:/app/configs:ro \
  streaming-server:arm64
```

## 📋 详细构建步骤

### 1. 检查环境

```bash
# 验证架构
uname -m
# 期望输出: arm64 或 aarch64

# 检查 Docker 版本
docker version

# 验证 buildx 可用
docker buildx version
```

### 2. 构建配置

项目已经配置了多架构支持：

```dockerfile
# Dockerfile 中的关键配置
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -a -installsuffix cgo \
    -ldflags="-s -w -X main.AppVersion=2.0.0-docker" \
    -o streaming-server \
    ./cmd/server
```

### 3. 构建选项

```bash
# 仅构建 ARM64
docker buildx build --platform linux/arm64 -t streaming-server:arm64 .

# 构建多架构镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t streaming-server:multi-arch \
  --push .
```

## ⚙️ 性能优化

### ARM64 特定优化

1. **Go 编译优化**

```bash
# 针对 ARM64 的编译标志
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -ldflags="-s -w" \
  -trimpath \
  ./cmd/server
```

2. **Docker 镜像优化**

```dockerfile
# 使用 Alpine Linux ARM64 镜像
FROM --platform=linux/arm64 alpine:latest
```

3. **性能配置调整**

```yaml
# config.yaml - ARM64 推荐配置
server:
  max_connections: 200  # ARM64 设备可适当降低
video:
  streaming:
    buffer_size: 16384   # 16KB 适合 ARM64
    chunk_size: 2097152  # 2MB 块大小
```

## 🔧 故障排除

### 常见问题

#### 1. 构建失败

```bash
# 错误: exec format error
# 解决: 确保使用正确的平台标志
docker buildx build --platform linux/arm64 ...
```

#### 2. 性能问题

```bash
# 在 ARM64 设备上监控资源使用
docker stats streaming-server

# 调整并发连接数
# 在 config.yaml 中降低 max_connections
```

#### 3. 依赖问题

```bash
# 清理 Docker 缓存
docker buildx prune

# 重新构建所有层
docker buildx build --no-cache --platform linux/arm64 ...
```

### 验证部署

```bash
# 检查容器架构
docker exec streaming-server uname -m
# 应输出: aarch64

# 检查 Go 程序架构
docker exec streaming-server file /app/streaming-server
# 应包含: ARM aarch64

# 性能测试
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:9000/health
```

## 📊 性能对比

### ARM64 vs x86_64 性能特点

| 指标 | ARM64 | x86_64 | 说明 |
|------|-------|--------|------|
| 能耗 | 低 | 高 | ARM64 功耗更低 |
| 并发连接 | 中等 | 高 | 建议调整 max_connections |
| 内存使用 | 低 | 中等 | ARM64 内存效率更高 |
| 启动时间 | 快 | 中等 | 启动速度更快 |

### 推荐配置

```yaml
# ARM64 优化配置示例
server:
  port: 9000
  host: "0.0.0.0"
  max_connections: 150    # 比 x86_64 略低
  read_timeout: "30s"
  write_timeout: "30s"

video:
  streaming:
    buffer_size: 16384     # 16KB 缓冲区
    chunk_size: 1572864    # 1.5MB 块大小
    connection_timeout: "45s"

logging:
  level: "info"
  format: "json"
```

## 🔍 监控和维护

### 系统监控

```bash
# CPU 使用率
top -p $(docker exec streaming-server pidof streaming-server)

# 内存使用
docker exec streaming-server cat /proc/meminfo

# 网络连接
docker exec streaming-server netstat -an | grep 9000
```

### 日志监控

```bash
# 查看应用日志
docker logs -f streaming-server

# 查看系统日志
journalctl -u docker -f
```

## 🚀 生产部署建议

### 1. 系统配置

```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化网络参数
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
sysctl -p
```

### 2. Docker 配置

```json
// /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "storage-driver": "overlay2"
}
```

### 3. 服务管理

```bash
# 创建 systemd 服务
sudo tee /etc/systemd/system/streaming-server.service << EOF
[Unit]
Description=Streaming Server
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=true
WorkingDirectory=/opt/streaming-server
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable streaming-server
sudo systemctl start streaming-server
```

## 📞 技术支持

遇到 ARM64 部署问题？

1. 检查 [GitHub Issues](https://github.com/taoyao-code/golang-streaming/issues)
2. 查看 [故障排除文档](./troubleshooting.md)
3. 提交新的 Issue 并标明 ARM64 标签

---

*最后更新: 2024年9月*
