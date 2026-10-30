package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

type intakeRouteSource struct {
	evidence control.AcceptanceEvidence
}

func (s intakeRouteSource) ReadAcceptanceEvidence(context.Context, string, string, []string) (control.AcceptanceEvidence, error) {
	return s.evidence, nil
}

func intakeRouteServer(t *testing.T, secret string) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	db := postgresFixture(t)
	c := config.Config{
		Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/",
		ForgejoInternalURL: "http://127.0.0.1:3000",
		DatabaseDSNFile:    filepath.Join(dir, "soda.dsn"),
		HostSocket:         filepath.Join(dir, "host.sock"),
		IdentitySocket:     filepath.Join(dir, "identity.sock"),
		OperatorID:         1,
	}
	if secret != "" {
		path := filepath.Join(dir, "intake-secret")
		if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
			t.Fatal(err)
		}
		c.FactoryIntakeSecretFile = path
	}
	server := New(c, db)
	return server, func() {}
}

func TestFactoryIntakeRouteServesVerifiedDeliveries(t *testing.T) {
	server, done := intakeRouteServer(t, "intake-secret")
	defer done()
	digest := strings.Repeat("d", 64)
	server.Coordinator.AcceptanceReads = intakeRouteSource{evidence: control.AcceptanceEvidence{
		Issue:    control.AcceptanceIssueView{Index: "3", TitleDigest: digest, ContentDigest: digest, Visible: true},
		Revision: 12,
	}}
	body := `{"action":"edited","number":3,"repository":{"id":7},"sender":{"id":5}}`
	sum := hmac.New(sha256.New, []byte("intake-secret"))
	_, _ = sum.Write([]byte(body))
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("verified intake refused:", w.Code, w.Body.String())
	}
	var outcome struct {
		Repository string `json:"repository"`
		Issue      string `json:"issue"`
		Readiness  string `json:"readiness"`
		Changed    bool   `json:"changed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Repository != "7" || outcome.Issue != "3" ||
		outcome.Readiness != "not_authorized" || !outcome.Changed {
		t.Fatal("intake outcome wrong:", outcome)
	}
	again := httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
	server.ServeHTTP(again, r)
	if again.Code != http.StatusOK {
		t.Fatal("duplicate intake refused:", again.Code)
	}
	t.Run("repository creation publishes one question without starting intake", func(t *testing.T) {
		deliver := func(action, delivery string, signed bool) int {
			body := `{"action":"` + action + `","repository":{"id":17,"created_at":"2026-10-29T20:00:00Z"}}`
			r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
			r.Header.Set("X-Forgejo-Event", "repository")
			r.Header.Set("X-Forgejo-Delivery", delivery)
			if signed {
				sum := hmac.New(sha256.New, []byte("intake-secret"))
				_, _ = sum.Write([]byte(body))
				r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
			}
			w := httptest.NewRecorder()
			server.ServeHTTP(w, r)
			return w.Code
		}
		if deliver("created", "unsigned", false) != http.StatusUnauthorized {
			t.Fatal("unsigned repository event was admitted")
		}
		if deliver("deleted", "deleted", true) != http.StatusOK {
			t.Fatal("unrelated repository action was not ignored")
		}
		query := factory.FeedQuery{Repositories: []int64{17}, Limit: 20}
		before, _, _, err := server.Store.SubscribedFeed(context.Background(), query)
		if err != nil || len(before) != 0 {
			t.Fatalf("unadmitted creation published a post: %+v %v", before, err)
		}
		for _, delivery := range []string{"created", "created", "redelivery"} {
			if deliver("created", delivery, true) != http.StatusOK {
				t.Fatal("signed repository event was refused")
			}
		}
		entries, _, _, err := server.Store.SubscribedFeed(context.Background(), query)
		if err != nil || len(entries) != 1 {
			t.Fatalf("opening question missing or duplicated: %+v %v", entries, err)
		}
		post, err := server.Store.FeedPost(context.Background(), entries[0].ID)
		if err != nil || post.Actor != "factory" || post.AuthorName != "Factory" || post.Body != store.RepositoryOpeningQuestion ||
			post.Payload.Communication.Kind != "ask" || post.Payload.Communication.QuestionState != "open" || post.Record.ParentID != "" {
			t.Fatalf("opening post attribution or content wrong: %+v %v", post, err)
		}
		pending, err := server.Store.PendingPostIntakes(context.Background(), 20)
		if err != nil || len(pending) != 0 {
			t.Fatalf("Factory question started human intake: %+v %v", pending, err)
		}
	})
}

func TestFactoryIntakeRouteRefusesUnauthenticated(t *testing.T) {
	server, done := intakeRouteServer(t, "intake-secret")
	defer done()
	body := `{"action":"edited","number":3,"repository":{"id":7}}`
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("unsigned intake served:", w.Code)
	}
}

func TestFactoryIntakeRouteUnavailableWithoutSecret(t *testing.T) {
	server, done := intakeRouteServer(t, "")
	defer done()
	body := `{"action":"edited","number":3,"repository":{"id":7}}`
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("secretless intake served:", w.Code)
	}
}
