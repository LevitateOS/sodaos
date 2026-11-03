package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
)

type issueWebSource struct {
	evidence control.AcceptanceEvidence
	err      error
	comments []string
	reads    int
	hook     func(context.Context, string, string)
}

func (s *issueWebSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, comments []string) (control.AcceptanceEvidence, error) {
	if s.hook != nil {
		s.hook(ctx, repository, issue)
	}
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

func issueAcceptanceBody(decisionID, predecessor string) string {
	return `{"decision_id":"` + decisionID + `","predecessor":"` + predecessor + `",` +
		`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","title_occurrence":"original","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
		`"sources":[{"id":"11","digest":"` + strings.Repeat("c", 64) + `","content_version":0}],` +
		`"prerequisites":[{"occurrence":"21","depends_on":"9","endpoint_repository":"7","endpoint_issue":"2","outcome":"result"}],` +
		`"resolutions":[{"id":"12","digest":"` + strings.Repeat("d", 64) + `","content_version":1}]}`
}

func TestSharedIssueAcceptanceAdmission(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("private PostgreSQL required")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	source := &issueWebSource{evidence: issueWebEvidence()}
	source.evidence.Dependencies = nil
	coordinator := control.NewCoordinator(db, nil, nil)
	coordinator.AcceptanceReads = source
	allowed, checks, denyAt := true, 0, 0
	denied := errors.New("declared current acceptance write authority denied")
	application := factoryservice.Acceptances{Writer: coordinator, Evidence: source, WriteAccess: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
		if admitted != actor || repo != repository {
			t.Fatal("acceptance changed canonical actor/repository", admitted, repo)
		}
		checks++
		if !allowed || checks == denyAt {
			return denied
		}
		return nil
	}}
	assertEvidenceScope := func(ctx context.Context, repo, issue string) {
		if repo != strconv.FormatInt(int64(repository), 10) || issue != "3" {
			t.Fatal("acceptance evidence changed scope", repo, issue)
		}
	}
	source.hook = assertEvidenceScope
	var in factoryservice.AcceptanceChange
	if err := json.Unmarshal([]byte(issueAcceptanceBody("d0123456789abcdef01234567", "")), &in); err != nil {
		t.Fatal(err)
	}
	in.Prerequisites = nil
	preview, err := application.Preview(ctx, actor, repository, "3", []string{"11"})
	if err != nil || preview.Repository != strconv.FormatInt(int64(repository), 10) || preview.Issue != "3" || len(preview.Comments) != 1 || preview.Comments[0].ID != "11" {
		t.Fatal("shared preview", preview, err)
	}
	encoded, _ := json.Marshal(preview)
	if strings.Contains(string(encoded), "poster") || strings.Contains(string(encoded), "body\"") {
		t.Fatal("preview exposed unselected native material", string(encoded))
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(d) ORDER BY id) FROM issue_acceptance_decisions d),
		 (SELECT jsonb_agg(to_jsonb(w) ORDER BY repository,issue,decision) FROM issue_acceptance_withdrawals w),
		 (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_runs),(SELECT count(*) FROM projects))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	source.hook = func(ctx context.Context, repo, issue string) { assertEvidenceScope(ctx, repo, issue); allowed = false }
	if result, err := application.Admit(ctx, actor, repository, "3", in); !errors.Is(err, denied) || result.DecisionID != "" {
		t.Fatal("late evidence permission loss admitted decision", result, err)
	}
	if _, err := db.AcceptanceDecision(ctx, in.DecisionID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("revoked evidence wrote acceptance", err)
	}
	allowed = true
	if result, err := application.Preview(ctx, actor, repository, "3", []string{"11"}); result != nil || !errors.Is(err, denied) {
		t.Fatal("late preview authority returned selected facts", result, err)
	}
	allowed = true
	source.hook = assertEvidenceScope
	cancelled, cancel := context.WithCancel(ctx)
	source.hook = func(ctx context.Context, repo, issue string) { assertEvidenceScope(ctx, repo, issue); cancel() }
	if _, err := application.Admit(cancelled, actor, repository, "3", in); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled evidence admitted acceptance", err)
	}
	source.hook = assertEvidenceScope
	missing := application
	missing.WriteAccess = nil
	if _, err := missing.Admit(ctx, actor, repository, "3", in); !errors.Is(err, factoryservice.ErrAcceptanceUnavailable) {
		t.Fatal("missing write authority admitted decision", err)
	}
	if snapshot() != before {
		t.Fatal("preview/refused/cancelled acceptance changed records")
	}
	result, err := application.Admit(ctx, actor, repository, "3", in)
	if err != nil || result.Depth != 1 || result.Head != in.DecisionID {
		t.Fatal("shared acceptance", result, err)
	}
	stored, err := db.AcceptanceDecision(ctx, in.DecisionID)
	if err != nil || stored.Repository != int64(repository) || stored.Approver != int64(actor) {
		t.Fatal("shared acceptance lost canonical attribution", stored, err)
	}
	if _, err := coordinator.AdmitAcceptance(ctx, stored, nil); !errors.Is(err, control.ErrIneffectiveAuthority) {
		t.Fatal("missing controller authorization admitted replay", err)
	}
	before = snapshot()
	source.evidence.Revision++
	source.evidence.Issue.ContentVer++
	if replay, err := application.Admit(ctx, actor, repository, "3", in); err != nil || replay.Head != in.DecisionID || snapshot() != before {
		t.Fatal("exact replay reapproved changed inputs", replay, err)
	}
	changed := in
	changed.ContentVersion++
	if _, err := application.Admit(ctx, actor, repository, "3", changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("changed inputs reused acceptance", err)
	}
	withdraw := factoryservice.AcceptanceWithdrawal{DecisionID: in.DecisionID}
	checks, denyAt = 0, 2
	if _, err := application.Withdraw(ctx, actor, repository, "3", withdraw); !errors.Is(err, denied) {
		t.Fatal("withdrawal bypassed final current authority", err)
	}
	denyAt = 0
	if snapshot() != before {
		t.Fatal("refused replay/withdrawal changed history")
	}
	withdrawn, err := application.Withdraw(ctx, actor, repository, "3", withdraw)
	if err != nil || !withdrawn.Withdrawn {
		t.Fatal("shared withdrawal", withdrawn, err)
	}
	if latched, by, err := db.AcceptanceWithdrawn(ctx, int64(repository), 3, in.DecisionID); err != nil || !latched || by != int64(actor) {
		t.Fatal("withdrawal lost canonical actor", latched, by, err)
	}
	if _, err := coordinator.WithdrawAcceptance(ctx, int64(repository), 3, in.DecisionID, int64(actor), nil); !errors.Is(err, control.ErrIneffectiveAuthority) {
		t.Fatal("withdrawal omitted controller authority", err)
	}
	before = snapshot()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	coordinator.Store = reopened
	allowed = false
	if _, err := application.Admit(ctx, actor, repository, "3", in); !errors.Is(err, denied) {
		t.Fatal("reopen bypassed current acceptance authority", err)
	}
	allowed = true
	if replay, err := application.Withdraw(ctx, actor, repository, "3", withdraw); err != nil || !replay.Withdrawn || snapshot() != before {
		t.Fatal("reopened withdrawal changed latch/history", replay, err)
	}
	var admitted atomic.Bool
	admitted.Store(true)
	var currentCalls, loseAt atomic.Int64
	shared := application
	shared.WriteAccess = func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
		if !admitted.Load() {
			return denied
		}
		return application.WriteAccess(ctx, selected, repo)
	}
	transport := factoryhttpapi.Acceptances{Service: shared,
		Host: collaboration.HostID(factory.NewID()), Actor: actor,
		Visibility: func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
			if selected != actor || repo != repository {
				return denied
			}
			return ctx.Err()
		},
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			check := currentCalls.Add(1)
			if !admitted.Load() || check == loseAt.Load() || r.Header.Get("Authorization") != "Bearer declared-acceptance-client" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
		AuthorityError: func(w http.ResponseWriter, err error) { w.WriteHeader(http.StatusForbidden) },
	}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	call := func(target *httptest.Server, method, path, data string, expected int) []byte {
		t.Helper()
		currentCalls.Store(0)
		req, err := http.NewRequestWithContext(ctx, method, target.URL+path, strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-acceptance-client")
		res, err := target.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != expected || (expected == 200 || expected == 201) && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("shared acceptance HTTP", method, path, res.StatusCode, string(body), err)
		}
		return body
	}
	base := "/api/repositories/9223372036854775807/factory/issues/3"
	preview, err = application.Preview(ctx, actor, repository, "3", []string{"11"})
	if err != nil {
		t.Fatal(err)
	}
	expectedPreview, _ := json.Marshal(preview)
	if body := call(server, http.MethodGet, base+"/acceptance-preview?comment_id=11", "", 200); strings.TrimSpace(string(body)) != string(expectedPreview) {
		t.Fatal("HTTP changed exact preview scope/selection", string(body))
	}
	encodedDecision, _ := json.Marshal(in)
	call(server, http.MethodPost, base+"/acceptances", string(encodedDecision), 201)
	if snapshot() != before {
		t.Fatal("HTTP replay rewrote captured acceptance after source changes")
	}
	fresh := in
	fresh.DecisionID, fresh.Predecessor = "d"+strings.Repeat("e", 24), in.DecisionID
	fresh.NativeRevision, fresh.ContentVersion = source.evidence.Revision, source.evidence.Issue.ContentVer
	encodedFresh, _ := json.Marshal(fresh)
	loseAt.Store(2) // Account changes after decoding, before the service command.
	call(server, http.MethodPost, base+"/acceptances", string(encodedFresh), 401)
	loseAt.Store(0)
	source.hook = func(ctx context.Context, repo, issue string) {
		assertEvidenceScope(ctx, repo, issue)
		admitted.Store(false)
	}
	call(server, http.MethodPost, base+"/acceptances", string(encodedFresh), 403)
	if _, err := reopened.AcceptanceDecision(ctx, fresh.DecisionID); !errors.Is(err, store.ErrNotFound) || snapshot() != before {
		t.Fatal("account withdrawal during evidence persisted a decision", err)
	}
	source.hook = assertEvidenceScope
	admitted.Store(true)
	first := call(server, http.MethodPost, base+"/acceptances", string(encodedFresh), 201)
	var accepted control.AcceptanceResult
	if json.Unmarshal(first, &accepted) != nil || accepted.Head != fresh.DecisionID || accepted.Depth != 2 {
		t.Fatal("HTTP did not admit the exact reviewed successor", string(first))
	}
	if stored, err := reopened.AcceptanceDecision(ctx, fresh.DecisionID); err != nil || stored.Approver != int64(actor) || stored.Repository != int64(repository) {
		t.Fatal("HTTP changed canonical decision attribution", stored, err)
	}
	if replay := call(server, http.MethodPost, base+"/acceptances", string(encodedFresh), 201); string(replay) != string(first) {
		t.Fatal("HTTP replay changed acceptance receipt", string(replay))
	}
	withdrawJSON := `{"decision_id":"` + fresh.DecisionID + `"}`
	call(server, http.MethodPost, base+"/withdrawal", withdrawJSON, 200)
	call(server, http.MethodPost, base+"/withdrawal", withdrawJSON, 200)
	if latched, by, err := reopened.AcceptanceWithdrawn(ctx, int64(repository), 3, fresh.DecisionID); err != nil || !latched || by != int64(actor) {
		t.Fatal("HTTP withdrawal lost canonical owner/latch", latched, by, err)
	}
	before = snapshot()
	for _, query := range []string{"?", "?unknown=1", "?comment_id=11&comment_id=11", "?comment_id=01", "?comment_id=" + strings.Repeat("1", 8193)} {
		call(server, http.MethodGet, base+"/acceptance-preview"+query, "", 400)
	}
	call(server, http.MethodGet, "/api/repositories/09223372036854775807/factory/issues/3/acceptance-preview", "", 404)
	call(server, http.MethodHead, base+"/acceptance-preview", "", 405)
	call(server, http.MethodPost, base+"/acceptances", string(encodedFresh)+string(encodedFresh), 400)
	call(server, http.MethodPost, base+"/acceptances", `{"approver":"9"}`, 400)
	call(server, http.MethodPost, base+"/acceptances", `{"decision_id":"`+strings.Repeat("x", 65536)+`"}`, 413)
	fresh.ContentVersion++
	changedJSON, _ := json.Marshal(fresh)
	call(server, http.MethodPost, base+"/acceptances", string(changedJSON), 409)
	allowed = false
	previousReads := source.reads
	call(server, http.MethodGet, base+"/acceptance-preview", "", 403)
	call(server, http.MethodPost, base+"/withdrawal", withdrawJSON, 403)
	if source.reads != previousReads {
		t.Fatal("denied HTTP acceptance reached evidence")
	}
	allowed = true
	loseAt.Store(2) // Current preview facts cannot reach a changed host account.
	if body := call(server, http.MethodGet, base+"/acceptance-preview", "", 401); len(body) != 0 {
		t.Fatal("late host withdrawal exposed acceptance preview", string(body))
	}
	loseAt.Store(0)
	source.err = errors.New("private declared evidence outage")
	if body := call(server, http.MethodGet, base+"/acceptance-preview", "", 503); strings.Contains(string(body), "private declared") {
		t.Fatal("HTTP exposed evidence failure detail", string(body))
	}
	source.err = nil
	missingAdmission := transport
	missingAdmission.Current = nil
	missingServer := httptest.NewServer(missingAdmission.Handler())
	t.Cleanup(missingServer.Close)
	call(missingServer, http.MethodGet, base+"/acceptance-preview", "", 503)
	if snapshot() != before {
		t.Fatal("HTTP previews/replay/refusals changed retained journals or work")
	}
}

func TestIssueAcceptanceUsesCanonicalScope(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("private PostgreSQL required")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	s, err := NewFactory(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `SELECT setval('collaboration_repositories_id_seq',9007199254740992,true),setval('collaboration_actors_id_seq',9007199254740992,true)`); err != nil {
		t.Fatal(err)
	}
	repository, err := s.API.FountainIdentities.LocalRepository(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := s.API.FountainIdentities.LocalActor(ctx, "1")
	if err != nil || actor <= 9007199254740991 {
		t.Fatal(actor, err)
	}
	if err := db.UpsertUser(ctx, store.User{ID: 1, Login: "alice", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	source := &issueWebSource{evidence: issueWebEvidence()}
	s.Coordinator.AcceptanceReads = source
	path := "/api/repositories/" + strconv.FormatInt(int64(repository), 10) + "/factory/issues/3"
	call := func(suffix, body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest(http.MethodPost, path+suffix, body, "alice"), func(in extensions.CallbackRequest) extensions.CallbackResponse {
			switch in.Operation {
			case extensions.OperationCurrentActor:
				return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}}
			case extensions.OperationRepository:
				if in.RepositoryID != "7" {
					return extensions.CallbackResponse{ErrorCode: "not_found"}
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "mapped", Permission: "write"}}
			default:
				return extensions.CallbackResponse{ErrorCode: "forbidden"}
			}
		})
		return w
	}
	decision := "d0123456789abcdef01234567"
	body := strings.ReplaceAll(issueAcceptanceBody(decision, ""), `"endpoint_repository":"7"`, `"endpoint_repository":"`+strconv.FormatInt(int64(repository), 10)+`"`)
	if response := call("/acceptances", body); response.Code != 201 {
		t.Fatal("canonical acceptance could not use native scope", response.Code, response.Body.String())
	}
	stored, err := db.AcceptanceDecision(ctx, decision)
	if err != nil || stored.Repository != int64(repository) || stored.Approver != int64(actor) {
		t.Fatal("acceptance confused canonical/native attribution", stored, err)
	}
	if response := call("/withdrawal", `{"decision_id":"`+decision+`"}`); response.Code != 200 {
		t.Fatal("canonical withdrawal", response.Code, response.Body.String())
	}
	if latched, by, err := db.AcceptanceWithdrawn(ctx, int64(repository), 3, decision); err != nil || !latched || by != int64(actor) {
		t.Fatal("native withdrawal lost canonical attribution", latched, by, err)
	}
}

func TestIssueAcceptanceJourney(t *testing.T) {
	s, source := issueWebFixture(t)
	decision := "d0123456789abcdef01234567"
	admit := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", issueAcceptanceBody(decision, ""), "alice")
	if admit.Code != 201 {
		t.Fatal(admit.Code, admit.Body.String())
	}
	var result control.AcceptanceResult
	decodeBody(t, admit, &result)
	if result.DecisionID != decision || result.Head != decision || result.Depth != 1 {
		t.Fatalf("acceptance result: %+v", result)
	}
	assertNoSecrets(t, admit.Body.String())
	if strings.Contains(admit.Body.String(), "command_id") {
		t.Fatal("acceptance returned a command receipt", admit.Body.String())
	}
	// Repeating an admitted decision reads its accepted inputs even after the
	// native issue changes. It must not accept the changed issue or advance twice.
	source.evidence.Revision = 42
	source.evidence.Issue.ContentVer++
	repeated := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", issueAcceptanceBody(decision, ""), "alice")
	if repeated.Code != 201 {
		t.Fatal("recorded decision required unchanged native inputs", repeated.Code, repeated.Body.String())
	}
	decodeBody(t, repeated, &result)
	stored, err := s.Store.AcceptanceDecision(t.Context(), decision)
	if err != nil || stored.NativeRev != 41 || stored.ContentVersion != 2 || result.Depth != 1 || result.Head != decision {
		t.Fatalf("repeat changed accepted inputs or advanced the chain: %+v %+v %v", stored, result, err)
	}
	changed := strings.Replace(issueAcceptanceBody(decision, ""), `"content_version":2`, `"content_version":3`, 1)
	conflict := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", changed, "alice")
	if conflict.Code != 409 {
		t.Fatal("changed inputs reused a recorded decision", conflict.Code, conflict.Body.String())
	}
	source.evidence.Issue.Visible = false
	hidden := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", issueAcceptanceBody(decision, ""), "alice")
	if hidden.Code != 409 || !strings.Contains(hidden.Body.String(), "issue_hidden") {
		t.Fatal("repeat bypassed current visibility", hidden.Code, hidden.Body.String())
	}
	source.evidence = issueWebEvidence()
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
		`{"decision_id":"`+decision+`"}`, "alice")
	if withdraw.Code != 200 {
		t.Fatal(withdraw.Code, withdraw.Body.String())
	}
	if strings.Contains(withdraw.Body.String(), "command_id") {
		t.Fatal("withdrawal returned a command receipt", withdraw.Body.String())
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
			issueAcceptanceBody("d0123456789abcdef01234567", ""), "alice")
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
		body := `{"decision_id":"d0123456789abcdef01234567",` +
			`"native_revision":41,"title_digest":"` + strings.Repeat("a", 64) + `","title_occurrence":"original","content_digest":"` + strings.Repeat("b", 64) + `","content_version":2,` +
			`"sources":[{"id":"99","digest":"` + strings.Repeat("c", 64) + `","content_version":0}]}`
		w := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", body, "alice")
		if w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
	})
	t.Run("approver-field", func(t *testing.T) {
		s, _ := issueWebFixture(t)
		body := `{"decision_id":"d0123456789abcdef01234567","approver":"9",` +
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
			issueAcceptanceBody("d0123456789abcdef01234567", ""), "alice")
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
			issueAcceptanceBody("d0123456789abcdef01234567", ""), "bob"), readOnly)
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
