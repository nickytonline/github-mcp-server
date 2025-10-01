package ghmcp

import (
	"context"
	stdErrors "errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pkgErrors "github.com/github/github-mcp-server/pkg/errors"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/server"
)

// HTTPServerConfig captures configuration for the HTTP transport.
type HTTPServerConfig struct {
	Version           string
	Host              string
	EnabledToolsets   []string
	DynamicToolsets   bool
	ReadOnly          bool
	ContentWindowSize int
	ListenAddress     string
	EndpointPath      string
	HealthPath        string
	ShutdownTimeout   time.Duration
	LogFilePath       string
}

const (
	defaultHTTPListenAddress = ":8080"
	defaultHTTPEndpoint      = "/mcp"
	defaultHTTPHealthPath    = "/health"
	defaultShutdownTimeout   = 10 * time.Second
)

// RunHTTPServer starts an MCP server over the Streamable HTTP transport.
func RunHTTPServer(cfg HTTPServerConfig) error {
	listenAddress := cfg.ListenAddress
	if strings.TrimSpace(listenAddress) == "" {
		listenAddress = defaultHTTPListenAddress
	}

	endpointPath := normalizePath(cfg.EndpointPath, defaultHTTPEndpoint)
	healthPath := normalizePath(cfg.HealthPath, defaultHTTPHealthPath)

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	translator, _ := translations.TranslationHelper()

	ghServer, err := NewMCPServer(MCPServerConfig{
		Version:           cfg.Version,
		Host:              cfg.Host,
		EnabledToolsets:   cfg.EnabledToolsets,
		DynamicToolsets:   cfg.DynamicToolsets,
		ReadOnly:          cfg.ReadOnly,
		Translator:        translator,
		ContentWindowSize: cfg.ContentWindowSize,
		TokenProvider:     TokenFromContext,
	})
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	var logOutput io.Writer
	var logFile *os.File
	var slogHandler slog.Handler

	if strings.TrimSpace(cfg.LogFilePath) != "" {
		file, fileErr := os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if fileErr != nil {
			return fmt.Errorf("failed to open log file: %w", fileErr)
		}
		logOutput = file
		logFile = file
		slogHandler = slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logOutput = os.Stderr
		slogHandler = slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	logger := slog.New(slogHandler)
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}
	httpServer := &http.Server{Addr: listenAddress}

	streamServer := server.NewStreamableHTTPServer(
		ghServer,
		server.WithStreamableHTTPServer(httpServer),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			return pkgErrors.ContextWithGitHubErrors(ctx)
		}),
	)

	mux := http.NewServeMux()
	mux.HandleFunc(healthPath, healthHandler)

	protectedHandler := tokenMiddleware(streamServer)
	mux.Handle(endpointPath, protectedHandler)
	if !strings.HasSuffix(endpointPath, "/") {
		mux.Handle(endpointPath+"/", protectedHandler)
	}

	httpServer.Handler = mux

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting HTTP server", "address", listenAddress, "endpoint", endpointPath, "health", healthPath, "dynamicToolsets", cfg.DynamicToolsets, "readOnly", cfg.ReadOnly)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !stdErrors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down HTTP server", "reason", ctx.Err())
	case serveErr := <-errCh:
		if serveErr != nil {
			logger.Error("error running HTTP server", "error", serveErr)
			return fmt.Errorf("error running HTTP server: %w", serveErr)
		}
		logger.Info("HTTP server stopped")
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if shutdownErr := streamServer.Shutdown(shutdownCtx); shutdownErr != nil && !stdErrors.Is(shutdownErr, http.ErrServerClosed) && !stdErrors.Is(shutdownErr, context.Canceled) {
		logger.Error("error during server shutdown", "error", shutdownErr)
		return fmt.Errorf("failed to shutdown HTTP server: %w", shutdownErr)
	}

	if serveErr := <-errCh; serveErr != nil && !stdErrors.Is(serveErr, http.ErrServerClosed) {
		logger.Error("error after server shutdown", "error", serveErr)
		return fmt.Errorf("error shutting down HTTP server: %w", serveErr)
	}

	logger.Info("HTTP server shutdown complete")
	return nil
}

func normalizePath(path string, fallback string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fallback
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	return trimmed
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte("ok\n"))
}

func tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			unauthorized(w, "missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(w, "invalid Authorization header")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			unauthorized(w, "missing bearer token")
			return
		}

		ctx := ContextWithToken(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer realm=\"github-mcp-server\"")
	http.Error(w, message, http.StatusUnauthorized)
}
