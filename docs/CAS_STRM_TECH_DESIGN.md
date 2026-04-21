# CAS 按需恢复播放技术设计文档

## 1. 设计目标

在 `cloudpan189pro` 中落地一套面向 `.cas` 文件的按需恢复播放体系，实现：

- 订阅阶段仅保存 `.cas`
- 为 `.cas` 生成 `.strm`
- 播放器请求 `.strm` 时触发恢复
- 恢复成功后提供真实媒体播放地址
- 超时后自动删除真实媒体，仅保留 `.cas`

该设计重点不是完整迁移 `cloud189-auto-save` 的任务系统，而是：

**按 `cloudpan189pro` 现有 Go 架构，重建一套 CAS 能力模块，并与订阅、媒体、STRM、存储、播放链路完成接线。**

---

## 2. 现有能力评估

## 2.1 `cloud189-auto-save` 已有可参考能力

可复用“逻辑”，不可直接复用“代码”的部分：

- `.cas` 文件识别
- `.cas` 内容解析（JSON / base64 / 多行）
- 原始文件名推导
- 天翼秒传恢复流程：
  - `initMultiUpload`
  - `checkTransSecond`
  - `commitMultiUploadFile`
- 黑名单 / 403 / 风控情况下的回退思路
- 本地 CAS 元数据缓存思路
- 中转 `.cas` 文件清理思路

## 2.2 `cloudpan189pro` 已有能力基础

从项目结构判断，当前已有这些适合作为接入点的模块：

- `internal/services/cloudbridge`：天翼能力接入候选位置
- `internal/services/storagefacade`：存储操作抽象层候选位置
- `internal/services/virtualfile`：虚拟文件系统候选接入点
- `internal/services/mediafile`：STRM / 媒体生成候选接入点
- `internal/services/autoingestplan`：订阅 / 自动导入相关候选接入点
- `internal/services/cloudtoken`：天翼登录态与 token 支撑

同时，项目当前已经具备一批可直接参照的 cloud189 文件能力：
- 文件浏览
- 文件定位
- 下载链接 / 播放地址获取
- 文件删除
- 存储抽象 / 挂载体系

这意味着：
- `.cas` 方案不需要从零重建完整文件系统操作链
- 回收模块可以复用现有“删除文件”能力，只补生命周期与状态管理
- 播放模块大概率可以复用现有“文件定位 + 下载地址生成”能力，只补“播放请求触发恢复”的前置逻辑
- 真正新增的重点进一步收敛到：`.cas` 解析、秒传恢复、播放触发、回收状态控制

说明：
- `.cas` 协议处理本身易迁移
- 秒传恢复链需要基于 Go 重新实现
- 业务接入应走 `cloudpan189pro` 自己的服务层，而不是直接模拟 JS 任务流

---

## 3. 总体架构

建议拆成 4 个核心模块。

### 3.1 CAS 解析模块

建议位置：
- `internal/services/casparser`

职责：
- 判断文件是否为 `.cas`
- 解析 `.cas` 内容
- 推导原始文件名

建议文件：
- `internal/services/casparser/types.go`
- `internal/services/casparser/parser.go`
- `internal/services/casparser/parser_test.go`

核心接口：

```go
type CasInfo struct {
    Name       string
    Size       int64
    MD5        string
    SliceMD5   string
    CreateTime string
}

func IsCasFile(name string) bool
func GetOriginalFileName(casFileName string, info *CasInfo) string
func ParseCasContent(content []byte) (*CasInfo, error)
```

---

### 3.2 CAS 恢复模块

建议位置：
- `internal/services/casrestore`

职责：
- 下载 `.cas` 文件内容
- 解析 `.cas`
- 调用天翼秒传接口恢复真实媒体
- 支持个人恢复优先、家庭中转回退
- 返回恢复后的目标媒体信息

建议文件：
- `internal/services/casrestore/service.go`
- `internal/services/casrestore/cloud189_restore.go`
- `internal/services/casrestore/state.go`

核心接口建议：

```go
type RestoreRequest struct {
    StorageID      uint
    CasFileID      string
    CasFileName    string
    TargetFolderID string
}

type RestoreResult struct {
    RestoredFileID   string
    RestoredFileName string
    TargetFolderID   string
}

type Service interface {
    EnsureRestored(ctx context.Context, req RestoreRequest) (*RestoreResult, error)
}
```

