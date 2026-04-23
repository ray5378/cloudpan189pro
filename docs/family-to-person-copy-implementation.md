# 家庭云盘 → 个人云盘（可指定目录）技术实现文档

## 目标

在现有 `cloudpan189pro` 中补一条新的官方转存链路：

- **源**：家庭云盘文件 / 文件夹
- **目标**：个人云盘指定目录
- **方式**：复用 189 官方 App 已在使用的 batch task 机制

该链路与当前项目里的 `DELETE / CLEAR_RECYCLE / SHARE_SAVE` 同属一套任务框架，但 task type 不同。

---

## 结论先行

根据 PC 网页、H5 网页、Android App 抓包交叉验证，已经确认：

### 1. 家庭 → 个人默认保存
网页 / H5 直接走：

```http
GET /open/family/manage/saveFileToMember.action?familyId={familyId}&fileIdList={fileIdList}
```

特征：

- 200 空响应
- 无 taskId
- 看起来保存到当前成员的个人空间默认位置
- **不适合做“指定目录”**

### 2. 家庭 → 个人指定目录
Android App 走的是 batch task：

#### 创建任务
```http
POST /batch/createBatchTask.action
```

请求体（multipart/form-data）：

```text
familyId=300000933227076
groupId=null
targetFolderId=323991244668921426
copyType=2
shareId=null
type=COPY
taskInfos=[{"fileId":"724431245208796587","fileName":"媒体库","isFolder":1,"srcParentId":0}]
```

#### 轮询任务
```http
POST /batch/checkBatchTask.action?taskId={taskId}&type=COPY
```

已观察到的状态：

- `taskStatus=3`：进行中
- `taskStatus=4`：任务结束

失败示例：

```json
{
  "res_code": 0,
  "res_message": "成功",
  "errorCode": "InsufficientStorageSpace",
  "failedCount": 1,
  "successedCount": 0,
  "taskStatus": 4
}
```

这说明任务链路本身成立，只是目标个人空间容量不足。

---

## 与参考项目的交叉验证

### 已匹配到的底座
当前项目 / 参考实现中已经存在并稳定使用：

- `createBatchTask.action`
- `checkBatchTask.action`

并且已有 task type：

- `DELETE`
- `CLEAR_RECYCLE`
- `SHARE_SAVE`

因此：

> App 抓到的 `COPY` 不是另一套体系，而是复用同一套 batch task 框架。

### 已验证一致的点

1. **任务创建入口一致**
   - 都是 `createBatchTask.action`
2. **任务轮询入口一致**
   - 都是 `checkBatchTask.action`
3. **完成态判断一致**
   - `taskStatus=4` 视为结束
4. **目标目录语义一致**
   - `targetFolderId` 在现有项目里已经被广泛当作最终落点目录使用

### 当前尚未在参考项目中直接看到的点

- `type=COPY + copyType=2 + familyId + targetFolderId`

也就是说：

> 底座和轮询方式与参考项目一致，但“家庭 → 个人指定目录”需要按抓包补一个新的 task type 实现。

---

## 已抓到的关键请求形态

## A. 网页 / H5：默认保存到个人空间

```http
GET /open/family/manage/saveFileToMember.action?familyId={familyId}&fileIdList={fileIdList}
```

### 已知特征

- 方法：`GET`
- 返回：`200` 空 body
- 参数：
  - `familyId`
  - `fileIdList`
- 未看到目标个人目录参数

### 适用场景

- 家庭文件保存到当前成员个人空间默认位置

### 限制

- 不能明确指定个人云盘目标目录

---

## B. App：保存到个人云盘指定目录

```http
POST /batch/createBatchTask.action
Content-Type: multipart/form-data
```

### 请求头（已抓到）

```text
sessionkey: f65122c0-a63a-4496-b2a3-6871f20b77fa_family
signature: a686c320c1a88cf2c33e61a4879e908c71431eaf
date: Thu, 23 Apr 2026 08:38:10 GMT
user-agent: Ecloud/10.3.12 (VRD-AL09; ; huawei) Android/29
```

### URL 查询参数

```text
rand=1776933457306
clientType=TELEANDROID
model=VRD-AL09
version=10.3.12
```

### 请求体

```text
familyId=300000933227076
groupId=null
targetFolderId=323991244668921426
copyType=2
shareId=null
type=COPY
taskInfos=[{"fileId":"724431245208796587","fileName":"媒体库","isFolder":1,"srcParentId":0}]
```

### 参数推断

| 字段 | 含义 |
|---|---|
| `familyId` | 源家庭空间 ID |
| `targetFolderId` | 个人云盘目标目录 ID |
| `copyType=2` | 高概率表示“家庭 → 个人”复制类型 |
| `type=COPY` | batch task 类型为复制 |
| `taskInfos[].fileId` | 源文件 / 源文件夹 ID |
| `taskInfos[].fileName` | 源名称 |
| `taskInfos[].isFolder` | `1`=文件夹，`0`=文件 |
| `taskInfos[].srcParentId` | 源父目录 ID，抓包里有观察到 `0` |
| `groupId=null` | 当前抓包未体现特殊作用 |
| `shareId=null` | 当前抓包未体现特殊作用 |

