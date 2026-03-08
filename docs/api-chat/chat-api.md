# 聊天对话功能 API 文档

## 概述

本模块实现了基于 LLM 的聊天对话功能，旨在帮助用户高效学习。支持与 MCP (Model Context Protocol) 集成，可连接 RAGFlow 知识库和内置功能。

- **统一消息存储**: 所有对话消息存储为 `[]*schema.Message` JSON 格式，支持完整的 eino 消息兼容性
- **智能缓存**: 使用 Redis 缓存消息，减少数据库查询，提高响应速度
- **流式处理**: 用户消息+历史消息自动合并，后端负责获取完整上下文并保存响应
- **简化 API**: 前端只需发送单个用户消息，后端自动处理上下文管理

## 配置

在 `.env` 文件中添加以下配置：

```bash
# RAGFlow MCP 配置
RAGFLOW_MCP_URL=http://your-ragflow-mcp-url/sse
RAGFLOW_API_KEY=your-ragflow-api-key

# LLM 配置
LLM_MODEL=gpt-4
LLM_API_KEY=your-openai-api-key
LLM_BASE_URL=https://api.openai.com/v1
```

## API 端点

所有端点都在 `/api/v0/chat` 路由组下，需要用户认证。

### 响应格式说明

所有 API 响应都遵循统一的包装格式：

```json
{
  "StatusCode": 0,                    // 0 表示成功，非0表示错误
  "StatusMessage": "Success",         // 状态描述信息
  "RequestId": "req-unique-id",       // 唯一请求ID，用于追踪
  "Result": {}                        // 实际数据，类型根据端点而异
}
```

**说明：**
- 对于成功请求，`StatusCode` 为 0，`Result` 包含实际数据
- 对于失败请求，`StatusCode` 为非 0 值，`Result` 为空，错误信息在 `StatusMessage` 中
- `RequestId` 对所有响应都存在，用于日志追踪和问题排查

### 1. 创建对话

**端点:** `POST /api/v0/chat/conversations`

**请求体:**
```json
{
  "title": "学习 Go 语言"
}
```

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": {
    "id": 1,
    "title": "学习 Go 语言",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "last_message_at": null
  }
}
```

### 2. 列出对话

**端点:** `GET /api/v0/chat/conversations`

**查询参数:**
- `page` (可选): 页码，默认 1
- `page_size` (可选): 每页数量，默认 20

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": {
    "total": 10,
    "page": 1,
    "page_size": 20,
    "conversations": [
      {
        "id": 1,
        "title": "学习 Go 语言",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "last_message_at": "2024-01-01T01:00:00Z"
      }
    ]
  }
}
```

### 3. 选择对话（获取历史消息）

**端点:** `GET /api/v0/chat/conversations/:id`

**说明:** 获取对话的所有历史消息（仅包含 User 和 Assistant 角色），使用缓存加速

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": [
    {
      "role": "user",
      "content": "什么是 Go 语言的 goroutine？"
    },
    {
      "role": "assistant",
      "content": "Goroutine 是 Go 语言中的轻量级线程...",
      "tool_calls": null
    }
  ]
}
```

### 4. 更新对话标题

**端点:** `PUT /api/v0/chat/conversations/:id`

**请求体:**
```json
{
  "title": "深入学习 Go 语言并发"
}
```

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": "ok"
}
```

### 5. 删除对话

**端点:** `DELETE /api/v0/chat/conversations/:id`

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": "ok"
}
```

### 6. 导出对话

**端点:** `GET /api/v0/chat/conversations/:id/export`

**说明:** 导出对话的完整信息，包括所有消息（仅 User 和 Assistant 角色）

**响应:**
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "req-123456",
  "Result": {
    "conversation": {
      "id": 1,
      "title": "学习 Go 语言",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "last_message_at": "2024-01-01T01:00:00Z"
    },
    "messages": [
      {
        "role": "user",
        "content": "什么是 Go 语言？"
      },
      {
        "role": "assistant",
        "content": "Go 语言是一门编译型的强类型编程语言..."
      }
    ]
  }
}
```

### 7. 流式对话（SSE）

**端点:** `POST /api/v0/chat/conversation`

**说明:** 
- 前端发送单个用户消息
- 后端自动加载历史消息，合并新消息，调用 LLM
- 流式返回响应，并自动保存完整对话到数据库和缓存
- **注意**: 此端点返回原始 SSE 流，不被包装在 `Response` 结构中

