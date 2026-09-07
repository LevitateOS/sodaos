package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginCascadeSurvivesPoolConnectionReplacement(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "db ?# keep"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.UpsertUser(t.Context(), User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateGrantedSession(t.Context(), "token", 1, "csrf", Grant{Access: "access", Refresh: "refresh", Expires: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	session, err := s.Session(t.Context(), "token")
	if err != nil {
		t.Fatal(err)
	}
	// database/sql may discard a connection after cancellation. Every replacement
	// must retain FK cascades, not just the connection used at startup.
	s.db.SetMaxIdleConns(0)
	if err = s.EndLoginContext(t.Context(), session.ContextID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Grant(t.Context(), "token", 1); !errors.Is(err, ErrGrantUnavailable) {
		t.Fatal("logout left an orphan usable grant", err)
	}
	var sessions, grants int
	if err = s.db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT count(*) FROM session_grants`).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || grants != 0 {
		t.Fatal("cascade lost", sessions, grants)
	}
}
