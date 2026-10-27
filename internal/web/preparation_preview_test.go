package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

type preparationPreviewFixture struct {
	server        *Server
	requirement   project.RequirementDecision
	states        map[string]project.PrepareState
	files         map[string][]byte
	contexts      []project.FactoryPreparationContextRequest
	administrator bool
	accessCalls   int
	missingRole   string
	delay         time.Duration
	failContext   bool
	hangContext   bool
	// badBatch, when non-empty, is served as the raw HTTP 200 body for
	// every multi-path batch while single-path reads stay valid. It
	// proves a malformed batch fails closed instead of halving onto
	// the servable singles.
	badBatch string
}

func newPreparationPreviewFixture(t *testing.T) *preparationPreviewFixture {
	t.Helper()
	return newPreparationPreviewFixtureFiles(t, map[string][]byte{
		"setup.sh": []byte("echo private-setup-content\n"),
		"check.sh": []byte("echo private-check-content\n"),
	})
}

func newPreparationPreviewFixtureFiles(t *testing.T, files map[string][]byte) *preparationPreviewFixture {
	t.Helper()
	return newPreparationPreviewFixtureFilesSelected(t, files, nil)
}

// newPreparationPreviewFixtureFilesSelected admits a requirement over the
// selected paths only, while the native peer still serves every file in
// files as setup, check or selected content. A nil selection selects every
// file, preserving the single-selection behavior for existing callers.
func newPreparationPreviewFixtureFilesSelected(t *testing.T, files map[string][]byte, selected []string) *preparationPreviewFixture {
	t.Helper()
	s, _ := preparationFixture(t)
	s.Config.OperatorID = 99 // Require each actor's actual native Project privilege.
	for i, login := range []string{"alice", "bob"} {
		require.NoError(t, s.Store.Join(t.Context(), preparationProject, int64(i+1), login))
	}
	f := &preparationPreviewFixture{
		server: s, states: map[string]project.PrepareState{},
		files: files,
	}
	f.requirement = project.RequirementDecision{
		ID: "d111111111111111111111111", Project: preparationProject, Approver: 1,
		SourceCommit: strings.Repeat("c", 40), SetupDigest: project.SetupDigestOf(f.files),
		InputsDigest: strings.Repeat("d", 64), SourceInputs: map[string]string{},
	}
	if selected == nil {
		for path := range f.files {
			selected = append(selected, path)
		}
	}
	for _, path := range selected {
		content, found := f.files[path]
		require.True(t, found, "selected path must exist in fixture files")
		digest := sha256.Sum256(content)
		f.requirement.SourceInputs[path] = hex.EncodeToString(digest[:])
	}
	require.NoError(t, s.Store.AdmitRequirementDecision(t.Context(), f.requirement))
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f" + strings.Repeat(string(rune('1'+i)), 24)
		zero := 0
		state := project.PrepareState{
			ID: id, Project: preparationProject, Role: role, Phase: project.PrepareReady,
			Container: strings.Repeat("a", 64), SourceCommit: f.requirement.SourceCommit,
			SetupDigest: f.requirement.SetupDigest, Ready: true, SetupExit: &zero, CheckExit: &zero,
			Tools:  []project.ResolvedTool{{Name: "git", Path: "/usr/bin/git", Version: "git version 2.47.3"}},
			Output: "secret-native-setup-log",
		}
		f.states[id] = state
		_, _, err := s.Store.AdmitPreparation(t.Context(), project.StoredPreparation{
			Preparation: project.Preparation{
				ID: id, Project: preparationProject, Role: role, SourceCommit: state.SourceCommit,
				SetupDigest: state.SetupDigest, Requirements: f.requirement.Ref(1),
				Approval: project.AdminApproval{ID: "d222222222222222222222222", Revision: 1, Approver: 2, EffectsDigest: strings.Repeat("e", 64)},
				Tools:    []string{"git"}, Credential: "private-credential-reference",
			}, State: state,
		})
		require.NoError(t, err)
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, r.Method)
		var response any
		switch r.URL.Path {
		case "/prepare-inspect":
			var in project.PrepareInspect
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			require.Equal(t, preparationProject, in.Project)
			state, found := f.states[in.ID]
			require.True(t, found)
			if state.Role == f.missingRole {
				return &http.Response{StatusCode: 404, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			response = state
		case "/prepare-context":
			if f.hangContext {
				<-r.Context().Done()
				return &http.Response{StatusCode: 500, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			if f.failContext {
				return &http.Response{StatusCode: 500, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			if f.delay > 0 {
				select {
				case <-time.After(f.delay):
				case <-r.Context().Done():
					return &http.Response{StatusCode: 500, Body: http.NoBody, Header: make(http.Header)}, nil
				}
			}
			var in project.FactoryPreparationContextRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			state, found := f.states[in.ID]
			require.True(t, found)
			require.Equal(t, preparationProject, in.Project)
			require.Equal(t, f.requirement.SourceCommit, in.SourceCommit)
			require.Equal(t, in.SourceCommit, in.ApprovedBase)
			require.Equal(t, in.SourceCommit, in.DiffBase)
			require.Equal(t, in.SourceCommit, in.Candidate)
			require.True(t, in.NotAfter.After(time.Now()))
			require.LessOrEqual(t, time.Until(in.NotAfter), 30*time.Second)
			f.contexts = append(f.contexts, in)
			if f.badBatch != "" && len(in.Paths) > 1 {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(f.badBatch))), Header: make(http.Header)}, nil
			}
			material := project.FactoryPreparationContext{
				Project: in.Project, ID: in.ID, Role: state.Role, SourceCommit: in.SourceCommit,
				ApprovedBase: in.ApprovedBase, DiffBase: in.DiffBase, Candidate: in.Candidate,
				Setup: f.files["setup.sh"], Check: f.files["check.sh"],
			}
			// Serve the requested paths plus the root AGENTS.md the real
			// native always selects, and refuse exactly like the real
			// route: any over-file, over-count or over-aggregate closure
			// is answered with the typed HTTP 413 signal before sending
			// bytes, never with an oversized 200 the native cannot emit.
			seen := map[string]bool{}
			var served []string
			for _, path := range in.Paths {
				if !seen[path] {
					seen[path] = true
					served = append(served, path)
				}
			}
			if _, found := f.files["AGENTS.md"]; found && !seen["AGENTS.md"] {
				served = append(served, "AGENTS.md")
			}
			total := len(material.Setup) + len(material.Check)
			refuse := len(served) > project.MaxFactoryContextFiles
			for _, path := range served {
				content := f.files[path]
				if len(content) > project.MaxFactoryContextFileBytes {
					refuse = true
				}
				total += len(content)
			}
			if total > project.MaxFactoryContextBytes {
				refuse = true
			}
			if refuse {
				return &http.Response{StatusCode: 413, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			for _, path := range served {
				material.Files = append(material.Files, project.FactoryContextFile{Path: path, Content: f.files[path]})
			}
			response = material
		case "/project-access":
			f.accessCalls++
			var in project.ProjectAccessRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			require.Equal(t, preparationProject, in.Project)
			response = project.ProjectAccessStatus{Project: in.Project, Login: in.Login, Identity: in.Identity, Administrator: &f.administrator}
		default:
			t.Fatalf("unexpected native path: %s", r.URL.Path)
		}
		data, err := json.Marshal(response)
		require.NoError(t, err)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}
	return f
}

type preparationPreviewResponse struct {
	Requirements string `json:"requirements"`
	Approval     string `json:"approval"`
	Preview      struct {
		Requirements *struct {
			SourceCommit string `json:"source_commit"`
			SetupDigest  string `json:"setup_digest"`
			InputsDigest string `json:"inputs_digest"`
			Predecessor  string `json:"predecessor"`
			SourceInputs []struct {
				Path   string `json:"path"`
				Digest string `json:"digest"`
			} `json:"source_inputs"`
		} `json:"requirements"`
		Approval *struct {
			Requirement     string                 `json:"requirement"`
			Predecessor     string                 `json:"predecessor"`
			EffectsDigest   string                 `json:"effects_digest"`
			ReadinessDigest string                 `json:"readiness_digest"`
			Roles           []project.PrepareState `json:"roles"`
		} `json:"approval"`
		Reason string `json:"reason"`
	} `json:"decision_preview"`
}

func preparationPreviewRead(t *testing.T, f *preparationPreviewFixture) preparationPreviewResponse {
	t.Helper()
	proxy := nativeProductProxy(t, f.server, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"})
	response := nativeProductRequest(t, proxy, f.server.Config.ForgejoURL, "GET", "/api/environments/"+preparationProject+"/preparation", nil)
	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assertNoSecrets(t, string(body))
	require.NotContains(t, string(body), "private-setup-content")
	require.NotContains(t, string(body), "private-check-content")
	var view preparationPreviewResponse
	require.NoError(t, json.Unmarshal(body, &view))
	return view
}

func preparationPreviewAction(t *testing.T, f *preparationPreviewFixture, endpoint, actor, permission string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		require.Equal(t, extensions.OperationRepository, in.Operation)
		require.Equal(t, "7", in.RepositoryID)
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "prep", Permission: permission}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, f.server, w, apiTestRequest("POST", "/api/environments/"+preparationProject+"/preparation/"+endpoint, string(body), actor), callback)
	assertNoSecrets(t, w.Body.String())
	return w
}

func previewAcceptance(f *preparationPreviewFixture, predecessor string) map[string]any {
	return map[string]any{
		"command_id": factory.NewID(), "decision_id": "d333333333333333333333333", "predecessor": predecessor,
		"source_commit": f.requirement.SourceCommit, "setup_digest": f.requirement.SetupDigest,
		"inputs_digest": f.requirement.InputsDigest, "source_inputs": maps.Clone(f.requirement.SourceInputs),
	}
}

func previewApproval(view preparationPreviewResponse, predecessor string) map[string]any {
	return map[string]any{
		"command_id": factory.NewID(), "decision_id": "d444444444444444444444444", "action": "approve",
		"predecessor": predecessor, "requirement": view.Preview.Approval.Requirement,
		"effects_digest": view.Preview.Approval.EffectsDigest, "readiness_digest": view.Preview.Approval.ReadinessDigest, "verified": true,
	}
}

func TestPreparationDecisionPreviewProxyJourney(t *testing.T) {
	f := newPreparationPreviewFixture(t)
	view := preparationPreviewRead(t, f)
	require.Empty(t, view.Preview.Reason)
	require.NotNil(t, view.Preview.Requirements)
	require.NotNil(t, view.Preview.Approval)
	require.Equal(t, f.requirement.ID, view.Preview.Requirements.Predecessor)
	require.Equal(t, f.requirement.SourceCommit, view.Preview.Requirements.SourceCommit)
	require.Equal(t, f.requirement.SetupDigest, view.Preview.Requirements.SetupDigest)
	require.Equal(t, f.requirement.InputsDigest, view.Preview.Requirements.InputsDigest)
	selected := map[string]string{}
	paths := []string{}
	for _, input := range view.Preview.Requirements.SourceInputs {
		selected[input.Path] = input.Digest
		paths = append(paths, input.Path)
	}
	require.Equal(t, f.requirement.SourceInputs, selected)
	require.Equal(t, []string{"check.sh", "setup.sh"}, paths)
	// The batched reader confirms each role's whole selection in one call
	// (both files fit the native per-request count and aggregate bounds),
	// so two files x two roles = two reads: each role reads its exact
	// prepared checkout once.
	require.Len(t, f.contexts, 2)
	coderID := "f" + strings.Repeat("1", 24)
	reviewerID := "f" + strings.Repeat("2", 24)
	expectedReads := []struct {
		id    string
		role  string
		paths []string
	}{
		{coderID, project.RoleCoder, paths},
		{reviewerID, project.RoleReviewer, paths},
	}
	covered := map[string][]string{}
	for i, request := range f.contexts {
		require.Equal(t, expectedReads[i].id, request.ID)
		require.Equal(t, expectedReads[i].paths, request.Paths)
		state, found := f.states[request.ID]
		require.True(t, found)
		require.Equal(t, expectedReads[i].role, state.Role)
		covered[request.ID] = append(covered[request.ID], request.Paths...)
	}
	require.Equal(t, map[string][]string{coderID: paths, reviewerID: paths}, covered)
	require.Len(t, view.Preview.Approval.Roles, 2)
	require.Equal(t, project.RoleCoder, view.Preview.Approval.Roles[0].Role)
	require.Equal(t, project.RoleReviewer, view.Preview.Approval.Roles[1].Role)
	for _, role := range view.Preview.Approval.Roles {
		require.True(t, role.Ready)
		require.Equal(t, 0, *role.SetupExit)
		require.Equal(t, 0, *role.CheckExit)
		require.Equal(t, "/usr/bin/git", role.Tools[0].Path)
	}
	require.True(t, project.ValidDigest(view.Preview.Approval.EffectsDigest))
	require.True(t, project.ValidDigest(view.Preview.Approval.ReadinessDigest))
	require.NotEqual(t, view.Preview.Approval.EffectsDigest, view.Preview.Approval.ReadinessDigest)
	require.Empty(t, view.Preview.Approval.Predecessor)
	_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound) // Reading creates no approval.

	acceptance := previewAcceptance(f, view.Preview.Requirements.Predecessor)
	acceptance["source_inputs"] = selected
	w := preparationPreviewAction(t, f, "acceptances", "alice", "write", acceptance)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	stored, err := f.server.Store.RequirementDecision(t.Context(), acceptance["decision_id"].(string))
	require.NoError(t, err)
	require.Equal(t, f.requirement.SourceInputs, stored.SourceInputs)
	refreshed := preparationPreviewRead(t, f)
	require.NotNil(t, refreshed.Preview.Approval) // Identical reacceptance retains native evidence.
	require.Equal(t, stored.ID, refreshed.Preview.Approval.Requirement)
	require.Equal(t, view.Preview.Approval.EffectsDigest, refreshed.Preview.Approval.EffectsDigest)
	require.Equal(t, view.Preview.Approval.ReadinessDigest, refreshed.Preview.Approval.ReadinessDigest)
	f.administrator = true
	approved := preparationPreviewAction(t, f, "actions", "bob", "read", previewApproval(refreshed, ""))
	require.Equal(t, http.StatusCreated, approved.Code, approved.Body.String())
	approval, err := f.server.Store.ApprovalDecision(t.Context(), "d444444444444444444444444")
	require.NoError(t, err)
	require.Equal(t, int64(2), approval.Approver)
	require.True(t, approval.Verified)
	require.Equal(t, refreshed.Preview.Approval.EffectsDigest, approval.EffectsDigest)
	require.Equal(t, refreshed.Preview.Approval.ReadinessDigest, approval.ReadinessDigest)
	final := preparationPreviewRead(t, f)
	require.Equal(t, approval.ID, final.Preview.Approval.Predecessor)
	// The preview follows fresh native tool evidence, rather than the durable
	// preparation's older observation or hashes supplied by the browser.
	for id, state := range f.states {
		state.Tools = []project.ResolvedTool{{Name: "git", Path: "/usr/bin/git", Version: "git version 2.48.0"}}
		f.states[id] = state
	}
	changed := preparationPreviewRead(t, f)
	require.Equal(t, "git version 2.48.0", changed.Preview.Approval.Roles[0].Tools[0].Version)
	require.NotEqual(t, final.Preview.Approval.EffectsDigest, changed.Preview.Approval.EffectsDigest)
	require.NotEqual(t, final.Preview.Approval.ReadinessDigest, changed.Preview.Approval.ReadinessDigest)
	require.Equal(t, final.Preview.Requirements.InputsDigest, changed.Preview.Requirements.InputsDigest)
}

func TestPreparationDecisionPreviewRefusesStalePredecessors(t *testing.T) {
	f := newPreparationPreviewFixture(t)
	view := preparationPreviewRead(t, f)
	first := preparationPreviewAction(t, f, "acceptances", "alice", "write", previewAcceptance(f, view.Preview.Requirements.Predecessor))
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
	stale := previewAcceptance(f, view.Preview.Requirements.Predecessor)
	stale["decision_id"] = "d555555555555555555555555"
	w := preparationPreviewAction(t, f, "acceptances", "alice", "write", stale)
	require.Equal(t, http.StatusPreconditionFailed, w.Code, w.Body.String())
	// An approval from the old screen also binds a superseded requirement.
	f.administrator = true
	w = preparationPreviewAction(t, f, "actions", "bob", "read", previewApproval(view, ""))
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	current := preparationPreviewRead(t, f)
	w = preparationPreviewAction(t, f, "actions", "bob", "read", previewApproval(current, ""))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	staleApproval := previewApproval(current, "")
	staleApproval["decision_id"] = "d666666666666666666666666"
	w = preparationPreviewAction(t, f, "actions", "bob", "read", staleApproval)
	require.Equal(t, http.StatusPreconditionFailed, w.Code, w.Body.String())
}

func TestPreparationDecisionPreviewKeepsWriteAndAdminSeparate(t *testing.T) {
	f := newPreparationPreviewFixture(t)
	view := preparationPreviewRead(t, f)
	f.administrator = true
	w := preparationPreviewAction(t, f, "acceptances", "bob", "read", previewAcceptance(f, view.Preview.Requirements.Predecessor))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "repository_write_required")
	require.Zero(t, f.accessCalls)
	f.administrator = false
	w = preparationPreviewAction(t, f, "actions", "alice", "write", previewApproval(view, ""))
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "project_administrator_required")
	require.Equal(t, 1, f.accessCalls)
	_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound)
	head, err := f.server.Store.RequirementHead(t.Context(), preparationProject)
	require.NoError(t, err)
	require.Equal(t, f.requirement.ID, head)
}

