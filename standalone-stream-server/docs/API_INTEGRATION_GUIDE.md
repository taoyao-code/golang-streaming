# 流媒体服务器 API 对接指南

## 📋 概述

本文档为第三方平台快速对接流媒体服务器提供完整的API集成指南。服务器已配置为内网环境，无需认证，可直接调用所有API。

## 🚀 快速开始

### 服务器信息

- **服务地址**: `http://your-server-ip:9000`
- **协议**: HTTP/1.1
- **认证**: 无需认证（内网环境）
- **CORS**: 已启用，支持跨域请求

### 启动服务器

```bash
cd standalone-stream-server
./streaming-server --config configs/config.yaml
```

## 📡 完整API列表

### 1. 健康检查和系统信息

#### 获取服务器状态

```http
GET /health
```

**响应示例**:

```json
{
  "status": "healthy",
  "server": {
    "version": "2.0.0",
    "uptime": "2h30m15s",
    "memory_usage": "45.2MB"
  },
  "video_service": {
    "directories_enabled": 2,
    "total_videos": 156
  },
  "active_connections": 5,
  "max_connections": 300
}
```

#### 简单ping检查

```http
GET /ping
```

**响应**: `pong`

#### API信息

```http
GET /api/info
```

### 2. 视频管理API

#### 获取所有视频列表

```http
GET /api/videos
```

**响应示例**:

```json
{
  "videos": [
    {
      "id": "movies:avatar",
      "name": "avatar",
      "directory": "movies",
      "path": "/path/to/videos/movies/avatar.mp4",
      "size": 2147483648,
      "duration": "02:42:35",
      "content_type": "video/mp4",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 156,
  "directories": ["movies", "series"]
}
```

#### 按目录获取视频

```http
GET /api/videos/{directory}
```

**示例**:

```http
GET /api/videos/movies
```

#### 获取视频目录信息

```http
GET /api/directories
```

**响应示例**:

```json
{
  "directories": [
    {
      "name": "movies",
      "path": "./videos/movies",
      "description": "Movie collection",
      "enabled": true,
      "video_count": 98
    },
    {
      "name": "series",
      "path": "./videos/series", 
      "description": "TV series collection",
      "enabled": true,
      "video_count": 58
    }
  ],
  "count": 2,
  "enabled_count": 2
}
```

#### 搜索视频

```http
GET /api/search?q={search_term}
```

**示例**:

```http
GET /api/search?q=avatar
```

#### 获取视频详细信息

```http
GET /api/video/{video-id}
```

**示例**:

```http
GET /api/video/movies:avatar
```

### 3. 视频流播放API

#### 播放视频（支持范围请求）

```http
GET /stream/{video-id}
```

**示例**:

```http
GET /stream/movies:avatar
```

**支持的请求头**:

- `Range: bytes=0-1023` - 获取指定字节范围
- `Accept-Ranges: bytes` - 服务器支持范围请求

#### 按目录播放视频（支持多级路径）

```http
GET /stream/{directory}/{video-path}
```

**示例**:

```http
GET /stream/movies/avatar
GET /stream/series/season1/episode1
```

### 4. 视频上传API

#### 上传单个视频

```http
POST /upload/{directory}/{video-id}
Content-Type: multipart/form-data
```

**请求体**: `file` 字段包含视频文件

#### 批量上传视频

```http
POST /upload/{directory}/batch
Content-Type: multipart/form-data
```

**请求体**: 多个 `files` 字段

### 5. 系统监控API

#### 获取系统统计

```http
GET /api/system/stats
```

#### 获取流控统计

```http
GET /api/streaming/stats
```

## 💻 集成示例

### JavaScript/Node.js 集成