关键点：
- 同一 `.cas` 恢复需要 inflight 去重
- 支持重试
- 支持恢复状态查询

---

### 3.3 CAS 播放模块

建议位置：
- `internal/services/casplayback`
- `internal/handler/http/cas_playback.go`

职责：
- 作为 `.strm` 指向的播放入口
- 接收播放器请求
- 判断真实媒体是否已存在
- 若不存在则触发恢复
- 恢复后返回真实媒体播放地址或代理流

推荐首版行为：
- 优先使用 **302 跳转到最终真实媒体播放 URL**
- 后续再按需要扩展反代流模式

建议播放接口：

```text
GET /api/cas/play/:recordId
```

可选接口：

```text
GET /api/cas/status/:recordId
POST /api/cas/restore/:recordId
POST /api/cas/recycle/:recordId
```

---

### 3.4 CAS 回收模块

建议位置：
- `internal/services/casrecycle`
- `internal/handler/scheduler`

职责：
- 定时扫描已恢复媒体
- 根据 TTL / 最后访问时间决定是否删除真实媒体
- 删除后更新状态
- 避免删除 `.cas` 和 `.strm`

建议策略：
- 默认按“最后访问时间 + 保留分钟数”计算回收时间
- 正在播放中的文件不删除
- 删除失败记录日志并重试

---

## 4. 数据模型设计

建议新增一张专用表：`cas_media_records`

建议字段：

- `id`
- `storage_id`：所属存储 / 挂载点
- `mount_point_id`：可选，所属挂载点
- `cas_file_id`：`.cas` 文件 ID
- `cas_file_name`
- `cas_file_path`
- `source_parent_id`：`.cas` 所在源目录 ID
- `restored_parent_id`：恢复目标目录 ID
- `original_file_name`
- `original_file_size`
- `file_md5`
- `slice_md5`
- `strm_relative_path`
- `restored_file_id`
- `restored_file_name`
- `restore_status`：`pending / restoring / restored / failed / recycling / recycled`
- `last_access_at`
- `restored_at`
- `recycle_after_at`
- `last_error`
- `created_at`
- `updated_at`

索引建议：
- `(storage_id, cas_file_id)` 唯一索引
- `restore_status` 普通索引
- `recycle_after_at` 普通索引

---

## 5. 核心流程设计

## 5.1 订阅 / 自动导入阶段

目标：只保存 `.cas` 与 `.strm`，不恢复真实媒体。

流程：
1. 订阅流程发现文件
2. 若文件为 `.cas`
3. 将 `.cas` 转存到本地网盘目录
4. 解析或登记 `.cas` 对应原始媒体名
5. 生成 `.strm`
6. `.strm` 内容写入 CAS 播放入口 URL
7. 写入 / 更新 `cas_media_records`

### `.strm` 内容建议

```text
http://<server>/api/cas/play/<recordId>
```

说明：
- 不应直接写真实媒体下载地址
- 不应直接写静态天翼分享地址
- 必须写系统控制入口，才能在播放时动态触发恢复

---

## 5.2 播放触发恢复流程

流程：
1. 播放器请求 `/api/cas/play/:recordId`
2. 查询 `cas_media_records`
3. 若当前状态为 `restored`，校验真实媒体是否仍存在
4. 若不存在或状态非 `restored`，进入恢复流程
5. 调用 `casrestore.EnsureRestored`
6. 恢复成功后更新：
   - `restore_status = restored`
   - `restored_file_id`
   - `restored_file_name`
   - `restored_at`
   - `last_access_at`
   - `recycle_after_at`
7. 获取真实媒体播放地址
8. 返回 302 跳转或代理流

---

## 5.3 恢复流程（内部）

推荐与 `cloud189-auto-save` 一致的恢复原则：

### 个人秒传优先
1. 获取上传 session
2. `initMultiUpload`（不携带 md5，使用 `lazyCheck=1`）
3. `checkTransSecond`
4. `commitMultiUploadFile`

### 回退策略
若出现以下情形，考虑回退家庭中转：
- 403
- 风控
- 黑名单
- `InfoSecurityErrorCode`
- `InvalidPartSize`

