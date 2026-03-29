# CloudPan189 Pro

一个基于天翼云盘分享链接的文件挂载、WebDAV、媒体映射与自动化管理系统。

当前主仓库：
- GitHub: `https://github.com/ray154856235/cloudpan189pro`
- Docker: `ray5378/cloudpan189pro`

---

## 项目现状

这份仓库当前已经包含以下增强能力：

- 外部接口创建存储：`POST /api/external/create-storage`
- 外部创建默认自动刷新策略（系统设置可配置）
- 自动刷新调度修复：按每个节点自己的开始时间 + 间隔计算，不再按全局整点误刷
- 批量刷新 / 批量改 token / 批量开关自动刷新 的并发统计修复
- 存储节点“持久检测存储”：每月指定日期和时间，对未启用自动刷新或已过期节点执行一次普通刷新
- 自动删除失效存储：支持关键词规则 + 成功刷新后 0 文件规则
- 自动删除前记录命中原因日志，便于审计和回溯

---

## 版本与发布

### 当前推荐版本

- Git tag: `r0.06`
- Docker tag: `ray5378/cloudpan189pro:r0.06`
- Docker latest: `ray5378/cloudpan189pro:latest`

已发布 digest：

- `ray5378/cloudpan189pro:r0.06`
  - `sha256:78e97cb693c89a696a63c754f8ce0834108b616148024a831ec7ef6d8388247a`
- `ray5378/cloudpan189pro:latest`
  - `sha256:c339b4c7108955267456e2f0817bd3b6676a43669f0461bc1c3690a00e7a890e`

### 历史标签

- `re0.01`
- `re0.03`
- `r0.06`

---

## 核心能力

### 1. 存储挂载
- 将天翼云盘分享链接挂载为站内目录
- 支持目录化浏览、文件统计、Web 管理
- 支持普通刷新 / 深度刷新

### 2. WebDAV
- 暴露标准 WebDAV 接口
- 可直接挂载到桌面系统、文件管理器、媒体工具

### 3. 媒体映射
- 可将云盘文件映射为 `strm`
- 可对接 Emby / Jellyfin / Plex

### 4. 外部自动化接入
- 提供 `POST /api/external/create-storage`
- 可由你自己的脚本、Webhook、外部任务系统直接创建存储节点

### 5. 自动运维能力
- 自动刷新
- 持久检测存储
- 自动删除失效存储

---

## 快速开始

## Docker 部署（推荐）

```bash
docker run -d \
  --name cloudpan189pro \
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
  ray5378/cloudpan189pro:latest
```

如需固定镜像内容，也可以直接使用 digest：

```bash
docker run -d \
  --name cloudpan189pro \
  -p 12395:12395 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/media_dir:/app/media_dir \
  --restart unless-stopped \
  ray5378/cloudpan189pro@sha256:c339b4c7108955267456e2f0817bd3b6676a43669f0461bc1c3690a00e7a890e
```

## docker-compose 示例

```yaml
services:
  189cloudshare:
    image: ray5378/cloudpan189pro:latest
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
      - ./data:/app/data
      - ./media_dir:/app/media_dir
    restart: always
```

### 访问地址

- Web 界面：`http://服务器IP:12395`
- WebDAV：`http://服务器IP:12395/dav`

---

## 初始化步骤

1. 首次打开进入初始化页面
2. 登录后台
3. 在 **令牌管理** 中添加天翼云盘令牌
4. 在 **系统设置** 中按需配置：
   - External API Key
   - 默认云盘令牌（tianyi）
   - 外部创建默认自动刷新
   - 持久检测存储
   - 自动删除失效存储
5. 在 **存储管理** 中添加或查看挂载点

---

## WebDAV 使用

### 地址格式

```text
http(s)://你的域名或IP:端口/dav
```

示例：

```text
http://localhost:12395/dav
```

### 支持客户端

- Windows：网络驱动器、RaiDrive、WinSCP
- macOS：Finder、Cyberduck
- Linux：davfs2、Nautilus、Dolphin
- 移动端：ES 文件浏览器、Solid Explorer、FE 文件管理器

---

## External API

## 创建存储

默认异步返回 `202 Accepted`。

- 路径：`POST /api/external/create-storage`
- 鉴权：External API Key
  - Header: `X-API-Key`
  - 或 Body: `apiKey` / `api-key`

### 请求参数

支持这些字段：

- `delayTime?`：延迟秒数，默认 0
- `shareLink` / `分享链接`
- `targetDir` / `目标文件夹`
- `tokenId?`

### shareLink 兼容格式

当前实现兼容以下输入：

- 标准链接
  - `https://cloud.189.cn/t/aiUVZz3QZjAn`
- 纯分享码
  - `aiUVZz3QZjAn`
- 混合文本
  - `网盘（tianyi）:https://cloud.189.cn/t/aiUVZz3QZjAn`
- 携带访问码文本
  - `aiUVZz3QZjAn（访问码：1234）`
  - `code:1234`

### 返回

- `202 Accepted`

示例：

```json
{
  "taskId": 123
}
```

### curl 示例

#### Header 传 API Key

