package api

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/codeatlas/codeatlas/internal/store"
	"github.com/codeatlas/codeatlas/internal/webui"
)

type Server struct {
	store    *store.Store
	dataDir  string
	logger   *slog.Logger
	shutdown func()
	// Hostnames accepted in addition to loopback. Populated from
	// CODEATLAS_ALLOWED_HOSTS (comma-separated) so a deployed instance can serve
	// its public domain; "*" allows any host. Empty keeps the local-first
	// loopback-only default.
	allowedHosts  map[string]bool
	allowAllHosts bool
}

func New(
	database *store.Store,
	dataDir string,
	logger *slog.Logger,
	shutdown func(),
) *Server {
	allowedHosts := make(map[string]bool)
	allowAll := false
	for _, host := range strings.Split(os.Getenv("CODEATLAS_ALLOWED_HOSTS"), ",") {
		host = strings.ToLower(strings.TrimSpace(host))
		switch host {
		case "":
			continue
		case "*":
			allowAll = true
		default:
			allowedHosts[host] = true
		}
	}
	return &Server{
		store:         database,
		dataDir:       dataDir,
		logger:        logger,
		shutdown:      shutdown,
		allowedHosts:  allowedHosts,
		allowAllHosts: allowAll,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("POST /api/v1/system/shutdown", s.stopApplication)

	mux.HandleFunc("GET /api/v1/projects", s.listProjects)
	mux.HandleFunc("POST /api/v1/projects", s.createProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}", s.getProject)
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}", s.deleteProject)

	mux.HandleFunc("POST /api/v1/projects/{projectID}/analyses", s.queueAnalysis)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/events", s.events)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/files", s.files)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/search", s.search)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/graph/architecture", s.architecture)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/flow/{entityID}", s.flow)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/impact/{entityID}", s.impact)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/impact-map/{entityID}", s.impactMap)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/changes", s.changes)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/entities/{entityID}", s.entity)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/entities/{entityID}/source", s.source)

	mux.Handle("/", webui.Handler())
	return s.security(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
