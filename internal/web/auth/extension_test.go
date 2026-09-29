package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

func TestExtensionContribution(t *testing.T) {
	for _, test := range []struct {
		name         string
		contribution extensions.Contribution
		want         bool
	}{
		{"spaces page", extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, true},
		{"runners page", extensions.Contribution{Kind: "page", ID: "runners", Scope: "global"}, true},
		{"tailnet page", extensions.Contribution{Kind: "page", ID: "tailnet", Scope: "global"}, true},
		{"workspace panel", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}, true},
		{"workspace panel with page scope", extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "global"}, false},
		{"other panel", extensions.Contribution{Kind: "panel", ID: "other", Scope: "panel"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ExtensionContribution(test.contribution); got != test.want {
				t.Fatalf("ExtensionContribution(%+v) = %t, want %t", test.contribution, got, test.want)
			}
		})
	}
}

func TestExtensionAuthorityFailsWithoutLiveHost(t *testing.T) {
	t.Setenv(extensions.CallbackEnv, "")
	r := httptest.NewRequest("GET", "/api/session", nil)
	if _, err := ExtensionAuthority(r); err == nil {
		t.Fatal("missing native context was accepted")
	}
	r.Header.Set(extensions.ContextHeader, `{"extension_id":"soda","instance_id":"untrusted","session_generation":"untrusted","contribution":{"id":"spaces","kind":"page","scope":"global","action":"get"},"actor":{"id":"42","username":"soda-tester","site_admin":false}}`)
	r.Header.Set(extensions.AdmissionHeader, "untrusted")
	if _, err := ExtensionAuthority(r); err == nil {
		t.Fatal("caller-authored context was accepted without the live host callback")
	}
	r.Header.Add(extensions.AdmissionHeader, "another")
	if _, err := ExtensionAuthority(r); err == nil {
		t.Fatal("duplicate admissions were accepted")
	}
}

func TestExtensionServiceRejectsUnverifiedMutation(t *testing.T) {
	t.Setenv(extensions.CallbackEnv, "")
	db, err := store.Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertUser(context.Background(), store.User{ID: 42, Login: "soda-tester", Name: "original"}); err != nil {
		t.Fatal(err)
	}
	s := New(&config.Config{ForgejoURL: "https://forgejo.test", OperatorID: 43}, db, nil)
	r := httptest.NewRequest(http.MethodPatch, "/api/me/preferences", strings.NewReader(`{"display_name":"forged"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Cookie", "soda_session=untrusted")
	r.Header.Set(extensions.ContextHeader, `{"extension_id":"soda","instance_id":"untrusted","session_generation":"untrusted","contribution":{"id":"spaces","kind":"page","scope":"global","action":"patch"},"actor":{"id":"42","username":"soda-tester","site_admin":false}}`)
	r.Header.Set(extensions.AdmissionHeader, "untrusted")
	w := httptest.NewRecorder()
	s.ExtensionHandler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unverified mutation returned %d", w.Code)
	}
	user, err := db.User(context.Background(), 42)
	if err != nil || user.Name != "original" {
		t.Fatalf("unverified mutation changed profile: %v", err)
	}
}
