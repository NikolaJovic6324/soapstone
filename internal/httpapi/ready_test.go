package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/urosevicvuk/soapstone/internal/store"
)

// notReady je memorijsko skladište koje nikad nije spremno, kao baza koja
// nije dostupna.
type notReady struct{ *store.Memory }

func (notReady) Ready(context.Context) error { return errors.New("database unavailable") }

func TestReadyz(t *testing.T) {
	rec := do(newTestServer(), "GET", "/readyz", "")
	if rec.Code != 200 || rec.Body.String() != "ready\n" {
		t.Errorf("memory: GET /readyz = %d %q", rec.Code, rec.Body)
	}

	s := New(notReady{store.NewMemory()}, slog.New(slog.DiscardHandler), "tomato")
	rec = do(s, "GET", "/readyz", "")
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "database unavailable") {
		t.Errorf("not ready: GET /readyz = %d %q", rec.Code, rec.Body)
	}

	// /healthz ne gleda skladište: proces radi, pa je 200.
	if rec := do(s, "GET", "/healthz", ""); rec.Code != 200 {
		t.Errorf("not ready: GET /healthz = %d", rec.Code)
	}
}
