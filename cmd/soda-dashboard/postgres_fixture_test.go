package main

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

// postgresFixture opens an isolated ephemeral PostgreSQL database for one
// test. Tests skip when the A10 disposable fixture environment is absent.
func postgresFixture(t *testing.T) *store.Store {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s
}
