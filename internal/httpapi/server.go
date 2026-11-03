// Paket httpapi je HTTP strana aplikacije: rute, handleri i middleware.
package httpapi

import (
	"encoding/json"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/urosevicvuk/soapstone/internal/store"
	"github.com/urosevicvuk/soapstone/web"
)

// Server drži sve što handleri koriste.
type Server struct {
	store    store.Store
	log      *slog.Logger
	color    string
	hostname string
	index    *template.Template
	mux      *http.ServeMux
}

// New pravi server sa svim rutama. Vraćeni Server je http.Handler, pa se
// prosleđuje direktno http.Server-u ili httptest-u.
func New(st store.Store, log *slog.Logger, color string) *Server {
	// Hostname je ime mašine, kontejnera ili poda na kom proces radi. Na
	// stranici se vidi koja kopija aplikacije je odgovorila.
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	s := &Server{
		store:    st,
		log:      log,
		color:    color,
		hostname: hostname,
		index:    template.Must(template.ParseFS(web.FS, "index.html")),
		mux:      http.NewServeMux(),
	}

	// Rute: metod, putanja i handler. {id} je deo putanje koji handler čita
	// sa r.PathValue("id"), a {$} znači „tačno /“, bez ičega posle.
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	s.mux.HandleFunc("GET /api/messages", s.handleList)
	s.mux.HandleFunc("POST /api/messages", s.handleCreate)
	s.mux.HandleFunc("GET /api/messages/{id}", s.handleGet)
	s.mux.HandleFunc("DELETE /api/messages/{id}", s.handleDelete)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)

	return s
}

// ServeHTTP propušta svaki zahtev kroz middleware, pa do rute.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logRequests(s.mux).ServeHTTP(w, r)
}

// handleHealthz kaže da proces radi i odgovara. Ne proverava ništa drugo,
// jer služi samo da se vidi da li proces treba restartovati.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := io.WriteString(w, "ok\n"); err != nil {
		s.log.Debug("writing response", "error", err)
	}
}

// writeJSON šalje vrednost v kao JSON, sa datim statusom.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Error("writing JSON response", "error", err)
	}
}

// writeError šalje grešku u obliku {"error": "…"}.
func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{"error": msg})
}
