// soapstone je mala oglasna tabla: neko ostavi poruku, drugi je čitaju.
// Ime dolazi od Orange Guidance Soapstone-a iz Dark Souls-a, kojim igrači
// ostavljaju poruke jedni drugima.
//
// Upotreba:
//
//	soapstone [serve]   start the HTTP server (default)
//	soapstone migrate   create or update the database schema
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/urosevicvuk/soapstone/internal/config"
	"github.com/urosevicvuk/soapstone/internal/httpapi"
	"github.com/urosevicvuk/soapstone/internal/store"
)

const usage = `usage: soapstone [command]

commands:
  serve     start the HTTP server (default)
  migrate   create or update the database schema (needs DATABASE_URL)
`

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "serve":
		err = serve()
	case "migrate":
		err = migrate()
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// serve pokreće server i radi dok ne stigne SIGINT (Ctrl+C) ili SIGTERM
// (ono što šalju `docker stop` i Kubernetes kad gase proces).
func serve() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg)

	// Bez DATABASE_URL poruke žive u memoriji, kao do sada; sa njim, u bazi.
	var st store.Store = store.NewMemory()
	if cfg.DatabaseURL != "" {
		pool, err := store.OpenPool(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		st = store.NewPostgres(pool)
	}

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           httpapi.New(st, log, cfg.AppColor),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ctx se otkazuje kad stigne signal za gašenje.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("start",
		"port", cfg.Port,
		"color", cfg.AppColor,
		"store", st.Kind(),
		"log_level", cfg.LogLevel.String(),
		"log_format", cfg.LogFormat,
	)

	// Server radi u svojoj gorutini, a ova čeka signal ili grešku servera,
	// na primer zauzet port.
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	// Gašenje: server prestaje da prima nove veze i čeka da se aktivni
	// zahtevi završe, najviše 10 sekundi.
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}

// migrate dovodi šemu baze na poslednju verziju i završava. serve to ne radi
// sam, pri startu: kad radi više kopija aplikacije, šemu menja jedan
// jednokratni proces, pre njih, a ne svaka kopija za sebe.
func migrate() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg)
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := store.Migrate(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if len(applied) == 0 {
		log.Info("schema is up to date")
	} else {
		log.Info("migrated", "applied", applied)
	}
	return nil
}

// newLogger pravi logger koji piše na stdout, u JSON-u ili kao tekst.
func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if cfg.LogFormat == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}
