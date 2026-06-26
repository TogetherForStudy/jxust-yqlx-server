# 后端错误码参考

本文面向前端联调与异常处理，内容以当前代码实现为准：

- 错误码定义：`pkg/constant/error_code.go`
- 统一错误响应：`internal/handlers/helper/response.go`

注意：仓库内部分旧设计文档仍保留早期 7 位错误码示例，当前服务实际使用的是本文所列的 5 位业务错误码。

## 统一响应格式

除少数特殊接口外，普通 JSON 接口统一返回以下结构：

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-xxxx",
  "Result": {}
}
```

字段说明：

- `StatusCode`：业务状态码，`0` 表示成功，非 `0` 表示失败
- `StatusMessage`：错误文案或成功文案
- `RequestId`：请求唯一标识，排查问题时请一并记录
- `Result`：成功时的业务数据；失败时通常不存在

## 当前实现规则

- 成功响应：HTTP 状态码固定为 `200`，`StatusCode = 0`
- 失败响应：会同时返回非 `0` 的 `StatusCode`，并配套返回对应的 HTTP 状态码
- 前端不要只判断其中一个，建议同时处理 HTTP 状态和 `StatusCode`

错误响应示例：

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json
```

```json
{
  "StatusCode": 11003,
  "StatusMessage": "无效的 Token",
  "RequestId": "req-xxxx"
}
```

## 非统一信封响应的例外接口

以下类型接口不适用本文的统一 JSON 错误结构：

- `/health`
- 聊天 SSE 流式接口
- 文件流下载接口
- MCP 相关接口
- MinIO 代理接口

## 错误码分组

当前错误码按前缀分组：

| 前缀 | 模块 |
| --- | --- |
| `0` | 成功 |
| `10xxx` | 通用错误 |
| `11xxx` | 鉴权与登录 |
| `12xxx` | 会话 |
| `13xxx` | 配置 |
| `20xxx` | 投稿 |
| `21xxx` | 倒数日 |
| `22xxx` | 课程表 |
| `23xxx` | 功能白名单 |
| `24xxx` | 英雄榜 |
| `25xxx` | 资料 |
| `26xxx` | 通知 |
| `27xxx` | 积分 |
| `28xxx` | 题库 |
| `29xxx` | 评价 |
| `30xxx` | 学习任务 |
| `31xxx` | 对象存储 |
| `32xxx` | 词典 |
| `33xxx` | 挂科率 |
| `34xxx` | 统计 |
| `35xxx` | 文件存储 |
| `36xxx` | 用户活跃度 |

## 完整错误码表

### 成功

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `0` | `200` | `SuccessCode` | `Success` |

### 通用错误

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `10001` | `404` | `CommonRouteNotFound` | `路由不存在` |
| `10002` | `405` | `CommonMethodNotAllowed` | `请求方法不允许` |
| `10003` | `400` | `CommonBadRequest` | `请求参数错误` |
| `10004` | `404` | `CommonNotFound` | `资源不存在` |
| `10005` | `409` | `CommonConflict` | `请求冲突` |
| `10006` | `403` | `CommonForbidden` | `权限不足` |
| `10007` | `401` | `CommonUnauthorized` | `未授权` |
| `10008` | `500` | `CommonInternal` | `服务器内部错误` |
| `10009` | `503` | `CommonServiceUnavailable` | `服务暂不可用` |
| `10010` | `404` | `CommonUserNotFound` | `用户不存在` |
| `10011` | `500` | `CommonRequestPanicked` | `服务器内部异常` |

