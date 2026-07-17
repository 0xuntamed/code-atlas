package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/codeatlas/codeatlas/internal/model"
	"github.com/codeatlas/codeatlas/internal/repository"
)

const (
	sourceContextLines = 3
	maxSourceLines     = 400
)

func (s *Server) source(w http.ResponseWriter, r *http.Request) {
	record, err := s.store.Evidence(
		r.Context(),
		r.PathValue("projectID"),
		r.PathValue("entityID"),
	)
	if handleStoreError(w, err) {
		return
	}

	absolutePath := filepath.Join(record.RootPath, filepath.FromSlash(record.RelativePath))
	if !isWithin(record.RootPath, absolutePath) {
		writeError(w, http.StatusForbidden, "path_escape", "Source path is outside the repository")
		return
	}

	currentHash, err := repository.HashFile(absolutePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_unavailable", "Source file is unavailable")
		return
	}
	if subtle.ConstantTimeCompare([]byte(currentHash), []byte(record.ContentHash)) != 1 {
		writeError(w, http.StatusConflict, "source_stale", "Source changed after analysis; reanalyze the project")
		return
	}

	content, err := os.ReadFile(absolutePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "source_unavailable", "Source file is unavailable")
		return
	}

	lines := strings.Split(string(content), "\n")
	startLine, endLine := evidenceWindow(
		record.Entity.Range.StartLine,
		record.Entity.Range.EndLine,
		len(lines),
	)
	code := strings.Join(lines[startLine-1:endLine], "\n")
	writeJSON(w, http.StatusOK, model.SourceEvidence{
		EntityID:  record.Entity.ID,
		FilePath:  record.RelativePath,
		Language:  record.Entity.Language,
		StartLine: startLine,
		EndLine:   endLine,
		Code:      code,
	})
}

func evidenceWindow(entityStart, entityEnd, lineCount int) (int, int) {
	start := entityStart
	if start < 1 {
		start = 1
	}
	start -= sourceContextLines
	if start < 1 {
		start = 1
	}

	end := entityEnd
	if end < start {
		end = start + 120
	}
	end += sourceContextLines
	if end > lineCount {
		end = lineCount
	}
	if end-start+1 > maxSourceLines {
		end = start + maxSourceLines - 1
	}
	return start, end
}
