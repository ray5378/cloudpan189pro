# CAS 按需恢复播放技术设计文档（确认版）

## 1. 设计目标

本设计不是迁移 `cloud189-auto-save` 的整个任务系统，而是：

**在 `cloud189pro` 中，基于已有 cloud189 读取链、STRM 写入能力和订阅框架，补齐 `.cas` 所需的恢复写链、播放触发链与回收链。**

最终实现：
- 订阅阶段仅保存 `.cas`
- 为 `.cas` 生成 `.strm`
- 播放请求到来时自动恢复真实媒体
- 播放完成后一段时间回收真实媒体
- 长期只保留 `.cas` 与 `.strm`

---

## 2. 重新核对代码后的现状结论

## 2.1 `cloud189pro` 当前已确认存在的能力

### A. cloud189 读取侧能力（位于 `internal/services/cloudbridge`）
已看到：
- 个人盘文件列表
- 家庭盘文件列表
- 分享文件列表
- 分享校验
- 下载链接生成

这说明 `cloud189pro` 不是完全没有 cloud189 接口基础，而是已经具备了 **读取链**。

### B. STRM 写入能力（位于 `internal/services/mediafile`）
已看到：
- `service_write_strm.go`

说明 `.strm` 文件写入能力已有明确接入点。

### C. 系统框架能力
已看到：
- `autoingestplan`
- `virtualfile`
- `storagefacade`
- `mountpoint`
- `mediaconfig`

说明：
- 订阅 / 记录 / 媒体映射 / 存储抽象这些系统框架已经存在。

## 2.2 `cloud189pro` 当前尚未看到的能力

### A. cloud189 恢复写链
当前未看到现成实现：
- 上传 session / 秒传上下文
- `initMultiUpload`
- `checkTransSecond`
- `commitMultiUploadFile`
- 家庭回退恢复
- 恢复后确认文件存在
- 恢复目标文件删除链

也就是说，当前 `cloud189pro` 的关键缺口不是读链，而是：

**`.cas` 所需的 cloud189 恢复写链。**

## 2.3 `cloud189-auto-save` 当前可直接参考的关键位置

### A. `src/services/casService.js`
可参考：
- `.cas` 判断
- 原始文件名推导
- `.cas` 内容解析
- 下载 `.cas`
- 个人秒传恢复
- 家庭回退恢复

### B. `src/services/lazyShareStrm.js`
可参考：
- 懒恢复思路
- `.cas` 元数据缓存
- 并发去重
- 中转 `.cas` 清理

### C. `src/services/task.js`
可参考：
- 普通任务场景下的 `.cas` 转存后恢复链

结论：
- `cloud189-auto-save` 适合作为 cloud189 `.cas` 恢复链的**逻辑参考**
- 不适合作为任务系统整体直接迁移目标

---

## 3. 最终架构定位

## 3.1 实现约束（必须遵守）

本方案在实现过程中，必须明确遵守以下约束，以避免形成技术债过重的大泥球：

- 必须按**功能边界**拆分组件，而不是按“先堆到一个 service 里再说”推进
- 不允许把 `.cas` 解析、cloud189 恢复、播放触发、回收调度、状态机全部塞进一个超大 service
- 不允许把 handler 直接写成业务主流程承载层
- 不允许为了追求短期可跑，把 `cloudbridge` 演化成“什么都塞”的超级包
- 不允许在订阅、播放、回收等不同链路里重复复制 `.cas` 核心逻辑
- 状态记录、恢复过程、回收过程必须有清晰职责边界

实现上应优先维持如下拆分：

- `casparser`：只负责 `.cas` 协议处理
- `casrestore`：只负责恢复链
- `casplayback`：只负责播放入口与恢复触发
- `casrecycle`：只负责回收链
- `CasMediaRecord`：只负责状态承载与生命周期记录

如果某个模块在推进过程中明显变厚，应优先继续拆分，而不是继续堆积。

## 3.2 不做的事情
- 不直接把 `cloud189-auto-save` JS 任务流整体翻译成 Go
- 不先做复杂流代理优化
- 不在恢复链未打通前先做完整播放器联调

