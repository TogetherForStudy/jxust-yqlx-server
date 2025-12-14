# 聊天对话功能 API 文档

## 概述

本模块实现了基于 LLM 的聊天对话功能，旨在帮助用户高效学习。支持与 MCP (Model Context Protocol) 集成，可连接 RAGFlow 知识库和内置功能。

## 配置

在 `.env` 文件中添加以下配置：

```bash
# RAGFlow MCP 配置
RAGFLOW_MCP_URL=http://your-ragflow-mcp-url
RAGFLOW_API_KEY=your-ragflow-api-key

# LLM 配置
LLM_MODEL=gpt-4
LLM_API_KEY=your-openai-api-key
LLM_BASE_URL=https://api.openai.com/v1
```

## API 端点

所有端点都在 `/api/v0/chat` 路由组下，需要用户认证。

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
  "id": 1,
  "title": "学习 Go 语言",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "last_message_at": null
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
```

### 3. 选择对话（获取历史消息）

**端点:** `GET /api/v0/chat/conversations/:id`

**响应:**
```json
{
  "messages": [
    {
      "id": 1,
      "conversation_id": 1,
      "role": "user",
      "content": "什么是 Go 语言的 goroutine？",
      "tool_calls": null,
      "token_count": 15,
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": 2,
      "conversation_id": 1,
      "role": "assistant",
      "content": "Goroutine 是 Go 语言中的轻量级线程...",
      "tool_calls": null,
      "token_count": 120,
      "created_at": "2024-01-01T00:00:05Z"
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
  "message": "Conversation updated successfully"
}
```

### 5. 删除对话

**端点:** `DELETE /api/v0/chat/conversations/:id`

**响应:**
```json
{
  "message": "Conversation deleted successfully"
}
```

### 6. 导出对话

**端点:** `GET /api/v0/chat/conversations/:id/export`

**响应:**
```json
{
  "conversation": {
    "id": 1,
    "title": "学习 Go 语言",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "last_message_at": "2024-01-01T01:00:00Z"
  },
  "messages": [...]
}
```

### 7. 流式对话（SSE）

**端点:** `POST /api/v0/chat/conversation`

**请求体:**
```json
{
  "conversation_id": 1,
  "messages": [
    {
      "role": "user",
      "content": "什么是 Go 语言的 channel？"
    }
  ]
}
```

**响应:** Server-Sent Events 流

```
data: {"type":"start"}

data: {"type":"content","content":"Channel 是"}

data: {"type":"content","content":" Go 语言中用于"}

data: {"type":"content","content":"goroutine 之间通信的管道..."}

data: {"type":"end"}
```

## 数据模型

### Conversation (对话)
- `id`: 对话 ID
- `user_id`: 用户 ID
- `title`: 对话标题
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `last_message_at`: 最后消息时间

### Message (消息)
- `id`: 消息 ID
- `conversation_id`: 所属对话 ID
- `role`: 角色 (user/assistant/tool)
- `content`: 消息内容
- `tool_calls`: 工具调用记录 (JSON)
- `token_count`: Token 数量
- `created_at`: 创建时间

## 使用示例

### 创建并使用对话

```bash
# 1. 创建对话
curl -X POST http://localhost:8085/api/v0/chat/conversations \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "学习计划"}'

# 2. 发送消息（流式）
curl -X POST http://localhost:8085/api/v0/chat/conversation \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": 1,
    "messages": [
      {"role": "user", "content": "帮我制定一个学习 Go 语言的计划"}
    ]
  }'

# 3. 获取历史消息
curl http://localhost:8085/api/v0/chat/conversations/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. 列出所有对话
curl http://localhost:8085/api/v0/chat/conversations \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## TODO

当前实现是 MVP 版本，以下功能待完善：

1. **LLM 集成**
   - 集成真实的 LLM 提供商 (OpenAI, Claude 等)
   - 实现真实的流式响应
   - Token 计数和成本追踪

2. **MCP 工具集成**
   - 集成 RAGFlow MCP 客户端
   - 实现工具调用记录和日志
   - 添加 gojxust 内置 MCP 工具

3. **高级功能**
   - 自动生成对话标题
   - 上下文窗口管理和消息压缩
   - 多轮对话的上下文保持
   - 工具调用可视化

4. **优化**
   - 消息分页加载
   - 流式响应的错误处理和重试
   - 并发请求的速率限制

## 开发指南

### 添加新的 MCP 工具

1. 在 `ChatService.InitializeLLM()` 中注册工具
2. 实现工具调用的处理逻辑
3. 在消息中记录工具调用结果

### 切换 LLM 提供商

修改 `ChatService` 的 `llm` 字段初始化，使用 eino 提供的不同模型适配器。

## 注意事项

- 所有 API 需要用户认证（JWT Token）
- 对话和消息会自动关联到当前用户
- 消息支持存储工具调用过程（JSON 格式）
- SSE 流式响应需要客户端正确处理 EventSource

## 参考文档

- [eino 文档](https://www.cloudwego.io/docs/eino/)
- [eino MCP 集成](https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/)
