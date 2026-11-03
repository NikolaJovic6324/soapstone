package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/urosevicvuk/soapstone/internal/message"
)

// TestPostgres traži pravu, praznu bazu i preskače se bez nje, pa
// `make test` i CI rade bez baze. Lokalno, uz bazu iz compose.yaml:
//
//	TEST_DATABASE_URL='postgres://soapstone:…@localhost:5432/soapstone?sslmode=disable' go test ./internal/store
//
// Test briše tabele messages i schema_migrations, pa ga ne pokrećite nad
// bazom čiji vam podaci trebaju.
func TestPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := OpenPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS messages, schema_migrations`); err != nil {
		t.Fatal(err)
	}
	p := NewPostgres(pool)

	// Pre migracija baza radi, ali aplikacija nije spremna.
	if err := p.Ready(ctx); err == nil {
		t.Fatal("Ready before migrate = nil, want error")
	}
	applied, err := Migrate(ctx, pool)
	if err != nil || len(applied) == 0 {
		t.Fatalf("Migrate = %v, %v", applied, err)
	}
	if err := p.Ready(ctx); err != nil {
		t.Fatalf("Ready after migrate = %v", err)
	}
	// Druga migracija nema šta da radi.
	if applied, err := Migrate(ctx, pool); err != nil || len(applied) != 0 {
		t.Fatalf("second Migrate = %v, %v", applied, err)
	}

	// Prazna lista je [], ne nil.
	if list, err := p.List(ctx, 10); err != nil || list == nil || len(list) != 0 {
		t.Fatalf("List on empty = %#v, %v", list, err)
	}

	// Isti tok kao TestMemory.
	for _, text := range []string{"first", "second", "third"} {
		if _, err := p.Create(ctx, message.Input{Author: "Ana", Text: text}); err != nil {
			t.Fatal(err)
		}
	}
	list, _ := p.List(ctx, 2)
	if len(list) != 2 || list[0].Text != "third" || list[1].Text != "second" {
		t.Fatalf("List(2) = %+v", list)
	}
	msg, err := p.Get(ctx, 1)
	if err != nil || msg.Text != "first" || msg.CreatedAt.Location().String() != "UTC" {
		t.Fatalf("Get(1) = %+v, %v", msg, err)
	}
	if err := p.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete(1) = %v", err)
	}
	if _, err := p.Get(ctx, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(1) after delete = %v, want ErrNotFound", err)
	}
	if err := p.Delete(ctx, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete(1) = %v, want ErrNotFound", err)
	}
	msg, _ = p.Create(ctx, message.Input{Author: "Ana", Text: "fourth"})
	if msg.ID != 4 {
		t.Fatalf("id after delete = %d, want 4", msg.ID)
	}
}
