# CloudPan189 Share

一个基于天翼云盘的文件分享和管理系统，提供 WebDAV 接口和 Web 管理界面。

## 🚧 提示
自动刷新有点问题，正在重构，有问题先提issue

## 🚀 项目简介

## 🧾 版本记录

- v3.0.9
  - 新增对外接口：POST /api/external/create-storage（外部创建挂载），默认异步 202，返回 { taskId }
  - 每次直读系统设置 Addition，不依赖内存；鉴权使用 External API Key（Header: X-API-Key 或 Body: apiKey/api-key）
  - 仅支持天翼 cloud.189.cn 分享链接；解析兼容“存储管理 → 文件分享”的混合文本（含访问码等）
  - 系统设置页新增：External API Key 输入/生成/保存（掩码回显）、默认云盘令牌（tianyi）下拉、外部创建默认自动刷新三项
  - Dockerfile 前端构建阶段修正（避免 corepack prepare 直连 registry.npmjs.org）
  - 镜像：ray5378/cloudpan189-share:v3.0.9（digest: sha256:ab2dc4daccbc8b985b03ace088bc8050e4d84ab6a871a0105a6702063ea1ad31）

- v2.0.2
  - 存储管理：后端支持按文件数量(fileCount)排序 + 分页（低内存，零表结构改动）
  - 前端：去除前端全量拉取排序，改为 sortBy/sortOrder 服务端排序
  - Docker：修复静态资源拷贝（fe/dist→/app/public），解决 404
  - Tag：ray5378/cloudpan189-share:v2.0.2


CloudPan189 Share 是一款专为天翼云盘设计的智能文件分享管理工具。该系统能够将天翼云盘的分享链接转换为标准的目录结构，并通过 WebDAV 协议提供统一的文件访问接口。

## ✨ 核心功能

**🔄 智能链接转换**
- 解析天翼云盘分享链接，转换为标准目录树结构
- 支持多层级文件夹映射，保持原有组织架构

**🌐 WebDAV 统一接口**
- 提供标准 WebDAV 协议支持，兼容主流客户端
- 统一文件访问入口，简化多链接管理流程
- 完整的文件锁定机制，确保并发安全

**💻 全功能网页端**
- 现代化文件浏览器界面，支持文件夹导航
- 支持在线下载、搜索功能
- 内置媒体播放器，支持视频、音频在线预览

**⚡ 高性能流媒体**
- 多线程并发传输，提升大文件访问速度
- 流式播放技术，实现视频无缓冲即时观看
- 智能带宽适配，确保播放流畅度

**📁 媒体目录映射**
- 将云盘文件通过strm形式映射到本地media_dir目录
- 完美兼容Emby、Jellyfin、Plex等媒体服务器
- 可自定义支持的视频格式列表
- 支持一键重建和批量管理

## 🚀 快速开始

### Docker 部署（推荐）

```sh
docker run -d \
  --name cloudpan189-share \
  -p 12395:12395 \
  -e GOMEMLIMIT=250MiB \
  -e GOGC=100 \
  -e HTTP_READ_HEADER_TIMEOUT=30s \
  -e HTTP_READ_TIMEOUT=5h \
  -e HTTP_WRITE_TIMEOUT=5h \
  -e HTTP_IDLE_TIMEOUT=10m \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/media_dir:/app/media_dir \
  --restart unless-stopped \
  ray5378/cloudpan189-share:v3.0.9
```
> 如需固定镜像内容，可使用 digest：`ray5378/cloudpan189-share@sha256:ab2dc4daccbc8b985b03ace088bc8050e4d84ab6a871a0105a6702063ea1ad31`

更多请参考文档：[CloudPan189 Share 快速开始文档](docs/1.quick_start.md)

### docker-compose 示例

