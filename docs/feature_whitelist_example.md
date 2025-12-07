# 功能白名单系统使用示例

本文档提供功能白名单系统的完整使用示例。

## 使用场景

假设我们要开发一个"AI学习助手"功能，仅对少量测试用户开放。

---

## 1. 管理员创建功能

### API 调用

```bash
curl -X POST http://localhost:8080/api/v0/admin/features \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "feature_key": "beta_ai_chat",
    "feature_name": "AI学习助手（测试版）",
    "description": "基于GPT的智能问答助手，帮助学生解答学习问题",
    "is_enabled": true
  }'
```

### 响应

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxx",
  "Result": {
    "id": 1,
    "feature_key": "beta_ai_chat",
    "feature_name": "AI学习助手（测试版）",
    "description": "基于GPT的智能问答助手，帮助学生解答学习问题",
    "is_enabled": true,
    "created_at": "2025-12-07T10:00:00Z",
    "updated_at": "2025-12-07T10:00:00Z"
  }
}
```

---

## 2. 添加测试用户到白名单

### 单个用户授权（7天后过期）

```bash
curl -X POST http://localhost:8080/api/v0/admin/features/beta_ai_chat/whitelist \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "user_id": 10001,
    "expires_at": "2025-12-14T10:00:00Z"
  }'
```

### 批量授权（永久有效）

```bash
curl -X POST http://localhost:8080/api/v0/admin/features/beta_ai_chat/whitelist/batch \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "user_ids": [10001, 10002, 10003]
  }'
```

---

## 3. 查看白名单用户列表

```bash
curl -X GET "http://localhost:8080/api/v0/admin/features/beta_ai_chat/whitelist?page=1&page_size=20" \
  -H "Authorization: Bearer {admin_token}"
```

### 响应

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

## 4. 后端实现受保护的 API

### 路由配置

```go
// internal/router/router.go

// 初始化 FeatureService
featureService := services.NewFeatureService(db)

// 在认证路由组下创建受保护的功能路由
betaAIChat := authorized.Group("/ai-chat")
betaAIChat.Use(middleware.RequireFeature(featureService, "beta_ai_chat"))
{
    betaAIChat.POST("/message", aiChatHandler.SendMessage)
    betaAIChat.GET("/history", aiChatHandler.GetHistory)
    betaAIChat.DELETE("/history", aiChatHandler.ClearHistory)
}
```

### Handler 实现

```go
// internal/handlers/ai_chat_handler.go

package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/TogetherForStudy/jxust-yqlx-server/internal/handlers/helper"
)

type AIChatHandler struct {
    // ... service
}

func (h *AIChatHandler) SendMessage(c *gin.Context) {
    // 能执行到这里，说明用户已通过 RequireFeature 中间件验证
    userID := c.GetUint("user_id")
    
    var req struct {
        Message string `json:"message" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        helper.ValidateResponse(c, "请求参数错误")
        return
    }
    
    // 调用 AI 服务处理消息
    response, err := h.aiService.Chat(userID, req.Message)
    if err != nil {
        helper.ErrorResponse(c, http.StatusInternalServerError, "AI服务暂时不可用")
        return
    }
    
    helper.SuccessResponse(c, gin.H{
        "reply": response,
    })
}
```

---

## 5. 前端集成

### 用户登录后获取权限

```javascript
// utils/auth.js

export async function fetchUserFeatures() {
  try {
    const token = wx.getStorageSync('token');
    const res = await wx.request({
      url: `${API_BASE_URL}/api/v0/user/features`,
      header: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    if (res.data.StatusCode === 0) {
      const features = res.data.Result.features;
      // 缓存到本地
      wx.setStorageSync('user_features', features);
      return features;
    }
    return [];
  } catch (error) {
    console.error('获取用户功能列表失败:', error);
    return [];
  }
}
```

### 在页面中检查权限

```javascript
// pages/index/index.js

Page({
  data: {
    showAIChat: false,
  },
  
  async onLoad() {
    // 获取用户功能列表
    const features = await fetchUserFeatures();
    
    // 检查是否有 AI 对话权限
    this.setData({
      showAIChat: features.includes('beta_ai_chat')
    });
  },
  
  // 跳转到 AI 对话页面
  goToAIChat() {
    if (!this.data.showAIChat) {
      wx.showToast({
        title: '暂无访问权限',
        icon: 'none'
      });
      return;
    }
    
    wx.navigateTo({
      url: '/pages/ai-chat/ai-chat'
    });
  }
});
```

### 页面 WXML

```xml
<!-- pages/index/index.wxml -->

<view class="container">
  <!-- 其他功能入口 -->
  
  <!-- AI 对话入口（仅有权限的用户可见）-->
  <view wx:if="{{showAIChat}}" class="feature-item" bindtap="goToAIChat">
    <image src="/images/ai-chat-icon.png" />
    <text>AI学习助手 (测试版)</text>
  </view>
</view>
```

### 调用受保护的 API

```javascript
// pages/ai-chat/ai-chat.js

async function sendMessage(message) {
  try {
    const token = wx.getStorageSync('token');
    const res = await wx.request({
      url: `${API_BASE_URL}/api/v0/ai-chat/message`,
      method: 'POST',
      header: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      data: { message }
    });
    
    if (res.data.StatusCode === 0) {
      return res.data.Result.reply;
    } else if (res.statusCode === 403) {
      wx.showModal({
        title: '提示',
        content: '您没有权限使用此功能',
        showCancel: false
      });
      return null;
    } else {
      throw new Error(res.data.StatusMessage);
    }
  } catch (error) {
    console.error('发送消息失败:', error);
    wx.showToast({
      title: '发送失败，请重试',
      icon: 'none'
    });
    return null;
  }
}
```

---

## 6. 管理员查看用户权限

```bash
# 查看用户 ID 为 10001 的所有功能权限
curl -X GET http://localhost:8080/api/v0/admin/users/10001/features \
  -H "Authorization: Bearer {admin_token}"
```

### 响应

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
    },
    {
      "feature_key": "beta_study_plan",
      "feature_name": "智能学习计划",
      "granted_by": 1,
      "granted_at": "2025-12-06T15:00:00Z",
      "expires_at": null,
      "is_expired": false
    }
  ]
}
```

---

## 7. 测试完成后关闭功能

### 方式 1：全局禁用功能

```bash
curl -X PUT http://localhost:8080/api/v0/admin/features/beta_ai_chat \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "is_enabled": false
  }'
