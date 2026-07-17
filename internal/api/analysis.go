package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/codeatlas/codeatlas/internal/id"
	"github.com/codeatlas/codeatlas/internal/model"
)

func (s *Server) queueAnalysis(w http.ResponseWriter, r *http.Request) {
	runID := id.New()
	projectID := r.PathValue("projectID")
	if handleStoreError(w, s.store.QueueAnalysis(r.Context(), projectID, runID)) {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"analysisRunId": runID,
		"status":        model.RunQueued,
	})
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "Streaming is unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()
	lastPayload := ""

	for {
		run, err := s.store.LatestRun(r.Context(), r.PathValue("projectID"))
		if err != nil {
			return
		}

		payload, _ := json.Marshal(run)
		payloadText := string(payload)
		if subtle.ConstantTimeCompare([]byte(payloadText), []byte(lastPayload)) != 1 {
			_, _ = fmt.Fprintf(w, "event: progress\ndata: %s\n\n", payload)
			flusher.Flush()
			lastPayload = payloadText
		}

		if run.Status == model.RunReady || run.Status == model.RunFailed {
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}
