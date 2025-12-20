# 聊天 API 手动测试文档（curl）

> !Warning
> 该文档适用于开发和测试环境。生产环境请使用正式认证流程获取 Token。

本文档提供使用 curl 命令手动测试聊天相关 API 的完整示例。

## 前置条件

1. 确保服务器在非 release 模式下运行（以启用 mock-login 端点）
2. 服务器默认运行在 `http://localhost:8085`
3. 已配置好环境变量（`.env` 文件）

## 环境变量设置

```bash
# 设置基础 URL（Windows PowerShell）
$BASE_URL = "http://localhost:8085"

# Linux/Mac
export BASE_URL="http://localhost:8085"
```

## 步骤 1: 获取测试 Token

使用 mock-login 端点获取测试用的 JWT token。

### 请求示例

```bash
# Windows PowerShell
curl -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" `
  -H "Content-Type: application/json" `
  -d '{\"user_id\": \"test-user-123\", \"nickname\": \"测试用户\"}'

# Linux/Mac
curl -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user-123", "nickname": "测试用户"}'
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "nickname": "测试用户",
      "avatar": "",
      "role": 1
    }
  }
}
```

**保存 token 到变量：**

```bash
# Windows PowerShell
$TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Linux/Mac
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## 步骤 2: 创建对话

### 请求示例

```bash
# Windows PowerShell
curl -X POST "$BASE_URL/api/v0/chat/conversations" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d '{\"title\": \"学习 Go 语言\"}'

# Linux/Mac
curl -X POST "$BASE_URL/api/v0/chat/conversations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "学习 Go 语言"}'
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-234567",
  "Result": {
    "id": 1,
    "title": "学习 Go 语言",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z",
    "last_message_at": null
  }
}
```

**保存对话 ID：**

```bash
# Windows PowerShell
$CONV_ID = 1

# Linux/Mac
export CONV_ID=1
```

## 步骤 3: 发送消息（流式对话）

### 请求示例

```bash
# Windows PowerShell
curl -X POST "$BASE_URL/api/v0/chat/conversation" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"什么是 Go 语言的 goroutine？\"}}"

# Linux/Mac
curl -X POST "$BASE_URL/api/v0/chat/conversation" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"什么是 Go 语言的 goroutine？\"}}"
```

### 响应示例（SSE 流）

**注意：** 此端点返回 Server-Sent Events (SSE) 流，不是标准 JSON 格式。

```
data: {"type":"start"}

data: {"type":"content","content":"Goroutine 是"}

data: {"type":"content","content":" Go 语言中的"}

data: {"type":"content","content":"轻量级线程"}

data: {"type":"content","content":"，可以"}

data: {"type":"content","content":"在程序中并发执行..."}

data: {"type":"end","message_count":2,"usage":{"prompt_tokens":50,"completion_tokens":100,"total_tokens":150}}
```

**事件类型说明：**

- `start`: 对话开始
- `content`: 内容增量（流式输出）
- `reasoning`: 模型推理过程（可选）
- `tool_call`: 工具调用事件
- `end`: 对话结束，包含消息计数和 token 使用统计

## 步骤 4: 获取对话历史消息

### 请求示例

```bash
# Windows PowerShell
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID" `
  -H "Authorization: Bearer $TOKEN"

# Linux/Mac
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-345678",
  "Result": [
    {
      "role": "user",
      "content": "什么是 Go 语言的 goroutine？"
    },
    {
      "role": "assistant",
      "content": "Goroutine 是 Go 语言中的轻量级线程，可以在程序中并发执行..."
    }
  ]
}
```

**注意：** 此接口只返回 `user` 和 `assistant` 角色的消息，不返回 `system` 和 `tool` 角色的消息。

## 步骤 5: 列出所有对话

### 请求示例（带分页）

```bash
# Windows PowerShell
curl -X GET "$BASE_URL/api/v0/chat/conversations?page=1&page_size=20" `
  -H "Authorization: Bearer $TOKEN"

# Linux/Mac
curl -X GET "$BASE_URL/api/v0/chat/conversations?page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN"
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-456789",
  "Result": {
    "total": 5,
    "page": 1,
    "page_size": 20,
    "conversations": [
      {
        "id": 1,
        "title": "学习 Go 语言",
        "created_at": "2024-01-01T10:00:00Z",
        "updated_at": "2024-01-01T10:05:00Z",
        "last_message_at": "2024-01-01T10:05:00Z"
      },
      {
        "id": 2,
        "title": "数据库设计",
        "created_at": "2024-01-01T11:00:00Z",
        "updated_at": "2024-01-01T11:10:00Z",
        "last_message_at": "2024-01-01T11:10:00Z"
      }
    ]
  }
}
```

