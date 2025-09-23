# Makefile for Standalone Video Streaming Server

# 项目配置
PROJECT_NAME := standalone-stream-server
BINARY_NAME := streaming-server
GO_VERSION := 1.25

# 目录配置
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage
SCRIPTS_DIR := scripts

# Go 配置
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GOPATH := $(shell go env GOPATH)

# 版本信息
VERSION := 2.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建标志
LDFLAGS := -ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
.PHONY: help
help: ## 显示帮助信息
	@echo "Standalone Video Streaming Server Makefile"
	@echo ""
	@echo "可用目标:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# 开发相关目标
.PHONY: setup
setup: ## 设置开发环境
	@echo "设置开发环境..."
	@go mod download
	@go mod verify
	@mkdir -p $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@echo "开发环境设置完成"

.PHONY: deps
deps: ## 下载并验证依赖
	@echo "下载依赖..."
	@go mod download
	@go mod verify
	@go mod tidy

.PHONY: fmt
fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "代码格式化完成"

.PHONY: vet
vet: ## 运行 go vet
	@echo "运行 go vet..."
	@go vet ./...
	@echo "go vet 检查完成"

.PHONY: lint
lint: vet ## 运行代码检查
	@echo "运行代码检查..."
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint 未安装，跳过 lint 检查"; \
	fi
#	@if command -v staticcheck >/dev/null 2>&1; then \
#		echo "运行 staticcheck..."; \
#		if staticcheck ./... 2>/dev/null; then \
#			echo "staticcheck 检查通过"; \
#		else \
#			echo "警告: staticcheck 检查失败，可能是版本兼容性问题"; \
#			echo "如果您使用的是较新的 Go 版本，这是正常现象"; \
#		fi; \
#	else \
#		echo "staticcheck 未安装，跳过静态检查"; \
#	fi

# 测试相关目标
.PHONY: test
test: ## 运行单元测试
	@echo "运行单元测试..."
	@$(SCRIPTS_DIR)/test.sh

.PHONY: test-unit
test-unit: ## 运行单元测试
	@echo "运行单元测试..."
	@go test -v $(shell go list ./... | grep -v /tests)

.PHONY: test-integration
test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	@$(SCRIPTS_DIR)/test.sh -i

.PHONY: test-coverage
test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	@$(SCRIPTS_DIR)/test.sh -c
	@echo "覆盖率报告已生成: coverage.html"

.PHONY: test-race
test-race: ## 运行竞态检测测试
	@echo "运行竞态检测测试..."
	@$(SCRIPTS_DIR)/test.sh -r

.PHONY: test-benchmark
test-benchmark: ## 运行基准测试
	@echo "运行基准测试..."
	@$(SCRIPTS_DIR)/test.sh -b

.PHONY: test-all
test-all: ## 运行所有测试
	@echo "运行所有测试..."
	@$(SCRIPTS_DIR)/test.sh -a

.PHONY: test-e2e
test-e2e: ## 运行端到端测试（需要服务器运行）
	@echo "运行端到端测试..."
	@$(SCRIPTS_DIR)/e2e_test.sh

