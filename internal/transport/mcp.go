package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/andr1an/marc-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPHandler struct {
	streamable http.Handler
}

func NewMCPHandler(registry *tools.Registry, version string) *MCPHandler {
	mcpServer := server.NewMCPServer(
		"marc-mcp",
		version,
		server.WithToolCapabilities(false),
	)

	for _, info := range registry.List() {
		toolName := info.Name
		mcpServer.AddTool(toMCPTool(info), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			input := map[string]any{}
			if args := req.GetArguments(); args != nil {
				input = args
			}

			rawInput, err := json.Marshal(input)
			if err != nil {
				return mcp.NewToolResultError("invalid tool arguments"), nil
			}

			result, err := registry.Invoke(ctx, toolName, rawInput)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return &mcp.CallToolResult{
				Content:           []mcp.Content{mcp.NewTextContent(toJSONString(result))},
				StructuredContent: result,
			}, nil
		})
	}

	return &MCPHandler{streamable: server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))}
}

func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.streamable.ServeHTTP(w, r)
}

func toMCPTool(info tools.ToolInfo) mcp.Tool {
	schemaBytes, err := json.Marshal(info.InputSchema)
	if err != nil {
		schemaBytes = []byte(`{"type":"object","properties":{},"required":[]}`)
	}

	return mcp.Tool{
		Name:           info.Name,
		Description:    info.Description,
		RawInputSchema: schemaBytes,
	}
}

func toJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
