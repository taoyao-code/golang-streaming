# 示例代码

本目录包含了流媒体服务器的各种集成示例和客户端代码。

## 📁 目录结构

```
examples/
├── README.md                    # 本文件
├── clients/                     # 客户端SDK示例
│   ├── javascript_client.js     # JavaScript/Node.js 客户端
│   └── python_client.py         # Python 客户端
└── integrations/                # 平台集成示例
    └── integration_examples.md  # 全平台集成指南
```

## 🚀 快速开始

### JavaScript 客户端

```javascript
// 浏览器环境
const client = new VideoStreamingClient('http://localhost:9000');
client.getVideos().then(videos => console.log(videos));

// Node.js环境
const VideoStreamingClient = require('./clients/javascript_client.js');
const client = new VideoStreamingClient('http://localhost:9000');
```

### Python 客户端

```python
from clients.python_client import VideoStreamingClient

client = VideoStreamingClient('http://localhost:9000')
videos = client.get_videos()
print(f"总视频数: {videos['count']}")
```

## 📚 更多示例

- **完整集成指南**: [integrations/integration_examples.md](./integrations/integration_examples.md)
- **API 文档**: [../docs/API_INTEGRATION_GUIDE.md](../docs/API_INTEGRATION_GUIDE.md)
- **快速开始**: [../docs/QUICK_START.md](../docs/QUICK_START.md)

## 🛠️ 自定义客户端

如需为其他语言创建客户端，请参考现有的示例代码结构。所有客户端都应该实现以下核心功能：

- [ ] 获取视频列表
- [ ] 搜索视频
- [ ] 获取流媒体URL
- [ ] 健康检查
- [ ] 视频上传（可选）
