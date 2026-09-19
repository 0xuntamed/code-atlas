package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/codeatlas/codeatlas/internal/analyzer"
	"github.com/codeatlas/codeatlas/internal/api"
	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/codeatlas/codeatlas/internal/store"
)

const (
	defaultListenAddress = "127.0.0.1:7331"
	defaultServerURL     = "http://127.0.0.1:7331"
	databaseFileName     = "codeatlas.db"

	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
	requestTimeout    = 30 * time.Second
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "codeatlas:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return serve(nil)
	}

	command, commandArgs := args[0], args[1:]
	switch command {
	case "serve":
		return serve(commandArgs)
	case "add":
		return add(commandArgs)
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := flags.String("listen", defaultListen(), "listen address host:port")
	databasePath := flags.String("database", env("CODEATLAS_DATABASE_PATH", ""), "SQLite database file path (defaults to <data-dir>/"+databaseFileName+")")
	dataDir := flags.String("data-dir", env("CODEATLAS_DATA_DIR", defaultDataDir()), "local application data directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	if *databasePath == "" {
		*databasePath = filepath.Join(*dataDir, databaseFileName)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	database, err := store.Open(ctx, *databasePath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(ctx); err != nil {
		return err
	}
	worker := analyzer.New(database, logger)
	go worker.Run(ctx)
	handler := api.New(database, *dataDir, logger, cancel).Handler()
	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), shutdownTimeout)
		defer done()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shut down HTTP server", "error", err)
		}
	}()
	if repo := flags.Arg(0); repo != "" {
		go registerOnStart(ctx, "http://"+*listen, repo, logger)
	}
	logger.Info("CodeAtlas ready", "url", "http://"+*listen, "data_dir", *dataDir, "database", *databasePath)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func add(args []string) error {
	flags := flag.NewFlagSet("add", flag.ContinueOnError)
	serverURL := flags.String("server", env("CODEATLAS_URL", defaultServerURL), "running CodeAtlas URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: codeatlas add <absolute-path>")
	}
	absolute, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	payload, err := json.Marshal(model.CreateProjectRequest{
		Source: model.ProjectSource{Type: model.SourceLocal, Path: absolute},
	})
	if err != nil {
		return fmt.Errorf("encode project request: %w", err)
	}
	endpoint := strings.TrimRight(*serverURL, "/") + "/api/v1/projects"
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create project request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: requestTimeout}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("contact local CodeAtlas server: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("server rejected the repository: %s", response.Status)
	}
	var created model.CreateProjectResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return fmt.Errorf("decode server response: %w", err)
	}
	fmt.Printf("Queued project %s (analysis %s)\n", created.ProjectID, created.AnalysisRunID)
	return nil
}

// registerOnStart waits for the just-started local server to become healthy and
// then registers the repository passed positionally to `serve`, so a first run
// needs no separate `add` step.
func registerOnStart(ctx context.Context, serverURL, repoPath string, logger *slog.Logger) {
	absolute, err := filepath.Abs(repoPath)
	if err != nil {
		logger.Error("resolve repository path", "error", err)
		return
	}
	base := strings.TrimRight(serverURL, "/")
	client := &http.Client{Timeout: requestTimeout}
	ready := false
	for attempt := 0; attempt < 50; attempt++ {
		if ctx.Err() != nil {
			return
		}
		if response, err := client.Get(base + "/api/v1/health"); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		logger.Error("register repository: server did not become ready", "path", absolute)
		return
	}
	payload, err := json.Marshal(model.CreateProjectRequest{
		Source: model.ProjectSource{Type: model.SourceLocal, Path: absolute},
	})
	if err != nil {
		logger.Error("encode project request", "error", err)
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/projects", bytes.NewReader(payload))
	if err != nil {
		logger.Error("create project request", "error", err)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		logger.Error("register repository", "error", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		logger.Error("register repository rejected", "status", response.Status, "path", absolute)
		return
	}
	logger.Info("registered repository", "path", absolute)
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// defaultListen resolves the address the server binds to. An explicit
// CODEATLAS_LISTEN_ADDR wins; otherwise a platform-provided PORT (Railway, Fly,
// Render, …) binds all interfaces on that port; otherwise it stays on the
// local-first loopback default so a plain `codeatlas serve` never exposes itself
// on the network.
func defaultListen() string {
	if addr := os.Getenv("CODEATLAS_LISTEN_ADDR"); addr != "" {
		return addr
	}
	if port := os.Getenv("PORT"); port != "" {
		return "0.0.0.0:" + port
	}
	return defaultListenAddress
}

func defaultDataDir() string {
	root, err := os.UserCacheDir()
	if err != nil {
		return ".codeatlas-data"
	}
	return filepath.Join(root, "CodeAtlas")
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:\n  codeatlas serve [flags]\n  codeatlas add <absolute-path>")
}