func TestPreparationDecisionPreviewRequiresNativeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*preparationPreviewFixture)
	}{
		{"missing-native-reviewer", func(f *preparationPreviewFixture) { f.missingRole = project.RoleReviewer }},
		{"wrong-native-source", func(f *preparationPreviewFixture) {
			for id, state := range f.states {
				state.SourceCommit = strings.Repeat("b", 40)
				f.states[id] = state
			}
		}},
		{"failed-native-check", func(f *preparationPreviewFixture) {
			for id, state := range f.states {
				nonzero := 1
				state.CheckExit = &nonzero
				f.states[id] = state
			}
		}},
		{"missing-native-setup-exit", func(f *preparationPreviewFixture) {
			for id, state := range f.states {
				state.SetupExit = nil
				f.states[id] = state
			}
		}},
		{"changed-selected-bytes", func(f *preparationPreviewFixture) { f.files["setup.sh"] = []byte("changed setup\n") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPreparationPreviewFixture(t)
			tc.mutate(f)
			view := preparationPreviewRead(t, f)
			require.Nil(t, view.Preview.Requirements)
			require.Nil(t, view.Preview.Approval)
			require.NotEmpty(t, view.Preview.Reason)
			require.Equal(t, f.requirement.ID, view.Requirements)
			_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
			require.ErrorIs(t, err, store.ErrNotFound)
		})
	}
}