```bash
curl -X POST 'http://<ip>:<port>/api/external/create-storage' \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: your_api_key_here' \
  -d '{
    "delayTime": 0,
    "shareLink": "https://cloud.189.cn/t/aiUVZz3QZjAn",
    "targetDir": "/手动2026/隐身的名字 (2026)"
  }'
```

#### Body 传 API Key（兼容中文键名）

```bash
curl -X POST 'http://<ip>:<port>/api/external/create-storage' \
  -H 'Content-Type: application/json' \
  -d '{
    "delayTime": 0,
    "api-key": "your_api_key_here",
    "分享链接": "网盘（tianyi）:https://cloud.189.cn/t/aiUVZz3QZjAn",
    "目标文件夹": "/手动2026/隐身的名字 (2026)"
  }'
```

### 任务查看

当前可以在页面中查看：
- **任务状态 → 文件任务日志**
- external create-storage 任务会记录到任务日志中

---

## 系统设置说明

## 1. External API Key
用于 `/api/external/create-storage` 的鉴权。

前端已支持：
- 输入
- 显示/隐藏
- 自动生成
- 保存

## 2. 默认云盘令牌（tianyi）
用于 external create-storage 未显式传 `tokenId` 时的默认令牌。

## 3. 外部创建默认自动刷新
用于 external 接口新建挂载时的默认自动刷新配置：
- 是否开启
- 刷新间隔（分钟）
- 生效天数

## 4. 持久检测存储
用于每月指定日期、指定时间，对以下节点执行一次**普通刷新**：

- 未启用自动刷新
- 自动刷新日期已过期

用途：
- 检查文件是否仍然存在
- 对长期不活跃节点做低频巡检

## 5. 自动删除失效存储
用于每天中午 `12:00` 自动删除符合规则的节点。

支持两类规则：

### 规则 A：失败关键词命中
根据存储节点**最新失败日志**里的内容进行匹配。

匹配字段：
- `errorMsg`
- `result`
- `desc`
- `title`

关键词输入框支持：

```text
资源不存在|文件不存在|目录不存在|分享已失效|分享不存在
```

使用 `|` 分隔多个字眼。

### 规则 B：成功刷新后仍然 0 文件
只有同时满足以下条件才会删除：

- 未启用自动刷新，或自动刷新已过期
- 文件数量 = 0
- 最新刷新状态 = 成功

这样可以避免因为网络波动、临时失败导致误删。

### 删除前日志
自动删除前会先记录一条任务日志，写明命中原因，例如：

- `命中自动删除关键词: 资源不存在`
- `未启用自动刷新或已过期，且最新刷新成功后文件数量仍为0`

---

## 自动刷新说明

当前自动刷新逻辑已经修复为：

- 以每个挂载点自己的 `AutoRefreshBeginAt` 为锚点
- 按 `RefreshInterval` 计算 slot
- 同一 slot 在单进程内只触发一次

这比“按当天零点全局整除”的老逻辑更稳定，不会误刷同 interval 的其他节点。

---

## 存储节点状态说明

当前后端已经支持：

- 最新任务状态筛选：`taskLogStatus`
- 失败细分筛选：`failureKind=permanent|transient`

语义如下：

### permanent
- 最新任务失败
- 且不在自动刷新有效期内

### transient
- 最新任务失败
- 但仍在自动刷新有效期内

---

## 开发部署

### 环境要求

- Go 1.25+
- Node.js 22+
- pnpm 10+
- SQLite / MySQL（按配置）

### 克隆项目

```bash
git clone https://github.com/ray154856235/cloudpan189pro.git
cd cloudpan189pro
```

### 后端开发

```bash
go mod tidy
go run cmd/main.go
```

### 前端开发

```bash
cd fe
pnpm install
pnpm dev
```

### 前端构建

```bash
cd fe
pnpm build
```

### 配置文件

编辑 `etc/config.yaml`：

```yaml
port: 12395
dbFile: "data/share.db"
logFile: "logs/share.log"
mediaDir: "media_dir"
```

---

## 项目结构

```text
cloudpan189pro/
├── cmd/
├── etc/
├── fe/
├── internal/
├── docs/
├── Dockerfile
├── Makefile
└── README.md
```

---

## 常见问题

### 1. WebDAV 连接失败
- 检查端口映射和防火墙
- 检查 `/dav` 路径是否正确
- Windows 某些客户端需要启用基础认证

### 2. 令牌失效
- 到“令牌管理”里重新登录或更新 token
- 建议保留备用 token

### 3. 外部接口创建失败
- 检查 External API Key
- 检查默认 token 是否配置
- 检查 `targetDir` 是否以 `/` 开头
- 检查 shareLink 是否为天翼云盘分享链接或可解析文本

### 4. 自动删除误删担忧
当前规则已经尽量收紧：
- 关键词规则可控
- 0 文件删除要求“最新刷新成功”
- 删除前写明命中原因日志

---

## 支持与反馈

如有问题，可在新仓库反馈：
- `https://github.com/ray154856235/cloudpan189pro`

如果这个项目对你有帮助，欢迎 Star。