### 鉴权与登录

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `11001` | `401` | `AuthMissingUserContext` | `未获取到用户信息` |
| `11002` | `401` | `AuthInvalidAuthorizationHeader` | `无效的 Authorization 头` |
| `11003` | `401` | `AuthInvalidToken` | `无效的 Token` |
| `11004` | `401` | `AuthInvalidTokenType` | `Token 类型无效` |
| `11005` | `401` | `AuthInvalidTokenClaims` | `无效的 Token Claims` |
| `11006` | `401` | `AuthAccountBlocked` | `用户账号已被封禁` |
| `11007` | `401` | `AuthSessionInvalid` | `当前会话已失效` |
| `11008` | `503` | `AuthCacheUnavailable` | `鉴权缓存未初始化` |
| `11009` | `503` | `AuthStateReadFailed` | `鉴权状态读取失败` |
| `11010` | `503` | `AuthStateParseFailed` | `鉴权状态解析失败` |
| `11011` | `401` | `AuthRefreshTokenInvalid` | `无效的 RefreshToken` |
| `11012` | `401` | `AuthRefreshTokenTypeInvalid` | `RefreshToken 类型无效` |
| `11013` | `401` | `AuthRefreshTokenSessionNotFound` | `RefreshToken 会话不存在` |
| `11014` | `401` | `AuthRefreshTokenExpired` | `RefreshToken 已失效` |
| `11015` | `401` | `AuthMissingSessionInfo` | `缺少会话信息` |
| `11016` | `400` | `AuthUnsupportedTestUserType` | `不支持的测试用户类型` |
| `11017` | `502` | `AuthWechatLoginFailed` | `微信登录失败` |
| `11018` | `401` | `AuthAccountDisabled` | `用户账号已被禁用` |
| `11019` | `401` | `AuthAccountTempBanned` | `用户账号已被临时封禁` |
| `11020` | `401` | `AuthAccountKicked` | `账号已被下线，请稍后重试` |

### 会话

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `12001` | `404` | `ConversationNotFound` | `会话不存在` |
| `12002` | `400` | `ConversationMessageRequired` | `新会话必须提供消息内容` |

### 配置

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `13001` | `409` | `ConfigKeyExists` | `配置键已存在` |
| `13002` | `404` | `ConfigKeyNotFound` | `配置项不存在` |

### 投稿

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `20001` | `404` | `ContributionNotFound` | `投稿不存在` |
| `20002` | `403` | `ContributionForbidden` | `无权限` |
| `20003` | `409` | `ContributionReviewStatusInvalid` | `只能审核待审核状态的投稿` |

### 倒数日

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `21001` | `400` | `CountdownTargetDateInvalid` | `目标日期格式错误` |
| `21002` | `404` | `CountdownNotAccessible` | `倒数日不存在或无权限访问` |

### 课程表

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `22001` | `409` | `CourseTableClassNotSet` | `用户尚未设置班级信息` |
| `22002` | `404` | `CourseTableScheduleNotFound` | `未找到该班级在指定学期的课程表` |
| `22003` | `404` | `CourseTableClassNotFound` | `指定的班级不存在` |
| `22004` | `404` | `CourseTablePersonalScheduleNotFound` | `未找到个人课表数据` |
| `22005` | `409` | `CourseTableBindLimitReached` | `仅可绑定2次` |

### 功能白名单

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `23001` | `404` | `FeatureNotFound` | `功能不存在` |
| `23002` | `409` | `FeatureIdentifierExists` | `功能标识已存在` |

### 英雄榜

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `24001` | `409` | `HeroNameExists` | `名称已存在` |
| `24002` | `404` | `HeroNotFound` | `未找到` |

### 资料

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `25001` | `404` | `MaterialNotFound` | `资料不存在` |
| `25002` | `404` | `MaterialDescriptionNotFound` | `资料描述不存在` |

### 通知

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `26001` | `404` | `NotificationNotFound` | `通知不存在` |
| `26002` | `409` | `NotificationDeletedCannotModify` | `已删除的通知不能修改` |
| `26003` | `403` | `NotificationNoPermissionModify` | `无权限修改` |
| `26004` | `409` | `NotificationDraftOnlyPublish` | `只能发布草稿状态的通知` |
| `26005` | `409` | `NotificationNotPublished` | `通知未发布` |
| `26006` | `409` | `NotificationReviewStatusInvalid` | `只能审核待审核状态的通知` |
| `26007` | `409` | `NotificationAlreadyReviewed` | `您已经审核过该通知` |
| `26008` | `409` | `NotificationOnlyPublishedCanPin` | `只有已发布的通知才能置顶` |
| `26009` | `409` | `NotificationAlreadyPinned` | `通知已经置顶` |
| `26010` | `409` | `NotificationNotPinned` | `通知未置顶` |
| `26011` | `404` | `CategoryNotFound` | `分类不存在` |

