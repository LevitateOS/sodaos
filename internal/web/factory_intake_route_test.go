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
	"github.com/levitateos/sodaos/internal/factory/control"
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
	db := postgresFixture(t, nil)
	c := config.Config{
		Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/",
		ForgejoInternalURL: "http://127.0.0.1:3000",
		DatabaseDSNFile:    filepath.Join(dir, "soda.dsn"),
		HostSocket:         filepath.Join(dir, "host.sock"),
		IdentitySocket:     filepath.Join(dir, "identity.sock"),
		GrantKeyFile:       filepath.Join(dir, "grant-key"),
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
