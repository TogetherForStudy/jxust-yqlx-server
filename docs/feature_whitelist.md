# 功能白名单系统 API 文档

## 概述

功能白名单系统允许管理员创建功能定义，并将特定用户添加到白名单中，使其能够访问这些功能。该系统支持功能的全局开关、用户级别的权限控制、以及基于时间的权限过期机制。

## 核心概念

- **功能（Feature）**：系统中的特定功能模块，由唯一的 `feature_key` 标识
- **白名单（Whitelist）**：被授予特定功能访问权限的用户列表
- **全局开关（Global Switch）**：功能级别的开关，关闭后所有用户都无法访问
- **权限过期（Expiration）**：可为用户设置临时权限，到期后自动失效
- **缓存机制**：用户权限查询结果会被缓存 5 分钟，提升性能

## 数据模型

### Feature（功能定义）

```go
type Feature struct {
    ID          uint           `json:"id"`
    FeatureKey  string         `json:"feature_key"`   // 功能唯一标识，如 "beta_ai_chat"
    FeatureName string         `json:"feature_name"`  // 功能显示名称
    Description string         `json:"description"`   // 功能描述
    IsEnabled   bool           `json:"is_enabled"`    // 全局开关
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-"`
}
```

### UserFeatureWhitelist（用户功能白名单）

```go
type UserFeatureWhitelist struct {
    ID         uint       `json:"id"`
    UserID     uint       `json:"user_id"`
    FeatureKey string     `json:"feature_key"`
    GrantedBy  uint       `json:"granted_by"`    // 授权人ID
    GrantedAt  time.Time  `json:"granted_at"`    // 授权时间
    ExpiresAt  *time.Time `json:"expires_at"`    // 过期时间，NULL 表示永久
    CreatedAt  time.Time  `json:"created_at"`
    UpdatedAt  time.Time  `json:"updated_at"`
}
```

## API 端点

### 用户端 API

#### 1. 获取当前用户的功能列表

**请求**

```http
GET /api/v0/user/features
Authorization: Bearer {token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "features": ["beta_ai_chat", "beta_study_plan"]
  }
}
```

**说明**
- 返回用户当前拥有的所有有效功能列表
- 只返回全局启用且未过期的功能
- 结果会被缓存 5 分钟

---

### 管理员 API

#### 2. 获取所有功能列表

**请求**

```http
GET /api/v0/admin/features
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": [
    {
      "id": 1,
      "feature_key": "beta_ai_chat",
      "feature_name": "AI学习助手（测试版）",
      "description": "基于AI的智能问答助手",
      "is_enabled": true,
      "created_at": "2025-12-07T10:00:00Z",
      "updated_at": "2025-12-07T10:00:00Z"
    }
  ]
}
```

---

#### 3. 获取功能详情

**请求**

```http
GET /api/v0/admin/features/:key
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "id": 1,
    "feature_key": "beta_ai_chat",
    "feature_name": "AI学习助手（测试版）",
    "description": "基于AI的智能问答助手",
    "is_enabled": true,
    "created_at": "2025-12-07T10:00:00Z",
    "updated_at": "2025-12-07T10:00:00Z"
  }
}
```

---

#### 4. 创建功能（幂等性保护）

**请求**

```http
POST /api/v0/admin/features
Authorization: Bearer {admin_token}
X-Idempotency-Key: {uuid}
Content-Type: application/json