### 恢复关键要求
- 必须支持分片大小动态计算
- 必须支持 commit 重试
- 必须支持同一 `.cas` 的恢复并发去重

---

## 5.4 自动回收流程

流程：
1. 定时任务扫描 `cas_media_records`
2. 找出 `restore_status = restored` 且 `recycle_after_at <= now()` 的记录
3. 若当前不在播放 / 不在恢复中，则删除真实媒体文件
4. 删除成功后更新：
   - `restore_status = recycled`
   - 清空 `restored_file_id`
   - 清空 `restored_file_name`
5. 保留 `.cas` 与 `.strm`

可选增强：
- 若删除失败，则记录 `last_error`
- 支持手工触发回收
- 支持按文件夹批量回收

---

## 6. 缓存与并发控制

## 6.1 恢复并发去重

为避免同一媒体被同时恢复多次，需要引入 inflight 控制：

建议 key：
- `storageID + casFileID`

行为：
- 若已有恢复任务在进行，则后续请求等待同一任务结果
- 避免重复调用天翼秒传接口

## 6.2 CAS 元数据缓存

可以支持两层：
- 数据库持久缓存（`cas_media_records`）
- 进程内短期缓存（可选）

首版建议：
- 以数据库字段为主
- 不急于做复杂进程内缓存

---

## 7. 接入点建议

## 7.1 订阅 / 自动导入接入点

候选位置：
- `internal/services/autoingestplan`
- `internal/services/storagefacade`
- `internal/services/mediafile`

建议原则：
- 订阅阶段只负责登记 `.cas` 与生成 `.strm`
- 不负责立即恢复真实媒体

## 7.2 播放接入点

候选位置：
- HTTP handler 层新增 CAS 播放接口
- 通过 `mediafile` 或专门 `casplayback` service 提供最终 URL

## 7.3 云盘能力接入点

候选位置：
- `internal/services/cloudbridge`

建议在此处补充：
- 获取 `.cas` 下载地址
- 上传会话 / 秒传接口
- 家庭恢复相关接口

---

## 8. 首版实现策略

## 阶段 1：基础层

目标：
- 落地 `.cas` 解析模块
- 完成 `.cas` 记录表建模
- 能识别 `.cas` 并生成正确原始文件名

交付物：
- `casparser`
- 数据表迁移
- 单元测试

## 阶段 2：手动恢复闭环

目标：
- 提供手工恢复接口
- 给定 `.cas` 记录即可恢复真实媒体
- 成功后可定位恢复出的文件

交付物：
- `casrestore`
- 恢复状态管理
- 恢复结果记录

## 阶段 3：STRM 播放联动

目标：
- `.strm` 指向播放入口
- 播放时自动恢复
- 返回真实播放地址

交付物：
- `casplayback`
- 新增播放 API
- 订阅侧 `.strm` 生成接线

## 阶段 4：自动回收

目标：
- TTL 清理
- 回收状态记录
- 删除失败重试

交付物：
- `casrecycle`
- 定时任务
- 回收日志

---

## 9. 风险与应对

### 9.1 天翼秒传接口不稳定
应对：
- 接口层独立封装
- 全链路日志
- commit 重试
- 家庭回退策略

### 9.2 首次播放耗时过长
应对：
- 首版接受“首次播放需要等待恢复”
- 后续可考虑元数据预热 / 恢复预热

### 9.3 恢复后的链接获取方式不稳定
应对：
- 优先走现有下载链接生成能力
- 首版优先 302 模式，减少代理复杂度

### 9.4 删除时机导致中断播放
应对：
- 基于最后访问时间回收
- 增加正在播放保护窗口
- 回收前再次校验最近访问时间

---

## 10. 当前设计结论

`.cas` 迁移到 `cloudpan189pro` 应采用以下原则：

1. **迁逻辑，不迁 JS 实现本体**
2. **先做解析 + 恢复最小闭环，再接 STRM 播放**
3. **订阅阶段只保存 `.cas` 与 `.strm`，不直接恢复媒体**
4. **播放时再恢复，恢复后自动回收**
5. **将其作为一个独立的 CAS 能力模块体系建设，而不是分散补丁式实现**

该设计与最终业务目标一致，适合作为 `cloudpan189pro` 后续新增主线能力推进。
