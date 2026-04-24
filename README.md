# CloudPan189 Pro

一个基于天翼云盘分享链接的文件挂载、WebDAV、媒体映射与自动化管理系统。

当前主仓库：
- GitHub: `https://github.com/ray5378/cloudpan189pro`
- Docker: `ray5378/cloudpan189pro`

---

## 新增/变更（2026-04）

- 内存与日志改进
  - 默认关闭第三方 Trace（HTTP_TRACE_LOG=false）
  - 响应体日志截断 64KB（LOG_RESP_BUFFER_SIZE 可调）
  - 支持 GODEBUG=madvdontneed=1，空闲堆更快归还 OS
  - 文件日志滚动：保留 15 天，到期直接删除（lumberjack）
- 任务/登录日志清理
  - 每日自动清理：
    - 任务日志：TASKLOG_RETENTION_DAYS（默认15）
    - 登录日志：LOGINLOG_RETENTION_DAYS（默认15）
  - 手动清理 API：
    - POST /api/task_state/file_log/cleanup
    - POST /api/login_log/cleanup
  - 前端：仪表盘 → 日志 → 任务/登录，刷新下方新增“清除日志”按钮

---

## 快速开始

### 本地源码编译（默认 Docker Go 1.25）

项目 `Makefile` 默认优先使用本地 Docker 中的 `golang:1.25-alpine` 来执行 Go 构建、运行和测试，避免宿主机 Go 版本不匹配导致的报错。

常用命令：

```bash
make go-env
make build-backend
make test
make dev
```

如需临时改回宿主机 Go：

```bash
USE_DOCKER_GO=0 make build-backend
```

如需更换镜像：

```bash
GO_IMAGE=golang:1.25-alpine make build-backend
```


### Docker 运行

```bash
docker run -d \
  --name cloudpan189pro \
  -p 12395:12395 \
  -e GOMEMLIMIT=200MiB \
  -e GOGC=100 \
  -e GIN_MODE=release \
  -e HTTP_TRACE_LOG=false \
  -e LOG_RESP_BUFFER_SIZE=65536 \
  -e PPROF_DISABLE=1 \
  -e TASKLOG_RETENTION_DAYS=15 \
  -e LOGINLOG_RETENTION_DAYS=15 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/media_dir:/app/media_dir \
  --restart unless-stopped \
  ray5378/cloudpan189pro:latest
```

可选（观察 RSS 更干净）：
- 增加 `-e GODEBUG=madvdontneed=1`

### docker-compose

```yaml
services:
  189cloudshare:
    image: ray5378/cloudpan189pro:latest
    container_name: 189cloudshare
    environment:
      - GOMEMLIMIT=200MiB
      - GOGC=100
      - GIN_MODE=release
      - HTTP_TRACE_LOG=false
      - LOG_RESP_BUFFER_SIZE=65536
      - PPROF_DISABLE=1
      - TASKLOG_RETENTION_DAYS=15
      - LOGINLOG_RETENTION_DAYS=15
      # 可选：更积极归还空闲内存给 OS
      # - GODEBUG=madvdontneed=1
      # 可选：周期性内存修剪（默认关闭，按需开启）
      # - MEM_TRIM_ENABLE=true
      # - MEM_TRIM_INTERVAL_MIN=10
      # - MEM_TRIM_THRESHOLD_MB=128
    ports:
      - "12395:12395"
    volumes:
      - ./data:/app/data
      - ./media_dir:/app/media_dir
      - ./local_cas:/local_cas
      - ./cas_strm:/cas_strm
    restart: always
```

### 目录挂载说明
- `./data` → `/app/data`：程序数据库与运行数据
- `./media_dir` → `/app/media_dir`：媒体映射目录
- `./local_cas` → `/local_cas`：本地 `.cas` 文件目录
- `./cas_strm` → `/cas_strm`：CAS 自动生成的 `.strm` 目录

### 访问地址
- Web：`http://<ip>:12395`
- WebDAV：`http://<ip>:12395/dav`

---

## 手动清理日志（API）

- 任务日志：
  - `POST /api/task_state/file_log/cleanup`
  - 响应：`{ deleted, retentionDays }`
- 登录日志：
  - `POST /api/login_log/cleanup`
  - 响应：`{ deleted, retentionDays }`

需要管理员鉴权（Bearer Token）。

---

## 运行参数说明

- GIN_MODE=release：发布模式，减少日志与开销
- HTTP_TRACE_LOG=false：关闭第三方请求 Trace，显著减小分配与日志量
- LOG_RESP_BUFFER_SIZE：响应体日志截断字节数（默认 65536）
- PPROF_DISABLE=1：关闭 pprof（仅排障时临时开启）
- TASKLOG_RETENTION_DAYS / LOGINLOG_RETENTION_DAYS：日志保留天数
- GODEBUG=madvdontneed=1：更积极归还空闲堆内存给 OS（可选）
- MEM_TRIM_ENABLE / MEM_TRIM_INTERVAL_MIN / MEM_TRIM_THRESHOLD_MB：周期性内存修剪（可选，默认关闭）

---

## 构建镜像

```bash
docker build --no-cache -t ray5378/cloudpan189pro:latest .
```

---

## 需求与技术设计文档

本轮新增 CAS 按需恢复播放方案文档：

- 需求文档：`docs/CAS_STRM_REQUIREMENTS.md`
- 技术设计：`docs/CAS_STRM_TECH_DESIGN.md`

---

## 其它保持不变的能力（摘）
- 外部接口创建存储：`POST /api/external/create-storage`
- 自动刷新/持久检测/自动删除失效存储
- WebDAV 映射 / 媒体映射 STRM

如需更详细的使用说明，请参考前文旧版 README 内容与界面指引。
