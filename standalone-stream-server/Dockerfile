# Multi-stage build for production-optimized container
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with multi-architecture support
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -a -installsuffix cgo \
    -ldflags="-s -w -X main.AppVersion=2.0.0-docker" \
    -o streaming-server \
    ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    curl \
    && rm -rf /var/cache/apk/*

# Create app user
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/streaming-server .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

# Create necessary directories with proper permissions
RUN mkdir -p videos thumbnails logs && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 9000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9000/health || exit 1

# Default command
CMD ["./streaming-server", "-config", "configs/config.yaml"]
