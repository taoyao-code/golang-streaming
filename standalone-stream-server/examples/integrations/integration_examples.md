# 第三方平台集成示例

## 📱 移动应用集成

### Android (Java/Kotlin)

```java
public class VideoStreamingClient {
    private String baseUrl;
    
    public VideoStreamingClient(String baseUrl) {
        this.baseUrl = baseUrl;
    }
    
    // 获取视频列表
    public void getVideos(Callback<VideoListResponse> callback) {
        Call<VideoListResponse> call = service.getVideos();
        call.enqueue(callback);
    }
    
    // 获取流媒体URL
    public String getStreamUrl(String videoId) {
        return baseUrl + "/stream/" + videoId;
    }
}

// 使用ExoPlayer播放
MediaItem mediaItem = MediaItem.fromUri(client.getStreamUrl("movies:avatar"));
player.setMediaItem(mediaItem);
player.prepare();
player.play();
```

### iOS (Swift)

```swift
class VideoStreamingAPI {
    private let baseURL: String
    
    init(baseURL: String) {
        self.baseURL = baseURL
    }
    
    func getVideos(completion: @escaping (Result<[Video], Error>) -> Void) {
        guard let url = URL(string: "\(baseURL)/api/videos") else { return }
        
        URLSession.shared.dataTask(with: url) { data, response, error in
            // 处理响应
        }.resume()
    }
    
    func streamURL(for videoId: String) -> URL? {
        return URL(string: "\(baseURL)/stream/\(videoId)")
    }
}

// 使用AVPlayer播放
let url = api.streamURL(for: "movies:avatar")!
let player = AVPlayer(url: url)
let playerViewController = AVPlayerViewController()
playerViewController.player = player
```

## 🌐 Web前端集成

### React组件

```jsx
import React, { useState, useEffect } from 'react';

const VideoPlayer = ({ apiBase }) => {
    const [videos, setVideos] = useState([]);
    const [currentVideo, setCurrentVideo] = useState(null);
    
    useEffect(() => {
        // 获取视频列表
        fetch(`${apiBase}/api/videos`)
            .then(res => res.json())
            .then(data => setVideos(data.videos));
    }, [apiBase]);
    
    return (
        <div>
            <video 
                controls 
                width="800" 
                height="450"
                src={currentVideo ? `${apiBase}/stream/${currentVideo}` : ''}
            />
            <div className="video-list">
                {videos.map(video => (
                    <div 
                        key={video.id}
                        onClick={() => setCurrentVideo(video.id)}
                        className="video-item"
                    >
                        {video.name}
                    </div>
                ))}
            </div>
        </div>
    );
};

export default VideoPlayer;
```

### Vue.js组件

```vue
<template>
  <div class="video-player">
    <video 
      ref="player"
      controls 
      width="800" 
      height="450"
      :src="currentVideoUrl"
    />
    <div class="video-list">
      <div 
        v-for="video in videos" 
        :key="video.id"
        @click="playVideo(video.id)"
        class="video-item"
      >
        {{ video.name }}
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      videos: [],
      currentVideoId: null,
      apiBase: 'http://your-server:9000'
    };
  },
  computed: {
    currentVideoUrl() {
      return this.currentVideoId ? 
        `${this.apiBase}/stream/${this.currentVideoId}` : '';
    }
  },
  async mounted() {
    const response = await fetch(`${this.apiBase}/api/videos`);
    const data = await response.json();
    this.videos = data.videos;
  },
  methods: {
    playVideo(videoId) {
      this.currentVideoId = videoId;
      this.$nextTick(() => {
        this.$refs.player.load();
        this.$refs.player.play();
      });
    }
  }
};
</script>
```

## 🖥️ 桌面应用集成

### Electron应用

```javascript
// main.js
const { app, BrowserWindow } = require('electron');
const path = require('path');

function createWindow() {
    const mainWindow = new BrowserWindow({
        width: 1200,
        height: 800,
        webPreferences: {
            nodeIntegration: true,
            contextIsolation: false
        }
    });
    
    mainWindow.loadFile('index.html');
}

// renderer.js
class VideoStreamingApp {
    constructor() {
        this.apiBase = 'http://192.168.1.100:9000';
        this.init();
    }
    
    async init() {
        await this.loadVideos();
        this.setupEventListeners();
    }
    
    async loadVideos() {
        try {
            const response = await fetch(`${this.apiBase}/api/videos`);
            const data = await response.json();
            this.renderVideoList(data.videos);
        } catch (error) {
            console.error('Failed to load videos:', error);
        }
    }
    
    renderVideoList(videos) {
        const container = document.getElementById('video-list');
        container.innerHTML = videos.map(video => `
            <div class="video-item" data-id="${video.id}">
                <h3>${video.name}</h3>
                <p>Directory: ${video.directory}</p>
                <p>Size: ${(video.size / 1024 / 1024).toFixed(2)} MB</p>
            </div>
        `).join('');
    }
}

new VideoStreamingApp();
```

