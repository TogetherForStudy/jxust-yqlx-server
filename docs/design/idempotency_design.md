# 幂等性设计文档

## 文档更新记录

| Code | Module       | Date       | Author  | PRI | Description                |
|------|--------------|------------|---------|-----|----------------------------|
| 1    | idempotency  | 2025-12-06 | Copilot | P0  | 初始设计文档创建           |

## 概述

幂等性（Idempotency）是指同一个请求被执行一次或多次所产生的效果是相同的。本系统通过基于Redis的幂等性中间件，防止因网络重试、用户重复点击等原因导致的重复提交问题。

## 设计目标

1. **防止重复提交**：同一请求多次提交只执行一次
2. **最小侵入**：通过中间件方式实现，不修改业务代码
3. **灵活配置**：支持严格模式和宽松模式
4. **高性能**：基于Redis内存存储，响应迅速
5. **分布式支持**：使用分布式锁防止并发问题

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端请求                              │
│                  Header: X-Idempotency-Key: <uuid>              │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      幂等性中间件                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  1. 读取 X-Idempotency-Key                              │   │
│  │  2. 构建缓存Key: idempotent:{userID}:{idempotencyKey}   │   │
│  │  3. 获取分布式锁                                         │   │
│  │  4. 检查缓存是否存在                                     │   │
│  │     - 存在: 返回缓存的响应                               │   │
│  │     - 不存在: 继续执行请求                               │   │
│  │  5. 缓存成功响应                                         │   │
│  │  6. 释放分布式锁                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        业务处理层                               │
└─────────────────────────────────────────────────────────────────┘
```

## 核心实现

### 1. 缓存接口扩展

```go
type Cache interface {
    // 基础操作
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value string, expiration *time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)

    // 分布式锁
    Lock(ctx context.Context, key string, expiration time.Duration) (bool, error)
    Unlock(ctx context.Context, key string) error
    SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)

    Close() error
}
```

### 2. 中间件函数

| 函数 | 说明 |
|------|------|
| `IdempotencyRecommended()` | 宽松模式：无Key时打印警告，继续处理 |
| `IdempotencyRequired()` | 严格模式：无Key时拒绝请求 |
| `IdempotencyWithTTL(ttl)` | 自定义过期时间的幂等性中间件 |

### 3. 常量定义

```go
const (
    IdempotencyKey           = "X-Idempotency-Key"      // Header名称
    IdempotencyCachePrefix   = "idempotent:"            // 缓存Key前缀
    IdempotencyExpiration    = 24 * time.Hour           // 默认过期时间
    IdempotencyLockTimeout   = 30 * time.Second         // 分布式锁超时
)
```

## 使用方式

### 前端集成

```javascript
// 生成幂等性Key
function generateIdempotencyKey() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        var r = Math.random() * 16 | 0;
        var v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

// 发送请求
async function createReview(data) {
    const idempotencyKey = generateIdempotencyKey();
    
    const response = await wx.request({
        url: '/api/v0/reviews',
        method: 'POST',
        header: {
            'Authorization': 'Bearer ' + token,
            'X-Idempotency-Key': idempotencyKey,
            'Content-Type': 'application/json'
        },
        data: data
    });
    
    return response;
}

// 重试时使用相同的Key
async function createReviewWithRetry(data, maxRetries = 3) {
    const idempotencyKey = generateIdempotencyKey();
    
    for (let i = 0; i < maxRetries; i++) {
        try {
            const response = await wx.request({
                url: '/api/v0/reviews',
                method: 'POST',
                header: {
                    'Authorization': 'Bearer ' + token,
                    'X-Idempotency-Key': idempotencyKey,  // 同一操作使用相同Key
                    'Content-Type': 'application/json'
                },
                data: data
            });
            return response;
        } catch (error) {
            if (i === maxRetries - 1) throw error;
            await sleep(1000 * (i + 1));  // 指数退避
        }
    }
}
```

### 后端路由配置

```go
// 创建类操作 - 推荐使用幂等性保护
reviews.POST("/", middleware.IdempotencyRecommended(), reviewHandler.CreateReview)

