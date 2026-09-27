package forgejo

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitIdentity(t *testing.T) {
	cases := []struct {
		name, user, emails string
		wantEmail          string
	}{
		{"verified primary", `{"id":42,"login":"soda-tester","full_name":"Soda Tester"}`, `[{"email":"secondary@example.test","verified":true},{"email":"soda-tester@example.test","verified":true,"primary":true}]`, "soda-tester@example.test"},
		{"wrong user", `{"id":43,"login":"another-user"}`, `[]`, ""},
		{"missing primary", `{"id":42,"login":"soda-tester"}`, `[{"email":"soda-tester@example.test","verified":true}]`, ""},
		{"unverified primary", `{"id":42,"login":"soda-tester"}`, `[{"email":"soda-tester@example.test","primary":true}]`, ""},
		{"invalid email response", `{"id":42,"login":"soda-tester"}`, `{"email":"soda-tester@example.test"}`, ""},
		{"invalid email address", `{"id":42,"login":"soda-tester"}`, `[{"email":"Soda Tester <soda-tester@example.test>","verified":true,"primary":true}]`, ""},
		{"ambiguous primary", `{"id":42,"login":"soda-tester"}`, `[{"email":"one@example.test","verified":true,"primary":true},{"email":"two@example.test","verified":true,"primary":true}]`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emailRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.Header.Get("Authorization") != "token fixture" {
					t.Error("wrong native request")
				}
				switch r.URL.Path {
				case "/api/v1/user":
					_, _ = fmt.Fprint(w, tc.user)
				case "/api/v1/user/emails":
					emailRequests++
					_, _ = fmt.Fprint(w, tc.emails)
				default:
					t.Error("unexpected native operation")
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			user, email, err := New(server.URL).GitIdentity(t.Context(), "fixture", 42)
			if tc.wantEmail == "" {
				if !errors.Is(err, ErrInvalidResponse) || user != (User{}) || email != "" {
					t.Fatalf("denial returned attribution: %+v %q %v", user, email, err)
				}
			} else if err != nil || email != tc.wantEmail || user.ID != 42 || user.Name != "Soda Tester" {
				t.Fatalf("native attribution differs: %+v %q %v", user, email, err)
			}
			if tc.name == "wrong user" && emailRequests != 0 {
				t.Fatal("email lookup followed identity mismatch")
			}
		})
	}
}

func TestGitIdentityProviderFailure(t *testing.T) {
	for _, path := range []string{"/api/v1/user", "/api/v1/user/emails"} {
		t.Run(path, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == path {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				_, _ = fmt.Fprint(w, `{"id":42,"login":"soda-tester"}`)
			}))
			defer server.Close()
			user, email, err := New(server.URL).GitIdentity(t.Context(), "fixture", 42)
			var native *HTTPError
			if !errors.As(err, &native) || native.Status != http.StatusForbidden || user != (User{}) || email != "" {
				t.Fatalf("native failure was lost: %+v %q %v", user, email, err)
			}
		})
	}
}