### 积分

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `27001` | `409` | `PointsInsufficient` | `积分不足` |

### 题库

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `28001` | `404` | `QuestionNotFound` | `题目不存在` |
| `28002` | `409` | `QuestionDisabled` | `题目已禁用` |

### 评价

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `29001` | `409` | `ReviewDuplicate` | `您已经评价过该教师的这门课程` |
| `29002` | `409` | `ReviewApproved` | `评价已审核通过，无需重复审核` |
| `29003` | `404` | `ReviewNotFound` | `评价不存在` |

### 学习任务

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `30001` | `400` | `StudyTaskDeadlineInvalid` | `截止时间格式错误` |
| `30002` | `400` | `StudyTaskDateInvalid` | `截止日期格式错误` |
| `30003` | `404` | `StudyTaskNotAccessible` | `学习任务不存在或无权限访问` |

### 对象存储

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `31001` | `400` | `URIInvalid` | `uri 必须以 / 开头，并且不可为空` |
| `31002` | `500` | `TokenSecretMissing` | `对象存储配置缺失` |

### 词典

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `32001` | `500` | `DictionaryRandomWordFailed` | `获取随机单词失败` |

### 挂科率

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `33001` | `500` | `FailRateQueryFailed` | `查询挂科率失败` |

### 统计

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `34001` | `503` | `StatServiceUnavailable` | `统计服务暂不可用` |

### 文件存储

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `35001` | `400` | `StoreFileUploadFailed` | `上传文件失败` |
| `35002` | `500` | `StoreFileOpenFailed` | `打开上传文件失败` |
| `35003` | `400` | `StoreInvalidTags` | `标签格式错误` |
| `35004` | `500` | `StoreFileStoreFailed` | `存储文件失败` |
| `35005` | `500` | `StoreFileDeleteFailed` | `删除文件失败` |
| `35006` | `500` | `StoreFileListFailed` | `获取文件列表失败` |
| `35007` | `500` | `StoreExpiredFileListFailed` | `获取过期文件列表失败` |
| `35008` | `500` | `StoreFileURLFailed` | `生成文件链接失败` |
| `35009` | `404` | `StoreFileNotFound` | `文件不存在` |
| `35010` | `500` | `StoreFileStreamFailed` | `文件流传输失败` |

### 用户活跃度

| 业务码 | HTTP | 后端常量 | 默认文案 |
| --- | --- | --- | --- |
| `36001` | `500` | `UserActivityQueryFailed` | `查询登录天数失败` |

## 前端处理建议

- `StatusCode = 0` 才视为业务成功
- 请求失败时优先保留 `RequestId`，便于后端排查
- `401` 或 `10007`、`11xxx`：按登录失效处理；若有刷新令牌机制，可先尝试刷新
- `403` 或 `10006`：提示无权限，不建议直接重试
- `404`：用于“资源不存在”或“无权限访问后按不存在处理”的场景，可展示空态或返回上一页
- `409`：表示状态冲突、重复提交、重复审核、额度限制等，优先展示后端文案
- `500`、`502`、`503`：统一提示“服务异常，请稍后再试”，同时上报 `RequestId`

## 推荐前端封装

```ts
export interface ApiResponse<T> {
  StatusCode: number;
  StatusMessage: string;
  RequestId: string;
  Result?: T;
}

export function assertBizSuccess<T>(resp: ApiResponse<T>) {
  if (resp.StatusCode !== 0) {
    const error = new Error(resp.StatusMessage);
    (error as Error & { code?: number; requestId?: string }).code = resp.StatusCode;
    (error as Error & { code?: number; requestId?: string }).requestId = resp.RequestId;
    throw error;
  }
  return resp.Result as T;
}
```

## 维护说明

- 本文应与 `pkg/constant/error_code.go` 保持同步
- 若新增错误码，请同时补充本文档
- 若与其他旧文档冲突，以当前代码实现和本文为准