```javascript
class VideoStreamingAPI {
    constructor(baseUrl) {
        this.baseUrl = baseUrl; // http://your-server:9000
    }

    // 获取所有视频
    async getVideos() {
        const response = await fetch(`${this.baseUrl}/api/videos`);
        return await response.json();
    }

    // 按目录获取视频
    async getVideosByDirectory(directory) {
        const response = await fetch(`${this.baseUrl}/api/videos/${directory}`);
        return await response.json();
    }

    // 搜索视频
    async searchVideos(query) {
        const response = await fetch(`${this.baseUrl}/api/search?q=${encodeURIComponent(query)}`);
        return await response.json();
    }

    // 获取视频信息
    async getVideoInfo(videoId) {
        const response = await fetch(`${this.baseUrl}/api/video/${videoId}`);
        return await response.json();
    }

    // 获取流媒体URL
    getStreamUrl(videoId) {
        return `${this.baseUrl}/stream/${videoId}`;
    }

    // 上传视频
    async uploadVideo(directory, videoId, file) {
        const formData = new FormData();
        formData.append('file', file);
        
        const response = await fetch(`${this.baseUrl}/upload/${directory}/${videoId}`, {
            method: 'POST',
            body: formData
        });
        return await response.json();
    }
}

// 使用示例
const api = new VideoStreamingAPI('http://192.168.1.100:9000');

// 获取视频列表
api.getVideos().then(data => {
    console.log('总视频数:', data.count);
    data.videos.forEach(video => {
        console.log(`${video.name} (${video.directory})`);
    });
});

// 创建视频播放器
function createVideoPlayer(videoId) {
    const video = document.createElement('video');
    video.src = api.getStreamUrl(videoId);
    video.controls = true;
    video.width = 800;
    return video;
}
```

### Python 集成

```python
import requests
import json
from typing import List, Dict, Optional

class VideoStreamingAPI:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        
    def get_videos(self) -> Dict:
        """获取所有视频列表"""
        response = requests.get(f"{self.base_url}/api/videos")
        response.raise_for_status()
        return response.json()
    
    def get_videos_by_directory(self, directory: str) -> Dict:
        """按目录获取视频"""
        response = requests.get(f"{self.base_url}/api/videos/{directory}")
        response.raise_for_status()
        return response.json()
    
    def search_videos(self, query: str) -> Dict:
        """搜索视频"""
        params = {'q': query}
        response = requests.get(f"{self.base_url}/api/search", params=params)
        response.raise_for_status()
        return response.json()
    
    def get_video_info(self, video_id: str) -> Dict:
        """获取视频详细信息"""
        response = requests.get(f"{self.base_url}/api/video/{video_id}")
        response.raise_for_status()
        return response.json()
    
    def get_stream_url(self, video_id: str) -> str:
        """获取流媒体URL"""
        return f"{self.base_url}/stream/{video_id}"
    
    def upload_video(self, directory: str, video_id: str, file_path: str) -> Dict:
        """上传视频"""
        with open(file_path, 'rb') as f:
            files = {'file': f}
            response = requests.post(
                f"{self.base_url}/upload/{directory}/{video_id}", 
                files=files
            )
        response.raise_for_status()
        return response.json()

# 使用示例
api = VideoStreamingAPI('http://192.168.1.100:9000')

# 获取视频列表
videos = api.get_videos()
print(f"总视频数: {videos['count']}")

# 搜索视频
results = api.search_videos('avatar')
for video in results['videos']:
    print(f"{video['name']} - {api.get_stream_url(video['id'])}")
```

### curl 命令示例

```bash
# 获取所有视频
curl -X GET http://192.168.1.100:9000/api/videos

# 按目录获取视频
curl -X GET http://192.168.1.100:9000/api/videos/movies

# 搜索视频
curl -X GET "http://192.168.1.100:9000/api/search?q=avatar"

# 获取视频信息
curl -X GET http://192.168.1.100:9000/api/video/movies:avatar

# 播放视频（下载文件）
curl -X GET http://192.168.1.100:9000/stream/movies:avatar -o avatar.mp4

# 范围请求（获取文件片段）
curl -H "Range: bytes=0-1023" http://192.168.1.100:9000/stream/movies:avatar

# 上传视频
curl -X POST -F "file=@video.mp4" http://192.168.1.100:9000/upload/movies/new-video

# 健康检查
curl -X GET http://192.168.1.100:9000/health
```

## 🎥 HTML5 视频播放器集成

### 基础播放器

```html
<!DOCTYPE html>
<html>
<head>
    <title>视频播放器</title>
</head>
<body>
    <video id="player" controls width="800" height="450">
        <source id="videoSource" type="video/mp4">
        您的浏览器不支持视频播放。
    </video>

    <script>
        const API_BASE = 'http://192.168.1.100:9000';
        
        // 加载视频
        function loadVideo(videoId) {
            const video = document.getElementById('player');
            const source = document.getElementById('videoSource');
            
            source.src = `${API_BASE}/stream/${videoId}`;
            video.load();
        }
        
        // 示例：加载电影
        loadVideo('movies:avatar');
    </script>
</body>
</html>
```

