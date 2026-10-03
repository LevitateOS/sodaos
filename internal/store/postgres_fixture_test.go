package store

import (
	"context"
	"testing"
)

// postgresFixture opens an isolated ephemeral PostgreSQL database for one
// test. key selects the encrypted entrypoint; nil opens without identity
// credentials. It returns the store and its connection URL, so close/reopen
// tests can reconnect to the same database. The database is dropped when
// the test finishes. Tests skip when the A10 disposable fixture
// environment (SODA_PG_HOST, SODA_PG_PORT, SODA_PG_SUPER_PASSWORD_FILE,
// as printed by scripts/pg-fixture.sh start) is absent.
func postgresFixture(t *testing.T, key []byte) (*Store, string) {
	t.Helper()
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := OpenEphemeral(context.Background(), admin, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s, dsn
}