---

## 标准实现链路（建议）

## 1. 创建 COPY 任务

### 请求

```http
POST /batch/createBatchTask.action
```

### 参数

```text
familyId={源家庭ID}
groupId=null
targetFolderId={目标个人目录ID}
copyType=2
shareId=null
type=COPY
taskInfos=[{"fileId":"{源ID}","fileName":"{源名称}","isFolder":{0|1},"srcParentId":{源父ID}}]
```

### 预期响应

返回 JSON，需拿到：

- `res_code`
- `taskId`

> 这部分在当前抓包中已确认发起成功，但仍建议后续再次补抓完整 `response_body` 保存样例。

---

## 2. 轮询任务状态

### 请求

```http
POST /batch/checkBatchTask.action?taskId={taskId}&type=COPY
```

### 轮询规则

建议继续沿用当前项目现有 batch task 轮询框架：

- 间隔：1s
- 超时：60~120s
- `taskStatus=4` 视为结束

### 结束态判断建议

#### 成功

满足：

- `taskStatus == 4`
- `failedCount == 0`
- `successedCount > 0`

#### 失败

满足任一：

- `taskStatus == 4 && failedCount > 0 && successedCount == 0`
- 返回明确 `errorCode`

### 失败码样例

- `InsufficientStorageSpace`
  - 个人空间容量不足

---

## 建议新增的服务层能力

建议新增一个专用服务，例如：

- `internal/services/familycopy/`
- 或并入当前 `casrestore` / `cloudbridge` 扩展模块

### 建议接口

```go
type FamilyCopyService interface {
    CopyFamilyToPerson(ctx context.Context, req CopyFamilyToPersonRequest) (*CopyFamilyToPersonResult, error)
}

type CopyFamilyToPersonRequest struct {
    TokenID        int64
    FamilyID       int64
    TargetFolderID string
    FileID         string
    FileName       string
    IsFolder       bool
    SrcParentID    string
}

type CopyFamilyToPersonResult struct {
    TaskID         string
    SuccessedCount int
    FailedCount    int
    ErrorCode      string
}
```

---

## 建议的实现细节

### 1. 不要优先走网页 `saveFileToMember.action`
原因：

- 无法明确指定目标目录
- 返回信息过少
- 不利于失败诊断

### 2. 优先走 App 的 `COPY` 任务链
原因：

- 支持 `targetFolderId`
- 支持文件夹
- 有 taskId
- 有错误码
- 更适合服务端稳定实现

### 3. 保留失败码透传
例如：

- `InsufficientStorageSpace`
- 未来可能还有：权限不足、目录不存在、非法源文件等

不要统一抹成“复制失败”。

---

## 当前最大的未解决项

## App 签名规则

App 请求头和网页不同，目前抓到的是：

- `sessionkey`
- `signature`
- `date`

并附带 URL 参数：

- `clientType=TELEANDROID`
- `version=10.3.12`
- `model=VRD-AL09`
- `rand=...`

### 当前能确定

- 这不是网页 `AccessToken + Timestamp + Signature + Sign-Type` 那套
- 是 App 自己的签名体系

### 当前不能 100% 确定

- `signature` 的计算算法
- `sessionkey` 与账号 token 的映射关系
- `date` 是否参与签名原文

因此：

> 现在已经能确定“接口形态”和“任务模型”，但真正落代码前，还需要继续反推 App 的签名算法，或找到现有库中可复用的 App 签名实现。

---

## 推荐开发路线

## 阶段 1：先做技术验证

1. 固化抓包样例
2. 写文档（本文件）
3. 从现有代码中复用 batch 轮询框架
4. 单独实现 `COPY` task 参数组织

## 阶段 2：攻克签名

优先查找：

- 当前仓库已有 app 直连接口实现
- `tickstep/cloudpan189-api` 是否已包含 `sessionkey + signature + date` 机制
- 其它参考项目是否已有 App API 适配

## 阶段 3：接入业务功能

可以先做成后端接口，例如：

```http
POST /api/media/copy_family_to_person
```

请求体：

```json
{
  "tokenId": 1,
  "familyId": 300000933227076,
  "targetFolderId": "323991244668921426",
  "fileId": "724431245208796587",
  "fileName": "媒体库",
  "isFolder": true,
  "srcParentId": "0"
}
```

---

## 现阶段可下的最终结论

### 已确认

- 家庭 → 个人默认保存：网页/H5 可走 `saveFileToMember.action`
- 家庭 → 个人指定目录：App 可走 `createBatchTask(COPY)`
- 这条 App 链路与参考项目当前使用的 batch task 底座一致
- `targetFolderId` 语义明确成立
- `taskStatus=4` 结束态判断与参考项目一致

### 尚未确认

- App `signature` 精确算法

### 因而当前结论

> **“家庭云盘 → 个人云盘指定目录”是可以实现的，且推荐按 App 的 `COPY` batch task 方案落地；当前唯一待补的是 App 签名实现。**
