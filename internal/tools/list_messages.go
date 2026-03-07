package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andr1an/marc-mcp/internal/marc"
)

type ListMessagesTool struct {
	client *marc.Client
}

type ListMessagesInput struct {
	List  string `json:"list"`
	Month string `json:"month,omitempty"`
	Page  int    `json:"page,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func NewListMessagesTool(client *marc.Client) Tool {
	return &ListMessagesTool{client: client}
}

func (t *ListMessagesTool) Name() string {
	return "list_messages"
}

func (t *ListMessagesTool) Description() string {
	return "List messages from a mailing list. Defaults to current month. Each page has ~30 messages."
}

func (t *ListMessagesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"list": map[string]any{
				"type":        "string",
				"description": "Name of the mailing list (e.g., 'git', 'linux-kernel')",
			},
			"month": map[string]any{
				"type":        "string",
				"description": "Month in YYYYMM format (e.g., '202602'). Defaults to current month.",
			},
			"page": map[string]any{
				"type":        "integer",
				"description": "Page number (1-based, default: 1). Each page has ~30 messages.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of messages to return from this page (default: all)",
			},
		},
		"required":             []string{"list"},
		"additionalProperties": false,
	}
}

func (t *ListMessagesTool) Invoke(ctx context.Context, input []byte) (any, error) {
	_ = ctx

	var req ListMessagesInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON body: %v", ErrInvalidArgument, err)
	}
	if req.List == "" {
		return nil, fmt.Errorf("%w: list is required", ErrInvalidArgument)
	}

	messages, err := t.client.ListMessagesWithOptions(marc.ListMessagesOptions{
		List:  req.List,
		Month: req.Month,
		Page:  req.Page,
		Limit: req.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return messages, nil
}
