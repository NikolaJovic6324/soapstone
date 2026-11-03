package httpapi

import (
	"context"
	"io"
	"net/http"
	"time"
)

// readyTimeout je koliko /readyz čeka skladište. Provera mora da bude brza:
// ko pita (Docker, Kubernetes) ionako pita ponovo za par sekundi.
const readyTimeout = 2 * time.Second

// handleReadyz kaže da li aplikacija može da služi zahteve: 200 kad je
// skladište spremno, 503 kad nije. Razlika od /healthz: proces može da radi
// (healthz 200), a da ne može ništa korisno (baza nedostupna, šema ne
// postoji). Takav proces ne treba restartovati, nego mu ne slati zahteve dok
// ne bude spreman.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.store.Ready(ctx); err != nil {
		s.log.Warn("not ready", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := io.WriteString(w, "not ready: "+err.Error()+"\n"); err != nil {
			s.log.Debug("writing response", "error", err)
		}
		return
	}
	if _, err := io.WriteString(w, "ready\n"); err != nil {
		s.log.Debug("writing response", "error", err)
	}
}