// TestPreparationDecisionPreviewSlowNativeRead is the FJ-v3-701 regression:
// a seven-input selection (~35KiB, mirroring the live multi-input selection)
// with a slow native reader must still confirm under the raised budget. Each
// role batches its whole selection into one host call, so the preview needs
// two 3s reads (~6s total): necessarily beyond the old 5s budget, where the
// old one-path-per-read fan-out would have needed fourteen such reads
// (~42s). Success still requires byte-exact SHA matching for every path.
func TestPreparationDecisionPreviewSlowNativeRead(t *testing.T) {
	files := map[string][]byte{
		"setup.sh": []byte("echo private-setup-content\n"),
		"check.sh": []byte("echo private-check-content\n"),
	}
	for i := range 5 {
		name := "input" + string(rune('a'+i)) + ".sh"
		files[name] = []byte("#!/bin/sh\n# selected input " + name + "\n" + strings.Repeat("echo payload\n", 500))
	}
	f := newPreparationPreviewFixtureFiles(t, files)
	f.delay = 3 * time.Second
	var total int
	for _, content := range files {
		total += len(content)
	}
	require.Greater(t, total, 30*1024) // Live-scale aggregate, far under the 128KiB native bound.
	start := time.Now()
	view := preparationPreviewRead(t, f)
	elapsed := time.Since(start)
	require.Empty(t, view.Preview.Reason)
	require.NotNil(t, view.Preview.Requirements)
	require.NotNil(t, view.Preview.Approval)
	selected := map[string]string{}
	for _, input := range view.Preview.Requirements.SourceInputs {
		selected[input.Path] = input.Digest
	}
	require.Equal(t, f.requirement.SourceInputs, selected)
	require.Len(t, selected, 7)
	require.Len(t, f.contexts, 2) // One batched read per role, not fourteen single-path reads.
	for _, request := range f.contexts {
		require.Len(t, request.Paths, 7)
	}
	// The two slow reads alone exceed the old 5s budget, so this passing
	// preview demonstrates the old fan-out could not have survived it.
	require.GreaterOrEqual(t, elapsed, 5*time.Second)
	_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound) // Reading creates no approval.
}

