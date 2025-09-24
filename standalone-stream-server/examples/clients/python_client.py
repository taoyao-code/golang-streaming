"""
流媒体服务器 Python 客户端
适用于Python 3.6+
"""

import requests
from typing import Dict, List, Optional
import json

class VideoStreamingClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
    
    def get_videos(self) -> Dict:
        """获取所有视频列表"""
        response = self.session.get(f"{self.base_url}/api/videos")
        response.raise_for_status()
        return response.json()
    
    def get_videos_by_directory(self, directory: str) -> Dict:
        """按目录获取视频"""
        response = self.session.get(f"{self.base_url}/api/videos/{directory}")
        response.raise_for_status()
        return response.json()
    
    def search_videos(self, query: str) -> Dict:
        """搜索视频"""
        params = {'q': query}
        response = self.session.get(f"{self.base_url}/api/search", params=params)
        response.raise_for_status()
        return response.json()
    
    def get_video_info(self, video_id: str) -> Dict:
        """获取视频详细信息"""
        response = self.session.get(f"{self.base_url}/api/video/{video_id}")
        response.raise_for_status()
        return response.json()
    
    def get_stream_url(self, video_id: str) -> str:
        """获取流媒体URL"""
        return f"{self.base_url}/stream/{video_id}"
    
    def check_health(self) -> Dict:
        """检查服务器健康状态"""
        response = self.session.get(f"{self.base_url}/health")
        response.raise_for_status()
        return response.json()
    
    def upload_video(self, directory: str, video_id: str, file_path: str) -> Dict:
        """上传视频"""
        with open(file_path, 'rb') as f:
            files = {'file': f}
            response = self.session.post(
                f"{self.base_url}/upload/{directory}/{video_id}", 
                files=files
            )
        response.raise_for_status()
        return response.json()
    
    def download_video(self, video_id: str, output_path: str) -> None:
        """下载视频文件"""
        url = self.get_stream_url(video_id)
        response = self.session.get(url, stream=True)
        response.raise_for_status()
        
        with open(output_path, 'wb') as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)

# 使用示例
if __name__ == "__main__":
    client = VideoStreamingClient('http://localhost:9000')
    
    try:
        # 检查服务器状态
        health = client.check_health()
        print(f"服务器状态: {health['status']}")
        
        # 获取视频列表
        videos = client.get_videos()
        print(f"总视频数: {videos['count']}")
        
        # 搜索视频
        if videos['count'] > 0:
            results = client.search_videos('test')
            print(f"搜索结果: {results['count']} 个视频")
            
            # 显示前5个视频的流媒体URL
            for video in videos['videos'][:5]:
                print(f"{video['name']}: {client.get_stream_url(video['id'])}")
                
    except requests.RequestException as e:
        print(f"请求错误: {e}")
    except Exception as e:
        print(f"其他错误: {e}")
