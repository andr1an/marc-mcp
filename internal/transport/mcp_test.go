package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andr1an/marc-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type testTool struct {
	name        string
	description string
	schema      map[string]any
	result      any
	err         error
}

func (t *testTool) Name() string                { return t.name }
func (t *testTool) Description() string         { return t.description }
func (t *testTool) InputSchema() map[string]any { return t.schema }
func (t *testTool) Invoke(ctx context.Context, input []byte) (any, error) {
	_ = ctx
	_ = input
	return t.result, t.err
}

func TestMCPHandlerListTools(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&testTool{
		name:        "test_tool",
		description: "test",
		schema:      map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
	})

	h := NewMCPHandler(reg, "test")
	s := httptest.NewServer(h)
	defer s.Close()

	c, err := client.NewStreamableHttpClient(s.URL)
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("start client failed: %v", err)
	}

	_, err = c.Initialize(ctx, mcp.InitializeRequest{Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, ClientInfo: mcp.Implementation{Name: "test", Version: "1.0.0"}}})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	res, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	if len(res.Tools) != 1 || res.Tools[0].Name != "test_tool" {
		t.Fatalf("unexpected tools response: %#v", res.Tools)
	}
}

func TestToMCPToolFallbackSchema(t *testing.T) {
	tool := toMCPTool(tools.ToolInfo{
		Name:        "bad_schema_tool",
		Description: "bad schema",
		InputSchema: map[string]any{"type": "object", "bad": make(chan int)},
	})

	var schema map[string]any
	if err := json.Unmarshal(tool.RawInputSchema, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Fatalf("expected fallback schema type object, got %#v", schema["type"])
	}
}

func TestMCPHandlerMethodNotAllowed(t *testing.T) {
	h := NewMCPHandler(tools.NewRegistry(), "test")
	req := httptest.NewRequest(http.MethodPut, "/mcp", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