// TestPreparationDecisionPreviewMaxInputsBoundedFallback proves the true
// worst-case call bound for the legal maximum of 64 selected inputs
// (project.MaxSourceInputs) against the genuine typed route signal: with
// the production-shaped boundary (32KiB setup + 32KiB check + 32KiB root
// AGENTS.md, which the native always selects), every multi-path batch
// genuinely overflows the native aggregate bound, so the peer refuses it
// with HTTP 413 exactly like the real route — and only that typed signal
// retries by halving. Every single-path read totals exactly 128KiB and
// fits, so each role falls back through the full binary tree down to
// singles and still confirms with byte-exact digests. A 12-batch costs 23
// calls, a 4-batch 7, so each role costs 5*23+7 = 122 and both roles 244
// native reads exactly. Any future halving change that issues more or
// fewer calls fails here.
func TestPreparationDecisionPreviewMaxInputsBoundedFallback(t *testing.T) {
	// The 96 KiB setup/check/AGENTS.md overhead rides along every native
	// read — facing only the aggregate bound, not the per-file bound —
	// while each of the 64 selected inputs fills the 32 KiB per-file bound
	// exactly: one selected file totals exactly 128KiB and fits, while two
	// selected files (160KiB) overflow. AGENTS.md is served but unselected,
	// mirroring the native auto-selection outside the accepted inputs.
	files := map[string][]byte{
		"setup.sh":  bytes.Repeat([]byte("s"), 32*1024),
		"check.sh":  bytes.Repeat([]byte("c"), 32*1024),
		"AGENTS.md": bytes.Repeat([]byte("a"), 32*1024),
	}
	var names []string
	for i := range 64 {
		name := fmt.Sprintf("input%02d.dat", i)
		files[name] = bytes.Repeat([]byte("x"), project.MaxFactoryContextFileBytes)
		names = append(names, name)
	}
	f := newPreparationPreviewFixtureFilesSelected(t, files, names)
	start := time.Now()
	view := preparationPreviewRead(t, f)
	elapsed := time.Since(start)
	require.Empty(t, view.Preview.Reason)
	require.NotNil(t, view.Preview.Requirements)
	require.NotNil(t, view.Preview.Approval)
	selected := map[string]string{}
	for _, input := range view.Preview.Requirements.SourceInputs {
		selected[input.Path] = input.Digest
	}
	require.Equal(t, f.requirement.SourceInputs, selected)
	require.Len(t, selected, 64)
	require.Less(t, elapsed, 30*time.Second)
	require.Len(t, f.contexts, 244) // True worst-case bound, both roles.
	coderID := "f" + strings.Repeat("1", 24)
	reviewerID := "f" + strings.Repeat("2", 24)
	perRole := map[string]int{}
	for _, request := range f.contexts {
		perRole[request.ID]++
	}
	require.Equal(t, map[string]int{coderID: 122, reviewerID: 122}, perRole)
	_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound) // Reading creates no approval.
}

