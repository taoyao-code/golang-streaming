/**
 * 流媒体服务器 JavaScript 客户端
 * 适用于Web浏览器和Node.js环境
 */

class VideoStreamingClient {
    constructor(baseUrl) {
        this.baseUrl = baseUrl.replace(/\/$/, ''); // 移除末尾斜杠
    }

    // 获取所有视频
    async getVideos() {
        const response = await fetch(`${this.baseUrl}/api/videos`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }

    // 按目录获取视频
    async getVideosByDirectory(directory) {
        const response = await fetch(`${this.baseUrl}/api/videos/${directory}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }

    // 搜索视频
    async searchVideos(query) {
        const response = await fetch(`${this.baseUrl}/api/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }

    // 获取视频信息
    async getVideoInfo(videoId) {
        const response = await fetch(`${this.baseUrl}/api/video/${videoId}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }

    // 获取流媒体URL
    getStreamUrl(videoId) {
        return `${this.baseUrl}/stream/${videoId}`;
    }

    // 检查服务器健康状态
    async checkHealth() {
        const response = await fetch(`${this.baseUrl}/health`);
        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }

    // 上传视频
    async uploadVideo(directory, videoId, file) {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch(`${this.baseUrl}/upload/${directory}/${videoId}`, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        return response.json();
    }
}

// 使用示例
if (typeof window !== 'undefined') {
    // 浏览器环境示例
    window.VideoStreamingClient = VideoStreamingClient;

    // 示例用法
    const client = new VideoStreamingClient('http://localhost:9000');

    // 创建简单的视频播放器
    window.createVideoPlayer = function (containerId, videoId) {
        const container = document.getElementById(containerId);
        const video = document.createElement('video');
        video.src = client.getStreamUrl(videoId);
        video.controls = true;
        video.style.width = '100%';
        video.style.maxWidth = '800px';
        container.appendChild(video);
        return video;
    };

} else if (typeof module !== 'undefined') {
    // Node.js环境
    module.exports = VideoStreamingClient;
}
