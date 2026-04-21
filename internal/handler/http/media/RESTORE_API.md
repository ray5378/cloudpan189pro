# CAS Restore API

这份说明对应：

- `POST /api/media/restore_cas`
- `GET /api/media/restore_status`

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

把恢复所需上下文全部手动传入：

```json
{
  "storageId": 1,
  "mountPointId": 1,
  "casFileId": "123456789",
  "casFileName": "movie.cas",
  "casVirtualId": 1001,
  "uploadRoute": "person",
  "destinationType": "family",
  "targetFolderId": "-11"
}
```

### 默认值

- `uploadRoute` 默认 = `family`
- `destinationType` 必填
- `targetFolderId` 必填
- `casVirtualId` / `casPath` 至少传一个（除非你手动显式把上下文都传全）

### curl 示例

#### 1) 家庭路线，最终落家庭目录

```bash
curl -X POST 'http://127.0.0.1:12395/api/media/restore_cas' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "casVirtualId": 1001,
    "uploadRoute": "family",
    "destinationType": "family",
    "targetFolderId": "-11"
  }'
```

#### 2) 家庭路线，最终落个人目录

```bash
curl -X POST 'http://127.0.0.1:12395/api/media/restore_cas' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "casPath": "/电影库/movie.cas",
    "uploadRoute": "family",
    "destinationType": "person",
    "targetFolderId": "123456"
  }'
```

#### 3) 个人路线，最终落个人目录

```bash
curl -X POST 'http://127.0.0.1:12395/api/media/restore_cas' \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "casVirtualId": 1001,
    "uploadRoute": "person",
    "destinationType": "person",
    "targetFolderId": "123456"
  }'
```

---

## `GET /api/media/restore_status`

查询单个 CAS 恢复记录。

### 支持三种定位方式

- `recordId`
- `casVirtualId`
- `casPath`

### curl 示例

#### 1) 按记录 ID

```bash
curl 'http://127.0.0.1:12395/api/media/restore_status?recordId=1' \
  -H 'Authorization: Bearer <token>'
```

#### 2) 按 CAS 虚拟文件 ID

```bash
curl 'http://127.0.0.1:12395/api/media/restore_status?casVirtualId=1001' \
  -H 'Authorization: Bearer <token>'
```

#### 3) 按 CAS 路径

```bash
curl 'http://127.0.0.1:12395/api/media/restore_status?casPath=/电影库/movie.cas' \
  -H 'Authorization: Bearer <token>'
```

### 返回里重点看什么

`data` 里重点字段：

- `restoreStatus`
  - `pending`
  - `restoring`
  - `restored`
  - `failed`
- `restoredFileId`
- `restoredFileName`
- `restoredParentId`
- `lastError`
- `restoredAt`

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
