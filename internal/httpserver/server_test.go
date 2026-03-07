package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/andr1an/marc-mcp/internal/config"
	"github.com/andr1an/marc-mcp/internal/tools"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestNewServer(t *testing.T) {
	t.Setenv("MARC_CACHE_DB", filepath.Join(t.TempDir(), "cache.db"))
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Fatalf("close tools: %v", err)
		}
	})

	cfg := config.Config{
		ListenAddr:      "127.0.0.1:8080",
		LogLevel:        "info",
		MaxHeaderBytes:  1 << 20,
		ReadTimeout:     0,
		WriteTimeout:    0,
		IdleTimeout:     0,
		ShutdownTimeout: 0,
	}

	logger := slog.Default()

	srv, err := New(cfg, logger, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected server")
	}
}
