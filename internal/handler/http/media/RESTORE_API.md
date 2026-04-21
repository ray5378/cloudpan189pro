# CAS Restore API

这份说明当前只对应：

- `POST /api/media/restore_cas`

目标很单纯：先手动打通 cloud189 `.cas` 恢复链。

## 实现约束（重要）

这条恢复链当前的实现目标不是“做一个差不多能工作的版本”，而是：

- **严格复刻参考实现**

参考文件：

- `/root/.openclaw/workspace/cloud189-auto-save/src/services/casService.js`
- `/root/.openclaw/workspace/cloud189-auto-save/src/services/cloud189.js`
- `/root/.openclaw/workspace/cloud189-auto-save/src/utils/UploadCryptoUtils.js`

因此凡是涉及云盘操作的：

- 命令名称
- 接口路径
- 参数名
- 字段提取顺序
- 签名方式
- 重试条件
- 轮询逻辑
- cleanup 顺序
- family / person 逻辑链路

都必须照参考实现搬，不接受“看起来等价”的 SDK 替代路线。

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

### 当前 reference-backed 支持范围

当前已经按参考实现收口、可以测试的组合只有：

- `uploadRoute=person`, `destinationType=person`
- `uploadRoute=family`, `destinationType=family`
- `uploadRoute=family`, `destinationType=person`

当前 **不支持**：

- `uploadRoute=person`, `destinationType=family`

原因不是产品语义不允许，而是：当前尚未找到可直接照搬的 reference-backed cloud-operation 主链，因此接口层会直接拒绝该组合。

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
