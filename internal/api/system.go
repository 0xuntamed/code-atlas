package api

import (
	"net/http"
	"time"
)

func (s *Server) stopApplication(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-CodeAtlas-Intent") != "shutdown" {
		writeError(w, http.StatusBadRequest, "shutdown_not_confirmed", "Shutdown must be explicitly confirmed")
		return
	}
	if s.shutdown == nil {
		writeError(w, http.StatusNotImplemented, "shutdown_unavailable", "Shutdown is controlled by the host process")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "stopping"})
	go func() {
		time.Sleep(150 * time.Millisecond)
		s.shutdown()
	}()
}