```yaml
services:
  189cloudshare:
    image: ray5378/cloudpan189-share:v3.0.9
    container_name: 189cloudshare
    environment:
      - GOMEMLIMIT=250MiB
      - GOGC=100
      - HTTP_READ_HEADER_TIMEOUT=30s
      - HTTP_READ_TIMEOUT=5h
      - HTTP_WRITE_TIMEOUT=5h
      - HTTP_IDLE_TIMEOUT=10m
    ports:
      - "12395:12395"
    volumes:
      - /vol1/1000/SSD/docker/189cloudshare/data:/app/data
      - /vol1/1000/SSD/docker/189cloudshare/media_dir:/app/media_dir
    restart: always
```

### 访问系统
- Web 界面: `http://服务器IP:12395`
- WebDAV 地址: `http://服务器IP:12395/dav`

### 初始化设置
1. 首次打开会进入初始化页面
2. 登录后在"令牌管理"中添加天翼云盘令牌
3. 在"系统设置 → External API" 配置 External API Key / 默认云盘令牌（tianyi）/ 外部创建默认自动刷新
4. 在"存储管理"中配置或查看存储源

## 📁 WebDAV 挂载说明

### 挂载地址格式
```
http(s)://你的网站地址/dav
```

### 示例地址
```
http://localhost:12395/dav
```

### 支持的客户端
- **Windows**: 网络驱动器映射、RaiDrive、WinSCP
- **macOS**: Finder 连接服务器、Cyberduck
- **Linux**: davfs2、文件管理器（Nautilus、Dolphin）
- **移动端**: ES文件浏览器、Solid Explorer、FE文件管理器
- **专业工具**: Cyberduck、FileZilla Pro

### 挂载步骤
1. 打开支持 WebDAV 的客户端
2. 输入服务器地址：`http://你的域名或IP:端口/dav`
3. 输入认证信息（如需要）
4. 连接成功后即可像本地磁盘一样使用

### 注意事项
- 确保服务正常运行且端口可访问
- 部分客户端可能需要启用不安全连接（HTTP）
- 建议在生产环境中使用 HTTPS 协议

## 🌐 外部接口（External API）

默认异步（202 Accepted），用于将“分享链接”挂载到指定目录。

- 路径：`POST /api/external/create-storage`
- 鉴权：系统设置页的 External API Key
  - Header: `X-API-Key: <key>`
  - Body: `apiKey` 或 `api-key`
- 入参（JSON）：
  - `delayTime?` number，默认 0（秒）
  - `shareLink | 分享链接` string（支持 cloud.189.cn 分享链接、纯分享码、含访问码混合文本）
  - `targetDir | 目标文件夹` string（必须以 `/` 开头，拒绝 `..`、非法路径）
  - `tokenId?` number（可选；默认使用“默认云盘令牌（tianyi）”）
- 返回：
  - 202 Accepted：`{ "taskId": number }`
  - 401 Unauthorized：API Key 无效
  - 403 Forbidden：未配置 External API Key
  - 400 Bad Request：参数不合法
- 幂等与兼容：
  - 链接解析兼容“存储管理 → 文件分享”；仅支持天翼 cloud.189.cn 分享
  - 同一路径已有挂载时由内部流程防重（若重复将记录在任务日志）

示例

- 使用 Header 传 Key：
```bash
curl -X POST 'http://<ip>:<port>/api/external/create-storage' \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: your_api_key_here' \
  -d '{
    "delayTime": 0,
    "shareLink": "https://cloud.189.cn/t/aiUVZz3QZjAn",
    "targetDir": "/手动2026/隐身的名字 (2026)"
  }'
# => 202 Accepted { "taskId": 123 }
```

- 使用 Body 传 Key（兼容中文键名）：
```bash
curl -X POST 'http://<ip>:<port>/api/external/create-storage' \
  -H 'Content-Type: application/json' \
  -d '{
    "delayTime": 0,
    "api-key": "your_api_key_here",
    "分享链接": "网盘（tianyi）:https://cloud.189.cn/t/aiUVZz3QZjAn",
    "目标文件夹": "/手动2026/隐身的名字 (2026)"
  }'
# => 202 Accepted { "taskId": 123 }
```

