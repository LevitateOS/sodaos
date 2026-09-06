package store

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestEncryptedGrantBindingAndLogout(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "soda.db")
	key := bytes.Repeat([]byte{42}, 32)
	s, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, uid := range []int64{1, 2} {
		if err = s.UpsertUser(ctx, User{ID: uid, Login: string(rune('a' + uid))}); err != nil {
			t.Fatal(err)
		}
	}
	grant := Grant{Access: "private-access", Refresh: "private-refresh", Scopes: "write:user", Expires: time.Now().Add(time.Hour).Unix()}
	for _, session := range []string{"one", "two"} {
		if err = s.CreateGrantedSession(ctx, session, 1, "csrf", grant); err != nil {
			t.Fatal(err)
		}
	}
	var ciphertext []byte
	if err = s.db.QueryRow(`SELECT ciphertext FROM session_grants WHERE session_token=?`, hash("one")).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("private")) {
		t.Fatal("plaintext grant stored")
	}
	if _, err = s.Grant(ctx, "one", 2); !errors.Is(err, ErrGrantUnavailable) {
		t.Fatal("cross-user grant accepted", err)
	}
	if _, err = s.db.Exec(`UPDATE session_grants SET ciphertext=? WHERE session_token=?`, ciphertext, hash("two")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Grant(ctx, "two", 1); !errors.Is(err, ErrGrantKey) {
		t.Fatal("cross-session ciphertext accepted", err)
	}
	if err = s.DeleteSession(ctx, "one"); err != nil {
		t.Fatal(err)
	}
	if err = s.ReplaceGrant(ctx, "one", 1, grant); !errors.Is(err, ErrGrantUnavailable) {
		t.Fatal("refresh resurrected logout", err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT count(*) FROM session_grants WHERE session_token=?`, hash("one")).Scan(&count); err != nil || count != 0 {
		t.Fatal("grant not deleted", err)
	}
}

func TestGrantKeyMustSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "soda.db")
	key := bytes.Repeat([]byte{7}, 32)
	s, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if s, err = Open(path); !errors.Is(err, ErrGrantKey) {
		if s != nil {
			s.Close()
		}
		t.Fatal("missing key accepted", err)
	}
	if s, err = OpenEncrypted(path, bytes.Repeat([]byte{8}, 32)); !errors.Is(err, ErrGrantKey) {
		if s != nil {
			s.Close()
		}
		t.Fatal("wrong key accepted", err)
	}
	s, err = OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
}

func TestLegacySessionRequiresGrant(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "soda.db"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.UpsertUser(ctx, User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateSession(ctx, "legacy", 1, "csrf"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Grant(ctx, "legacy", 1); !errors.Is(err, ErrGrantUnavailable) {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, "legacy"); err != nil {
		t.Fatal("legacy product session lost", err)
	}
}