## 🔧 后端服务集成

### Java Spring Boot

```java
@RestController
@RequestMapping("/api/proxy")
public class VideoProxyController {
    
    private final String streamingServerUrl = "http://192.168.1.100:9000";
    private final RestTemplate restTemplate = new RestTemplate();
    
    @GetMapping("/videos")
    public ResponseEntity<String> getVideos() {
        String url = streamingServerUrl + "/api/videos";
        return restTemplate.getForEntity(url, String.class);
    }
    
    @GetMapping("/stream/{videoId}")
    public ResponseEntity<Resource> streamVideo(@PathVariable String videoId, 
                                              HttpServletRequest request) {
        String url = streamingServerUrl + "/stream/" + videoId;
        
        // 转发Range请求头
        HttpHeaders headers = new HttpHeaders();
        String rangeHeader = request.getHeader("Range");
        if (rangeHeader != null) {
            headers.set("Range", rangeHeader);
        }
        
        HttpEntity<String> entity = new HttpEntity<>(headers);
        return restTemplate.exchange(url, HttpMethod.GET, entity, Resource.class);
    }
}
```

### Python Flask代理

```python
from flask import Flask, request, Response
import requests

app = Flask(__name__)
STREAMING_SERVER = 'http://192.168.1.100:9000'

@app.route('/api/videos')
def get_videos():
    response = requests.get(f'{STREAMING_SERVER}/api/videos')
    return response.json()

@app.route('/api/stream/<video_id>')
def stream_video(video_id):
    # 转发请求到流媒体服务器
    headers = {}
    if 'Range' in request.headers:
        headers['Range'] = request.headers['Range']
    
    response = requests.get(
        f'{STREAMING_SERVER}/stream/{video_id}',
        headers=headers,
        stream=True
    )
    
    def generate():
        for chunk in response.iter_content(chunk_size=8192):
            yield chunk
    
    return Response(
        generate(),
        status=response.status_code,
        headers={
            'Content-Type': response.headers.get('Content-Type'),
            'Content-Length': response.headers.get('Content-Length'),
            'Accept-Ranges': response.headers.get('Accept-Ranges'),
            'Content-Range': response.headers.get('Content-Range')
        }
    )

if __name__ == '__main__':
    app.run(debug=True)
```

## 📊 数据管理集成

### 视频管理系统

```python
import requests
import os
import json
from datetime import datetime

class VideoManager:
    def __init__(self, api_base):
        self.api_base = api_base
    
    def sync_video_library(self, local_path):
        """同步本地视频库到流媒体服务器"""
        videos = self.get_videos()
        server_videos = {v['id']: v for v in videos['videos']}
        
        for root, dirs, files in os.walk(local_path):
            for file in files:
                if file.endswith(('.mp4', '.avi', '.mov', '.mkv')):
                    rel_path = os.path.relpath(os.path.join(root, file), local_path)
                    directory = os.path.dirname(rel_path) or 'default'
                    video_id = f"{directory}:{os.path.splitext(file)[0]}"
                    
                    if video_id not in server_videos:
                        self.upload_video(directory, file, os.path.join(root, file))
    
    def upload_video(self, directory, video_id, file_path):
        """上传视频到服务器"""
        with open(file_path, 'rb') as f:
            files = {'file': f}
            response = requests.post(
                f"{self.api_base}/upload/{directory}/{video_id}",
                files=files
            )
        return response.json()
    
    def get_videos(self):
        """获取服务器视频列表"""
        response = requests.get(f"{self.api_base}/api/videos")
        return response.json()
    
    def generate_playlist(self, directory=None):
        """生成播放列表"""
        videos = self.get_videos()
        
        if directory:
            videos['videos'] = [v for v in videos['videos'] if v['directory'] == directory]
        
        playlist = {
            'name': f'Playlist - {directory or "All"}',
            'created': datetime.now().isoformat(),
            'videos': [
                {
                    'title': v['name'],
                    'url': f"{self.api_base}/stream/{v['id']}",
                    'duration': v.get('duration', ''),
                    'size': v['size']
                }
                for v in videos['videos']
            ]
        }
        
        return playlist

# 使用示例
manager = VideoManager('http://192.168.1.100:9000')
playlist = manager.generate_playlist('movies')
print(json.dumps(playlist, indent=2))
```

## 🎮 游戏引擎集成

### Unity C #

