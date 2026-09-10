package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestV9PreservesPendingCancellationAndExistingGrant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v8.db")
	key := bytes.Repeat([]byte{8}, 32)
	s, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if err = s.UpsertUser(ctx, User{ID: 1, Login: "alice", Name: "Keep"}); err != nil {
		t.Fatal(err)
	}
	grant := Grant{Access: "fixture-access", Refresh: "fixture-refresh", Scopes: "read:user", Expires: time.Now().Add(time.Hour).Unix()}
	if err = s.CreateGrantedSession(ctx, "existing", 1, "existing-csrf", grant); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOAuth(ctx, "pending", OAuthLogin{Verifier: "keep-verifier", SpacesReturn: true}, "existing", ""); err != nil {
		t.Fatal(err)
	}
	before, err := s.Session(ctx, "existing")
	if err != nil {
		t.Fatal(err)
	}
	// Remove only the v9 additions from this synthetic database. Existing v8
	// session/grant/pending rows must survive the actual upgrade on reopen.
	if _, err = s.db.Exec(`DROP INDEX login_context_oauth_cookie; ALTER TABLE login_contexts DROP COLUMN oauth_cookie; UPDATE schema_version SET version=8;`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	after, err := s.Session(ctx, "existing")
	if err != nil || before != after {
		t.Fatal("session changed", err)
	}
	kept, err := s.Grant(ctx, "existing", 1)
	if err != nil || kept != grant {
		t.Fatal("grant changed", err)
	}
	id, actor, err := s.OAuthCancellationContext(ctx, "pending")
	if err != nil || id != before.ContextID || actor != 1 {
		t.Fatal("pending context lost", err)
	}
	a, err := s.ConsumeOAuth(ctx, "pending", "existing")
	if err != nil || a.Verifier != "keep-verifier" || !a.SpacesReturn {
		t.Fatal("transaction changed", err)
	}
	if err = s.EndLoginContext(ctx, id); err != nil {
		t.Fatal(err)
	}
	if s.FinishOAuth(ctx, a, User{ID: 1, Login: "alice"}, "late", "csrf", grant) == nil {
		t.Fatal("cancelled migrated callback committed")
	}
}

func TestLoginCancellationSupersessionAndAtomicFailure(t *testing.T) {
	for _, mode := range []string{"logout", "superseded", "expired", "transaction failure", "success"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db")
			key := bytes.Repeat([]byte{9}, 32)
			s, err := OpenEncrypted(path, key)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { s.Close() }()
			ctx := t.Context()
			user := User{ID: 1, Login: "alice", Name: "keep"}
			if err = s.UpsertUser(ctx, user); err != nil {
				t.Fatal(err)
			}
			grant := Grant{Access: "fixture-access", Refresh: "fixture-refresh", Scopes: "read:user", Expires: time.Now().Add(time.Hour).Unix()}
			if err = s.CreateGrantedSession(ctx, "old", 1, "csrf", grant); err != nil {
				t.Fatal(err)
			}
			old, err := s.Session(ctx, "old")
			if err != nil {
				t.Fatal(err)
			}
			if err = s.BeginOAuth(ctx, "attempt", OAuthLogin{Verifier: "v", ExpectedUserID: 1}, "old", ""); err != nil {
				t.Fatal(err)
			}
			a, err := s.ConsumeOAuth(ctx, "attempt", "old")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ConsumeOAuth(ctx, "attempt", "old"); err == nil {
				t.Fatal("replayed claim")
			}
			switch mode {
			case "logout":
				if err = s.EndLoginContext(ctx, old.ContextID); err != nil {
					t.Fatal(err)
				}
				s.Close()
				s, err = OpenEncrypted(path, key)
				if err != nil {
					t.Fatal(err)
				}
			case "superseded":
				if err = s.BeginOAuth(ctx, "new-attempt", OAuthLogin{Verifier: "new"}, "old", ""); err != nil {
					t.Fatal(err)
				}
			case "expired":
				a.expires = time.Now().Add(-time.Minute).Unix()
			case "transaction failure":
				if _, err = s.db.Exec(`CREATE TRIGGER reject_grant BEFORE INSERT ON session_grants BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			err = s.FinishOAuth(ctx, a, User{ID: 1, Login: "renamed", Name: "ignored"}, "replacement", "new-csrf", grant)
			if mode == "success" {
				if err != nil {
					t.Fatal(err)
				}
				replacement, err := s.Session(ctx, "replacement")
				if err != nil || replacement.ContextID != old.ContextID {
					t.Fatal("lost context", err)
				}
				if err = s.EndLoginContext(ctx, old.ContextID); err != nil {
					t.Fatal(err)
				}
				if _, err = s.Grant(ctx, "replacement", 1); !errors.Is(err, ErrGrantUnavailable) {
					t.Fatal("rotated grant survived logout", err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid finalization succeeded")
			}
			got, err := s.User(ctx, 1)
			if err != nil || got != user {
				t.Fatal("partial profile write", err)
			}
			if _, err = s.Session(ctx, "replacement"); err == nil {
				t.Fatal("partial session write")
			}
			if mode != "logout" {
				if _, err = s.Grant(ctx, "old", 1); err != nil {
					t.Fatal("old grant lost", err)
				}
			}
		})
	}
}

func TestAnonymousAttemptSharesCancellationContextUntilClaimCompletes(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "db"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := t.Context()
	if err = s.BeginOAuth(ctx, "first", OAuthLogin{Verifier: "v"}, "", ""); err != nil {
		t.Fatal(err)
	}
	a, err := s.ConsumeOAuth(ctx, "first", "")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOAuth(ctx, "second", OAuthLogin{Verifier: "v2"}, "", "first"); err != nil {
		t.Fatal(err)
	}
	b, err := s.ConsumeOAuth(ctx, "second", "")
	if err != nil || a.contextID != b.contextID {
		t.Fatal("anonymous attempt escaped context", err)
	}
	grant := Grant{Access: "access", Refresh: "refresh", Expires: time.Now().Add(time.Hour).Unix()}
	if err = s.FinishOAuth(ctx, a, User{ID: 1, Login: "alice"}, "stale", "csrf", grant); !errors.Is(err, ErrLoginContext) {
		t.Fatal(err)
	}
	if err = s.FinishOAuth(ctx, b, User{ID: 1, Login: "alice"}, "fresh", "csrf", grant); err != nil {
		t.Fatal(err)
	}
	if err = s.EndLoginContext(ctx, b.contextID); err != nil {
		t.Fatal(err)
	}
	if err = s.FinishOAuth(ctx, b, User{ID: 1, Login: "changed"}, "replay", "csrf", grant); !errors.Is(err, ErrLoginContext) {
		t.Fatal(err)
	}
	if err = s.BeginOAuth(ctx, "bad", OAuthLogin{Verifier: "v"}, "", "second"); !errors.Is(err, ErrLoginContext) {
		t.Fatal("cancelled anonymous context revived", err)
	}
}
