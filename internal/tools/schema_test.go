package tools

import "testing"

func TestMarcToolsHaveBasicSchemaShape(t *testing.T) {
	// Constructors do not use client for metadata methods.
	all := []Tool{
		NewListMailingListsTool(nil),
		NewListMessagesTool(nil),
		NewGetMessageTool(nil),
		NewSearchMessagesTool(nil),
	}

	r := NewRegistry()
	for _, tool := range all {
		r.Register(tool)
	}

	for _, info := range r.List() {
		if info.Name == "" {
			t.Fatal("tool name must not be empty")
		}
		if info.Description == "" {
			t.Fatalf("tool %q has empty description", info.Name)
		}
		if info.InputSchema == nil {
			t.Fatalf("tool %q has nil input schema", info.Name)
		}

		typ, ok := info.InputSchema["type"].(string)
		if !ok || typ != "object" {
			t.Fatalf("tool %q schema.type must be object", info.Name)
		}

		if _, ok := info.InputSchema["properties"].(map[string]any); !ok {
			t.Fatalf("tool %q schema.properties must be map[string]any", info.Name)
		}
		if _, ok := info.InputSchema["additionalProperties"]; !ok {
			t.Fatalf("tool %q schema.additionalProperties must be set", info.Name)
		}
	}
}
