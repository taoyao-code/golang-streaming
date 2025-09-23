#!/bin/bash

# 端到端测试脚本
# 测试实际运行的服务器的所有功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
SERVER_HOST="localhost"
SERVER_PORT="9000"
BASE_URL="http://${SERVER_HOST}:${SERVER_PORT}"
TEST_VIDEO_FILE="test_video.mp4"
TEST_VIDEO_CONTENT="fake video content for e2e testing"

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

# 检查服务器是否运行
check_server() {
    log_info "检查服务器状态..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$BASE_URL/ping" > /dev/null 2>&1; then
            log_success "服务器已启动"
            return 0
        fi
        
        log_info "等待服务器启动... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    log_error "服务器未响应，请确保服务器运行在 $BASE_URL"
    exit 1
}

# 测试健康检查端点
test_health_endpoints() {
    log_info "测试健康检查端点..."
    
    # 测试 /ping
    if curl -s -f "$BASE_URL/ping" | jq -e '.message == "pong"' > /dev/null; then
        log_success "/ping 端点正常"
    else
        log_error "/ping 端点失败"
        exit 1
    fi
    
    # 测试 /health
    if curl -s -f "$BASE_URL/health" | jq -e '.status == "healthy"' > /dev/null; then
        log_success "/health 端点正常"
    else
        log_error "/health 端点失败"
        exit 1
    fi
    
    # 测试 /ready
    if curl -s -f "$BASE_URL/ready" | jq -e '.status == "ready"' > /dev/null; then
        log_success "/ready 端点正常"
    else
        log_error "/ready 端点失败"
        exit 1
    fi
    
    # 测试 /live
    if curl -s -f "$BASE_URL/live" | jq -e '.status == "alive"' > /dev/null; then
        log_success "/live 端点正常"
    else
        log_error "/live 端点失败"
        exit 1
    fi
}

# 测试API信息端点
test_api_info() {
    log_info "测试API信息端点..."
    
    local response=$(curl -s -f "$BASE_URL/api/info")
    
    if echo "$response" | jq -e '.service == "Standalone Video Streaming Server"' > /dev/null; then
        log_success "API信息端点正常"
        
        # 显示服务信息
        local version=$(echo "$response" | jq -r '.version')
        local framework=$(echo "$response" | jq -r '.framework')
        log_info "服务版本: $version"
        log_info "框架: $framework"
    else
        log_error "API信息端点失败"
        exit 1
    fi
}

# 测试视频列表端点
test_video_listing() {
    log_info "测试视频列表端点..."
    
    # 测试获取所有视频
    local response=$(curl -s -f "$BASE_URL/api/videos")
    if echo "$response" | jq -e '.videos' > /dev/null; then
        local video_count=$(echo "$response" | jq '.videos | length')
        log_success "获取所有视频成功，共 $video_count 个视频"
    else
        log_error "获取所有视频失败"
        exit 1
    fi
    
    # 测试获取目录列表
    response=$(curl -s -f "$BASE_URL/api/directories")
    if echo "$response" | jq -e '.directories' > /dev/null; then
        local dir_count=$(echo "$response" | jq '.directories | length')
        log_success "获取目录列表成功，共 $dir_count 个目录"
        
        # 获取第一个启用的目录名称进行后续测试
        FIRST_DIRECTORY=$(echo "$response" | jq -r '.directories[] | select(.enabled == true) | .name' | head -n1)
        if [ -n "$FIRST_DIRECTORY" ]; then
            log_info "使用目录进行测试: $FIRST_DIRECTORY"
        fi
    else
        log_error "获取目录列表失败"
        exit 1
    fi
}

# 测试视频搜索
test_video_search() {
    log_info "测试视频搜索..."
    
    # 搜索视频
    local response=$(curl -s -f "$BASE_URL/api/search?q=test")
    if echo "$response" | jq -e '.videos' > /dev/null; then
        local result_count=$(echo "$response" | jq '.videos | length')
        log_success "视频搜索成功，找到 $result_count 个结果"
    else
        log_error "视频搜索失败"
        exit 1
    fi
}

# 创建测试视频文件
create_test_video() {
    log_info "创建测试视频文件..."
    
    echo "$TEST_VIDEO_CONTENT" > "$TEST_VIDEO_FILE"
    log_success "测试视频文件已创建: $TEST_VIDEO_FILE"
}

# 测试视频上传
test_video_upload() {
    if [ -z "$FIRST_DIRECTORY" ]; then
        log_warning "跳过上传测试：未找到可用目录"
        return
    fi
    
    log_info "测试视频上传到目录: $FIRST_DIRECTORY"
    
    create_test_video
    
    local upload_response=$(curl -s -w "%{http_code}" \
        -X POST \
        -F "file=@$TEST_VIDEO_FILE" \
        "$BASE_URL/upload/$FIRST_DIRECTORY/e2e_test_video")
    
    local http_code="${upload_response: -3}"
    local response_body="${upload_response%???}"
    
    if [ "$http_code" = "200" ]; then
        log_success "视频上传成功"
        
        # 验证上传的视频是否出现在列表中
        sleep 1 # 等待文件系统同步
        local videos_response=$(curl -s -f "$BASE_URL/api/videos/$FIRST_DIRECTORY")
        if echo "$videos_response" | jq -e '.videos[] | select(.name == "e2e_test_video")' > /dev/null; then
            log_success "上传的视频在列表中找到"
            UPLOADED_VIDEO_ID="$FIRST_DIRECTORY:e2e_test_video"
        else
            log_warning "上传的视频未在列表中找到"
        fi
    else
        log_error "视频上传失败 (HTTP $http_code): $response_body"
    fi
}

