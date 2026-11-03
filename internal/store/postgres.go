package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/urosevicvuk/soapstone/internal/message"
)

// Postgres čuva poruke u bazi Postgres. Za razliku od memorije, poruke
// preživljavaju restart aplikacije, a sve kopije aplikacije vide iste poruke.
type Postgres struct {
	// pool drži nekoliko otvorenih veza ka bazi i daje ih zahtevima redom.
	// Veze se otvaraju tek kad zatrebaju, pa aplikacija krene i kad baza još
	// nije dostupna; tada je samo /readyz 503.
	pool *pgxpool.Pool
}

// OpenPool pravi pool veza za dati URL, oblika
// postgres://korisnik:lozinka@host:5432/baza?sslmode=disable. Ne povezuje
// se odmah, pa greška znači samo da URL nije ispravan.
func OpenPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// Greška parsiranja ne sme da ispiše URL, jer je u njemu lozinka.
		return nil, errors.New("DATABASE_URL is not a valid Postgres URL")
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

// NewPostgres pravi skladište nad postojećim pool-om. Šemu pravi
// `soapstone migrate`, ne ovaj kod.
func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) Kind() string { return "postgres" }

// Ready proverava da li je baza dostupna i da li je šema na poslednjoj
// verziji. Bez migracija tabela messages ne postoji, pa aplikacija nije
// spremna, iako proces radi.
func (p *Postgres) Ready(ctx context.Context) error {
	want, err := latestVersion()
	if err != nil {
		return err
	}
	got, err := schemaVersion(ctx, p.pool)
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}
	if got < want {
		return fmt.Errorf("schema is at version %d, want %d: run soapstone migrate", got, want)
	}
	return nil
}

func (p *Postgres) List(ctx context.Context, limit int) ([]message.Message, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, author, text, created_at FROM messages ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	// CollectRows čita sve redove i zatvara ih; prazan rezultat je prazna
	// lista, ne nil, pa API vraća [] kao i memorija.
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (message.Message, error) {
		return scanMessage(r)
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []message.Message{}
	}
	return out, nil
}

func (p *Postgres) Create(ctx context.Context, in message.Input) (message.Message, error) {
	row := p.pool.QueryRow(ctx,
		`INSERT INTO messages (author, text) VALUES ($1, $2) RETURNING id, author, text, created_at`,
		in.Author, in.Text)
	return scanMessage(row)
}

func (p *Postgres) Get(ctx context.Context, id int64) (message.Message, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT id, author, text, created_at FROM messages WHERE id = $1`, id)
	msg, err := scanMessage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return message.Message{}, ErrNotFound
	}
	return msg, err
}

func (p *Postgres) Delete(ctx context.Context, id int64) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanMessage čita jedan red u poruku. Vreme iz baze stiže u lokalnoj zoni
// procesa, a API ga uvek vraća u UTC.
func scanMessage(row pgx.Row) (message.Message, error) {
	var m message.Message
	if err := row.Scan(&m.ID, &m.Author, &m.Text, &m.CreatedAt); err != nil {
		return message.Message{}, err
	}
	m.CreatedAt = m.CreatedAt.UTC()
	return m, nil
}
