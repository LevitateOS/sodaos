package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

type issueWebSource struct {
	evidence control.AcceptanceEvidence
	err      error
}

func (s *issueWebSource) ReadAcceptanceEvidence(context.Context, string, string, []string) (control.AcceptanceEvidence, error) {
	return s.evidence, s.err
}

func issueWebEvidence() control.AcceptanceEvidence {
	return control.AcceptanceEvidence{
		Revision: 41,
		Issue: control.AcceptanceIssueView{Index: "3",
			TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVer: 2,
			PosterID: "1", Verified: true, FirstCreated: true, Visible: true},
		Comments: []control.AcceptanceComment{
			{ID: "11", Digest: strings.Repeat("c", 64), ContentVer: 0, Visible: true},
			{ID: "12", Digest: strings.Repeat("d", 64), ContentVer: 1, Visible: true},
		},
		Dependencies: []control.AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
}

func issueWebFixture(t *testing.T) (*Server, *issueWebSource) {
	t.Helper()
	s := lifecycleWebFixture(t)
	source := &issueWebSource{evidence: issueWebEvidence()}
	s.Coordinator.AcceptanceReads = source
	return s, source
}

func issueAcceptanceBody(commandID, decisionID, predecessor string) string {
	return `{"command_id":"` + commandID + `","decision_id":"` + decisionID + `","predecessor":"` + predecessor + `",` +
		`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
		`"sources":[{"id":"11","digest":"` + strings.Repeat("c", 64) + `","content_version":0}],` +
		`"prerequisites":[{"occurrence":"21","depends_on":"9","endpoint_repository":"7","endpoint_issue":"2","outcome":"result"}],` +
		`"resolutions":[{"id":"12","digest":"` + strings.Repeat("d", 64) + `","content_version":1}]}`
}

func TestIssueAcceptanceJourney(t *testing.T) {
	s, _ := issueWebFixture(t)
	decision := "d0123456789abcdef01234567"
	admit := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", issueAcceptanceBody(factory.NewID(), decision, ""), "alice")
	if admit.Code != 201 {
		t.Fatal(admit.Code, admit.Body.String())
	}
	var receipt control.AcceptanceReceipt
	decodeBody(t, admit, &receipt)
	if receipt.DecisionID != decision || receipt.Head != decision || receipt.Depth != 1 {
		t.Fatalf("receipt: %+v", receipt)
	}
	assertNoSecrets(t, admit.Body.String())
	view := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/3", "", "alice")
	if view.Code != 200 {
		t.Fatal(view.Code, view.Body.String())
	}
	var seen struct {
		Status struct {
			Acceptance *factory.Acceptance `json:"acceptance"`
			Valid      bool                `json:"valid"`
		} `json:"status"`
	}
	decodeBody(t, view, &seen)
	if !seen.Status.Valid || seen.Status.Acceptance == nil || seen.Status.Acceptance.Approver != 1 {
		t.Fatalf("view: %+v", seen.Status)
	}
	withdraw := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/withdrawal",
		`{"command_id":"`+factory.NewID()+`","decision_id":"`+decision+`"}`, "alice")
	if withdraw.Code != 200 {
		t.Fatal(withdraw.Code, withdraw.Body.String())
	}
	stale := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/3", "", "alice")
	if stale.Code != 200 {
		t.Fatal(stale.Code, stale.Body.String())
	}
	var after struct {
		Status struct {
			Valid     bool     `json:"valid"`
			Withdrawn bool     `json:"withdrawn"`
			Reasons   []string `json:"reasons"`
		} `json:"status"`
	}
	decodeBody(t, stale, &after)
	if after.Status.Valid || !after.Status.Withdrawn {
		t.Fatalf("withdrawn: %+v", after.Status)
	}
}

func TestIssueAcceptanceRefusals(t *testing.T) {
	t.Run("stale-screen", func(t *testing.T) {
		s, source := issueWebFixture(t)
		source.evidence.Revision = 42
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances",
			issueAcceptanceBody(factory.NewID(), "d0123456789abcdef01234567", ""), "alice")
		if w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
		var refusal struct {
			Error string `json:"error"`
		}
		decodeBody(t, w, &refusal)
		if refusal.Error != control.RefusalStaleEvidence {
			t.Fatalf("refusal: %+v", refusal)
		}
	})
	t.Run("fabricated-source", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		body := `{"command_id":"` + factory.NewID() + `","decision_id":"d0123456789abcdef01234567",` +
			`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
			`"sources":[{"id":"99","digest":"` + strings.Repeat("c", 64) + `","content_version":0}]}`
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", body, "alice")
		if w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("approver-field", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		body := `{"command_id":"` + factory.NewID() + `","decision_id":"d0123456789abcdef01234567","approver":"9",` +
			`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2}`
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", body, "alice")
		if w.Code != 400 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("unavailable", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		s.Coordinator.AcceptanceReads = nil
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances",
			issueAcceptanceBody(factory.NewID(), "d0123456789abcdef01234567", ""), "alice")
		if w.Code != 503 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("read-only", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		readOnly := func(in extensions.CallbackRequest) extensions.CallbackResponse {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
		}
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/repositories/7/factory/issues/3/acceptances",
			issueAcceptanceBody(factory.NewID(), "d0123456789abcdef01234567", ""), "bob"), readOnly)
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("unknown-issue", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		w := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/9", "", "alice")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var seen struct {
			Status struct {
				Valid   bool     `json:"valid"`
				Reasons []string `json:"reasons"`
			} `json:"status"`
		}
		decodeBody(t, w, &seen)
		if seen.Status.Valid || len(seen.Status.Reasons) != 1 || seen.Status.Reasons[0] != "no_acceptance" {
			t.Fatalf("view: %+v", seen.Status)
		}
		bad := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/0", "", "alice")
		if bad.Code != 404 {
			t.Fatal(bad.Code, bad.Body.String())
		}
	})
}
