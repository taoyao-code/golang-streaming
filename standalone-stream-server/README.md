# 独立视频流媒体服务器

基于 Go 和 GoFiber 构建的高性能、功能丰富的视频流媒体服务器。该服务器提供高效的视频流服务，具有多目录管理、范围请求支持、可配置身份验证和全面的 YAML 配置等高级功能。

## ✨ 功能特性

### 🎬 视频管理

- **多目录视频管理**：配置多个视频源目录
- **范围请求支持**：支持视频快进而无需完整下载
- **多种视频格式支持**：MP4、AVI、MOV、MKV、WebM、FLV、M4V、3GP
- **视频上传功能**：上传视频到指定目录
- **批量上传支持**：一次上传多个视频
- **视频搜索**：跨所有目录搜索视频

### 🚀 性能与扩展性

- **GoFiber 框架**：高性能 Web 框架
- **连接限制**：防止资源耗尽
- **速率限制**：可配置的请求速率限制
- **高效流媒体**：针对视频流优化，支持可配置的块大小
- **优雅关闭**：服务器关闭时正确清理资源

### 🔧 配置与管理

- **YAML 配置**：使用 Viper 进行全面配置
- **环境变量覆盖**：使用环境变量覆盖任何配置
- **多配置源**：文件、环境变量、默认值
- **热重载就绪**：结构支持配置热重载

### 🔒 安全与监控

- **CORS 支持**：可配置的跨域资源共享
- **身份验证选项**：无验证、API 密钥或基本身份验证
- **安全头**：全面的安全头配置
- **健康监控**：多个健康检查端点
- **结构化日志**：JSON 或文本格式日志

## 🏗️ 项目结构

```
standalone-stream-server/
├── cmd/
│   └── server/
│       └── main.go           # 应用程序入口点
├── internal/
│   ├── config/
│   │   └── config.go         # Viper YAML 配置管理
│   ├── handlers/
│   │   ├── health.go         # 健康检查处理器
│   │   ├── video.go          # 视频流和列表处理器
│   │   └── upload.go         # 视频上传处理器
│   ├── middleware/
│   │   └── middleware.go     # CORS、速率限制、认证中间件
│   ├── services/
│   │   └── video.go          # 视频管理业务逻辑
│   └── models/
│       └── config.go         # 配置数据结构
├── configs/
│   └── config.yaml           # 默认 YAML 配置
├── docs/                     # 文档目录
│   ├── API_INTEGRATION_GUIDE.md  # API对接完整指南
│   ├── QUICK_START.md            # 快速开始指南
│   ├── architecture.md           # 系统架构文档
│   └── application-flow.md       # 应用流程文档
├── examples/                 # 示例代码
│   ├── clients/              # 客户端SDK
│   │   ├── javascript_client.js  # JavaScript客户端
│   │   └── python_client.py      # Python客户端
│   └── integrations/         # 集成示例
│       └── integration_examples.md  # 全平台集成指南
├── scripts/                  # 脚本目录
│   ├── deploy.sh             # 部署脚本
│   ├── test.sh               # 测试脚本
│   └── e2e_test.sh           # 端到端测试脚本
├── videos/                   # 视频存储目录
├── web/                      # Web界面
├── build-arm64.sh            # ARM64构建脚本
├── go.mod                    # Go 模块定义
├── go.sum                    # Go 模块校验和
└── README.md                 # 本文件
```

## 🚀 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/taoyao-code/golang-streaming.git
cd golang-streaming/standalone-stream-server

# 构建服务器
go build -o streaming-server ./cmd/server

# 或直接安装
go install ./cmd/server
```

### 基本使用

```bash
# 使用默认配置运行
./streaming-server

# 使用自定义配置文件运行
./streaming-server --config /path/to/config.yaml

# 显示示例配置
./streaming-server --show-config

# 显示版本信息
./streaming-server --version
```

### Docker 部署

#### 标准 x86_64 部署

```bash
# 使用 Docker Compose 启动
docker-compose up -d

# 查看日志
docker-compose logs -f streaming-server

# 停止服务
docker-compose down
```

#### ARM64 架构部署

```bash
# 构建 ARM64 镜像
./build-arm64.sh

# 启动服务
docker-compose up -d

# 验证架构
docker exec streaming-server uname -m
# 输出: aarch64

# 手动构建 ARM64 镜像（可选）
docker buildx build --platform linux/arm64 -t streaming-server:arm64 --load .
```

#### 多架构构建

```bash
# 构建支持多架构的镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t streaming-server:multi-arch \
  --push .
```

### 📚 详细文档

- **[快速开始指南](./docs/QUICK_START.md)** - 5分钟快速部署和测试
- **[ARM64部署指南](./docs/ARM64_DEPLOYMENT.md)** - Apple Silicon、树莓派等ARM64架构部署
- **[API对接指南](./docs/API_INTEGRATION_GUIDE.md)** - 完整的第三方集成文档
- **[客户端示例](./examples/clients/)** - JavaScript和Python客户端代码
- **[集成示例](./examples/integrations/)** - 全平台集成参考

### 配置

创建 `config.yaml` 文件或修改 `configs/config.yaml`：

```yaml
server:
  port: 9000
  host: "0.0.0.0"
  max_connections: 100

