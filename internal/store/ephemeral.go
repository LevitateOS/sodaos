package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
)

// TestFixtureDSN builds a superuser connection URL from the A10 disposable
// fixture environment: SODA_PG_HOST, SODA_PG_PORT and
// SODA_PG_SUPER_PASSWORD_FILE, as printed by soda-pg-fixture start.
// It reports ok=false when the fixture is absent; tests skip instead of
// guessing a database. The password travels from its restricted file into
// the returned URL only; it is never logged.
func TestFixtureDSN() (dsn string, ok bool) {
	host, port, passwordFile := os.Getenv("SODA_PG_HOST"), os.Getenv("SODA_PG_PORT"), os.Getenv("SODA_PG_SUPER_PASSWORD_FILE")
	if host == "" || port == "" || passwordFile == "" {
		return "", false
	}
	raw, err := os.ReadFile(passwordFile)
	if err != nil {
		return "", false
	}
	password := strings.TrimSpace(string(raw))
	if password == "" || strings.ContainsAny(password, "\r\n\x00") {
		return "", false
	}
	u := url.URL{Scheme: "postgres", Host: host + ":" + port, Path: "/postgres", User: url.UserPassword("postgres", password)}
	query := url.Values{}
	query.Set("sslmode", "disable")
	u.RawQuery = query.Encode()
	return u.String(), true
}

// OpenEphemeral creates a uniquely-named database on the given server and
// opens a Store on it for one test. key selects the encrypted production
// entrypoint; nil opens without identity credentials. The returned cleanup
// closes the store and drops the database. Tests resolve the server URL
// through TestFixtureDSN and skip when it is absent.
func OpenEphemeral(ctx context.Context, superDSN string, key []byte) (s *Store, dsn string, cleanup func(), err error) {
	dsn, drop, err := createEphemeralDatabase(ctx, superDSN)
	if err != nil {
		return nil, "", nil, err
	}
	if key == nil {
		s, err = Open(dsn)
	} else {
		s, err = OpenEncrypted(dsn, key)
	}
	if err != nil {
		drop()
		return nil, "", nil, err
	}
	cleanup = func() {
		_ = s.Close()
		drop()
	}
	return s, dsn, cleanup, nil
}

// createEphemeralDatabase creates a uniquely-named empty database and
// returns its connection URL plus a drop cleanup. Schema tests use it to
// stage legacy version markers without opening a Store.
func createEphemeralDatabase(ctx context.Context, superDSN string) (string, func(), error) {
	name, err := ephemeralName()
	if err != nil {
		return "", nil, err
	}
	admin, err := sql.Open("pgx", superDSN)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = admin.Close() }()
	if _, err = admin.ExecContext(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		return "", nil, err
	}
	u, err := url.Parse(superDSN)
	if err != nil {
		return "", nil, err
	}
	u.Path = "/" + name
	drop := func() {
		drop, err := sql.Open("pgx", superDSN)
		if err != nil {
			return
		}
		defer func() { _ = drop.Close() }()
		_, _ = drop.ExecContext(context.Background(), `DROP DATABASE IF EXISTS "`+name+`"`)
	}
	return u.String(), drop, nil
}

func ephemeralName() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	// Lowercase hex only, so the quoted identifier needs no escaping.
	return "soda_ephem_" + hex.EncodeToString(random[:]), nil
}
