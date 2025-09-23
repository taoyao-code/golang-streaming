#!/bin/bash

# 视频流媒体服务器 Docker 管理脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
COMPOSE_FILE="docker-compose.yml"
SERVICE_NAME="video-streaming-server"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 和 Docker Compose
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或未在 PATH 中"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装或未在 PATH 中"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 构建镜像
build() {
    log_info "构建视频流媒体服务器镜像..."
    docker-compose build --no-cache
    log_success "镜像构建完成"
}

# 启动服务（基础模式）
start() {
    log_info "启动视频流媒体服务器..."
    docker-compose up -d
    log_success "服务启动完成"
    show_status
}

# 启动服务（生产模式，包含 Nginx）
start_production() {
    log_info "启动生产环境（包含 Nginx）..."
    docker-compose --profile production up -d
    log_success "生产环境启动完成"
    show_status
}

# 启动服务（包含监控）
start_monitoring() {
    log_info "启动服务和监控..."
    docker-compose --profile monitoring up -d
    log_success "服务和监控启动完成"
    show_status
}

# 启动所有服务
start_all() {
    log_info "启动所有服务..."
    docker-compose --profile production --profile monitoring up -d
    log_success "所有服务启动完成"
    show_status
}

# 停止服务
stop() {
    log_info "停止服务..."
    docker-compose down
    log_success "服务已停止"
}

# 重启服务
restart() {
    log_info "重启服务..."
    stop
    start
}

# 查看状态
show_status() {
    log_info "服务状态:"
    docker-compose ps
    
    echo ""
    log_info "服务访问地址:"
    echo "  - 视频流媒体服务: http://localhost:9000"
    echo "  - 管理面板: http://localhost:9000/dashboard"
    echo "  - API 文档: http://localhost:9000/api/info"
    echo "  - 健康检查: http://localhost:9000/health"
    
    if docker-compose ps | grep -q nginx; then
        echo "  - Nginx 反向代理: http://localhost:80"
    fi
    
    if docker-compose ps | grep -q prometheus; then
        echo "  - Prometheus 监控: http://localhost:9090"
    fi
    
    if docker-compose ps | grep -q grafana; then
        echo "  - Grafana 仪表板: http://localhost:3000 (admin/admin123)"
    fi
}

# 查看日志
logs() {
    local service=${1:-$SERVICE_NAME}
    log_info "查看 $service 服务日志..."
    docker-compose logs -f $service
}

# 进入容器
shell() {
    local service=${1:-$SERVICE_NAME}
    log_info "进入 $service 容器..."
    docker-compose exec $service /bin/sh
}

# 清理
cleanup() {
    log_warning "这将删除所有容器、镜像和数据卷"
    read -p "确定要继续吗? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理 Docker 资源..."
        docker-compose down -v --rmi all
        docker system prune -f
        log_success "清理完成"
    else
        log_info "取消清理操作"
    fi
}

# 更新服务
update() {
    log_info "更新服务..."
    git pull
    build
    restart
    log_success "服务更新完成"
}

# 备份数据
backup() {
    local backup_dir="backups/$(date +%Y%m%d_%H%M%S)"
    log_info "备份数据到 $backup_dir..."
    
    mkdir -p $backup_dir
    
    # 备份数据卷
    docker run --rm -v standalone-stream-server_streaming_data:/data -v $(pwd)/$backup_dir:/backup alpine tar czf /backup/streaming_data.tar.gz /data
    
    # 备份配置
    cp -r configs $backup_dir/
    
    log_success "备份完成: $backup_dir"
}

# 恢复数据
restore() {
    local backup_path=$1
    if [ -z "$backup_path" ]; then
        log_error "请提供备份路径"
        exit 1
    fi
    
    log_warning "这将覆盖现有数据"
    read -p "确定要继续吗? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "恢复数据从 $backup_path..."
        
        # 停止服务
        stop
        
        # 恢复数据卷
        docker run --rm -v standalone-stream-server_streaming_data:/data -v $(pwd)/$backup_path:/backup alpine tar xzf /backup/streaming_data.tar.gz -C /
        
        # 恢复配置
        cp -r $backup_path/configs/* configs/
        
        # 重启服务
        start
        
        log_success "数据恢复完成"
    else
        log_info "取消恢复操作"
    fi
}

# 显示帮助
show_help() {
    echo "视频流媒体服务器 Docker 管理脚本"
    echo ""
    echo "用法: $0 <命令> [参数]"
    echo ""
    echo "可用命令:"
    echo "  build              构建镜像"
    echo "  start              启动基础服务"
    echo "  start-prod         启动生产环境（包含 Nginx）"
    echo "  start-monitoring   启动服务和监控"
    echo "  start-all          启动所有服务"
    echo "  stop               停止所有服务"
    echo "  restart            重启服务"
    echo "  status             显示服务状态"
    echo "  logs [service]     查看日志"
    echo "  shell [service]    进入容器"
    echo "  update             更新服务"
    echo "  backup             备份数据"
    echo "  restore <path>     恢复数据"
    echo "  cleanup            清理所有资源"
    echo "  help               显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 start              # 启动基础服务"
    echo "  $0 logs               # 查看主服务日志"
    echo "  $0 logs nginx         # 查看 Nginx 日志"
    echo "  $0 shell              # 进入主服务容器"
    echo "  $0 backup             # 备份当前数据"
    echo "  $0 restore backups/20231201_120000  # 恢复指定备份"
}

# 主函数
main() {
    check_dependencies
    
    case "${1:-help}" in
        build)
            build
            ;;
        start)
            start
            ;;
        start-prod|start-production)
            start_production
            ;;
        start-monitoring)
            start_monitoring
            ;;
        start-all)
            start_all
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        status)
            show_status
            ;;
        logs)
            logs $2
            ;;
        shell)
            shell $2
            ;;
        update)
            update
            ;;
        backup)
            backup
            ;;
        restore)
            restore $2
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"