```

**效果**：所有用户（包括白名单用户）都无法访问该功能。

### 方式 2：撤销单个用户权限

```bash
curl -X DELETE http://localhost:8080/api/v0/admin/features/beta_ai_chat/whitelist/10001 \
  -H "Authorization: Bearer {admin_token}"
```

**效果**：只有该用户失去权限，其他白名单用户不受影响。

### 方式 3：软删除功能

```bash
curl -X DELETE http://localhost:8080/api/v0/admin/features/beta_ai_chat \
  -H "Authorization: Bearer {admin_token}"
```

**效果**：功能被软删除，所有相关权限失效，但数据库记录保留。

---

## 8. 权限验证流程

```
用户请求 /api/v0/ai-chat/message
       |
       v
JWT 认证中间件（验证 token）
       |
       v
RequireFeature 中间件（检查 beta_ai_chat 权限）
       |
       +-- 检查功能全局开关（从缓存或数据库）
       |
       +-- 检查用户白名单（从缓存或数据库）
       |
       +-- 检查是否过期
       |
       v
   有权限？
   /      \
 YES      NO
  |        |
  v        v
处理请求  返回 403
```

---

## 9. 常见问题

### Q1: 用户刚被授权，但前端仍显示无权限？

**A**: 由于缓存机制，权限变更最多需要 5 分钟生效。如需立即生效：
- 让用户重新登录
- 或清除 Redis 缓存：`DEL user_features:{user_id}`

### Q2: 如何设置永久权限？

**A**: 授权时不提供 `expires_at` 字段，或设置为 `null`。

### Q3: 如何批量导入测试用户？

**A**: 使用批量授权接口：

```bash
curl -X POST http://localhost:8080/api/v0/admin/features/beta_ai_chat/whitelist/batch \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [10001, 10002, 10003, 10004, 10005]
  }'
```

### Q4: 过期的权限记录会自动删除吗？

**A**: 不会自动删除，但会失效。过期记录保留在数据库中便于审计。

### Q5: 前端如何优雅地处理无权限情况？

**A**: 
1. 登录后立即获取功能列表，隐藏无权限的功能入口
2. API 调用返回 403 时，提示用户并引导到其他功能

---

## 10. 性能优化建议

### 缓存预热

用户登录时，主动调用 `/user/features` 预加载权限列表：

```javascript
// 登录成功后
wx.login({
  success: async (res) => {
    const loginResult = await loginAPI(res.code);
    // 保存 token
    wx.setStorageSync('token', loginResult.token);
    // 立即获取功能列表（缓存）
    await fetchUserFeatures();
  }
});
```

### 减少重复查询

前端将功能列表缓存到本地存储，避免每次进入页面都查询：

```javascript
export async function getUserFeatures() {
  // 先从本地缓存读取
  let features = wx.getStorageSync('user_features');
  
  // 如果缓存为空或过期（例如每小时刷新一次）
  const lastUpdate = wx.getStorageSync('user_features_updated_at');
  const now = Date.now();
  
  if (!features || !lastUpdate || (now - lastUpdate > 3600000)) {
    features = await fetchUserFeatures();
    wx.setStorageSync('user_features_updated_at', now);
  }
  
  return features;
}
```

---

## 相关文档

- [功能白名单 API 文档](../docs/feature_whitelist.md)
- [中间件设计](../docs/design/middleware_design.md)
- [缓存设计](../docs/design/cache_design.md)
