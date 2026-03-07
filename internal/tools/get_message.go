package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andr1an/marc-mcp/internal/marc"
)

type GetMessageTool struct {
	client *marc.Client
}

type GetMessageInput struct {
	List      string `json:"list"`
	MessageID string `json:"message_id"`
}

func NewGetMessageTool(client *marc.Client) Tool {
	return &GetMessageTool{client: client}
}

func (t *GetMessageTool) Name() string {
	return "get_message"
}

func (t *GetMessageTool) Description() string {
	return "Get the full content of a specific message"
}

func (t *GetMessageTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"list": map[string]any{
				"type":        "string",
				"description": "Name of the mailing list",
			},
			"message_id": map[string]any{
				"type":        "string",
				"description": "Message ID from list_messages results",
			},
		},
		"required":             []string{"list", "message_id"},
		"additionalProperties": false,
	}
}

func (t *GetMessageTool) Invoke(ctx context.Context, input []byte) (any, error) {
	_ = ctx

	var req GetMessageInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON body: %v", ErrInvalidArgument, err)
	}
	if req.List == "" {
		return nil, fmt.Errorf("%w: list is required", ErrInvalidArgument)
	}
	if req.MessageID == "" {
		return nil, fmt.Errorf("%w: message_id is required", ErrInvalidArgument)
	}

	message, err := t.client.GetMessage(req.List, req.MessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}
