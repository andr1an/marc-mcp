package tools

import (
	"fmt"
	"sync"

	"github.com/andr1an/marc-mcp/internal/marc"
)

var (
	clientMu   sync.Mutex
	clientInst *marc.Client
	clientErr  error
)

func getClient() (*marc.Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if clientInst != nil {
		return clientInst, nil
	}
	if clientErr != nil {
		return nil, clientErr
	}

	clientInst, clientErr = marc.NewClient()
	return clientInst, clientErr
}

func RegisterBuiltins(registry *Registry) error {
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("create marc client: %w", err)
	}

	registry.Register(NewListMailingListsTool(client))
	registry.Register(NewListMessagesTool(client))
	registry.Register(NewGetMessageTool(client))
	registry.Register(NewSearchMessagesTool(client))
	return nil
}

func Close() error {
	clientMu.Lock()
	client := clientInst
	clientInst = nil
	clientErr = nil
	clientMu.Unlock()

	if client == nil {
		return nil
	}

	return client.Close()
}