## 步骤 6: 更新对话标题

### 请求示例

```bash
# Windows PowerShell
curl -X PUT "$BASE_URL/api/v0/chat/conversations/$CONV_ID" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d '{\"title\": \"深入学习 Go 语言并发\"}'

# Linux/Mac
curl -X PUT "$BASE_URL/api/v0/chat/conversations/$CONV_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "深入学习 Go 语言并发"}'
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-567890",
  "Result": "ok"
}
```

## 步骤 7: 导出对话

### 请求示例

```bash
# Windows PowerShell
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" `
  -H "Authorization: Bearer $TOKEN"

# Linux/Mac
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" \
  -H "Authorization: Bearer $TOKEN"
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-678901",
  "Result": {
    "conversation": {
      "id": 1,
      "title": "深入学习 Go 语言并发",
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T10:15:00Z",
      "last_message_at": "2024-01-01T10:05:00Z"
    },
    "messages": [
      {
        "role": "user",
        "content": "什么是 Go 语言的 goroutine？"
      },
      {
        "role": "assistant",
        "content": "Goroutine 是 Go 语言中的轻量级线程..."
      }
    ]
  }
}
```

## 步骤 8: 继续对话（发送第二条消息）

### 请求示例

```bash
# Windows PowerShell
curl -X POST "$BASE_URL/api/v0/chat/conversation" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"goroutine 和 channel 如何配合使用？\"}}"

# Linux/Mac
curl -X POST "$BASE_URL/api/v0/chat/conversation" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"goroutine 和 channel 如何配合使用？\"}}"
```

### 响应示例（SSE 流）

```
data: {"type":"start"}

data: {"type":"content","content":"Goroutine 和 channel"}

data: {"type":"content","content":" 的配合使用是 Go 并发编程的核心..."}

data: {"type":"end","message_count":4,"usage":{"prompt_tokens":150,"completion_tokens":200,"total_tokens":350}}
```

**说明：** 
- 后端会自动加载之前的对话历史
- `message_count` 为 4 表示现在对话中有 4 条消息（2 条用户消息 + 2 条助手回复）

## 步骤 9: 删除对话

### 请求示例

```bash
# Windows PowerShell
curl -X DELETE "$BASE_URL/api/v0/chat/conversations/$CONV_ID" `
  -H "Authorization: Bearer $TOKEN"

# Linux/Mac
curl -X DELETE "$BASE_URL/api/v0/chat/conversations/$CONV_ID" \
  -H "Authorization: Bearer $TOKEN"
```

### 响应示例

```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-789012",
  "Result": "ok"
}
```

## 完整测试脚本示例

### Windows PowerShell

```powershell
# 设置变量
$BASE_URL = "http://localhost:8085"

# 1. 获取 Token
$response = curl -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" `
  -H "Content-Type: application/json" `
  -d '{\"user_id\": \"test-user-123\", \"nickname\": \"测试用户\"}' | ConvertFrom-Json

$TOKEN = $response.Result.token
Write-Host "Token: $TOKEN"

# 2. 创建对话
$response = curl -X POST "$BASE_URL/api/v0/chat/conversations" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d '{\"title\": \"测试对话\"}' | ConvertFrom-Json

$CONV_ID = $response.Result.id
Write-Host "对话 ID: $CONV_ID"

# 3. 发送消息
Write-Host "发送消息..."
curl -X POST "$BASE_URL/api/v0/chat/conversation" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"你好\"}}"

# 4. 等待 2 秒后获取历史消息
Start-Sleep -Seconds 2
Write-Host "`n获取历史消息..."
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID" `
  -H "Authorization: Bearer $TOKEN"

# 5. 列出所有对话
Write-Host "`n列出所有对话..."
curl -X GET "$BASE_URL/api/v0/chat/conversations" `
  -H "Authorization: Bearer $TOKEN"

# 6. 导出对话
Write-Host "`n导出对话..."
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" `
  -H "Authorization: Bearer $TOKEN"

# 7. 删除对话
Write-Host "`n删除对话..."
curl -X DELETE "$BASE_URL/api/v0/chat/conversations/$CONV_ID" `
  -H "Authorization: Bearer $TOKEN"
```

### Linux/Mac Bash

