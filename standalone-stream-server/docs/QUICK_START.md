# 流媒体服务器快速对接指南

## 🚀 5分钟快速开始

### 部署方式选择

#### 方式一：Docker 部署（推荐）

**标准 x86_64 部署：**

```bash
cd standalone-stream-server
docker-compose up -d
```

**ARM64 架构部署：**

```bash
cd standalone-stream-server
# 构建 ARM64 镜像
./build-arm64.sh
# 启动服务
docker-compose up -d
```

#### 方式二：直接运行

### 1. 启动服务器

```bash
cd standalone-stream-server
./streaming-server --config configs/config.yaml
```

**控制台输出示例**:

```
🚀 Starting Standalone Video Streaming Server v2.0.0
📡 Server listening on 0.0.0.0:9000
🎬 Video directories:
   - movies: ./videos/movies (✅ enabled)
   - series: ./videos/series (✅ enabled)
✨ Ready to serve video streams!
```

### 2. 验证服务器

```bash
curl http://localhost:9000/health
```

**期望响应**:

```json
{
  "status": "healthy",
  "server": {
    "version": "2.0.0",
    "uptime": "30s"
  }
}
```

### 3. 获取视频列表

```bash
curl http://localhost:9000/api/videos
```

### 4. 播放视频

在浏览器中访问:

```
http://localhost:9000/player
```

## 📋 核心API速查

| API | 方法 | 描述 | 示例 |
|-----|------|------|------|
| `/health` | GET | 健康检查 | `curl http://localhost:9000/health` |
| `/api/videos` | GET | 获取所有视频 | `curl http://localhost:9000/api/videos` |
| `/api/videos/{dir}` | GET | 按目录获取视频 | `curl http://localhost:9000/api/videos/movies` |
| `/api/search?q={term}` | GET | 搜索视频 | `curl "http://localhost:9000/api/search?q=test"` |
| `/stream/{video-id}` | GET | 播放视频 | `curl http://localhost:9000/stream/movies:test` |
| `/upload/{dir}/{id}` | POST | 上传视频 | `curl -F "file=@video.mp4" http://localhost:9000/upload/movies/new` |

## 🔧 配置要点

**内网环境已优化**:

- ❌ 无需认证
- ❌ 无速率限制
- ✅ 允许所有跨域请求
- ✅ 支持所有HTTP方法

## 💻 快速集成代码

### JavaScript

```javascript
const API_BASE = 'http://your-server:9000';

// 获取视频列表
fetch(`${API_BASE}/api/videos`)
  .then(r => r.json())
  .then(data => console.log(data.videos));

// 播放视频
const video = document.createElement('video');
video.src = `${API_BASE}/stream/movies:test`;
video.controls = true;
document.body.appendChild(video);
```

### Python

```python
import requests

# 获取视频列表
response = requests.get('http://your-server:9000/api/videos')
videos = response.json()['videos']

# 获取流媒体URL
stream_url = f'http://your-server:9000/stream/{videos[0]["id"]}'
```

### curl测试

```bash
# 完整测试流程
curl http://localhost:9000/health                    # 健康检查
curl http://localhost:9000/api/videos                # 获取视频
curl http://localhost:9000/stream/movies:test        # 播放视频
curl -F "file=@test.mp4" http://localhost:9000/upload/movies/new  # 上传视频
```

## 🎯 视频ID格式

**格式**: `目录:文件名`（不含扩展名）

**示例**:

- `movies:avatar` → `./videos/movies/avatar.mp4`
- `series:s01e01` → `./videos/series/s01e01.mp4`

## ⚡ 性能特性

- **并发连接**: 最大300个
- **支持格式**: MP4, AVI, MOV, MKV, WebM, FLV, M4V, 3GP
- **范围请求**: 支持视频快进/后退
- **分块传输**: 3MB块大小
- **上传限制**: 最大1GB

## 🔍 故障排除

### 常见问题

1. **端口被占用**: 修改配置文件中的端口号
2. **视频不存在**: 检查视频文件路径和ID格式
3. **权限问题**: 确保视频目录可读
4. **跨域问题**: 已配置允许所有源

### 日志查看

```bash
# 实时查看日志
./streaming-server --config configs/config.yaml 2>&1 | tee server.log
```

## 📞 技术支持

如遇到问题，请提供：

1. 服务器启动日志
2. API请求和响应
3. 错误信息截图
4. 网络环境信息

---

**下一步**: 查看完整的 [API集成指南](./API_INTEGRATION_GUIDE.md) 了解更多高级功能。

**其他资源**:

- [客户端示例代码](../examples/clients/) - 现成的JavaScript和Python客户端
- [全平台集成指南](../examples/integrations/integration_examples.md) - 移动端、Web、桌面应用集成