**请求体:**
```json
{
  "conversation_id": 1,
  "message": {
    "role": "user",
    "content": "什么是 Go 语言的 channel？"
  }
}
```

**响应:** Server-Sent Events 流（原始格式，未包装）

```
data: {"type":"start"}

data: {"type":"content","content":"Channel 是"}

data: {"type":"content","content":" Go 语言中用于"}

data: {"type":"content","content":"goroutine 之间通信的管道..."}

data: {"type":"tool_call","function":"search","arguments":{"query":"go channel"}}

data: {"type":"end","message_count":15,"usage":{"prompt_tokens":100,"completion_tokens":150,"total_tokens":250}}
```

**事件类型:**
- `start`: 对话开始
- `content`: 内容增量（可多次发送）
- `reasoning`: 模型推理过程（可选）
- `tool_call`: 工具调用事件
- `end`: 对话结束，包含统计信息

POST /api/v0/chat/conversation

```json
{
    "conversation_id": 123,
    "message": {
        "role": "user",
        "content": "Hello"
    }
}
```

Resume Interrupted Conversation:

POST /api/v0/chat/conversation

```json
{
    "conversation_id": 123,
    "checkpoint_id": "123:456:1234567890",
    "resume_input": "user's additional input for the tool"
}
```
## 数据模型

### 响应格式速查表

| 端点 | 方法 | 响应类型 | Result 数据结构 |
|------|------|--------|-----------------|
| `/conversations` | POST | JSON (包装) | `ConversationResponse` |
| `/conversations` | GET | JSON (包装) | `ConversationListResponse` |
| `/conversations/:id` | GET | JSON (包装) | `[]*schema.Message` |
| `/conversations/:id` | PUT | JSON (包装) | `string` ("ok") |
| `/conversations/:id` | DELETE | JSON (包装) | `string` ("ok") |
| `/conversations/:id/export` | GET | JSON (包装) | `ExportConversationResponse` |
| `/conversation` | POST | SSE 流 | N/A (原始流) |

**包装格式** = `Response { StatusCode, StatusMessage, RequestId, Result }`

### Conversation (对话)
- `id`: 对话 ID
- `user_id`: 用户 ID
- `title`: 对话标题
- `messages`: 完整的消息列表 JSON，格式为 `[]*schema.Message`
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `last_message_at`: 最后消息时间

### Message (schema.Message - JSON 格式)
```json
{
  "role": "user|assistant|tool|system",
  "content": "消息内容",
  "tool_calls": [
    {
      "type": "function",
      "id": "call_123",
      "function": {
        "name": "search",
        "arguments": "{\"query\": \"key\"}"
      }
    }
  ],
  "tool_call_id": "call_123",
  "reasoning_content": "模型的推理过程"
}
```

## 使用示例

### 创建并使用对话

