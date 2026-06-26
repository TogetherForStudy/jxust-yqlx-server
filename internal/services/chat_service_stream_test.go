package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestStreamEventProcessorEmitsStreamingToolResult(t *testing.T) {
	t.Parallel()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	outputChan := make(chan string, 10)
	errChan := make(chan error, 1)

	processor := &streamEventProcessor{
		ctx:            context.Background(),
		conv:           &models.Conversation{},
		outputChan:     outputChan,
		errChan:        errChan,
		mcpClients:     mcpClient{},
		startEventType: "start",
	}

	go processor.process(iter)
	gen.Send(adk.EventFromMessage(nil, schema.StreamReaderFromArray([]*schema.Message{
		schema.ToolMessage("tool ", "call_1", schema.WithToolName("search")),
		schema.ToolMessage("result", "call_1", schema.WithToolName("search")),
	}), schema.Tool, "search"))
	gen.Close()

	var events []map[string]any
	for raw := range outputChan {
		payload := strings.TrimSpace(strings.TrimPrefix(raw, "data: "))
		if payload == "" {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			t.Fatalf("unmarshal event %q: %v", payload, err)
		}
		events = append(events, event)
	}

	for err := range errChan {
		if err != nil {
			t.Fatalf("unexpected processor error: %v", err)
		}
	}

	var foundToolResult bool
	for _, event := range events {
		if event["type"] != "tool_result" {
			continue
		}
		foundToolResult = true
		if event["tool_call_id"] != "call_1" {
			t.Fatalf("unexpected tool_call_id: %v", event["tool_call_id"])
		}
		if event["tool_name"] != "search" {
			t.Fatalf("unexpected tool_name: %v", event["tool_name"])
		}
		if event["content"] != "tool result" {
			t.Fatalf("unexpected content: %v", event["content"])
		}
	}

	if !foundToolResult {
		t.Fatalf("expected tool_result event, got %#v", events)
	}
}

type failingAgentTool struct{}

func (f failingAgentTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "failingTool"}, nil
}

func (f failingAgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error) {
	return "", errors.New("backend unavailable")
}

func TestWrapAgentToolsWithFailureResultsConvertsErrorToResult(t *testing.T) {
	t.Parallel()

	tools := wrapAgentToolsWithFailureResults(context.Background(), []einotool.BaseTool{failingAgentTool{}}, "test", 1, 2)
	if len(tools) != 1 {
		t.Fatalf("expected one wrapped tool, got %d", len(tools))
	}

	invokable, ok := tools[0].(einotool.InvokableTool)
	if !ok {
		t.Fatalf("wrapped tool is not invokable")
	}

	result, err := invokable.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("expected tool failure to be converted to result, got error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("unmarshal failure result %q: %v", result, err)
	}
	if payload["success"] != false {
		t.Fatalf("expected success=false, got %#v", payload["success"])
	}
	if payload["tool_name"] != "failingTool" {
		t.Fatalf("unexpected tool_name: %#v", payload["tool_name"])
	}
	if payload["error"] != "backend unavailable" {
		t.Fatalf("unexpected error: %#v", payload["error"])
	}
}
