package control_test

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

// postgresFixture opens an isolated ephemeral PostgreSQL database for one
// native test; see the internal-test twin for the contract.
func postgresFixture(t *testing.T, key []byte) (*store.Store, string) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s, dsn
}
