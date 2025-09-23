# 使用官方 Golang 镜像构建
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# 最终镜像
FROM alpine:3.18

# 安装运行时依赖（包括 ffmpeg 用于视频处理）
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    && rm -rf /var/cache/apk/*

# 创建非 root 用户
RUN addgroup -g 1001 appuser && \
    adduser -u 1001 -G appuser -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/server .

# 复制配置文件和 web 资源
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

# 创建必要的目录
RUN mkdir -p data/thumbnails data/tasks videos && \
    chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 9000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/health || exit 1

# 启动命令
CMD ["./server", "--config", "./configs/config.yaml"]