任务查询
- 系统 → 任务状态 → 文件任务日志（类型：external_create_storage）

系统设置页（前端已提供）
- External API Key：输入/显示隐藏/生成（32 位强随机）/保存
- 默认云盘令牌（tianyi）：下拉选择（无分页加载）
- 外部创建默认自动刷新：Enabled / Interval(默认60分钟) / Days(默认60天)

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.25
- **框架**: Gin
- **数据库**: SQLite (GORM)
- **认证**: JWT

### 前端
- **框架**: Vue 3 + TypeScript
- **构建工具**: Vite
- **状态管理**: Pinia

## 📦 开发部署

### 环境要求
- Go 1.25+
- Node.js 22+
- npm 或 yarn 或 pnpm

### 1. 克隆项目
```bash
git clone https://github.com/xxcheng123/cloudpan189-share.git
cd cloudpan189-share
```

### 2. 后端部署
```bash
# 安装依赖
go mod tidy

# 构建项目
make build

# 或直接运行
go run cmd/main.go
```

### 3. 前端部署
```bash
# 进入前端目录
cd fe

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建生产版本
npm run build
```

### 4. 配置文件

编辑 `etc/config.yaml` 配置文件：

```yaml
port: 12395          # 服务端口
dbFile: "data/share.db"   # 数据库文件路径
logFile: "logs/share.log" # 日志文件路径
mediaDir: "media_dir"  # 媒体文件映射目录
```

### 5. 启动服务
```bash
# 启动后端服务
go run cmd/main.go

# 启动前端开发服务器（另一个终端）
cd fe && npm run dev
```

### 6. 访问开发环境
- 前端界面: http://localhost:5173
- 后端 API: http://localhost:12395
- WebDAV 地址: http://localhost:12395/dav

## 🔧 开发指南

### 项目结构
```
cloudpan189-share/
├── cmd/                 # 主程序入口
├── configs/             # 配置管理
├── etc/                 # 配置文件
├── fe/                  # 前端项目
│   ├── src/
│   │   ├── api/         # API 接口
│   │   ├── components/  # 组件
│   │   ├── stores/      # 状态管理
│   │   ├── utils/       # 工具函数
│   │   └── views/       # 页面组件
├── internal/            # 内部模块
│   ├── jobs/           # 后台任务
│   ├── models/         # 数据模型
│   ├── router/         # 路由
│   └── services/       # 业务服务
└── logs/               # 日志文件
```

### API 接口
- `/api/user/*` - 用户管理
- `/api/cloudtoken/*` - 令牌管理
- `/api/storage/*` - 存储管理
- `/api/setting/*` - 系统设置
- `/api/external/create-storage` - 外部创建挂载（默认 202 异步）
- `/dav/*` - WebDAV 接口

## ❓ 常见问题

### WebDAV 连接失败
- 检查防火墙设置，确保端口开放
- 某些客户端需要在地址末尾添加 `/`
- Windows 网络驱动器可能需要启用基本认证

### 文件播放卡顿
- 检查网络带宽和服务器性能
- 尝试降低播放质量
- 确保天翼云盘令牌有效

### 令牌失效问题
- 定期检查令牌状态
- 及时更新过期令牌
- 建议配置多个备用令牌

### Docker 相关问题
- 确保 Docker 服务正常运行
- 检查端口映射是否正确
- 数据卷挂载路径是否有权限

## 🤝 贡献

我们欢迎各种形式的贡献，包括但不限于提交 Bug 报告、功能请求、文档改进和代码贡献。请在提交之前阅读我们的贡献指南（如果可用）。

## 📄 许可证

本项目采用 MIT 许可证。详情请参阅 `LICENSE` 文件。

## 🙏 致谢

感谢所有为本项目做出贡献的开发者和社区成员。

## 💬 支持

如果您在使用过程中遇到任何问题，可以通过以下方式获得支持：

- 提交 GitHub Issue
- 查阅项目文档
- 参与社区讨论

---

⭐ 如果这个项目对您有帮助，请给它一个 Star！
