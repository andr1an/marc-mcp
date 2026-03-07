package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/andr1an/marc-mcp/internal/marc"
)

type ListMailingListsTool struct {
	client *marc.Client
}

type ListMailingListsInput struct {
	Category string `json:"category,omitempty"`
	Filter   string `json:"filter,omitempty"`
}

func NewListMailingListsTool(client *marc.Client) Tool {
	return &ListMailingListsTool{client: client}
}

func (t *ListMailingListsTool) Name() string {
	return "list_mailing_lists"
}

func (t *ListMailingListsTool) Description() string {
	return "List all available mailing lists from marc.info, optionally filtered by category or name regex"
}

func (t *ListMailingListsTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category": map[string]any{
				"type":        "string",
				"description": "Filter by category name (e.g., 'Development', 'Linux', 'Security')",
			},
			"filter": map[string]any{
				"type":        "string",
				"description": "Filter list names by regular expression (e.g., 'git.*', '^linux', 'kernel')",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
	}
}

func (t *ListMailingListsTool) Invoke(ctx context.Context, input []byte) (any, error) {
	_ = ctx

	var req ListMailingListsInput
	if len(input) > 0 {
		if err := json.Unmarshal(input, &req); err != nil {
			return nil, fmt.Errorf("%w: invalid JSON body: %v", ErrInvalidArgument, err)
		}
	}

	var filterRe *regexp.Regexp
	if req.Filter != "" {
		re, err := regexp.Compile(req.Filter)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid filter regex: %v", ErrInvalidArgument, err)
		}
		filterRe = re
	}

	lists, err := t.client.ListMailingLists()
	if err != nil {
		return nil, fmt.Errorf("failed to list mailing lists: %w", err)
	}

	if req.Category != "" {
		filtered := make([]marc.MailingList, 0, len(lists))
		for _, l := range lists {
			if l.Category == req.Category {
				filtered = append(filtered, l)
			}
		}
		lists = filtered
	}

	if filterRe != nil {
		filtered := make([]marc.MailingList, 0, len(lists))
		for _, l := range lists {
			if filterRe.MatchString(l.Name) {
				filtered = append(filtered, l)
			}
		}
		lists = filtered
	}

	return lists, nil
}