.PHONY: test-clean
test-clean: ## 清理测试文件
	@echo "清理测试文件..."
	@$(SCRIPTS_DIR)/test.sh --clean
	@rm -rf $(COVERAGE_DIR)/*
	@echo "测试文件清理完成"

# 构建相关目标
.PHONY: build
build: ## 构建二进制文件
	@echo "构建 $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## 构建 Linux 二进制文件
	@echo "构建 Linux 版本..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server
	@echo "Linux 版本构建完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-windows
build-windows: ## 构建 Windows 二进制文件
	@echo "构建 Windows 版本..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	@echo "Windows 版本构建完成: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

.PHONY: build-darwin
build-darwin: ## 构建 macOS 二进制文件
	@echo "构建 macOS 版本..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	@echo "macOS 版本构建完成"

.PHONY: build-all
build-all: build-linux build-windows build-darwin ## 构建所有平台的二进制文件

.PHONY: install
install: build ## 安装到本地
	@echo "安装 $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "安装完成: $(GOPATH)/bin/$(BINARY_NAME)"

# 运行相关目标
.PHONY: run
run: ## 运行服务器
	@echo "启动服务器..."
	@go run ./cmd/server

.PHONY: run-dev
run-dev: ## 以开发模式运行
	@echo "以开发模式启动服务器..."
	@go run ./cmd/server --config configs/config.yaml

.PHONY: run-build
run-build: build ## 运行构建的二进制文件
	@echo "运行构建的服务器..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Docker 相关目标
.PHONY: docker-build
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	@docker build -t $(PROJECT_NAME):$(VERSION) .
	@docker tag $(PROJECT_NAME):$(VERSION) $(PROJECT_NAME):latest
	@echo "Docker 镜像构建完成"

.PHONY: docker-run
docker-run: ## 运行 Docker 容器
	@echo "运行 Docker 容器..."
	@docker run -p 9000:9000 -v $(PWD)/videos:/app/videos $(PROJECT_NAME):latest

.PHONY: docker-stop
docker-stop: ## 停止 Docker 容器
	@echo "停止 Docker 容器..."
	@docker stop $(shell docker ps -q --filter ancestor=$(PROJECT_NAME):latest) 2>/dev/null || true

# 发布相关目标
.PHONY: dist
dist: build-all ## 创建发布包
	@echo "创建发布包..."
	@rm -rf $(DIST_DIR)/*
	@mkdir -p $(DIST_DIR)
	
	# Linux 包
	@mkdir -p $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64
	@cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64/$(BINARY_NAME)
	@cp -r configs $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64/
	@cp README.md LICENSE $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-linux-amd64/ 2>/dev/null || true
	@cd $(DIST_DIR) && tar -czf $(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz $(PROJECT_NAME)-$(VERSION)-linux-amd64
	
	# Windows 包
	@mkdir -p $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-windows-amd64
	@cp $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-windows-amd64/$(BINARY_NAME).exe
	@cp -r configs $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-windows-amd64/
	@cp README.md LICENSE $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-windows-amd64/ 2>/dev/null || true
	@cd $(DIST_DIR) && zip -r $(PROJECT_NAME)-$(VERSION)-windows-amd64.zip $(PROJECT_NAME)-$(VERSION)-windows-amd64
	
	# macOS 包
	@mkdir -p $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-darwin-amd64
	@cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-darwin-amd64/$(BINARY_NAME)
	@cp -r configs $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-darwin-amd64/
	@cp README.md LICENSE $(DIST_DIR)/$(PROJECT_NAME)-$(VERSION)-darwin-amd64/ 2>/dev/null || true
	@cd $(DIST_DIR) && tar -czf $(PROJECT_NAME)-$(VERSION)-darwin-amd64.tar.gz $(PROJECT_NAME)-$(VERSION)-darwin-amd64
	
	@echo "发布包创建完成:"
	@ls -la $(DIST_DIR)/*.tar.gz $(DIST_DIR)/*.zip

# 清理相关目标
.PHONY: clean
clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)/*
	@rm -rf $(DIST_DIR)/*
	@rm -f coverage.out coverage.html
	@echo "清理完成"

.PHONY: clean-all
clean-all: clean test-clean ## 清理所有生成的文件
	@echo "清理所有生成的文件..."
	@go clean -cache -testcache -modcache
	@echo "全部清理完成"

# 开发工具
.PHONY: watch
watch: ## 监视文件变化并自动重新构建
	@echo "监视文件变化..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	elif command -v reflex >/dev/null 2>&1; then \
		reflex -r '\.go$$' -s -- sh -c 'make build && echo "重新构建完成"'; \
	else \
		echo "请安装 air 或 reflex 来使用文件监视功能"; \
		echo "  go install github.com/cosmtrek/air@latest"; \
		echo "  go install github.com/cespare/reflex@latest"; \
	fi

.PHONY: mod-update
mod-update: ## 更新依赖到最新版本
	@echo "更新依赖..."
	@go get -u ./...
	@go mod tidy
	@echo "依赖更新完成"

.PHONY: security-check
security-check: ## 运行安全检查
	@echo "运行安全检查..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec 未安装，请运行: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# 信息目标
.PHONY: info
info: ## 显示项目信息
	@echo "项目信息:"
	@echo "  名称: $(PROJECT_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  Go 版本: $(GO_VERSION)"
	@echo "  Git 提交: $(GIT_COMMIT)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  操作系统: $(GOOS)"
	@echo "  架构: $(GOARCH)"

.PHONY: deps-check
deps-check: ## 检查依赖状态
	@echo "检查依赖状态..."
	@go mod verify
	@go list -m -u all

# 性能分析
.PHONY: profile-cpu
profile-cpu: ## CPU 性能分析
	@echo "启动 CPU 性能分析..."
	@go test -cpuprofile=cpu.prof -bench=. ./...
	@echo "使用 'go tool pprof cpu.prof' 查看结果"

.PHONY: profile-mem
profile-mem: ## 内存性能分析
	@echo "启动内存性能分析..."
	@go test -memprofile=mem.prof -bench=. ./...
	@echo "使用 'go tool pprof mem.prof' 查看结果"

# 快捷组合目标
.PHONY: check
check: fmt vet lint test ## 运行所有检查

.PHONY: ci
ci: deps check test-coverage ## CI 流水线目标

.PHONY: release
release: clean check test-all build-all dist ## 发布流程

# 确保目录存在
$(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR):
	@mkdir -p $@