// 积分消费 - 强烈推荐使用幂等性保护
points.POST("/spend", middleware.IdempotencyRecommended(), pointsHandler.SpendPoints)

// 审核操作 - 推荐使用幂等性保护
adminReviews.POST("/:id/approve", middleware.IdempotencyRecommended(), reviewHandler.ApproveReview)
```

## 已保护的接口

| 接口路径 | 方法 | 说明 | 保护级别 |
|----------|------|------|----------|
| `/api/v0/reviews` | POST | 创建评价 | 推荐 |
| `/api/v0/reviews/:id/approve` | POST | 审核通过 | 推荐 |
| `/api/v0/reviews/:id/reject` | POST | 审核拒绝 | 推荐 |
| `/api/v0/points/spend` | POST | 消费积分 | 推荐 |
| `/api/v0/contributions` | POST | 创建投稿 | 推荐 |
| `/api/v0/contributions/:id/review` | POST | 审核投稿 | 推荐 |
| `/api/v0/countdowns` | POST | 创建倒数日 | 推荐 |
| `/api/v0/study-tasks` | POST | 创建学习任务 | 推荐 |
| `/api/v0/admin/notifications` | POST | 创建通知 | 推荐 |
| `/api/v0/admin/notifications/:id/publish` | POST | 发布通知 | 推荐 |
| `/api/v0/admin/notifications/:id/approve` | POST | 审核通知 | 推荐 |
| `/api/v0/admin/categories` | POST | 创建分类 | 推荐 |
| `/api/v0/heroes` | POST | 创建英雄榜 | 推荐 |
| `/api/v0/config` | POST | 创建配置 | 推荐 |

## 响应说明

### 正常响应

首次请求和重复请求都返回相同的响应，重复请求会增加响应头：

```
HTTP/1.1 200 OK
X-Idempotency-Replayed: true
Content-Type: application/json

{
    "request_id": "xxx",
    "status_message": "Success",
    "result": { ... }
}
```

### 错误响应

#### 缺少幂等性Key（严格模式）

```json
{
    "request_id": "xxx",
    "status_code": 400,
    "status_message": "缺少幂等性Key，请在Header中添加 X-Idempotency-Key"
}
```

#### 请求正在处理中

```json
{
    "request_id": "xxx",
    "status_code": 409,
    "status_message": "请求正在处理中，请稍后重试"
}
```

## 配置说明

### Redis配置

在 `.env` 文件中配置：

```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### 缓存过期时间

默认24小时，可通过 `IdempotencyWithTTL` 自定义：

```go
// 自定义1小时过期
router.POST("/custom", middleware.IdempotencyWithTTL(time.Hour), handler)
```

## 注意事项

1. **Key的唯一性**：不同操作必须使用不同的幂等性Key
2. **Key的复用**：同一操作的重试必须使用相同的Key
3. **Key的格式**：推荐使用UUID v4格式
4. **缓存可用性**：Redis不可用时，幂等性检查会被跳过，请求正常处理
5. **响应缓存**：仅缓存成功（HTTP状态码<400）的响应
6. **失败重试**：请求失败时会清除缓存状态，允许重试

## 监控指标

日志中会记录以下信息：

- `[Idempotency] 请求缺少幂等性Key` - 警告级别
- `[Idempotency] 命中缓存，返回已缓存的响应` - 信息级别
- `[Idempotency] 响应已缓存` - 信息级别
- `[Idempotency] 获取分布式锁失败` - 错误级别

## 测试

运行单元测试：

```bash
go test -v ./internal/middleware/... -run TestIdempotency
```

## 参考资料

- [Stripe Idempotent Requests](https://stripe.com/docs/api/idempotent_requests)
- [RFC 7231 - HTTP/1.1 Semantics and Content](https://tools.ietf.org/html/rfc7231)
