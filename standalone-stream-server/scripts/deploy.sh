#!/bin/bash

# Standalone Video Streaming Server Deployment Script

set -e

PROJECT_NAME="standalone-stream-server"
DEFAULT_PORT=9000
DEFAULT_VIDEO_DIR="./videos"
DEFAULT_MAX_CONNS=100

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.19 or later."
        exit 1
    fi
    
    go_version=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
    print_info "Go version: $go_version"
}

# Build the server
build_server() {
    print_info "Building streaming server..."
    go build -o streaming-server -ldflags="-s -w"
    
    if [ $? -eq 0 ]; then
        print_success "Server built successfully"
    else
        print_error "Failed to build server"
        exit 1
    fi
}

# Create necessary directories
setup_directories() {
    print_info "Setting up directories..."
    
    if [ ! -d "$DEFAULT_VIDEO_DIR" ]; then
        mkdir -p "$DEFAULT_VIDEO_DIR"
        print_info "Created video directory: $DEFAULT_VIDEO_DIR"
    fi
    
    if [ ! -d "logs" ]; then
        mkdir -p "logs"
        print_info "Created logs directory"
    fi
}

# Generate systemd service file
generate_systemd_service() {
    local user=$(whoami)
    local working_dir=$(pwd)
    
    cat > "${PROJECT_NAME}.service" << EOF
[Unit]
Description=Standalone Video Streaming Server
After=network.target

[Service]
Type=simple
User=$user
WorkingDirectory=$working_dir
ExecStart=$working_dir/streaming-server -config $working_dir/config.json
Restart=always
RestartSec=10
StandardOutput=append:$working_dir/logs/streaming-server.log
StandardError=append:$working_dir/logs/streaming-server.error.log

[Install]
WantedBy=multi-user.target
EOF

    print_info "Generated systemd service file: ${PROJECT_NAME}.service"
    print_warning "To install as system service, run:"
    print_warning "  sudo cp ${PROJECT_NAME}.service /etc/systemd/system/"
    print_warning "  sudo systemctl enable ${PROJECT_NAME}"
    print_warning "  sudo systemctl start ${PROJECT_NAME}"
}

# Generate Docker files
generate_docker() {
    cat > Dockerfile << 'EOF'
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o streaming-server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/streaming-server .
COPY config.json .

# Create videos directory
RUN mkdir -p videos

EXPOSE 9000

CMD ["./streaming-server", "-config", "config.json"]
EOF

    cat > docker-compose.yml << EOF
version: '3.8'

services:
  streaming-server:
    build: .
    ports:
      - "9000:9000"
    volumes:
      - ./videos:/root/videos
      - ./logs:/root/logs
    restart: unless-stopped
    environment:
      - TZ=Asia/Shanghai
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
EOF

    cat > .dockerignore << 'EOF'
.git
*.log
logs/
videos/
streaming-server
test.html
README.md
*.service
EOF

    print_info "Generated Docker files: Dockerfile, docker-compose.yml, .dockerignore"
}

# Show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  build       Build the streaming server"
    echo "  run         Build and run the server"
    echo "  service     Generate systemd service file"
    echo "  docker      Generate Docker files"
    echo "  clean       Clean build artifacts"
    echo "  help        Show this help message"
    echo ""
    echo "Options (for run command):"
    echo "  -p, --port PORT           Server port (default: $DEFAULT_PORT)"
    echo "  -d, --video-dir DIR       Video directory (default: $DEFAULT_VIDEO_DIR)"
    echo "  -c, --max-conns NUM       Max connections (default: $DEFAULT_MAX_CONNS)"
    echo "  -f, --config FILE         Config file path"
    echo ""
    echo "Examples:"
    echo "  $0 build                               # Just build"
    echo "  $0 run                                 # Build and run with defaults"
    echo "  $0 run -p 8080 -d /var/videos         # Custom port and directory"
    echo "  $0 run -f production.json              # Use config file"
    echo "  $0 service                             # Generate systemd service"
    echo "  $0 docker                              # Generate Docker files"
}

# Run the server
run_server() {
    local port=$DEFAULT_PORT
    local video_dir=$DEFAULT_VIDEO_DIR
    local max_conns=$DEFAULT_MAX_CONNS
    local config_file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--port)
                port="$2"
                shift 2
                ;;
            -d|--video-dir)
                video_dir="$2"
                shift 2
                ;;
            -c|--max-conns)
                max_conns="$2"
                shift 2
                ;;
            -f|--config)
                config_file="$2"
                shift 2
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    setup_directories
    build_server
    
    print_info "Starting streaming server..."
    print_info "Port: $port"
    print_info "Video Directory: $video_dir"
    print_info "Max Connections: $max_conns"
    
    if [ -n "$config_file" ]; then
        print_info "Config File: $config_file"
    fi
    
    # Build command
    cmd="./streaming-server -port $port -video-dir $video_dir -max-conns $max_conns"
    if [ -n "$config_file" ]; then
        cmd="$cmd -config $config_file"
    fi
    
    print_success "Server starting..."
    print_info "Access the server at: http://localhost:$port"
    print_info "Health check: http://localhost:$port/health"
    print_info "API info: http://localhost:$port/api/info"
    print_info "Test page: file://$(pwd)/test.html"
    print_info ""
    print_info "Press Ctrl+C to stop the server"
    
    exec $cmd
}

# Clean build artifacts
clean() {
    print_info "Cleaning build artifacts..."
    rm -f streaming-server
    rm -f *.service
    rm -f Dockerfile docker-compose.yml .dockerignore
    print_success "Clean completed"
}

# Main script logic
main() {
    case "${1:-help}" in
        build)
            check_go
            setup_directories
            build_server
            print_success "Build completed. Run './streaming-server' to start."
            ;;
        run)
            check_go
            shift
            run_server "$@"
            ;;
        service)
            setup_directories
            generate_systemd_service
            ;;
        docker)
            generate_docker
            ;;
        clean)
            clean
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"