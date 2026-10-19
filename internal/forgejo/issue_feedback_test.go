package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssueFeedbackReconcilesLostReplyWithoutEditingRequirements(t *testing.T) {
	marker := "<!-- soda-feed:" + strings.Repeat("a", 32) + ":r0 -->"
	var saved string
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token synthetic-feedback-token" {
			t.Fatal("missing native actor authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/user":
			_ = json.NewEncoder(w).Encode(User{ID: 91, Login: "worker"})
		case r.URL.Path == "/api/v1/repositories/7":
			_ = json.NewEncoder(w).Encode(Repository{ID: 7, Name: "demo", Owner: User{Login: "owner"}})
		case r.URL.Path == "/api/v1/repos/owner/demo/issues/55/comments" && r.Method == "GET":
			if saved == "" {
				_, _ = w.Write([]byte("[]"))
			} else {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 12, "body": saved, "user": User{ID: 91}}})
			}
		case r.URL.Path == "/api/v1/repos/owner/demo/issues/55/comments" && r.Method == "POST":
			posts++
			var in struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatal(err)
			}
			saved = in.Body
			w.WriteHeader(503) // The comment committed, but its reply was lost.
		default:
			t.Errorf("unexpected native mutation or route: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	credential := filepath.Join(t.TempDir(), "actor")
	if err := os.WriteFile(credential, []byte("synthetic-feedback-token"), 0600); err != nil {
		t.Fatal(err)
	}
	w := NewIssueGraphWriter(&graphClientFixture{}, New(server.URL), credential)
	if err := w.DeliverIssueFeedback(context.Background(), 7, 55, 91, marker, "Please fix the filter."); err == nil {
		t.Fatal("lost reply reported confirmed")
	}
	if err := w.DeliverIssueFeedback(context.Background(), 7, 55, 91, marker, "Please fix the filter."); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || saved != marker+"\nPlease fix the filter." {
		t.Fatalf("feedback was duplicated or changed: %d", posts)
	}
}
