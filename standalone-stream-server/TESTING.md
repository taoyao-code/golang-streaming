# 测试指南

本文档提供了 Standalone Video Streaming Server 的完整测试指南，包括单元测试、集成测试、端到端测试和性能测试。

## 📋 目录

- [测试概述](#测试概述)
- [测试环境设置](#测试环境设置)
- [运行测试](#运行测试)
- [测试类型](#测试类型)
- [测试工具](#测试工具)
- [CI/CD 集成](#cicd-集成)
- [最佳实践](#最佳实践)
- [故障排除](#故障排除)

## 🎯 测试概述

我们的测试策略包括：

- **单元测试**: 测试独立的函数和方法
- **集成测试**: 测试组件之间的交互
- **端到端测试**: 测试完整的用户场景
- **性能测试**: 测试系统性能和负载能力
- **安全测试**: 测试安全漏洞和配置

## 🛠️ 测试环境设置

### 前提条件

```bash
# 1. Go 1.21+
go version

# 2. 测试工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# 3. 系统工具（用于端到端测试）
# macOS
brew install curl jq

# Ubuntu/Debian
sudo apt-get install curl jq

# 4. 可选：性能测试工具
brew install wrk  # 或从源码编译
```

### 项目依赖

```bash
# 进入项目目录
cd standalone-stream-server

# 下载依赖
make deps
# 或
go mod download && go mod verify
```

## 🚀 运行测试

### 使用 Makefile（推荐）

```bash
# 查看所有可用目标
make help

# 运行所有检查和测试
make check

# 单独运行不同类型的测试
make test-unit          # 单元测试
make test-integration   # 集成测试
make test-coverage     # 覆盖率测试
make test-race         # 竞态检测
make test-benchmark    # 基准测试
make test-all          # 所有测试

# 端到端测试（需要服务器运行）
make run &              # 后台运行服务器
make test-e2e          # 运行端到端测试
```

### 使用测试脚本

```bash
# 基本单元测试
./scripts/test.sh

# 带覆盖率的测试
./scripts/test.sh -c

# 集成测试
./scripts/test.sh -i

# 所有测试
./scripts/test.sh -a

# 查看脚本帮助
./scripts/test.sh --help
```

### 使用原生 Go 命令

```bash
# 运行所有单元测试
go test ./...

# 详细输出
go test -v ./...

# 覆盖率测试
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 竞态检测
go test -race ./...

# 基准测试
go test -bench=. ./...

# 仅运行特定测试
go test -run TestVideoService_ListVideos ./internal/services

# 运行集成测试
go test ./tests/...
```

## 🧪 测试类型

### 1. 单元测试

测试独立的函数和方法：

```bash
# 运行所有单元测试
go test $(go list ./... | grep -v /tests)

# 测试特定包
go test ./internal/services
go test ./internal/handlers
go test ./internal/config
```

**单元测试覆盖的模块：**

- `internal/services/video_test.go` - 视频服务逻辑
- `internal/handlers/health_test.go` - 健康检查处理器
- `internal/config/config_test.go` - 配置管理
- `internal/middleware/` - 中间件功能

### 2. 集成测试

测试组件之间的交互：

```bash
# 运行集成测试
go test ./tests/...
# 或
make test-integration
```

**集成测试包括：**

- HTTP API 端点测试
- 文件上传和下载流程
- 视频流媒体功能
- 错误处理机制
- CORS 和安全配置

### 3. 端到端测试

测试完整的用户场景：

```bash
# 1. 启动服务器
make run &

# 2. 运行端到端测试
./scripts/e2e_test.sh

# 3. 或者指定自定义服务器
./scripts/e2e_test.sh --host localhost --port 8080
```

**端到端测试场景：**

- 服务器启动和健康检查
- 视频列表和搜索
- 视频上传流程
- 视频流媒体播放
- 范围请求支持
- 错误处理
- 基本性能测试

### 4. 性能测试

测试系统性能：

```bash
# 基准测试
make test-benchmark

# CPU 性能分析
make profile-cpu

# 内存性能分析
make profile-mem

# 使用 wrk 进行负载测试（需要服务器运行）
wrk -t12 -c400 -d30s http://localhost:9000/api/videos
```

### 5. 安全测试

检查安全漏洞：

```bash
# 静态安全分析
make security-check

# 或直接使用 gosec
gosec ./...

# 检查依赖漏洞
go list -json -m all | nancy sleuth
```

## 🔧 测试工具

### 内置工具

| 工具 | 用途 | 命令示例 |
|------|------|----------|
| `go test` | 运行测试 | `go test -v ./...` |
| `go test -cover` | 覆盖率分析 | `go test -coverprofile=coverage.out ./...` |
| `go test -race` | 竞态检测 | `go test -race ./...` |
| `go test -bench` | 基准测试 | `go test -bench=. ./...` |
| `go vet` | 静态分析 | `go vet ./...` |

### 第三方工具

```bash
# 安装常用测试工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/cosmtrek/air@latest  # 热重载开发

# 使用 golangci-lint 进行全面代码检查
golangci-lint run

# 使用 gosec 进行安全检查
gosec ./...
```

## 📊 测试报告

### 覆盖率报告

```bash
# 生成覆盖率报告
make test-coverage

# 查看文本报告
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
```

### 测试摘要

测试脚本会自动生成测试摘要：

```bash
./scripts/test.sh -a
# 生成 test_summary_YYYYMMDD_HHMMSS.md

./scripts/e2e_test.sh
# 生成 e2e_test_report_YYYYMMDD_HHMMSS.md
```

## 🔄 CI/CD 集成

### GitHub Actions 示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.21
    
    - name: Download dependencies
      run: make deps
    
    - name: Run tests
      run: make ci
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

### GitLab CI 示例

```yaml
stages:
  - test
  - integration

test:
  stage: test
  image: golang:1.21
  script:
    - make deps
    - make check
    - make test-coverage
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml

integration:
  stage: integration
  script:
    - make run &
    - sleep 5
    - make test-e2e
```

## 📝 编写测试的最佳实践

### 1. 测试结构

```go
func TestServiceFunction(t *testing.T) {
    // Arrange - 准备测试数据
    service := NewService(config)
    input := "test input"
    
    // Act - 执行被测试的功能
    result, err := service.Function(input)
    
    // Assert - 验证结果
    if err != nil {
        t.Fatal(err)
    }
    
    if result != expectedResult {
        t.Errorf("Expected %v, got %v", expectedResult, result)
    }
}
```

### 2. 表驱动测试

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    bool
        shouldError bool
    }{
        {"valid input", "valid", true, false},
        {"invalid input", "invalid", false, false},
        {"empty input", "", false, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Validate(tt.input)
            
            if tt.shouldError && err == nil {
                t.Error("Expected error but got none")
            }
            
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### 3. 测试辅助函数

```go
// 创建测试服务器
func setupTestServer(t *testing.T) (*fiber.App, *models.Config) {
    t.Helper()
    
    config := &models.Config{
        // 测试配置
    }
    
    app := fiber.New()
    // 设置路由
    
    return app, config
}

// 创建测试文件
func createTestFile(t *testing.T, dir, filename, content string) {
    t.Helper()
    
    filePath := filepath.Join(dir, filename)
    err := os.WriteFile(filePath, []byte(content), 0644)
    if err != nil {
        t.Fatal(err)
    }
}
```

### 4. 模拟和存根

```go
// 模拟接口
type MockVideoService struct {
    videos []Video
    err    error
}

func (m *MockVideoService) ListVideos(directory string) ([]Video, error) {
    return m.videos, m.err
}

// 在测试中使用
func TestHandler(t *testing.T) {
    mockService := &MockVideoService{
        videos: []Video{{Name: "test"}},
        err:    nil,
    }
    
    handler := NewHandler(mockService)
    // 测试处理器
}
```

## 🐛 故障排除

### 常见问题

#### 1. 测试文件权限错误

```bash
# 确保测试目录可写
chmod -R 755 ./tests/
mkdir -p ./tmp/test/
```

#### 2. 竞态条件检测

```bash
# 如果发现竞态条件，使用详细输出查看详情
go test -race -v ./...

# 查看竞态条件报告
GORACE="log_path=./race_report" go test -race ./...
```

#### 3. 内存泄漏检测

```bash
# 使用内存分析检查泄漏
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

#### 4. 端到端测试失败

```bash
# 检查服务器是否运行
curl http://localhost:9000/ping

# 检查端口是否被占用
lsof -i :9000

# 查看服务器日志
./build/streaming-server 2>&1 | tee server.log
```

#### 5. 依赖问题

```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 验证依赖
go mod verify
```

### 调试技巧

#### 1. 详细测试输出

```bash
# 显示所有测试输出
go test -v ./...

# 显示测试覆盖的包
go test -v -cover ./...
```

#### 2. 只运行失败的测试

```bash
# 运行特定测试
go test -run TestSpecificFunction ./...

# 使用正则表达式过滤测试
go test -run "TestVideo.*" ./...
```

#### 3. 测试超时设置

```bash
# 设置测试超时
go test -timeout 30s ./...

# 针对特定慢测试
go test -timeout 5m ./tests/...
```

## 📈 性能测试详解

### 基准测试

```go
func BenchmarkVideoListing(b *testing.B) {
    service := setupService()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.ListVideos("")
        if err != nil {
            b.Fatal(err)
        }
    }
}

// 运行基准测试
go test -bench=BenchmarkVideoListing -benchmem ./...
```

### 负载测试

```bash
# 使用 wrk 进行 HTTP 负载测试
wrk -t12 -c400 -d30s --script=load_test.lua http://localhost:9000/

# 使用 Apache Bench
ab -n 1000 -c 10 http://localhost:9000/api/videos

# 使用 hey
hey -z 30s -c 50 http://localhost:9000/stream/movies:sample
```

### 内存和 CPU 分析

```bash
# CPU 分析
go test -cpuprofile cpu.prof -bench=. ./...
go tool pprof cpu.prof

# 内存分析
go test -memprofile mem.prof -bench=. ./...
go tool pprof mem.prof

# 阻塞分析
go test -blockprofile block.prof -bench=. ./...
go tool pprof block.prof
```

## 🏆 测试指标和目标

### 覆盖率目标

- **单元测试覆盖率**: ≥ 80%
- **集成测试覆盖率**: ≥ 70%
- **关键路径覆盖率**: ≥ 95%

### 性能目标

- **API 响应时间**: < 100ms (P95)
- **视频流启动时间**: < 500ms
- **并发连接数**: ≥ 100
- **内存使用**: < 100MB (空闲状态)

### 质量门槛

- **所有测试通过**: 100%
- **无竞态条件**: 0 检测
- **无内存泄漏**: 0 检测
- **安全漏洞**: 0 高危/中危

## 🔗 相关资源

- [Go 测试官方文档](https://golang.org/pkg/testing/)
- [GoFiber 测试指南](https://docs.gofiber.io/guide/testing)
- [Go 基准测试最佳实践](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [竞态检测器使用指南](https://golang.org/doc/articles/race_detector.html)

---

如有测试相关问题，请查看 [故障排除](#故障排除) 部分或创建 issue。