video:
  directories:
    - name: "movies"
      path: "./videos/movies"
      description: "电影收藏"
      enabled: true
    - name: "series"
      path: "./videos/series"  
      description: "电视剧收藏"
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

## 📡 API 端点

### 健康与监控

- `GET /health` - 包含服务器状态的全面健康检查
- `GET /ping` - 简单的 ping 端点
- `GET /ready` - 就绪探针
- `GET /live` - 活性探针
- `GET /api/info` - API 信息和功能

### 视频管理

- `GET /api/videos` - 列出所有目录中的所有视频
- `GET /api/videos/:directory` - 列出特定目录中的视频
- `GET /api/directories` - 列出所有视频目录和统计信息
- `GET /api/search?q=term` - 按名称搜索视频
- `GET /api/video/:video-id` - 获取详细的视频信息

### 视频流

- `GET /stream/:video-id` - 流式传输视频（支持范围请求）

### 视频上传

- `POST /upload/:directory/:video-id` - 上传单个视频
- `POST /upload/:directory/batch` - 上传多个视频

## 🎥 视频管理

### 视频 ID 格式

视频使用以下格式标识：`目录:文件名`（不包含扩展名）

示例：

- `movies:avatar`
- `series:breaking-bad-s01e01`

### 多目录支持

配置多个视频目录以便更好地组织：

```yaml
video:
  directories:
    - name: "movies"
      path: "/media/movies"
      description: "电影收藏"
      enabled: true
    - name: "tv-shows"
      path: "/media/tv"
      description: "电视剧"
      enabled: true
    - name: "documentaries"
      path: "/media/docs"
      description: "纪录片"
      enabled: false
```

## 🔒 安全配置

### CORS 配置

```yaml
security:
  cors:
    enabled: true
    allowed_origins: ["https://yourdomain.com", "http://localhost:3000"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Range", "Authorization"]
```

### 身份验证选项

#### API 密钥身份验证

```yaml
security:
  auth:
    enabled: true
    type: "api_key"
    api_key: "your-secret-api-key"
```

使用请求头：`X-API-Key: your-secret-api-key`

#### 基本身份验证

```yaml
security:
  auth:
    enabled: true
    type: "basic"
    basic_auth:
      username: "admin"
      password: "secret"
```

### 速率限制

```yaml
security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    cleanup_time: "5m"
```

## 🌍 环境变量

使用 `STREAMING_` 前缀的环境变量覆盖任何配置：

```bash
export STREAMING_SERVER_PORT=8080
export STREAMING_VIDEO_MAX_UPLOAD_SIZE=209715200  # 200MB
export STREAMING_SECURITY_AUTH_ENABLED=true
export STREAMING_SECURITY_AUTH_API_KEY=my-secret-key
```

## 📊 监控与日志

### 健康检查

```bash
# 基本健康检查
curl http://localhost:9000/health

# 就绪探针（用于 Kubernetes）
curl http://localhost:9000/ready

# 活性探针（用于 Kubernetes）
curl http://localhost:9000/live
```

### 日志配置

```yaml
logging:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, text
  output: "stdout"   # stdout, stderr, file
  access_log: true
  error_log: true
```

## 🔧 高级配置

### 流媒体设置

```yaml
video:
  streaming:
    cache_control: "public, max-age=3600"
    buffer_size: 32768     # 32KB
    range_support: true
    chunk_size: 1048576    # 1MB
    connection_timeout: "60s"
```

### 服务器超时

```yaml
server:
  read_timeout: "30s"
  write_timeout: "30s"
  graceful_timeout: "30s"
```

## 🐳 Docker 部署

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

## 🚀 生产部署

### Systemd 服务

```ini
[Unit]
Description=独立视频流媒体服务器
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

### 性能调优

1. **使用 SSD 存储**：用于视频文件
2. **调整 max_connections**：根据您的带宽调整
3. **配置 chunk_size**：优化流媒体性能
4. **启用缓存**：使用适当的缓存头
5. **使用反向代理**：（nginx、traefik）进行 SSL 终止

## 📈 性能考虑

- **连接限制**：防止服务器过载
- **范围请求支持**：高效的视频快进
- **流媒体优化**：可配置的块大小
- **内存管理**：高效的文件流，无需加载整个文件
- **并发处理**：GoFiber 的高性能请求处理

## 🆚 从 v1.x 迁移

v2.0 版本代表了完全重写，具有显著改进：

### 主要变化

- **框架**：从 httprouter 迁移到 GoFiber
- **配置**：JSON → YAML，使用 Viper
- **结构**：单体 → 模块化架构
- **功能**：增加了多目录支持、高级认证、速率限制

### 迁移步骤

1. **更新配置**：将 JSON 配置转换为 YAML 格式
2. **更新 API 调用**：某些端点路径已更改
3. **重新组织视频**：利用多目录支持
4. **配置安全性**：根据需要设置身份验证和速率限制

## 🤝 贡献

1. Fork 仓库
2. 创建功能分支
3. 进行更改
4. 如适用，添加测试
5. 提交拉取请求

## 📄 许可证

此项目是 golang-streaming 仓库的一部分，遵循相同的许可条款。

## 🔗 相关项目

- [golang-streaming](https://github.com/taoyao-code/golang-streaming) - 主仓库
- [video_server](../video_server) - 替代视频服务器实现
- [webserver](../webserver) - 视频管理 Web 界面
