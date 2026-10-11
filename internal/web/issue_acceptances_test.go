package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

type issueWebSource struct {
	evidence control.AcceptanceEvidence
	err      error
	comments []string
	reads    int
}

func (s *issueWebSource) ReadAcceptanceEvidence(_ context.Context, _, _ string, comments []string) (control.AcceptanceEvidence, error) {
	s.reads++
	s.comments = append([]string(nil), comments...)
	if s.err != nil {
		return control.AcceptanceEvidence{}, s.err
	}
	evidence := s.evidence
	evidence.Comments = nil
	selected := make(map[string]bool, len(comments))
	for _, id := range comments {
		selected[id] = true
	}
	for _, comment := range s.evidence.Comments {
		if selected[comment.ID] {
			evidence.Comments = append(evidence.Comments, comment)
		}
	}
	return evidence, nil
}

func issueWebEvidence() control.AcceptanceEvidence {
	return control.AcceptanceEvidence{
		Revision: 41,
		Issue: control.AcceptanceIssueView{
			Index:       "3",
			TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVer: 2,
			TitleOccurrence: factory.OriginalOccurrence, PosterID: "1", Visible: true,
		},
		Comments: []control.AcceptanceComment{
			{ID: "11", Digest: strings.Repeat("c", 64), ContentVer: 0, Visible: true},
			{ID: "12", Digest: strings.Repeat("d", 64), ContentVer: 1, Visible: true},
		},
		Dependencies: []control.AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
}

func TestIssueAcceptancePreviewProxyIsReadOnlyAndBounded(t *testing.T) {
	s, source := issueWebFixture(t)
	proxy := nativeProductProxy(t, s, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"})
	response := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", "/api/repositories/7/factory/issues/3/acceptance-preview?comment_id=11", nil)
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("preview: %d %s", response.StatusCode, body)
	}
	var view struct {
		Repository      string `json:"repository"`
		Issue           string `json:"issue"`
		TitleDigest     string `json:"title_digest"`
		TitleOccurrence string `json:"title_occurrence"`
		BodyDigest      string `json:"body_digest"`
		ContentVersion  int    `json:"content_version"`
		NativeRevision  int64  `json:"native_revision"`
		Comments        []struct {
			ID             string `json:"id"`
			Digest         string `json:"digest"`
			ContentVersion int    `json:"content_version"`
		} `json:"comments"`
		Dependencies []struct {
			Occurrence string `json:"occurrence"`
			DependsOn  string `json:"depends_on"`
		} `json:"dependencies"`
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecrets(t, string(raw))
	if strings.Contains(string(raw), "private-issue-body") {
		t.Fatal("preview exposed native text")
	}
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatal(err)
	}
	if view.Repository != "7" || view.Issue != "3" || view.TitleDigest != strings.Repeat("a", 64) || view.TitleOccurrence != factory.OriginalOccurrence || view.BodyDigest != strings.Repeat("b", 64) || view.ContentVersion != 2 || view.NativeRevision != 41 {
		t.Fatalf("preview identity/evidence: %+v", view)
	}
	if len(view.Comments) != 1 || view.Comments[0].ID != "11" || view.Comments[0].Digest != strings.Repeat("c", 64) || view.Comments[0].ContentVersion != 0 {
		t.Fatalf("selected comments: %+v", view.Comments)
	}
	if len(view.Dependencies) != 1 || view.Dependencies[0].Occurrence != "21" || view.Dependencies[0].DependsOn != "9" {
		t.Fatalf("dependency edges: %+v", view.Dependencies)
	}
	if source.reads != 1 || len(source.comments) != 1 || source.comments[0] != "11" {
		t.Fatalf("native read selection: reads=%d comments=%v", source.reads, source.comments)
	}
	for _, path := range []string{
		"/api/repositories/7/factory/issues/3/acceptance-preview?comment_id=11&comment_id=11",
		"/api/repositories/7/factory/issues/3/acceptance-preview?unknown=1",
		"/api/repositories/7/factory/issues/3/acceptance-preview?comment_id=0",
	} {
		bad := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", path, nil)
		if bad.StatusCode != 400 {
			t.Fatalf("%s: %d", path, bad.StatusCode)
		}
	}
	tooMany := make([]string, factory.MaxAcceptedSources+1)
	for i := range tooMany {
		tooMany[i] = "comment_id=" + strconv.Itoa(i+1)
	}
	bad := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", "/api/repositories/7/factory/issues/3/acceptance-preview?"+strings.Join(tooMany, "&"), nil)
	if bad.StatusCode != 400 {
		t.Fatalf("too many selected comments: %d", bad.StatusCode)
	}
	if source.reads != 1 {
		t.Fatalf("invalid query reached native reader: %d", source.reads)
	}

	if _, err := s.Store.AcceptanceHead(t.Context(), 7, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("preview changed acceptance state: %v", err)
	}

	source.evidence.Issue.Visible = false
	hidden := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", "/api/repositories/7/factory/issues/3/acceptance-preview", nil)
	if hidden.StatusCode != 409 {
		t.Fatalf("hidden issue: %d", hidden.StatusCode)
	}
	source.evidence.Issue.Visible = true
	source.err = &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	incomplete := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", "/api/repositories/7/factory/issues/3/acceptance-preview", nil)
	if incomplete.StatusCode != 503 {
		t.Fatalf("incomplete evidence: %d", incomplete.StatusCode)
	}
	source.err = nil
	source.evidence.Dependencies[0].Visible = false
	hiddenEdge := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", "/api/repositories/7/factory/issues/3/acceptance-preview", nil)
	if hiddenEdge.StatusCode != 409 {
		t.Fatalf("hidden dependency edge: %d", hiddenEdge.StatusCode)
	}
	readOnlyProxy := nativeProductProxyForActorWithCallback(t, s, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, "bob", func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation == extensions.OperationRepository {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
		}
		return extensions.CallbackResponse{}
	})
	denied := nativeProductRequestForActor(t, readOnlyProxy, s.Config.ForgejoURL, "bob", "GET", "/api/repositories/7/factory/issues/3/acceptance-preview", nil)
	if denied.StatusCode != 403 {
		t.Fatalf("read-only caller: %d", denied.StatusCode)
	}
	if source.reads != 4 {
		t.Fatalf("read-only caller reached native reader: %d", source.reads)
	}
}

func nativeProductRequestForActor(t *testing.T, proxy *httptest.Server, origin, actor, method, path string, body []byte) *http.Response {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, proxy.URL+strings.TrimPrefix(path, "/api"), strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	r.Header = nativeProductHeadersForActor(method, origin, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, actor)
	response, err := proxy.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
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
		`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","title_occurrence":"original","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
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
			`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","title_occurrence":"original","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
			`"sources":[{"id":"99","digest":"` + strings.Repeat("c", 64) + `","content_version":0}]}`
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", body, "alice")
		if w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("approver-field", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		body := `{"command_id":"` + factory.NewID() + `","decision_id":"d0123456789abcdef01234567","approver":"9",` +
			`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","title_occurrence":"original","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2}`
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
