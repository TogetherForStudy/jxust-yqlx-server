/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func initRAGFlowMCP(ctx context.Context) (*client.Client, error) {
	// TODO: 根据 eino 的 MCP 集成文档实现
	// 参考: https://www.cloudwego.io/docs/eino/ecosystem_integration/tool/tool_mcp/
	// todo: ragflow sse endpoint 还需要 session_id 参数
	apiKey := os.Getenv("RAGFLOW_API_KEY")
	mcpClient, err := client.NewSSEMCPClient(os.Getenv("RAGFLOW_MCP_URL"), //fmt.Sprintf("%s/messages/?session_id=%s", s.cfg.LLM.RAGFlowMCPURL, sessionId),
		transport.WithHeaders(
			map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", apiKey), // RAGFlow API key with Bearer format
			}),
		//transport.WithHTTPTimeout(30*time.Second),
		transport.WithSSELogger(logger.L()),
	)
	if err != nil {
		logger.Errorf("RequestID[%s]:Failed to initialize RAG flow MCP client: %v ", utils.GetRequestID(ctx), err)
		return nil, fmt.Errorf("failed to create ragflow mcp client: %w", err)
	}
	err = mcpClient.Start(ctx)
	if err != nil {
		logger.Errorf("RequestID[%s]:Failed to start RAG flow MCP client: %v ", utils.GetRequestID(ctx), err)
		return nil, err
	}
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{})
	if err != nil {
		logger.Errorf("RequestID[%s]:Failed to initialize RAG flow MCP client connection: %v ", utils.GetRequestID(ctx), err)
		return nil, err
	}
	return mcpClient, nil
}

func main() {
	chatModel, err := einoopenai.NewChatModel(context.Background(), &einoopenai.ChatModelConfig{
		Model:   os.Getenv("OPENAI_MODEL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	if err != nil {
		log.Fatalf("failed to create chat model: %v", err)
	}
	ctx := context.Background()

	flowMCP, err := initRAGFlowMCP(ctx)
	if err != nil {
		log.Fatalf("failed to initialize RAG flow MCP: %v", err)
	}

	var allTools []einotool.BaseTool

	tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: flowMCP})
	if err != nil {
		log.Fatalf("failed to get tools: %v", err)
	}
	allTools = append(allTools, tools...)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "StudyAssistant",
		Description: "A helpful assistant for students.",
		Instruction: constant.ChatSystemPrompt,
		Model:       chatModel, // 底层大模型（DeepSeek/OpenAI）
		// 挂载工具箱
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		// 关键：设置最大迭代轮数，防止 Agent 陷入死循环
		MaxIterations: 15,
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}
	iter := agent.Run(ctx, &adk.AgentInput{
		Messages:        []adk.Message{schema.UserMessage("江理一起来学是个什么组织？")},
		EnableStreaming: true,
	})
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}
		st := event.Output.MessageOutput.MessageStream
		log.Printf("Event Message Role: %s\n", event.Output.MessageOutput.Role)
		if event.Output.MessageOutput.IsStreaming {
			for {
				recv, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%s", recv.Content)
			}
		} else {
			// 工具调用结果等是非流式内容
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(msg.Content)
		}

		log.Println("======")
	}

	// 以下代码是非流式调用的示例
	// 	runner := adk.NewRunner(ctx, adk.RunnerConfig{
	//		Agent: agent,
	//	})
	//iter := runner.Query(ctx, "江理一起来学是个什么组织？")
	//for {
	//	event, ok := iter.Next()
	//	if !ok {
	//		break
	//	}
	//	if event.Err != nil {
	//		log.Fatal(event.Err)
	//	}
	//	msg, err := event.Output.MessageOutput.GetMessage()
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Printf("\nmessage:\n%v\n======", msg)
	//}
}
