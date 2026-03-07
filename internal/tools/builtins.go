package tools

import (
	"fmt"
	"sync"

	"github.com/andr1an/marc-mcp/internal/marc"
)

var (
	clientOnce sync.Once
	clientInst *marc.Client
	clientErr  error
)

func getClient() (*marc.Client, error) {
	clientOnce.Do(func() {
		clientInst, clientErr = marc.NewClient()
	})
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
	if clientInst == nil {
		return nil
	}
	return clientInst.Close()
}
