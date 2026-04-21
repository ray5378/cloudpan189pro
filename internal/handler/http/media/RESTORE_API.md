# CAS Restore API

这份说明对应：

- `POST /api/media/restore_cas`
- `POST /api/media/retry_restore`
- `GET /api/media/restore_status`
- `GET /api/media/restore_list`

用于手动联调 cloud189 `.cas` 恢复链。

---

## 先记住两个概念

### 1. `uploadRoute`
表示秒传/上传时优先走哪条路线：

- `family`：家庭路线，**默认**
- `person`：个人路线

### 2. `destinationType`
表示文件最终落在哪类目录：

- `family`
- `person`

> `uploadRoute` 和 `destinationType` 是两个独立维度，不要混用。

---

## `POST /api/media/restore_cas`

手动触发一次 CAS 恢复。

### 请求头

```http
Authorization: Bearer <token>
Content-Type: application/json
```

### 支持三种请求模式

#### 模式 A：最简模式（推荐）

只传：

- `casVirtualId`
- `destinationType`
- `targetFolderId`
- `uploadRoute` 可选

```json
{
  "casVirtualId": 1001,
  "uploadRoute": "family",
  "destinationType": "family",
  "targetFolderId": "-11"
}
```

#### 模式 B：路径模式

只传：

- `casPath`
- `destinationType`
- `targetFolderId`
- `uploadRoute` 可选

```json
{
  "casPath": "/电影库/movie.cas",
  "uploadRoute": "family",
  "destinationType": "person",
  "targetFolderId": "123456"
}
```

#### 模式 C：显式模式

把恢复所需上下文全部手动传入。

### 默认值

- `uploadRoute` 默认 = `family`
- `destinationType` 必填
- `targetFolderId` 必填
- `casVirtualId` / `casPath` 至少传一个（除非你手动显式把上下文都传全）

---

## `POST /api/media/retry_restore`

基于已有恢复记录重新触发恢复。

### 兼容策略

当前 retry 会按下面顺序重新定位 `.cas` 虚拟文件：

1. 优先使用记录里的 `casFilePath`
2. 如果旧记录没有 `casFilePath`，则尝试在同一挂载点下按 `casFileId` 精确匹配
3. 还不行则按 `casFileName(.cas)` 缩窄匹配

所以旧记录现在不一定必须有 `casFilePath` 才能重试；但如果三种定位都失败，接口仍会返回错误。

### 请求体

```json
{
  "recordId": 1,
  "uploadRoute": "family",
  "destinationType": "family",
  "targetFolderId": "-11"
}
```

### 字段说明

- `recordId` 必填
- `uploadRoute` 可选，默认 `family`
- `destinationType` 必填
- `targetFolderId` 可选；不传时默认沿用记录中的 `restoredParentId`

---

## `GET /api/media/restore_status`

查询单个 CAS 恢复记录。

### 支持三种定位方式

- `recordId`
- `casVirtualId`
- `casPath`

---

## `GET /api/media/restore_list`

分页查询恢复记录列表。

### 支持筛选字段

- `storageId`
- `mountPointId`
- `restoreStatus`
- `casFileName`
- `beginAt`
- `endAt`
- `currentPage`
- `pageSize`

---

## 额外说明

### `storageId` 的语义

在这条恢复链里，`storageId` 兜底沿用 `/api/storage/list` 的现有语义：

- `storageId = mountPoint.file_id`

### 为什么最简模式只要 `casVirtualId` / `casPath`

因为接口内部会自动反查：

1. CAS 虚拟文件
2. 所属挂载点 top
3. mount point
4. `storageId / mountPointId / casFileId / casFileName`

所以手动联调时一般不需要自己再拼这些上下文。