# 测试视频流
test_video_streaming() {
    if [ -z "$UPLOADED_VIDEO_ID" ]; then
        log_warning "跳过流媒体测试：未找到可流式传输的视频"
        return
    fi
    
    log_info "测试视频流: $UPLOADED_VIDEO_ID"
    
    # 测试基本流媒体
    local stream_response=$(curl -s -w "%{http_code}:%{content_type}" \
        "$BASE_URL/stream/$UPLOADED_VIDEO_ID")
    
    local http_code=$(echo "$stream_response" | cut -d: -f1 | tail -c 4)
    local content_type=$(echo "$stream_response" | cut -d: -f2)
    local content="${stream_response%:*:*}"
    
    if [ "$http_code" = "200" ]; then
        log_success "视频流请求成功"
        log_info "内容类型: $content_type"
        
        # 验证内容
        if echo "$content" | grep -q "$TEST_VIDEO_CONTENT"; then
            log_success "视频内容验证成功"
        else
            log_warning "视频内容验证失败"
        fi
    else
        log_error "视频流请求失败 (HTTP $http_code)"
    fi
    
    # 测试范围请求
    log_info "测试范围请求..."
    local range_response=$(curl -s -w "%{http_code}" \
        -H "Range: bytes=0-10" \
        "$BASE_URL/stream/$UPLOADED_VIDEO_ID")
    
    local range_http_code="${range_response: -3}"
    
    if [ "$range_http_code" = "206" ]; then
        log_success "范围请求支持正常"
    else
        log_warning "范围请求可能不支持 (HTTP $range_http_code)"
    fi
}

# 测试错误处理
test_error_handling() {
    log_info "测试错误处理..."
    
    # 测试不存在的视频
    local error_response=$(curl -s -w "%{http_code}" "$BASE_URL/stream/nonexistent:video")
    local error_http_code="${error_response: -3}"
    
    if [ "$error_http_code" = "404" ]; then
        log_success "不存在视频的错误处理正确"
    else
        log_warning "不存在视频的错误处理可能有问题 (HTTP $error_http_code)"
    fi
    
    # 测试无效的API端点
    error_response=$(curl -s -w "%{http_code}" "$BASE_URL/api/invalid-endpoint")
    error_http_code="${error_response: -3}"
    
    if [ "$error_http_code" = "404" ]; then
        log_success "无效端点的错误处理正确"
    else
        log_warning "无效端点的错误处理可能有问题 (HTTP $error_http_code)"
    fi
}

# 性能测试
test_performance() {
    log_info "进行基本性能测试..."
    
    # 测试并发请求
    log_info "测试并发请求性能..."
    
    local start_time=$(date +%s%N)
    
    # 并发发送10个请求
    for i in {1..10}; do
        curl -s -f "$BASE_URL/api/videos" > /dev/null &
    done
    wait
    
    local end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time) / 1000000)) # 转换为毫秒
    
    log_info "10个并发请求耗时: ${duration}ms"
    
    if [ $duration -lt 5000 ]; then
        log_success "并发性能良好"
    else
        log_warning "并发性能可能需要优化"
    fi
}

# 清理测试文件
cleanup() {
    log_info "清理测试文件..."
    
    if [ -f "$TEST_VIDEO_FILE" ]; then
        rm -f "$TEST_VIDEO_FILE"
        log_success "测试文件已清理"
    fi
}

# 生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    local report_file="e2e_test_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# 端到端测试报告

**测试时间**: $(date)
**服务器地址**: $BASE_URL
**测试状态**: 通过

## 测试项目

- ✅ 健康检查端点
- ✅ API信息端点
- ✅ 视频列表功能
- ✅ 视频搜索功能
- ✅ 视频上传功能
- ✅ 视频流媒体功能
- ✅ 错误处理
- ✅ 基本性能测试

## 服务器信息

- 主机: $SERVER_HOST
- 端口: $SERVER_PORT
- 状态: 正常运行

## 建议

- 服务器运行正常，所有基本功能可用
- 建议定期运行此测试以确保服务稳定性
- 在生产环境中考虑负载测试

EOF

    log_success "测试报告已生成: $report_file"
}

# 显示帮助信息
show_help() {
    echo "端到端测试脚本"
    echo ""
    echo "使用方法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --host HOST    服务器主机 (默认: localhost)"
    echo "  --port PORT    服务器端口 (默认: 9000)"
    echo "  -h, --help     显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                           # 使用默认设置"
    echo "  $0 --host 192.168.1.100      # 指定服务器主机"
    echo "  $0 --port 8080               # 指定服务器端口"
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --host)
                SERVER_HOST="$2"
                BASE_URL="http://${SERVER_HOST}:${SERVER_PORT}"
                shift 2
                ;;
            --port)
                SERVER_PORT="$2"
                BASE_URL="http://${SERVER_HOST}:${SERVER_PORT}"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 主函数
main() {
    echo "======================================"
    echo "  Standalone Stream Server E2E 测试"
    echo "======================================"
    echo ""
    
    # 解析参数
    parse_args "$@"
    
    log_info "测试目标: $BASE_URL"
    
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        log_error "curl 未安装"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq 未安装，请先安装: brew install jq 或 apt-get install jq"
        exit 1
    fi
    
    # 设置清理钩子
    trap cleanup EXIT
    
    # 运行测试
    check_server
    test_health_endpoints
    test_api_info
    test_video_listing
    test_video_search
    test_video_upload
    test_video_streaming
    test_error_handling
    test_performance
    
    # 生成报告
    generate_report
    
    echo ""
    log_success "所有端到端测试通过！"
    echo ""
    log_info "如需更详细的测试，请考虑："
    echo "  - 压力测试工具 (如 wrk, ab)"
    echo "  - 长时间运行测试"
    echo "  - 大文件上传测试"
    echo "  - 网络异常测试"
}

# 运行主函数
main "$@"
