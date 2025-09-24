#!/bin/bash

# Docker management script for Standalone Stream Server
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
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

# Show usage
show_usage() {
    cat << EOF
Docker Management Script for Standalone Stream Server

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    build           Build the streaming server image
    up              Start all services
    down            Stop all services
    restart         Restart all services
    logs            Show logs for all services
    logs <service>  Show logs for specific service
    status          Show status of all services
    clean           Clean up containers, images, and volumes
    backup          Backup video files and configurations
    restore         Restore from backup
    update          Update and restart services
    scale           Scale streaming server instances
    monitor         Open monitoring dashboard

Options:
    --dev           Use development configuration
    --prod          Use production configuration (default)
    --force         Force operation without confirmation
    -f              Follow logs output

Examples:
    $0 build --dev
    $0 up --prod
    $0 logs streaming-server -f
    $0 scale 3
    $0 backup /backup/location
