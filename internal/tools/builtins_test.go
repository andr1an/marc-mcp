package tools

import (
	"path/filepath"
	"testing"
)

func TestCloseAllowsReinit(t *testing.T) {
	t.Setenv("MARC_CACHE_DB", filepath.Join(t.TempDir(), "cache.db"))

	first, err := getClient()
	if err != nil {
		t.Fatalf("first getClient() failed: %v", err)
	}
	if first == nil {
		t.Fatal("first getClient() returned nil client")
	}

	if err := Close(); err != nil {
		t.Fatalf("first Close() failed: %v", err)
	}

	second, err := getClient()
	if err != nil {
		t.Fatalf("second getClient() failed: %v", err)
	}
	if second == nil {
		t.Fatal("second getClient() returned nil client")
	}
	if first == second {
		t.Fatal("expected a new client instance after Close()")
	}

	if err := Close(); err != nil {
		t.Fatalf("second Close() failed: %v", err)
	}
}
