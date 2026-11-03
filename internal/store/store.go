// Paket store čuva poruke. Ostatak aplikacije zna samo za interfejs Store,
// pa ne zna, niti ga zanima, gde poruke stvarno žive.
package store

import (
	"context"
	"errors"

	"github.com/urosevicvuk/soapstone/internal/message"
)

// ErrNotFound vraćaju Get i Delete kad poruka sa tim id-jem ne postoji.
var ErrNotFound = errors.New("message not found")

// Store je sve što aplikacija traži od skladišta.
type Store interface {
	// List vraća najviše limit poruka, od najnovije.
	List(ctx context.Context, limit int) ([]message.Message, error)
	// Create upisuje novu poruku i vraća je sa dodeljenim id-jem i vremenom.
	// Poruka mora već da bude validirana.
	Create(ctx context.Context, in message.Input) (message.Message, error)
	// Get vraća jednu poruku, ili ErrNotFound.
	Get(ctx context.Context, id int64) (message.Message, error)
	// Delete briše jednu poruku, ili vraća ErrNotFound.
	Delete(ctx context.Context, id int64) error
	// Ready kaže da li skladište može da radi. Memorija je uvek spremna, a
	// Postgres kad je baza dostupna i šema na poslednjoj verziji.
	Ready(ctx context.Context) error
	// Kind je kratko ime skladišta, na primer "memory", za prikaz.
	Kind() string
}