```csharp
using UnityEngine;
using UnityEngine.Video;
using System.Collections;
using UnityEngine.Networking;

public class VideoStreamingManager : MonoBehaviour
{
    [SerializeField] private VideoPlayer videoPlayer;
    [SerializeField] private string apiBaseUrl = "http://192.168.1.100:9000";
    
    void Start()
    {
        StartCoroutine(LoadVideoList());
    }
    
    IEnumerator LoadVideoList()
    {
        string url = $"{apiBaseUrl}/api/videos";
        using (UnityWebRequest request = UnityWebRequest.Get(url))
        {
            yield return request.SendWebRequest();
            
            if (request.result == UnityWebRequest.Result.Success)
            {
                VideoListResponse response = JsonUtility.FromJson<VideoListResponse>(request.downloadHandler.text);
                // 处理视频列表
            }
        }
    }
    
    public void PlayVideo(string videoId)
    {
        string streamUrl = $"{apiBaseUrl}/stream/{videoId}";
        videoPlayer.url = streamUrl;
        videoPlayer.Play();
    }
}

[System.Serializable]
public class VideoListResponse
{
    public VideoInfo[] videos;
    public int count;
}

[System.Serializable]
public class VideoInfo
{
    public string id;
    public string name;
    public string directory;
    public long size;
}
```

## 📺 智能电视应用

### TV Web应用

```javascript
// 针对电视遥控器优化的界面
class TVVideoApp {
    constructor() {
        this.apiBase = 'http://192.168.1.100:9000';
        this.currentIndex = 0;
        this.videos = [];
        this.init();
    }
    
    async init() {
        await this.loadVideos();
        this.setupKeyboardNavigation();
        this.renderGrid();
    }
    
    async loadVideos() {
        const response = await fetch(`${this.apiBase}/api/videos`);
        const data = await response.json();
        this.videos = data.videos;
    }
    
    setupKeyboardNavigation() {
        document.addEventListener('keydown', (e) => {
            switch(e.code) {
                case 'ArrowUp':
                    this.navigate(-4); // 假设4列网格
                    break;
                case 'ArrowDown':
                    this.navigate(4);
                    break;
                case 'ArrowLeft':
                    this.navigate(-1);
                    break;
                case 'ArrowRight':
                    this.navigate(1);
                    break;
                case 'Enter':
                    this.playSelected();
                    break;
            }
        });
    }
    
    navigate(delta) {
        this.currentIndex = Math.max(0, Math.min(this.videos.length - 1, this.currentIndex + delta));
        this.updateSelection();
    }
    
    playSelected() {
        const video = this.videos[this.currentIndex];
        const player = document.getElementById('player');
        player.src = `${this.apiBase}/stream/${video.id}`;
        player.play();
    }
    
    renderGrid() {
        const container = document.getElementById('video-grid');
        container.innerHTML = this.videos.map((video, index) => `
            <div class="video-card ${index === this.currentIndex ? 'selected' : ''}" data-index="${index}">
                <div class="video-title">${video.name}</div>
                <div class="video-info">${video.directory}</div>
            </div>
        `).join('');
    }
    
    updateSelection() {
        document.querySelectorAll('.video-card').forEach((card, index) => {
            card.classList.toggle('selected', index === this.currentIndex);
        });
    }
}

// 页面加载后初始化
document.addEventListener('DOMContentLoaded', () => {
    new TVVideoApp();
});
```

## 🚀 微服务架构集成

### Docker Compose配置

```yaml
version: '3.8'
services:
  video-streaming:
    build: .
    ports:
      - "9000:9000"
    volumes:
      - ./videos:/app/videos
      - ./configs:/app/configs
    environment:
      - CONFIG_PATH=/app/configs/config.yaml
    networks:
      - video-network
  
  web-frontend:
    image: nginx
    ports:
      - "80:80"
    volumes:
      - ./frontend:/usr/share/nginx/html
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - video-streaming
    networks:
      - video-network

networks:
  video-network:
    driver: bridge
```

### Kubernetes部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: video-streaming-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: video-streaming
  template:
    metadata:
      labels:
        app: video-streaming
    spec:
      containers:
      - name: streaming-server
        image: video-streaming:latest
        ports:
        - containerPort: 9000
        volumeMounts:
        - name: video-storage
          mountPath: /app/videos
        env:
        - name: CONFIG_PATH
          value: "/app/configs/config.yaml"
      volumes:
      - name: video-storage
        persistentVolumeClaim:
          claimName: video-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: video-streaming-service
spec:
  selector:
    app: video-streaming
  ports:
  - port: 9000
    targetPort: 9000
  type: LoadBalancer
```

这些示例涵盖了主流平台和技术栈的集成方案，可以根据具体需求进行调整和扩展。