// TestPreparationDecisionPreviewFailsClosedOnNativeErrors proves actual host
// failures refuse the preview with the fixed reason instead of partial data:
// no requirements, no approval, and no durable approval written.
func TestPreparationDecisionPreviewFailsClosedOnNativeErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*preparationPreviewFixture)
	}{
		{"host-error", func(f *preparationPreviewFixture) { f.failContext = true }},
		{"host-timeout", func(f *preparationPreviewFixture) { f.hangContext = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPreparationPreviewFixture(t)
			tc.mutate(f)
			view := preparationPreviewRead(t, f)
			require.Nil(t, view.Preview.Requirements)
			require.Nil(t, view.Preview.Approval)
			require.Contains(t, view.Preview.Reason, "must be confirmed before reviewing decisions")
			require.Equal(t, f.requirement.ID, view.Requirements)
			_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
			require.ErrorIs(t, err, store.ErrNotFound)
		})
	}
}

// TestPreparationDecisionPreviewMalformedBatchFailsClosed is the FJ-v3-729
// regression: an HTTP 200 batch carrying null or malformed JSON, followed
// by servable valid singles, must fail closed. The shape error
// ("invalid native response") is not the typed bounds signal, so no
// subset retry may halve onto the valid singles and mask the failed
// batch — the preview refuses with the fixed reason and writes nothing.
func TestPreparationDecisionPreviewMalformedBatchFailsClosed(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"null-batch", "null"},
		{"malformed-batch", "{not json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPreparationPreviewFixture(t)
			f.badBatch = tc.body
			view := preparationPreviewRead(t, f)
			require.Nil(t, view.Preview.Requirements)
			require.Nil(t, view.Preview.Approval)
			require.Contains(t, view.Preview.Reason, "must be confirmed before reviewing decisions")
			require.Equal(t, f.requirement.ID, view.Requirements)
			// Fail-closed means no subset retry: exactly one batch read —
			// the coder role refuses and the preview returns before the
			// reviewer role reads — never the halved singles the peer
			// would have served validly.
			require.Len(t, f.contexts, 1)
			_, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
			require.ErrorIs(t, err, store.ErrNotFound)
		})
	}
}
