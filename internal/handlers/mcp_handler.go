package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPHandler handles MCP protocol requests for LLM tool calling
type MCPHandler struct {
	server  *mcp.Server
	handler *mcp.StreamableHTTPHandler
}

// NewMCPHandler creates a new MCP handler with placeholder tools
func NewMCPHandler() *MCPHandler {
	// Create MCP server with GoJxust implementation info
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gojxust-mcp-server",
		Title:   "江理一起来学智能助理 MCP 服务，提供各种校园服务、学习类服务工具调用接口",
		Version: "0.1.0",
	}, &mcp.ServerOptions{HasTools: true})

	// TODO: Add GoJxust service tools here
	// Example:
	// mcp.AddTool(server, &mcp.Tool{
	//     Name:        "getCourseTable",
	//     Description: "Get course table for a student",
	// }, getCourseTableHandler)

	// Create streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	return &MCPHandler{
		server:  server,
		handler: handler,
	}
}

// Handle processes MCP requests via Gin context
func (h *MCPHandler) Handle(c *gin.Context) {
	h.handler.ServeHTTP(c.Writer, c.Request)
}