{
  "feature_key": "beta_ai_chat",
  "feature_name": "AI学习助手（测试版）",
  "description": "基于AI的智能问答助手",
  "is_enabled": true
}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "id": 1,
    "feature_key": "beta_ai_chat",
    "feature_name": "AI学习助手（测试版）",
    "description": "基于AI的智能问答助手",
    "is_enabled": true,
    "created_at": "2025-12-07T10:00:00Z",
    "updated_at": "2025-12-07T10:00:00Z"
  }
}
```

**注意**
- `feature_key` 必须唯一，建议使用 `beta_` 前缀标识测试功能
- `is_enabled` 可选，默认为 `true`

---

#### 5. 更新功能

**请求**

```http
PUT /api/v0/admin/features/:key
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "feature_name": "AI学习助手（正式版）",
  "description": "更新后的描述",
  "is_enabled": false
}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "message": "更新成功"
  }
}
```

**注意**
- 所有字段都是可选的，只更新提供的字段
- 更新功能后会清除相关缓存

---

#### 6. 删除功能（软删除）

**请求**

```http
DELETE /api/v0/admin/features/:key
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "message": "删除成功"
  }
}
```

**注意**
- 软删除，数据库记录不会真正删除
- 删除后会清除该功能的所有缓存

---

#### 7. 获取功能的白名单列表

**请求**

```http
GET /api/v0/admin/features/:key/whitelist?page=1&page_size=20
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "Data": [
      {
        "id": 1,
        "user_id": 10001,
        "student_id": "202012345",
        "real_name": "张三",
        "granted_by": 1,
        "granted_at": "2025-12-07T10:00:00Z",
        "expires_at": "2025-12-14T10:00:00Z",
        "is_expired": false,
        "created_at": "2025-12-07T10:00:00Z"
      }
    ],
    "Total": 1,
    "Page": 1,
    "Size": 20
  }
}
```

---

#### 8. 授予用户功能权限（幂等性保护）

**请求**

```http
POST /api/v0/admin/features/:key/whitelist
Authorization: Bearer {admin_token}
X-Idempotency-Key: {uuid}
Content-Type: application/json

{
  "user_id": 10001,
  "expires_at": "2025-12-14T10:00:00Z"
}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "message": "授权成功"
  }
}
```

**注意**
- `expires_at` 可选，不提供则表示永久有效
- 如果用户已存在该权限，则更新过期时间

---

#### 9. 批量授予功能权限（幂等性保护）

**请求**

```http
POST /api/v0/admin/features/:key/whitelist/batch
Authorization: Bearer {admin_token}
X-Idempotency-Key: {uuid}
Content-Type: application/json

{
  "user_ids": [10001, 10002, 10003],
  "expires_at": "2025-12-14T10:00:00Z"
}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "message": "批量授权成功"
  }
}
```

---

#### 10. 撤销用户功能权限

**请求**

```http
DELETE /api/v0/admin/features/:key/whitelist/:uid
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "message": "撤销成功"
  }
}
```

**注意**
- 撤销后用户立即失去权限（最多 5 分钟缓存延迟）

---

#### 11. 查看用户的功能权限详情

**请求**

```http
GET /api/v0/admin/users/:id/features
Authorization: Bearer {admin_token}
```

**响应**

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": [
    {
      "feature_key": "beta_ai_chat",
      "feature_name": "AI学习助手（测试版）",
      "granted_by": 1,
      "granted_at": "2025-12-07T10:00:00Z",
      "expires_at": "2025-12-14T10:00:00Z",
      "is_expired": false
    }
  ]
}
```

---

## 中间件使用

### RequireFeature 中间件

用于保护需要特定功能权限的路由。

**示例**

```go
// 在路由中使用
betaGroup := authorized.Group("/ai-chat")
betaGroup.Use(middleware.RequireFeature(featureService, "beta_ai_chat"))
{
    betaGroup.POST("/message", aiChatHandler.SendMessage)
    betaGroup.GET("/history", aiChatHandler.GetHistory)
}
```

**行为**
- 检查用户是否有 `beta_ai_chat` 功能权限
- 无权限返回 `403 Forbidden`
- 有权限则继续处理请求

---

## 使用场景

### 场景 1：开发新功能的灰度测试

1. 管理员创建功能定义：`beta_ai_chat`
2. 添加测试用户到白名单，设置 7 天后过期
3. 前端根据 `/user/features` 返回的列表显示/隐藏功能入口
4. 测试完成后，更新 `is_enabled` 为 `false` 禁用功能

### 场景 2：VIP 功能限制

1. 创建功能：`vip_advanced_study`
2. 只有 VIP 用户才添加到白名单（永久有效）
3. 前端根据权限控制高级功能的显示

### 场景 3：临时活动功能

