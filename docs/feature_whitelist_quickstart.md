# 功能白名单系统快速开始指南

本指南帮助你在 5 分钟内快速上手功能白名单系统。

---

## 前置条件

- 已部署运行的 GoJxust API 服务
- 管理员账号的 JWT Token
- Redis 已配置并运行（用于缓存）

---

## 步骤 1: 创建功能定义

使用管理员账号创建一个新功能：

```bash
# 替换 {admin_token} 为你的管理员 token
curl -X POST http://localhost:8080/api/v0/admin/features \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "feature_key": "beta_my_feature",
    "feature_name": "我的测试功能",
    "description": "这是一个测试功能",
    "is_enabled": true
  }'
```

**成功响应示例**：
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "Result": {
    "id": 1,
    "feature_key": "beta_my_feature",
    "feature_name": "我的测试功能",
    "is_enabled": true
  }
}
```

---

## 步骤 2: 添加测试用户到白名单

将用户 ID 为 1 的用户添加到白名单：

```bash
curl -X POST http://localhost:8080/api/v0/admin/features/beta_my_feature/whitelist \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "user_id": 1
  }'
```

**提示**：不提供 `expires_at` 字段表示永久有效。

---

## 步骤 3: 验证用户权限

使用普通用户的 token 查询自己的功能列表：

```bash
# 替换 {user_token} 为用户的 token
curl -X GET http://localhost:8080/api/v0/user/features \
  -H "Authorization: Bearer {user_token}"
```

**成功响应示例**：
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "Result": {
    "features": ["beta_my_feature"]
  }
}
```

---

## 步骤 4: 在代码中使用中间件保护路由

编辑 `internal/router/router.go`：

```go
// 初始化 FeatureService
featureService := services.NewFeatureService(db)

// 创建受保护的路由组
betaFeature := authorized.Group("/my-feature")
betaFeature.Use(middleware.RequireFeature(featureService, "beta_my_feature"))
{
    betaFeature.GET("/data", myFeatureHandler.GetData)
    betaFeature.POST("/action", myFeatureHandler.DoAction)
}
```

---

## 步骤 5: 测试受保护的 API

### 有权限的用户访问（成功）

```bash
curl -X GET http://localhost:8080/api/v0/my-feature/data \
  -H "Authorization: Bearer {authorized_user_token}"
```

**响应**: 200 OK

### 无权限的用户访问（失败）

```bash
curl -X GET http://localhost:8080/api/v0/my-feature/data \
  -H "Authorization: Bearer {unauthorized_user_token}"
```

**响应**: 
```json
{
  "StatusCode": 403,
  "StatusMessage": "无权访问此功能"
}
```

---

## 步骤 6: 前端集成（可选）

在小程序中获取用户权限：

```javascript
// 获取用户功能列表
async function checkFeatureAccess() {
  const token = wx.getStorageSync('token');
  const res = await wx.request({
    url: 'http://localhost:8080/api/v0/user/features',
    header: { 'Authorization': `Bearer ${token}` }
  });
  
  const features = res.data.Result.features;
  
  // 检查是否有权限
  if (features.includes('beta_my_feature')) {
    console.log('用户有权限访问该功能');
    return true;
  } else {
    console.log('用户无权限');
    return false;
  }
}
```

---

## 常用管理命令

### 查看所有功能

```bash
curl -X GET http://localhost:8080/api/v0/admin/features \
  -H "Authorization: Bearer {admin_token}"
```

### 查看某功能的白名单用户

```bash
curl -X GET "http://localhost:8080/api/v0/admin/features/beta_my_feature/whitelist?page=1&page_size=20" \
  -H "Authorization: Bearer {admin_token}"
```

### 批量添加用户

```bash
curl -X POST http://localhost:8080/api/v0/admin/features/beta_my_feature/whitelist/batch \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $(uuidgen)" \
  -d '{
    "user_ids": [1, 2, 3, 4, 5]
  }'
```

### 撤销用户权限

```bash
curl -X DELETE http://localhost:8080/api/v0/admin/features/beta_my_feature/whitelist/1 \
  -H "Authorization: Bearer {admin_token}"
```

### 禁用功能

```bash
curl -X PUT http://localhost:8080/api/v0/admin/features/beta_my_feature \
  -H "Authorization: Bearer {admin_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "is_enabled": false
  }'
```

### 删除功能（软删除）

```bash
curl -X DELETE http://localhost:8080/api/v0/admin/features/beta_my_feature \
  -H "Authorization: Bearer {admin_token}"
```

---

## 故障排查

### 问题 1: 用户被授权后仍无权限

**原因**: 缓存未过期  
**解决**: 
1. 等待 5 分钟缓存自动过期
2. 或手动清除 Redis 缓存：`DEL user_features:{user_id}`
3. 或让用户重新登录

### 问题 2: 中间件返回 500 错误

**原因**: FeatureService 未正确初始化  
**解决**: 确保在 `router.go` 中正确创建了 `featureService` 实例

### 问题 3: 创建功能时提示"功能标识已存在"

**原因**: feature_key 重复  
**解决**: 使用不同的 feature_key

---

## 下一步

- 📖 阅读 [完整 API 文档](./feature_whitelist.md)
- 💡 查看 [使用示例](./feature_whitelist_example.md)
- 🔍 了解 [设计文档](./design/middleware_design.md)

---

## 需要帮助？

如有问题，请查看：
1. [常见问题](./feature_whitelist.md#常见问题)
2. [完整文档](./feature_whitelist.md)
3. 提交 Issue 到 GitHub 仓库
