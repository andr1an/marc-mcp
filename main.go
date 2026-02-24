package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/andr1an/marc-mcp/internal/marc"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	client, err := marc.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	s := server.NewMCPServer(
		"marc-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// list_mailing_lists tool
	listMailingListsTool := mcp.NewTool("list_mailing_lists",
		mcp.WithDescription("List all available mailing lists from marc.info, optionally filtered by category"),
		mcp.WithString("category",
			mcp.Description("Filter by category name (e.g., 'Development', 'Linux', 'Security')"),
		),
	)
	s.AddTool(listMailingListsTool, listMailingListsHandler(client))

	// list_messages tool
	listMessagesTool := mcp.NewTool("list_messages",
		mcp.WithDescription("List messages from a mailing list. Defaults to current month. Each page has ~30 messages."),
		mcp.WithString("list",
			mcp.Required(),
			mcp.Description("Name of the mailing list (e.g., 'git', 'linux-kernel')"),
		),
		mcp.WithString("month",
			mcp.Description("Month in YYYYMM format (e.g., '202602'). Defaults to current month."),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number (1-based, default: 1). Each page has ~30 messages."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of messages to return from this page (default: all)"),
		),
	)
	s.AddTool(listMessagesTool, listMessagesHandler(client))

	// get_message tool
	getMessageTool := mcp.NewTool("get_message",
		mcp.WithDescription("Get the full content of a specific message"),
		mcp.WithString("list",
			mcp.Required(),
			mcp.Description("Name of the mailing list"),
		),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("Message ID from list_messages results"),
		),
	)
	s.AddTool(getMessageTool, getMessageHandler(client))

	// search_messages tool
	searchMessagesTool := mcp.NewTool("search_messages",
		mcp.WithDescription("Search messages in a mailing list"),
		mcp.WithString("list",
			mcp.Required(),
			mcp.Description("Name of the mailing list to search"),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query string"),
		),
		mcp.WithString("search_type",
			mcp.Description("Type of search: 's' for subject (default), 'a' for author, 'b' for body"),
			mcp.Enum("s", "a", "b"),
		),
	)
	s.AddTool(searchMessagesTool, searchMessagesHandler(client))

	// Start the server
	addr := os.Getenv("MCP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	httpServer := server.NewStreamableHTTPServer(s)
	log.Printf("Starting MCP server on %s", addr)
	if err := httpServer.Start(addr); err != nil {
		log.Fatal(err)
	}
}

func listMailingListsHandler(client *marc.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		category := req.GetString("category", "")

		lists, err := client.ListMailingLists()
		if err != nil {
			return nil, fmt.Errorf("failed to list mailing lists: %w", err)
		}

		if category != "" {
			var filtered []marc.MailingList
			for _, l := range lists {
				if l.Category == category {
					filtered = append(filtered, l)
				}
			}
			lists = filtered
		}

		data, err := json.MarshalIndent(lists, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func listMessagesHandler(client *marc.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list, err := req.RequireString("list")
		if err != nil {
			return nil, err
		}

		opts := marc.ListMessagesOptions{
			List:  list,
			Month: req.GetString("month", ""),
			Page:  req.GetInt("page", 1),
			Limit: req.GetInt("limit", 0),
		}

		messages, err := client.ListMessagesWithOptions(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}

		data, err := json.MarshalIndent(messages, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func getMessageHandler(client *marc.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list, err := req.RequireString("list")
		if err != nil {
			return nil, err
		}
		messageID, err := req.RequireString("message_id")
		if err != nil {
			return nil, err
		}

		message, err := client.GetMessage(list, messageID)
		if err != nil {
			return nil, fmt.Errorf("failed to get message: %w", err)
		}

		data, err := json.MarshalIndent(message, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func searchMessagesHandler(client *marc.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list, err := req.RequireString("list")
		if err != nil {
			return nil, err
		}
		query, err := req.RequireString("query")
		if err != nil {
			return nil, err
		}
		searchType := req.GetString("search_type", "s")

		messages, err := client.Search(list, query, searchType)
		if err != nil {
			return nil, fmt.Errorf("failed to search messages: %w", err)
		}

		data, err := json.MarshalIndent(messages, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