## 3.3 要做的事情
在 `cloud189pro` 中新增一条纵向能力链：

### 第一层：CAS 协议层
负责：
- 识别 `.cas`
- 解析 `.cas`
- 推导原始文件名

### 第二层：CAS 恢复层
负责：
- 下载 `.cas`
- 解析元数据
- 调 cloud189 秒传恢复真实媒体
- 家庭回退
- 恢复并发去重
- 恢复状态管理

### 第三层：CAS 播放层
负责：
- 接收 `.strm` 指向的播放请求
- 检查是否已恢复
- 必要时触发恢复
- 返回真实播放地址

### 第四层：CAS 回收层
负责：
- 按 TTL 删除恢复出来的真实媒体
- 更新恢复状态
- 保留 `.cas` 与 `.strm`

---

## 4. 模块拆分建议

## 4.1 `casparser`
建议位置：
- `internal/services/casparser`

建议文件：
- `types.go`
- `parser.go`
- `parser_test.go`

职责：
- `IsCasFile(name string) bool`
- `GetOriginalFileName(casFileName string, info *CasInfo) string`
- `ParseCasContent(content []byte) (*CasInfo, error)`

该模块只做 `.cas` 协议，不碰 cloud189 API。

---

## 4.2 `casrestore`
建议位置：
- `internal/services/casrestore`

建议文件：
- `service.go`
- `cloud189_restore.go`
- `state.go`
- `download.go`

职责：
- 根据 `.cas` 文件内容恢复真实媒体
- 调用 cloud189 恢复写链
- 执行个人恢复优先 + 家庭回退
- 记录恢复状态
- 执行 inflight 去重