### 高级播放器（带视频列表）

```html
<!DOCTYPE html>
<html>
<head>
    <title>视频播放平台</title>
    <style>
        .video-list { display: flex; flex-wrap: wrap; gap: 10px; margin: 20px 0; }
        .video-item { 
            border: 1px solid #ddd; 
            padding: 10px; 
            cursor: pointer; 
            border-radius: 5px; 
        }
        .video-item:hover { background-color: #f0f0f0; }
    </style>
</head>
<body>
    <h1>视频播放平台</h1>
    
    <video id="player" controls width="800" height="450">
        <source id="videoSource" type="video/mp4">
        您的浏览器不支持视频播放。
    </video>
    
    <div id="videoList" class="video-list"></div>

    <script>
        const API_BASE = 'http://192.168.1.100:9000';
        
        // 加载视频列表
        async function loadVideoList() {
            try {
                const response = await fetch(`${API_BASE}/api/videos`);
                const data = await response.json();
                
                const listContainer = document.getElementById('videoList');
                listContainer.innerHTML = '';
                
                data.videos.forEach(video => {
                    const item = document.createElement('div');
                    item.className = 'video-item';
                    item.innerHTML = `
                        <h4>${video.name}</h4>
                        <p>目录: ${video.directory}</p>
                        <p>大小: ${(video.size / 1024 / 1024).toFixed(2)} MB</p>
                    `;
                    item.onclick = () => playVideo(video.id);
                    listContainer.appendChild(item);
                });
            } catch (error) {
                console.error('加载视频列表失败:', error);
            }
        }
        
        // 播放视频
        function playVideo(videoId) {
            const video = document.getElementById('player');
            const source = document.getElementById('videoSource');
            
            source.src = `${API_BASE}/stream/${videoId}`;
            video.load();
            video.play();
        }
        
        // 页面加载时获取视频列表
        document.addEventListener('DOMContentLoaded', loadVideoList);
    </script>
</body>
</html>
```

## ⚙️ 配置优化

### 内网环境配置

服务器已针对内网环境优化配置：

- ✅ 禁用认证机制
- ✅ 禁用速率限制  
- ✅ 允许所有CORS源
- ✅ 允许所有HTTP方法
- ✅ 支持范围请求

### 性能调优建议

1. **并发连接数**: 默认300，可根据服务器性能调整
2. **上传大小限制**: 默认1GB，可根据需要调整
3. **分块大小**: 默认3MB，平衡内存使用和传输效率
4. **缓存控制**: 默认1小时，减少重复请求

## 🐛 常见问题

### Q: 视频无法播放？

**A**: 检查视频ID格式，应为 `目录:文件名`（不含扩展名）

### Q: 跨域请求被阻止？

**A**: 服务器已启用CORS，检查客户端是否正确发送请求

### Q: 上传失败？

**A**: 检查文件大小是否超过1GB限制，检查目录是否存在

### Q: 范围请求不生效？

**A**: 确保使用HTTP/1.1协议，正确设置Range请求头

## 📊 API响应格式

### 成功响应

所有成功的API请求返回JSON格式，状态码200：

```json
{
  "data": {},
  "status": "success"
}
```

### 错误响应

错误响应包含错误信息和状态码：

```json
{
  "error": "错误描述",
  "details": "详细错误信息",
  "timestamp": 1640995200
}
```

### 常见状态码

- `200` - 成功
- `206` - 部分内容（范围请求）
- `400` - 请求参数错误
- `404` - 资源不存在
- `429` - 请求过于频繁（已禁用）
- `500` - 服务器内部错误

## 🔄 集成测试

### 快速验证

```bash
# 1. 检查服务器状态
curl http://your-server:9000/health

# 2. 获取视频列表
curl http://your-server:9000/api/videos

# 3. 播放第一个视频
curl -I http://your-server:9000/stream/movies:test
```

### 功能测试清单

- [ ] 健康检查响应正常
- [ ] 视频列表获取成功
- [ ] 视频信息查询正常
- [ ] 视频流播放正常
- [ ] 范围请求支持正常
- [ ] 视频搜索功能正常
- [ ] 视频上传功能正常

---

**支持联系**: 如有技术问题，请检查服务器日志或联系技术支持团队。
