package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migracije su SQL fajlovi u direktorijumu migrations, ugrađeni u binarni
// fajl kroz go:embed, pa image ne mora da nosi ništa osim njega. Ime fajla
// počinje brojem verzije: 001_messages.sql je verzija 1. Fajl se nikad ne
// menja kad jednom stigne na main; izmena šeme je nov fajl sa sledećim brojem.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// migration je jedan SQL fajl i njegova verzija.
type migration struct {
	version int
	name    string
	sql     string
}

// migrations vraća sve migracije, sortirane po verziji.
func migrations() ([]migration, error) {
	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, f := range files {
		name := path.Base(f)
		prefix, _, _ := strings.Cut(name, "_")
		v, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration %s: name must start with a version number", name)
		}
		b, err := migrationFS.ReadFile(f)
		if err != nil {
			return nil, err
		}
		out = append(out, migration{version: v, name: name, sql: string(b)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// latestVersion je verzija poslednje migracije: šema je spremna kad je baza
// na njoj.
func latestVersion() (int, error) {
	all, err := migrations()
	if err != nil {
		return 0, err
	}
	if len(all) == 0 {
		return 0, nil
	}
	return all[len(all)-1].version, nil
}

// migrateLock je proizvoljan broj za pg_advisory_xact_lock: dve migracije
// pokrenute u isto vreme ne smeju da rade istovremeno, pa druga čeka prvu.
const migrateLock = 4242

// Migrate dovodi šemu baze na poslednju verziju i vraća imena migracija
// koje je primenio; prazna lista znači da je šema već bila na poslednjoj
// verziji. Sve se radi u jednoj transakciji: ako jedna migracija padne, baza
// ostaje kakva je bila pre poziva.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	all, err := migrations()
	if err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Posle Commit-a Rollback ne radi ništa, pa je defer bezbedan.
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrateLock); err != nil {
		return nil, err
	}
	// U schema_migrations se pamti koje su verzije već primenjene.
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    integer     PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, err
	}

	var current int
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return nil, err
	}

	var applied []string
	for _, m := range all {
		if m.version <= current {
			continue
		}
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			return nil, fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.version); err != nil {
			return nil, err
		}
		applied = append(applied, m.name)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return applied, nil
}

// schemaVersion vraća verziju šeme u bazi, ili 0 ako migracije nikad nisu
// pokrenute.
func schemaVersion(ctx context.Context, q *pgxpool.Pool) (int, error) {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	var v int
	err := q.QueryRow(ctx, `SELECT coalesce(max(version), 0) FROM schema_migrations`).Scan(&v)
	return v, err
}
