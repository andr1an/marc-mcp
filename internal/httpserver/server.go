package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/andr1an/marc-mcp/internal/config"
	"github.com/andr1an/marc-mcp/internal/middleware"
	"github.com/andr1an/marc-mcp/internal/tools"
	"github.com/andr1an/marc-mcp/internal/transport"
)

type Server struct {
	*http.Server
}

func New(cfg config.Config, logger *slog.Logger, version string) (*Server, error) {
	mux := http.NewServeMux()

	registry, err := tools.NewRegistryWithBuiltins()
	if err != nil {
		return nil, err
	}

	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/mcp", chain(
		transport.NewMCPHandler(registry, version),
		middleware.RequestID,
		middleware.Logging(logger),
	))

	srv := &http.Server{
		Addr:           cfg.Address(),
		Handler:        mux,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	return &Server{Server: srv}, nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func chain(final http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	h := final
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
