# OpenAPI

项目接口规范文件位于 `docs/openapi/openapi.json`。

重新生成：

```bash
go run ./scripts/generate_openapi
```

或使用：

```bash
make openapi
```

说明：

- 文档基于当前 `router`、`handler`、DTO 与 service 返回结构生成。
- 统一 JSON 响应默认使用 `StatusCode`、`StatusMessage`、`RequestId`、`Result`。
- `/health`、聊天 SSE、文件流、MCP、MinIO 代理不是统一信封响应。
