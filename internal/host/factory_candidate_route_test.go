package host

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

// The candidate routes must be reachable through the daemon: direct client
// or runtime calls do not prove dispatch, admission, clean-path or body-cap
// wiring.
func TestCandidateRoutesRejectUncleanRequests(t *testing.T) {
	d := testDaemonPtr(&noExec{}, Config{})
	for _, path := range []string{"/prepare-candidate", "/factory-candidate-inspect"} {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s accepted GET: %d", path, w.Code)
		}
		w = httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path+"?x=1", strings.NewReader("{}")))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s accepted query: %d", path, w.Code)
		}
	}
}

func candidateRoutePreparation() project.Preparation {
	return project.Preparation{
		ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Role: project.RoleReviewer,
		Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Approver: 7, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)},
		Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Approver: 9, EffectsDigest: strings.Repeat("b", 64)},
		SourceCommit: strings.Repeat("c", 40), SetupDigest: strings.Repeat("d", 64),
	}
}

func TestPrepareCandidateRouteReachesRuntime(t *testing.T) {
	n := &noExec{}
	d := testDaemonPtr(n, Config{})
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prepare-candidate", strings.NewReader(`{"preparation":{}}`)))
	if w.Code == http.StatusNotFound || n.called {
		t.Fatalf("invalid candidate: status=%d executed=%v", w.Code, n.called)
	}
	// A 512 KiB bundle (~700 KiB of JSON) must pass the route body cap and
	// reach the runtime; the 64 KiB default cap would refuse it before exec.
	bundle := bytes.Repeat([]byte("c"), project.MaxSourceBundle)
	body, err := json.Marshal(project.FactoryCandidate{Preparation: candidateRoutePreparation(), SourcePreparation: "f111111111111111111111111", Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) <= 65536 {
		t.Fatal("candidate body does not exceed the default route cap")
	}
	n.called = false
	w = httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prepare-candidate", bytes.NewReader(body)))
	if w.Code == http.StatusNotFound || !n.called {
		t.Fatalf("valid candidate did not reach runtime: status=%d executed=%v", w.Code, n.called)
	}
}

func TestFactoryCandidateInspectRoute(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	body, _ := json.Marshal(project.FactoryCandidateInspect{Project: "p123456789012345678901234", ID: strings.Repeat("a", 32)})
	r := httptest.NewRequest(http.MethodPost, "/factory-candidate-inspect", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || exec.called {
		t.Fatal("unknown candidate run was not a clean miss", w.Code)
	}
	exec.called = false
	r = httptest.NewRequest(http.MethodPost, "/factory-candidate-inspect", strings.NewReader(`{"project":"../../other","id":"short"}`))
	w = httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code == http.StatusOK || w.Code == http.StatusNotFound || exec.called {
		t.Fatal("untrusted candidate address reached native execution", w.Code)
	}
	bare := &Daemon{}
	w = httptest.NewRecorder()
	bare.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/factory-candidate-inspect", bytes.NewReader(body)))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("candidate inspect without runtime reported", w.Code)
	}
}
