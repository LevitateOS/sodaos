package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

type stubIntakeCoordinator struct {
	hint    control.IntakeHint
	control factory.IssueControl
	changed bool
	calls   int
	err     error
}

func (s *stubIntakeCoordinator) ObserveIssueEvent(_ context.Context, hint control.IntakeHint) (factory.IssueControl, bool, error) {
	s.hint = hint
	s.calls++
	return s.control, s.changed, s.err
}

func signIntake(secret, body string) string {
	sum := hmac.New(sha256.New, []byte(secret))
	_, _ = sum.Write([]byte(body))
	return hex.EncodeToString(sum.Sum(nil))
}

func intakeRequest(t *testing.T, event, delivery, body string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", event)
	r.Header.Set("X-Forgejo-Delivery", delivery)
	r.Header.Set("X-Forgejo-Signature", signIntake("intake-secret", body))
	return r
}

func TestFactoryIntakeOpenedCarriesCreatorAuthority(t *testing.T) {
	coordinator := &stubIntakeCoordinator{control: factory.IssueControl{
		Repository: 7, Issue: 3, Readiness: factory.ReadinessQueued,
	}, changed: true}
	handler := IntakeHandler{Coordinator: coordinator, Secret: []byte("intake-secret")}
	body := `{"action":"opened","number":3,"issue":{"index":3},` +
		`"repository":{"id":7,"permissions":{"push":true}},` +
		`"sender":{"id":5},"extra_future_field":{"nested":true}}`
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, intakeRequest(t, "issues", "delivery-1", body))
	if w.Code != http.StatusOK {
		t.Fatal("opened intake refused:", w.Code, w.Body.String())
	}
	if coordinator.calls != 1 {
		t.Fatal("coordinator not called once:", coordinator.calls)
	}
	hint := coordinator.hint
	if !hint.Created || hint.Creator != 5 || !hint.CreatorWrite || hint.Repository != 7 || hint.Issue != 3 {
		t.Fatal("creation hint wrong:", hint)
	}
	var outcome intakeOutcome
	if err := json.Unmarshal(w.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Readiness != factory.ReadinessQueued || !outcome.Changed {
		t.Fatal("intake outcome wrong:", outcome)
	}
}

func TestFactoryIntakeHints(t *testing.T) {
	coordinator := &stubIntakeCoordinator{}
	handler := IntakeHandler{Coordinator: coordinator, Secret: []byte("intake-secret")}
	edited := `{"action":"edited","number":3,"repository":{"id":7},"sender":{"id":5}}`
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, intakeRequest(t, "issues", "delivery-2", edited))
	if w.Code != http.StatusOK || coordinator.hint.Created {
		t.Fatal("edited hint wrong:", w.Code, coordinator.hint)
	}
	comment := `{"issue":{"index":4},"repository":{"id":7},"is_pull":false}`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, intakeRequest(t, "issue_comment", "delivery-3", comment))
	if w.Code != http.StatusOK || coordinator.hint.Issue != 4 {
		t.Fatal("comment hint wrong:", w.Code, coordinator.hint)
	}
	before := coordinator.calls
	pullComment := `{"issue":{"index":4},"repository":{"id":7},"is_pull":true}`
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, intakeRequest(t, "issue_comment", "delivery-4", pullComment))
	if w.Code != http.StatusOK || coordinator.calls != before {
		t.Fatal("pull comment must ignore without assessing:", w.Code, coordinator.calls)
	}
	var outcome intakeOutcome
	if err := json.Unmarshal(w.Body.Bytes(), &outcome); err != nil || !outcome.Ignored {
		t.Fatal("ignored outcome wrong:", outcome, err)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, intakeRequest(t, "push", "delivery-5", `{"ref":"refs/heads/main"}`))
	if w.Code != http.StatusOK || coordinator.calls != before {
		t.Fatal("push event must ignore:", w.Code, coordinator.calls)
	}
}

func TestFactoryIntakeRejectsForgedDeliveries(t *testing.T) {
	coordinator := &stubIntakeCoordinator{}
	handler := IntakeHandler{Coordinator: coordinator, Secret: []byte("intake-secret")}
	body := `{"action":"opened","number":3,"repository":{"id":7},"sender":{"id":5}}`
	forged := intakeRequest(t, "issues", "delivery-9", body)
	forged.Header.Set("X-Forgejo-Signature", signIntake("wrong-secret", body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, forged)
	if w.Code != http.StatusUnauthorized || coordinator.calls != 0 {
		t.Fatal("forged delivery assessed:", w.Code, coordinator.calls)
	}
	unsigned := intakeRequest(t, "issues", "delivery-9", body)
	unsigned.Header.Del("X-Forgejo-Signature")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, unsigned)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("unsigned delivery accepted:", w.Code)
	}
}

func TestFactoryIntakeFailures(t *testing.T) {
	coordinator := &stubIntakeCoordinator{}
	handler := IntakeHandler{Coordinator: coordinator, Secret: []byte("intake-secret")}
	bad := intakeRequest(t, "issues", "delivery-1", `{"action":"opened"}`)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, bad)
	if w.Code != http.StatusBadRequest {
		t.Fatal("malformed hint accepted:", w.Code)
	}
	inconsistent := intakeRequest(t, "issues", "delivery-1",
		`{"action":"edited","number":3,"issue":{"index":4},"repository":{"id":7}}`)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, inconsistent)
	if w.Code != http.StatusBadRequest {
		t.Fatal("inconsistent index accepted:", w.Code)
	}
	broken := intakeRequest(t, "issues", "delivery-1", `{"action":`)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, broken)
	if w.Code != http.StatusBadRequest {
		t.Fatal("broken JSON accepted:", w.Code)
	}
	coordinator.err = errors.New("observation unavailable")
	ok := intakeRequest(t, "issues", "delivery-6", `{"action":"edited","number":3,"repository":{"id":7}}`)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ok)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("assessment failure not retryable:", w.Code)
	}
	coordinator.err = nil
	unconfigured := IntakeHandler{Coordinator: coordinator}
	w = httptest.NewRecorder()
	unconfigured.ServeHTTP(w, ok)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("unconfigured intake served:", w.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/factory/intake", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, get)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("GET intake served:", w.Code)
	}
	huge := intakeRequest(t, "issues", "delivery-7", strings.Repeat("x", 70000))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, huge)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatal("oversized delivery accepted:", w.Code)
	}
}

func TestServiceReadinessSourceNilGuards(t *testing.T) {
	var source *ServiceReadinessSource
	if _, err := source.ReadAcceptanceEvidence(context.Background(), "7", "3", nil); err == nil {
		t.Fatal("nil source read evidence")
	}
	if _, _, err := source.ObserveNativeRevision(context.Background()); err == nil {
		t.Fatal("nil source observed revision")
	}
	source = &ServiceReadinessSource{}
	if _, err := source.ReadAcceptanceEvidence(context.Background(), "7", "3", nil); err == nil {
		t.Fatal("observerless source read evidence")
	}
}