```bash
# 1. 创建对话
curl -X POST http://localhost:8085/api/v0/chat/conversations \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "学习计划"}'

# 2. 发送消息（流式）- 后端自动处理上下文
curl -X POST http://localhost:8085/api/v0/chat/conversation \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": 1,
    "message": {
      "role": "user",
      "content": "帮我制定一个学习 Go 语言的计划"
    }
  }'

# 3. 获取历史消息
curl http://localhost:8085/api/v0/chat/conversations/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. 列出所有对话
curl http://localhost:8085/api/v0/chat/conversations \
  -H "Authorization: Bearer YOUR_TOKEN"

# 5. 导出对话
curl http://localhost:8085/api/v0/chat/conversations/1/export \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 工作流程说明

1. **创建对话**: 前端调用创建 API，获得 `conversation_id`
2. **发送消息**: 
   - 前端构建 `schema.Message` 对象（仅需 role 和 content）
   - 发送到后端，包含 `conversation_id`
3. **后端处理**:
   - 从缓存/数据库加载完整的 `[]*schema.Message`
   - 合并新消息（用户消息）到消息列表
   - 调用 LLM 进行对话
   - 流式返回 SSE 事件
   - 保存完整的消息列表到数据库和缓存
4. **前端渲染**: 
   - 接收 SSE 事件，实时显示内容
   - 加载对话时，获取历史消息并渲染（仅 User 和 Assistant 角色）

## TODO

当前实现是 MVP 版本，以下功能待完善：

1. **缓存优化**
   - 实现缓存失效策略（对话更新时清理缓存）
   - 支持缓存预热和过期时间动态调整
   - 添加缓存统计和监控

2. **LLM 集成**
   - 集成真实的 LLM 提供商 (OpenAI, Claude 等)
   - Token 计数和成本追踪
   - 流式响应的超时和重试机制

3. **MCP 工具集成**
   - 集成 RAGFlow MCP 客户端
   - 实现工具调用记录和日志
   - 添加 gojxust 内置 MCP 工具
   - 工具调用结果的自动集成

4. **高级功能**
   - 自动生成对话标题
   - 上下文窗口管理和消息压缩
   - 对话搜索和过滤
   - 工具调用可视化和结果展示

5. **优化**
   - 大消息列表的分页加载支持
   - 并发请求的速率限制
   - 数据库查询优化（索引、分区）
   - 消息存储压缩

## 开发指南

### 架构设计

**消息存储流程：**

```
Frontend                Backend                Database/Cache
   |                       |                          |
   |-- 发送单个消息 ------->|                          |
   |                       |                          |
   |                       |-- 加载历史消息 --------->|
   |                       |<-- 返回 []*Message ---|
   |                       |                          |
   |                       |-- 合并新消息到列表      |
   |                       |                          |
   |                       |-- 调用 LLM 流式处理   |
   |<-- 流式响应 SSE -------|                          |
   |                       |                          |
   |                       |-- 保存完整消息列表 ---->|
   |                       |<-- 确认保存 ----------|
```

### 关键实现细节

1. **GetMessages()**: 
   - 优先从 Redis 缓存读取 `conversation:messages:{id}`
   - 缓存未命中，从数据库 `Conversation.Messages` JSON 字段读取
   - 更新 Redis 缓存（30分钟过期）
   - 返回 `[]*schema.Message`

2. **StreamChat()**:
   - 接收单个用户消息 `*schema.Message`
   - 调用 `GetMessages()` 获取历史消息
   - 将新消息追加到列表
   - 传递给 LLM 进行推理
   - 流式返回 SSE 事件
   - 在后台 Goroutine 中保存完整消息列表

3. **SaveMessages()**:
   - 将 `[]*schema.Message` Marshal 为 JSON
   - 更新 `Conversation.Messages` 字段
   - 更新 `last_message_at` 和 `updated_at` 时间戳
   - 更新 Redis 缓存

### 添加新的 MCP 工具

1. 在 `ChatService.prepareUserMcpClient()` 中注册工具
2. 工具调用会自动记录到 `schema.Message.ToolCalls`
3. 工具结果包含在对话历史中

### 消息过滤

- `ChooseConversation()`: 返回前端时仅过滤 User 和 Assistant 角色
- 系统消息和工具调用消息存储在数据库，但不返回给前端
- 支持工具调用过程的完整记录和复现

## 注意事项

- 所有 API 需要用户认证（JWT Token）
- 对话和消息会自动关联到当前用户
- 消息列表以 JSON 格式存储，完全兼容 eino 的 `schema.Message`
- 系统消息、工具消息存储在数据库中保持完整上下文，但不返回给前端
- 缓存使用 Redis，Key 格式: `conversation:messages:{conversation_id}`
- SSE 流式响应需要客户端正确处理 EventSource API
- 并发对同一对话的更新是安全的（数据库级别的最后一个更新获胜）

## 性能考虑

| 操作 | 时间复杂度 | 备注 |
|------|----------|------|
| GetMessages (缓存命中) | O(1) | 网络 I/O |
| GetMessages (缓存未命中) | O(n) | n = 消息数量，包含 JSON 解析 |
| SaveMessages | O(n) | n = 消息数量，包含 JSON 序列化 |
| StreamChat | O(n) | 取决于 LLM 响应流速 |

**优化建议：**
- 对于大对话（>1000条消息），考虑实现消息分页存储
- 定期清理过期缓存，避免内存溢出
- 使用数据库连接池，配置合理的连接数

## 参考文档

- [eino 文档](https://www.cloudwego.io/docs/eino/)
- [eino MCP 集成](https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/)
- [Redis 缓存策略](https://redis.io/topics/lru-cache)
- [SSE 规范](https://html.spec.whatwg.org/multipage/server-sent-events.html)
