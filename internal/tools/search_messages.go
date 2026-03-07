package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andr1an/marc-mcp/internal/marc"
)

type SearchMessagesTool struct {
	client *marc.Client
}

type SearchMessagesInput struct {
	List       string `json:"list"`
	Query      string `json:"query"`
	SearchType string `json:"search_type,omitempty"`
}

func NewSearchMessagesTool(client *marc.Client) Tool {
	return &SearchMessagesTool{client: client}
}

func (t *SearchMessagesTool) Name() string {
	return "search_messages"
}

func (t *SearchMessagesTool) Description() string {
	return "Search messages in a mailing list"
}

func (t *SearchMessagesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"list": map[string]any{
				"type":        "string",
				"description": "Name of the mailing list to search",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query string",
			},
			"search_type": map[string]any{
				"type":        "string",
				"description": "Type of search: 's' for subject (default), 'a' for author, 'b' for body",
				"enum":        []string{"s", "a", "b"},
			},
		},
		"required":             []string{"list", "query"},
		"additionalProperties": false,
	}
}

func (t *SearchMessagesTool) Invoke(ctx context.Context, input []byte) (any, error) {
	_ = ctx

	var req SearchMessagesInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON body: %v", ErrInvalidArgument, err)
	}
	if req.List == "" {
		return nil, fmt.Errorf("%w: list is required", ErrInvalidArgument)
	}
	if req.Query == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidArgument)
	}
	if req.SearchType == "" {
		req.SearchType = "s"
	}
	if req.SearchType != "s" && req.SearchType != "a" && req.SearchType != "b" {
		return nil, fmt.Errorf("%w: search_type must be one of s, a, b", ErrInvalidArgument)
	}

	messages, err := t.client.Search(req.List, req.Query, req.SearchType)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	return messages, nil
}
