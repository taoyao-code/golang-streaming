#!/bin/bash

# standalone-stream-server 测试脚本
# 使用方法: ./scripts/test.sh [选项]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认参数
VERBOSE=false
COVERAGE=false
RACE=false
INTEGRATION=false
BENCHMARK=false
CLEAN=false

# 帮助信息
show_help() {
    echo "standalone-stream-server 测试脚本"
    echo ""
    echo "使用方法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -v, --verbose      显示详细输出"
    echo "  -c, --coverage     生成覆盖率报告"
    echo "  -r, --race         启用竞态检测"
    echo "  -i, --integration  运行集成测试"
    echo "  -b, --benchmark    运行基准测试"
    echo "  -a, --all          运行所有测试（单元+集成+基准）"
    echo "  --clean           清理测试缓存和输出文件"
    echo "  -h, --help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                 # 运行单元测试"
    echo "  $0 -c              # 运行单元测试并生成覆盖率报告"
    echo "  $0 -i              # 运行集成测试"
    echo "  $0 -a              # 运行所有测试"
    echo "  $0 --clean         # 清理测试文件"
}

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

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -r|--race)
                RACE=true
                shift
                ;;
            -i|--integration)
                INTEGRATION=true
                shift
                ;;
            -b|--benchmark)
                BENCHMARK=true
                shift
                ;;
            -a|--all)
                INTEGRATION=true
                BENCHMARK=true
                COVERAGE=true
                shift
                ;;
            --clean)
                CLEAN=true
                shift
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

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go 版本: $GO_VERSION"
    
    # 检查 Go 模块
    if [ ! -f "go.mod" ]; then
        log_error "go.mod 文件不存在"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 清理函数
clean_test_files() {
    log_info "清理测试文件..."
    
    # 清理覆盖率文件
    rm -f coverage.out coverage.html
    
    # 清理测试缓存
    go clean -testcache
    
    # 清理临时文件
    find . -name "*.tmp" -delete 2>/dev/null || true
    find . -name "test_*.log" -delete 2>/dev/null || true
    
    log_success "清理完成"
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    local test_flags=""
    
    if [ "$VERBOSE" = true ]; then
        test_flags="$test_flags -v"
    fi
    
    if [ "$RACE" = true ]; then
        test_flags="$test_flags -race"
        log_info "启用竞态检测"
    fi
    
    if [ "$COVERAGE" = true ]; then
        test_flags="$test_flags -coverprofile=coverage.out"
        log_info "启用覆盖率收集"
    fi
    
    # 运行单元测试（排除 tests 目录）
    if go test $test_flags $(go list ./... | grep -v /tests); then
        log_success "单元测试通过"
        
        # 生成覆盖率报告
        if [ "$COVERAGE" = true ] && [ -f "coverage.out" ]; then
            generate_coverage_report
        fi
    else
        log_error "单元测试失败"
        exit 1
    fi
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    local test_flags=""
    
    if [ "$VERBOSE" = true ]; then
        test_flags="$test_flags -v"
    fi
    
    if [ "$RACE" = true ]; then
        test_flags="$test_flags -race"
    fi
    
    # 运行集成测试
    if go test $test_flags ./tests/...; then
        log_success "集成测试通过"
    else
        log_error "集成测试失败"
        exit 1
    fi
}

# 运行基准测试
run_benchmark_tests() {
    log_info "运行基准测试..."
    
    local bench_flags="-bench=."
    
    if [ "$VERBOSE" = true ]; then
        bench_flags="$bench_flags -v"
    fi
    
    # 创建基准测试结果目录
    mkdir -p benchmarks
    
    # 运行基准测试并保存结果
    local bench_file="benchmarks/benchmark_$(date +%Y%m%d_%H%M%S).txt"
    
    if go test $bench_flags -run=^$ ./... | tee "$bench_file"; then
        log_success "基准测试完成，结果保存到: $bench_file"
    else
        log_error "基准测试失败"
        exit 1
    fi
}

# 生成覆盖率报告
generate_coverage_report() {
    log_info "生成覆盖率报告..."
    
    # 显示覆盖率统计
    go tool cover -func=coverage.out
    
    # 生成HTML报告
    go tool cover -html=coverage.out -o coverage.html
    
    # 计算总覆盖率
    local total_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    log_success "总覆盖率: $total_coverage"
    log_info "HTML报告已生成: coverage.html"
    
    # 如果覆盖率低于阈值，发出警告
    local threshold=80
    local coverage_num=$(echo $total_coverage | sed 's/%//')
    if (( $(echo "$coverage_num < $threshold" | bc -l) )); then
        log_warning "覆盖率 ($total_coverage) 低于推荐阈值 (${threshold}%)"
    fi
}

# 运行静态分析
run_static_analysis() {
    log_info "运行静态分析..."
    
    # 检查 go fmt
    if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
        log_warning "代码格式不符合 go fmt 标准:"
        gofmt -l .
        log_info "运行 'go fmt ./...' 来修复格式问题"
    else
        log_success "代码格式检查通过"
    fi
    
    # 检查 go vet
    if go vet ./...; then
        log_success "go vet 检查通过"
    else
        log_error "go vet 检查失败"
        exit 1
    fi
    
    # 如果安装了 golint，运行它
    if command -v golint &> /dev/null; then
        log_info "运行 golint..."
        golint ./...
    fi
    
    # 如果安装了 staticcheck，运行它
    # if command -v staticcheck &> /dev/null; then
    #     log_info "运行 staticcheck..."
    #     staticcheck ./...
    # fi
}

# 创建测试摘要
create_test_summary() {
    log_info "创建测试摘要..."
    
    local summary_file="test_summary_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$summary_file" << EOF
# 测试摘要

**测试时间**: $(date)
**Go 版本**: $(go version)

## 测试配置
- 详细输出: $VERBOSE
- 覆盖率收集: $COVERAGE
- 竞态检测: $RACE
- 集成测试: $INTEGRATION
- 基准测试: $BENCHMARK

## 测试结果
EOF
    
    if [ "$COVERAGE" = true ] && [ -f "coverage.out" ]; then
        local total_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        echo "- **覆盖率**: $total_coverage" >> "$summary_file"
    fi
    
    echo "" >> "$summary_file"
    echo "## 建议" >> "$summary_file"
    echo "- 定期运行完整测试套件" >> "$summary_file"
    echo "- 保持测试覆盖率在80%以上" >> "$summary_file"
    echo "- 在生产部署前运行集成测试" >> "$summary_file"
    
    log_success "测试摘要已生成: $summary_file"
}

# 主函数
main() {
    echo "=================================="
    echo "  Standalone Stream Server 测试"
    echo "=================================="
    echo ""
    
    # 解析参数
    parse_args "$@"
    
    # 如果只是清理，执行清理后退出
    if [ "$CLEAN" = true ]; then
        clean_test_files
        exit 0
    fi
    
    # 检查依赖
    check_dependencies
    
    # 清理旧的测试文件
    clean_test_files
    
    # 运行静态分析
    run_static_analysis
    
    # 运行单元测试
    run_unit_tests
    
    # 运行集成测试
    if [ "$INTEGRATION" = true ]; then
        run_integration_tests
    fi
    
    # 运行基准测试
    if [ "$BENCHMARK" = true ]; then
        run_benchmark_tests
    fi
    
    # 创建测试摘要
    create_test_summary
    
    echo ""
    log_success "所有测试完成！"
    
    if [ "$COVERAGE" = true ]; then
        log_info "查看覆盖率报告: open coverage.html"
    fi
}

# 运行主函数
main "$@"
