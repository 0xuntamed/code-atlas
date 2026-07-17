package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/codeatlas/codeatlas/internal/store"
)

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON request")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func handleStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found")
	} else {
		writeError(w, http.StatusInternalServerError, "database_error", "Database operation failed")
	}
	return true
}

func parseLimit(r *http.Request, fallback, maximum int) int {
	value, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func parseDepth(r *http.Request, fallback int) int {
	value, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if value <= 0 {
		return fallback
	}
	if value > 12 {
		return 12
	}
	return value
}