```bash
#!/bin/bash

# 设置变量
BASE_URL="http://localhost:8085"

# 1. 获取 Token
echo "获取 Token..."
response=$(curl -s -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user-123", "nickname": "测试用户"}')

TOKEN=$(echo $response | jq -r '.Result.token')
echo "Token: $TOKEN"

# 2. 创建对话
echo -e "\n创建对话..."
response=$(curl -s -X POST "$BASE_URL/api/v0/chat/conversations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "测试对话"}')

CONV_ID=$(echo $response | jq -r '.Result.id')
echo "对话 ID: $CONV_ID"

# 3. 发送消息
echo -e "\n发送消息..."
curl -X POST "$BASE_URL/api/v0/chat/conversation" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"你好\"}}"

# 4. 等待 2 秒后获取历史消息
sleep 2
echo -e "\n\n获取历史消息..."
curl -s -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID" \
  -H "Authorization: Bearer $TOKEN" | jq

# 5. 列出所有对话
echo -e "\n列出所有对话..."
curl -s -X GET "$BASE_URL/api/v0/chat/conversations" \
  -H "Authorization: Bearer $TOKEN" | jq

# 6. 导出对话
echo -e "\n导出对话..."
curl -s -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" \
  -H "Authorization: Bearer $TOKEN" | jq

# 7. 删除对话
echo -e "\n删除对话..."
curl -s -X DELETE "$BASE_URL/api/v0/chat/conversations/$CONV_ID" \
  -H "Authorization: Bearer $TOKEN" | jq
```

## 常见问题

### 1. 如何测试 SSE 流式输出？

curl 默认会缓冲输出，要实时查看流式响应，可以使用：

```bash
# Windows PowerShell
curl -N -X POST "$BASE_URL/api/v0/chat/conversation" `
  -H "Authorization: Bearer $TOKEN" `
  -H "Content-Type: application/json" `
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"测试流式输出\"}}"

# Linux/Mac
curl -N -X POST "$BASE_URL/api/v0/chat/conversation" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\": $CONV_ID, \"message\": {\"role\": \"user\", \"content\": \"测试流式输出\"}}"
```

`-N` 参数禁用输出缓冲。

### 2. 如何保存响应到文件？

```bash
# Windows PowerShell
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" `
  -H "Authorization: Bearer $TOKEN" `
  -o conversation_export.json

# Linux/Mac
curl -X GET "$BASE_URL/api/v0/chat/conversations/$CONV_ID/export" \
  -H "Authorization: Bearer $TOKEN" \
  -o conversation_export.json
```

### 3. Token 过期怎么办？

重新执行步骤 1 获取新的 token。JWT token 的有效期在服务器配置中设置。

### 4. 如何查看详细的请求/响应信息？

使用 `-v` 参数：

```bash
# Windows PowerShell
curl -v -X GET "$BASE_URL/api/v0/chat/conversations" `
  -H "Authorization: Bearer $TOKEN"

# Linux/Mac
curl -v -X GET "$BASE_URL/api/v0/chat/conversations" \
  -H "Authorization: Bearer $TOKEN"
```

### 5. 如何测试不同用户的对话？

使用不同的 `user_id` 创建不同的 mock login：

```bash
# 用户 A
curl -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "nickname": "用户A"}'

# 用户 B
curl -X POST "$BASE_URL/api/v0/auth/mock-wechat-login" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "nickname": "用户B"}'
```

## API 端点速查表

| 端点 | 方法 | 描述 | 需要认证 |
|------|------|------|----------|
| `/api/v0/auth/mock-wechat-login` | POST | 获取测试 Token | 否 |
| `/api/v0/chat/conversations` | POST | 创建对话 | 是 |
| `/api/v0/chat/conversations` | GET | 列出对话 | 是 |
| `/api/v0/chat/conversations/:id` | GET | 获取历史消息 | 是 |
| `/api/v0/chat/conversations/:id` | PUT | 更新对话标题 | 是 |
| `/api/v0/chat/conversations/:id` | DELETE | 删除对话 | 是 |
| `/api/v0/chat/conversations/:id/export` | GET | 导出对话 | 是 |
| `/api/v0/chat/conversation` | POST | 流式对话（SSE） | 是 |

## 注意事项

1. **认证**: 所有聊天相关 API 都需要在 Header 中携带 `Authorization: Bearer <token>`
2. **流式响应**: `/api/v0/chat/conversation` 端点返回 SSE 流，不是标准 JSON
3. **消息过滤**: 获取历史消息时，只返回 `user` 和 `assistant` 角色的消息
4. **上下文管理**: 后端自动处理对话上下文，前端只需发送单条消息
5. **幂等性**: 创建对话接口使用了幂等性中间件，相同的请求 ID 不会重复创建

## 参考文档

- [聊天 API 完整文档](./chat-api.md)
- [项目 README](../README.md)
