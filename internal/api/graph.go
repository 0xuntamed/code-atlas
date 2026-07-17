package api

import (
	"net/http"
	"strings"

	"github.com/codeatlas/codeatlas/internal/model"
)

func (s *Server) files(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Files(
		r.Context(),
		r.PathValue("projectID"),
		r.URL.Query().Get("classification"),
		parseLimit(r, 1_000, 5_000),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": items})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]any{"entities": []model.Entity{}})
		return
	}

	items, err := s.store.Search(
		r.Context(),
		r.PathValue("projectID"),
		query,
		parseLimit(r, 30, 100),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entities": items})
}

func (s *Server) architecture(w http.ResponseWriter, r *http.Request) {
	graph, err := s.store.ArchitectureGraph(
		r.Context(),
		r.PathValue("projectID"),
		strings.TrimSpace(r.URL.Query().Get("scope")),
		parseLimit(r, 80, 120),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) flow(w http.ResponseWriter, r *http.Request) {
	graph, err := s.store.FlowGraph(
		r.Context(),
		r.PathValue("projectID"),
		r.PathValue("entityID"),
		parseDepth(r, 6),
		parseLimit(r, 120, 500),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) impact(w http.ResponseWriter, r *http.Request) {
	graph, err := s.store.ImpactGraph(
		r.Context(),
		r.PathValue("projectID"),
		r.PathValue("entityID"),
		r.URL.Query().Get("direction"),
		parseDepth(r, 4),
		parseLimit(r, 120, 500),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) entity(w http.ResponseWriter, r *http.Request) {
	entity, err := s.store.Entity(
		r.Context(),
		r.PathValue("projectID"),
		r.PathValue("entityID"),
	)
	if handleStoreError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, entity)
}
