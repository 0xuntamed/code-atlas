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
	"syscall"
	"time"

	"github.com/codeatlas/codeatlas/internal/analyzer"
	"github.com/codeatlas/codeatlas/internal/api"
	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/codeatlas/codeatlas/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "codeatlas:", err)
		os.Exit(1)
	}
}

func run() error {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "serve":
		return serve(os.Args[2:])
	case "add":
		return add(os.Args[2:])
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
	listen := flags.String("listen", env("CODEATLAS_LISTEN_ADDR", "127.0.0.1:7331"), "loopback listen address")
	databaseURL := flags.String("database-url", env("CODEATLAS_DATABASE_URL", "postgres://codeatlas:codeatlas@127.0.0.1:5433/codeatlas?sslmode=disable"), "PostgreSQL connection URL")
	dataDir := flags.String("data-dir", env("CODEATLAS_DATA_DIR", defaultDataDir()), "local application data directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	database, err := store.Open(ctx, *databaseURL)
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
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = server.Shutdown(shutdownCtx)
	}()
	logger.Info("CodeAtlas ready", "url", "http://"+*listen, "data_dir", *dataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func add(args []string) error {
	flags := flag.NewFlagSet("add", flag.ContinueOnError)
	serverURL := flags.String("server", env("CODEATLAS_URL", "http://127.0.0.1:7331"), "running CodeAtlas URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("usage: codeatlas add <absolute-path>")
	}
	absolute, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(model.CreateProjectRequest{Source: model.ProjectSource{Type: model.SourceLocal, Path: absolute}})
	request, err := http.NewRequest(http.MethodPost, *serverURL+"/api/v1/projects", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("contact local CodeAtlas server: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("server rejected the repository: %s", response.Status)
	}
	var created model.CreateProjectResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return err
	}
	fmt.Printf("Queued project %s (analysis %s)\n", created.ProjectID, created.AnalysisRunID)
	return nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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