1. 创建功能：`activity_2025_spring`
2. 批量添加参与用户，设置活动结束时间为过期时间
3. 活动结束后自动失效，无需手动撤销

---

## 缓存策略

### 用户功能列表缓存

- **缓存 Key**: `user_features:{user_id}`
- **过期时间**: 5 分钟
- **失效触发**: 授权/撤销权限时

### 功能全局开关缓存

- **缓存 Key**: `feature_enabled:{feature_key}`
- **过期时间**: 10 分钟
- **失效触发**: 更新/删除功能时

### 注意事项

- 管理员修改权限后，用户最多 5 分钟后生效（缓存过期）
- 如需立即生效，可手动清除 Redis 缓存

---

## 权限检查流程

```
用户请求 -> JWT认证 -> RequireFeature中间件
                             |
                             v
                    检查功能全局开关（缓存）
                             |
                             v
                    检查用户白名单（缓存）
                             |
                   +---------+---------+
                   |                   |
                 有权限              无权限
                   |                   |
                   v                   v
              继续处理             403 Forbidden
```

---

## 错误码

| 错误码 | 说明 |
|-------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（未登录） |
| 403 | 无权访问此功能 |
| 404 | 功能不存在 |
| 500 | 服务器内部错误 |

---

## 最佳实践

1. **功能命名规范**：使用 `beta_` 前缀标识测试功能，如 `beta_ai_chat`
2. **设置过期时间**：临时测试功能建议设置 7-30 天过期
3. **前端配合**：用户登录后调用 `/user/features` 获取权限列表，控制页面渲染
4. **性能考虑**：避免在高频接口中进行权限检查，应在模块入口处检查
5. **日志记录**：`user_feature_whitelist` 表记录了授权人和授权时间，可追溯变更历史

---

## 前端集成示例

```javascript
// 1. 用户登录后获取功能列表
async function fetchUserFeatures() {
  const response = await fetch('/api/v0/user/features', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const data = await response.json();
  return data.Result.features; // ["beta_ai_chat", "beta_study_plan"]
}

// 2. 根据权限控制页面渲染
const features = await fetchUserFeatures();

if (features.includes('beta_ai_chat')) {
  showAIChatEntry(); // 显示AI对话入口
}

if (features.includes('beta_study_plan')) {
  showStudyPlanFeature(); // 显示学习计划功能
}

// 3. 调用受保护的 API
async function sendAIMessage(message) {
  const response = await fetch('/api/v0/ai-chat/message', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ message })
  });
  
  if (response.status === 403) {
    alert('您没有权限使用此功能');
  }
  
  return response.json();
}
```

---

## 数据库表结构

### features 表

| 字段 | 类型 | 说明 |
|-----|------|-----|
| id | INT UNSIGNED | 主键 |
| feature_key | VARCHAR(50) | 功能唯一标识 |
| feature_name | VARCHAR(100) | 功能显示名称 |
| description | VARCHAR(500) | 功能描述 |
| is_enabled | TINYINT | 全局开关 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |
| deleted_at | DATETIME | 软删除时间 |

**索引**
- `idx_feature_key`: UNIQUE (feature_key)
- `idx_is_enabled`: (is_enabled)

### user_feature_whitelist 表

| 字段 | 类型 | 说明 |
|-----|------|-----|
| id | INT UNSIGNED | 主键 |
| user_id | INT UNSIGNED | 用户ID |
| feature_key | VARCHAR(50) | 功能标识 |
| granted_by | INT UNSIGNED | 授权人ID |
| granted_at | DATETIME | 授权时间 |
| expires_at | DATETIME | 过期时间（NULL=永久） |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

**索引**
- `uk_user_feature`: UNIQUE (user_id, feature_key)
- `idx_feature_key`: (feature_key)
- `idx_user_id`: (user_id)
- `idx_expires_at`: (expires_at)

---

## 相关文档

- [数据库设计](./design/database_design.md)
- [缓存设计](./design/cache_design.md)
- [RBAC 设计](./design/rbac_design.md)
- [中间件设计](./design/middleware_design.md)
