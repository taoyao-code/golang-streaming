#!/bin/bash

# ARM64 构建脚本
# 用于构建可在 ARM64 架构上运行的 Docker 镜像

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
IMAGE_NAME="streaming-server"
IMAGE_TAG="latest"
PLATFORM="linux/arm64"

echo -e "${BLUE}🚀 开始构建 ARM64 流媒体服务器镜像...${NC}"

# 检查 Docker 和 buildx
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ 错误: Docker 未安装${NC}"
    exit 1
fi

# 启用 buildx（如果需要）
if ! docker buildx version &> /dev/null; then
    echo -e "${YELLOW}⚠️  启用 Docker buildx...${NC}"
    docker buildx create --use
fi

echo -e "${BLUE}📦 构建镜像: ${IMAGE_NAME}:${IMAGE_TAG}${NC}"
echo -e "${BLUE}🏗️  目标架构: ${PLATFORM}${NC}"

# 构建 ARM64 镜像
docker buildx build \
    --platform ${PLATFORM} \
    --tag ${IMAGE_NAME}:${IMAGE_TAG} \
    --tag ${IMAGE_NAME}:arm64-${IMAGE_TAG} \
    --load \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ ARM64 镜像构建成功!${NC}"
    echo -e "${GREEN}📋 镜像信息:${NC}"
    docker images | grep ${IMAGE_NAME}
    
    echo -e "\n${BLUE}🎯 使用方法:${NC}"
    echo -e "${YELLOW}启动容器:${NC}"
    echo "docker run -d -p 9000:9000 -v \$(pwd)/videos:/app/videos -v \$(pwd)/configs:/app/configs:ro ${IMAGE_NAME}:${IMAGE_TAG}"
    
    echo -e "\n${YELLOW}使用 Docker Compose:${NC}"
    echo "docker-compose up -d"
    
    echo -e "\n${YELLOW}查看日志:${NC}"
    echo "docker logs -f streaming-server"
    
else
    echo -e "${RED}❌ 构建失败${NC}"
    exit 1
fi