建议核心接口：

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
```

---

## 4.3 `casplayback`
建议位置：
- `internal/services/casplayback`
- `internal/handler/http/cas_playback.go`

职责：
- 作为 `.strm` 的播放入口
- 查询 CAS 记录
- 必要时触发恢复
- 恢复成功后返回真实播放 URL

建议首版接口：

```text
GET /api/cas/play/:recordId
```

首版建议行为：
- 优先返回 **302 跳转** 到真实媒体地址
- 暂不优先做流代理

---

## 4.4 `casrecycle`
建议位置：
- `internal/services/casrecycle`
- `internal/handler/scheduler`

职责：
- 定时扫描已恢复媒体
- 根据 TTL 删除真实媒体
- 更新状态
- 删除失败记录并重试

---

## 5. 数据模型设计

建议新增表：`cas_media_records`

建议字段：
- `id`
- `storage_id`
- `mount_point_id`
- `cas_file_id`
- `cas_file_name`
- `cas_file_path`
- `source_parent_id`
- `restored_parent_id`
- `original_file_name`
- `original_file_size`
- `file_md5`
- `slice_md5`
- `strm_relative_path`
- `restored_file_id`
- `restored_file_name`
- `restore_status`（`pending / restoring / restored / failed / recycling / recycled`）
- `last_access_at`
- `restored_at`
- `recycle_after_at`
- `last_error`
- `created_at`
- `updated_at`

索引建议：
- `(storage_id, cas_file_id)` 唯一索引
- `restore_status` 索引
- `recycle_after_at` 索引

---

## 6. 核心流程设计（确认版）

## 6.1 第一阶段：恢复最小闭环

这是当前最应该优先落地的阶段。

### 目标
先不接播放器，先做到：
- 能识别 `.cas`
- 能解析 `.cas`
- 能根据 `.cas` 手动恢复真实媒体
- 能返回恢复成功结果

### 流程
1. 给定 `.cas` 文件标识
2. 获取 `.cas` 下载链接
3. 下载 `.cas` 内容
4. 解析出 `name / size / md5 / sliceMd5`
5. 调用个人秒传恢复链：
   - `initMultiUpload`
   - `checkTransSecond`
   - `commitMultiUploadFile`
6. 必要时走家庭回退
7. 恢复完成后查找到真实媒体文件
8. 返回恢复结果并记录状态

### 为什么先做这个
因为这是整条链风险最高、价值最高的部分。
只有恢复闭环打通，STRM 播放与自动回收才值得接。

---

## 6.2 第二阶段：STRM 播放联动

### 目标
- `.strm` 指向 CAS 播放入口
- 播放请求到来时自动恢复
- 恢复成功后返回真实播放地址

### 流程
1. 订阅阶段生成 `.strm`
2. `.strm` 内容为：

```text
http://<server>/api/cas/play/<recordId>
```

3. 播放器请求播放入口
4. 查询 CAS 记录
5. 若已恢复且真实文件仍存在，直接返回真实播放地址
6. 否则触发恢复
7. 恢复成功后更新状态并返回真实播放地址

---

## 6.3 第三阶段：自动回收

### 目标
- 让恢复出来的真实媒体在 TTL 后自动删除
- 长期仅保留 `.cas` 与 `.strm`

### 流程
1. 定时任务扫描 `cas_media_records`
2. 找出 `restore_status = restored` 且 `recycle_after_at <= now()` 的记录
3. 若当前不在恢复中、也未命中访问保护，则删除真实媒体
4. 更新状态为 `recycled`
5. 保留 `.cas` 与 `.strm`

---

## 7. cloud189 恢复写链实现重点

这部分是当前最大的新增模块。

### 7.1 必须补出的能力
- cloud189 上传 / 秒传 session
- `initMultiUpload`
- `checkTransSecond`
- `commitMultiUploadFile`
- 恢复后的文件查询
- 家庭回退恢复
- 失败重试

### 7.2 直接参考来源
参考：
- `cloud189-auto-save/src/services/casService.js`

重点参考逻辑：
- init 阶段不带 md5，使用 `lazyCheck=1`
- commit 阶段重试
- 黑名单 / 403 / 风控 / `InvalidPartSize` 回退家庭恢复
- 分片大小动态计算

---

## 8. 并发与状态控制

## 8.1 恢复并发去重
同一 `.cas` 不能同时恢复多次。

建议 key：
- `storageID + casFileID`

行为：
- 若已有 inflight 恢复任务，则后续请求等待同一结果

## 8.2 状态流转
建议状态：
- `pending`
- `restoring`
- `restored`
- `failed`
- `recycling`
- `recycled`

### 首版要求
- 状态必须持久化
- 失败原因必须记录
- 回收状态必须可追踪

---

## 9. 订阅阶段接线原则

### 当前原则
订阅阶段只做：
- 保存 `.cas`
- 建立 CAS 记录
- 生成 `.strm`

### 当前不做
- 订阅时立刻恢复真实媒体
- 订阅时直接下载真实媒体

原因：
- 你的目标就是长期只保留 `.cas`
- 恢复应当延迟到播放触发时发生

---

## 10. 当前确认后的实施顺序

## 第一优先级
在 `cloud189pro` 中落地：
1. `.cas` 解析模块
2. CAS 记录表
3. cloud189 秒传恢复写链
4. 手动恢复最小闭环

## 第二优先级
在恢复闭环稳定后再落地：
1. `.strm` 播放入口
2. 播放触发恢复
3. 返回真实播放地址

## 第三优先级
最后再落地：
1. 自动回收
2. 播放保护
3. 删除失败重试

---

## 11. 当前结论（确认版）

重新读完两边代码后的最明确结论是：

### `cloud189pro` 现状
- 已有 cloud189 读链
- 已有 STRM 写入能力
- 已有订阅 / 存储 / 媒体映射框架
- 尚无 `.cas` 所需的恢复写链

### `cloud189-auto-save` 价值
- 是 cloud189 `.cas` 恢复链的主要逻辑参考来源
- 尤其适合参考：下载 `.cas`、解析 `.cas`、秒传恢复、家庭回退、懒恢复思路

### 当前最正确的起手式
- 先在 `cloud189pro` 落 `.cas` 解析模块
- 再补 cloud189 秒传恢复写链
- 做成“手动恢复成功”的最小闭环
- 最后再接 `.strm` 播放与自动回收

这就是当前两边代码重新核对后的确认版技术路线。
