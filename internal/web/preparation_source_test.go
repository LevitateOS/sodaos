package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/stretchr/testify/require"
)

type preparationSourceTest struct {
	server         *Server
	files          map[string][]byte
	commit         string
	writable       bool
	admin          bool
	prepared       []string
	state          map[string]project.PrepareState
	loseReply      bool
	sourceRequests []extensions.CallbackRequest
	writerResults  map[string]bool
}

func newPreparationSourceTest(t *testing.T) *preparationSourceTest {
	t.Helper()
	s, _ := preparationFixture(t)
	s.Config.OperatorID = 99
	require.NoError(t, s.Store.Join(t.Context(), preparationProject, 1, "alice"))
	require.NoError(t, s.Store.Join(t.Context(), preparationProject, 2, "bob"))
	f := &preparationSourceTest{
		server: s, commit: strings.Repeat("a", 40), writable: true, admin: true, state: map[string]project.PrepareState{}, writerResults: map[string]bool{},
		files: map[string][]byte{
			"setup.sh":  []byte("#!/bin/sh\nexit 0\n"),
			"check.sh":  []byte("#!/bin/sh\ngit --version\n"),
			"README.md": []byte("reviewed source input\n"),
		},
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		var out any
		switch r.URL.Path {
		case "/project-access":
			var in project.ProjectAccessRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			allowed := f.admin
			out = project.ProjectAccessStatus{Project: in.Project, Login: in.Login, Identity: in.Identity, Administrator: &allowed}
		case "/prepare-inspect":
			var in project.PrepareInspect
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			state, ok := f.state[in.ID]
			if !ok {
				return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			out = state
		case "/prepare":
			var in project.Prepare
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			f.prepared = append(f.prepared, in.Preparation.ID)
			state := project.PrepareState{
				ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role, Phase: project.PrepareRunning,
				Container: strings.Repeat("b", 64), SourceCommit: in.Preparation.SourceCommit, SetupDigest: in.Preparation.SetupDigest,
			}
			f.state[in.Preparation.ID] = state
			if f.loseReply {
				f.loseReply = false
				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody, Header: make(http.Header)}, nil
			}
			out = state
		default:
			t.Fatalf("unexpected host operation %s", r.URL.Path)
		}
		data, err := json.Marshal(out)
		require.NoError(t, err)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}
	return f
}

func (f *preparationSourceTest) callback(t *testing.T, permission string) func(extensions.CallbackRequest) extensions.CallbackResponse {
	t.Helper()
	return func(in extensions.CallbackRequest) extensions.CallbackResponse {
		switch in.Operation {
		case extensions.OperationRepository:
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "prep", Permission: permission}}
		case extensions.OperationRepositorySource:
			f.sourceRequests = append(f.sourceRequests, in)
			if in.RepositoryID != "7" || in.SourceCommit != "" && in.SourceCommit != f.commit {
				return extensions.CallbackResponse{ErrorCode: "not_found"}
			}
			files := make(map[string][]byte, len(in.SourcePaths))
			for _, path := range in.SourcePaths {
				content, ok := f.files[path]
				if !ok {
					return extensions.CallbackResponse{ErrorCode: "not_found"}
				}
				files[path] = content
			}
			writers := make(map[string]bool, len(in.SourceActors))
			for _, actor := range in.SourceActors {
				writers[actor] = f.writable && actor == "1"
				f.writerResults[actor] = writers[actor]
			}
			return extensions.CallbackResponse{Source: &extensions.RepositorySource{RepositoryID: "7", Commit: f.commit, Bundle: []byte("verified native git bundle"), Files: files, Writers: writers}}
		default:
			t.Fatalf("unexpected Forgejo callback operation %s", in.Operation)
			return extensions.CallbackResponse{}
		}
	}
}

func preparationSourceProxy(t *testing.T, f *preparationSourceTest, login, permission string) *httptest.Server {
	t.Helper()
	return nativeProductProxyForActorWithCallback(t, f.server, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}, login, f.callback(t, permission))
}

func preparationSourcePost(t *testing.T, proxy *httptest.Server, origin, path string, payload any) *http.Response {
	return preparationSourcePostAs(t, proxy, origin, path, "alice", payload)
}

func preparationSourcePostAs(t *testing.T, proxy *httptest.Server, origin, path, login string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	path = strings.TrimPrefix(path, "/api")
	r, err := http.NewRequestWithContext(t.Context(), http.MethodPost, proxy.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	contribution := extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global", Action: "post"}
	r.Header = nativeProductHeadersForActor(http.MethodPost, origin, contribution, login)
	r.Header.Set("Content-Type", "application/json")
	response, err := proxy.Client().Do(r)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func requirePreparationStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		t.Fatalf("HTTP status %d, want %d: %s", response.StatusCode, want, readResponse(t, response))
	}
}

func establishSourceDecisions(t *testing.T, f *preparationSourceTest) (string, string, []string, []string, []string) {
	t.Helper()
	paths := []string{"check.sh", "README.md", "setup.sh"}
	setupFiles := []string{"check.sh", "setup.sh"}
	tools := []string{"git"}
	alice := preparationSourceProxy(t, f, "alice", "write")
	preview := preparationSourcePost(t, alice, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", map[string]any{
			"action": "source-preview", "source_paths": paths, "setup_files": setupFiles, "required_tools": tools,
		})
	requirePreparationStatus(t, preview, http.StatusOK)
	var selected struct {
		Requirements struct {
			SourceCommit string                          `json:"source_commit"`
			SetupDigest  string                          `json:"setup_digest"`
			InputsDigest string                          `json:"inputs_digest"`
			Predecessor  string                          `json:"predecessor"`
			SourceInputs []struct{ Path, Digest string } `json:"source_inputs"`
		} `json:"requirements"`
	}
	require.NoError(t, json.NewDecoder(preview.Body).Decode(&selected))
	inputDigests := make(map[string]string, len(selected.Requirements.SourceInputs))
	for _, input := range selected.Requirements.SourceInputs {
		inputDigests[input.Path] = input.Digest
	}
	requirementID := "d" + strings.Repeat("3", 24)
	accepted := preparationSourcePost(t, alice, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/acceptances", map[string]any{
			"command_id": factory.NewID(), "decision_id": requirementID,
			"predecessor": selected.Requirements.Predecessor, "source_commit": selected.Requirements.SourceCommit,
			"setup_digest": selected.Requirements.SetupDigest, "inputs_digest": selected.Requirements.InputsDigest,
			"source_inputs": inputDigests,
		})
	requirePreparationStatus(t, accepted, http.StatusCreated)
	bob := preparationSourceProxy(t, f, "bob", "read")
	approvalID := "d" + strings.Repeat("4", 24)
	approved := preparationSourcePostAs(t, bob, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", "bob", map[string]any{
			"action": "approve-initial", "command_id": factory.NewID(), "decision_id": approvalID,
			"requirement": requirementID, "verified": true,
			"effects_evidence":   "Reviewed the selected privileged effects and installed prerequisites.",
			"readiness_evidence": "Checked current native Project services and assigned resources.",
		})
	requirePreparationStatus(t, approved, http.StatusCreated)
	return requirementID, approvalID, paths, setupFiles, tools
}

func preparationSourceStart(t *testing.T, f *preparationSourceTest, proxy *httptest.Server, login, requirement, approval string, paths, setupFiles, tools []string, coder, reviewer string) *http.Response {
	t.Helper()
	return preparationSourcePostAs(t, proxy, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", login, map[string]any{
			"action": "prepare", "requirement": requirement, "approval": approval,
			"source_paths": paths, "setup_files": setupFiles, "required_tools": tools,
			"coder_id": coder, "reviewer_id": reviewer,
		})
}

func TestPreparationSourceProxyJourneyStartsFromEmptyDecisions(t *testing.T) {
	f := newPreparationSourceTest(t)
	_, err := f.server.Store.RequirementHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound)
	_, err = f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound)
	rows, err := f.server.Store.ProjectPreparations(t.Context(), preparationProject)
	require.NoError(t, err)
	require.Empty(t, rows)

	alice := preparationSourceProxy(t, f, "alice", "write")
	previewBody, err := json.Marshal(map[string]any{
		"action": "source-preview", "source_paths": []string{"README.md", "setup.sh", "check.sh"},
		"setup_files": []string{"setup.sh", "check.sh"}, "required_tools": []string{"git"},
	})
	require.NoError(t, err)
	preview := nativeProductRequest(t, alice, f.server.Config.ForgejoURL, http.MethodPost,
		"/api/environments/"+preparationProject+"/preparation/actions", previewBody)
	require.Equal(t, http.StatusOK, preview.StatusCode)
	var previewData struct {
		Requirements struct {
			SourceCommit string                          `json:"source_commit"`
			SetupDigest  string                          `json:"setup_digest"`
			InputsDigest string                          `json:"inputs_digest"`
			Predecessor  string                          `json:"predecessor"`
			SourceInputs []struct{ Path, Digest string } `json:"source_inputs"`
		} `json:"requirements"`
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Bytes   int    `json:"bytes"`
		} `json:"files"`
	}
	require.NoError(t, json.NewDecoder(preview.Body).Decode(&previewData))
	require.Equal(t, f.commit, previewData.Requirements.SourceCommit)
	require.Empty(t, previewData.Requirements.Predecessor)
	require.Len(t, previewData.Requirements.SourceInputs, 3)
	selected := make(map[string]string, 3)
	for _, input := range previewData.Requirements.SourceInputs {
		sum := sha256.Sum256(f.files[input.Path])
		require.Equal(t, hex.EncodeToString(sum[:]), input.Digest)
		selected[input.Path] = input.Digest
	}
	require.Len(t, previewData.Files, 3)
	for _, file := range previewData.Files {
		require.Equal(t, string(f.files[file.Path]), file.Content)
		require.Equal(t, len(f.files[file.Path]), file.Bytes)
	}

	requirementID := "d" + strings.Repeat("1", 24)
	accept := preparationSourcePost(t, alice, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/acceptances", map[string]any{
			"command_id": factory.NewID(), "decision_id": requirementID, "source_commit": previewData.Requirements.SourceCommit,
			"setup_digest": previewData.Requirements.SetupDigest, "inputs_digest": previewData.Requirements.InputsDigest,
			"source_inputs": selected,
		})
	require.Equal(t, http.StatusCreated, accept.StatusCode)
	requirement, err := f.server.Store.RequirementDecision(t.Context(), requirementID)
	require.NoError(t, err)
	require.Equal(t, selected, requirement.SourceInputs)

	bob := preparationSourceProxy(t, f, "bob", "read")
	approve := preparationSourcePostAs(t, bob, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", "bob", map[string]any{
			"action": "approve-initial", "command_id": factory.NewID(), "decision_id": "d" + strings.Repeat("2", 24),
			"requirement": requirementID, "verified": true,
			"effects_evidence":   "Reviewed installed prerequisites and assigned resources.",
			"readiness_evidence": "Checked current native Project readiness and service state.",
		})
	requirePreparationStatus(t, approve, http.StatusCreated)
	approval, err := f.server.Store.ApprovalDecision(t.Context(), "d"+strings.Repeat("2", 24))
	require.NoError(t, err)
	require.True(t, approval.Verified)
	require.Equal(t, int64(2), approval.Approver)

	// The native callback endpoint is process-scoped in this test fixture;
	// install Alice's callback again before returning to Alice's proxy.
	alice = preparationSourceProxy(t, f, "alice", "write")
	prepared := preparationSourcePost(t, alice, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", map[string]any{
			"action": "prepare", "requirement": requirementID, "approval": approval.ID,
			"source_paths": []string{"README.md", "check.sh", "setup.sh"},
			"setup_files":  []string{"check.sh", "setup.sh"}, "required_tools": []string{"git"},
			"coder_id": "f" + strings.Repeat("1", 24), "reviewer_id": "f" + strings.Repeat("2", 24),
		})
	requirePreparationStatus(t, prepared, http.StatusOK)
	require.Equal(t, []string{"f" + strings.Repeat("1", 24), "f" + strings.Repeat("2", 24)}, f.prepared)
	rows, err = f.server.Store.ProjectPreparations(t.Context(), preparationProject)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestPreparationSourceProxyRequiresCurrentRepositoryWriterAndNativeAdministrator(t *testing.T) {
	f := newPreparationSourceTest(t)
	readonly := preparationSourceProxy(t, f, "bob", "read")
	preview := preparationSourcePostAs(t, readonly, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", "bob", map[string]any{
			"action": "source-preview", "source_paths": []string{"setup.sh", "check.sh"},
			"setup_files": []string{"setup.sh", "check.sh"},
		})
	requirePreparationStatus(t, preview, http.StatusOK)
	require.Len(t, f.sourceRequests, 1)
	require.Empty(t, f.sourceRequests[0].SourceActors)
	readOnlyAcceptance := preparationSourcePostAs(t, readonly, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/acceptances", "bob", map[string]any{
			"command_id": factory.NewID(), "decision_id": "d" + strings.Repeat("5", 24),
			"source_commit": f.commit, "setup_digest": strings.Repeat("a", 64), "inputs_digest": strings.Repeat("b", 64),
			"source_inputs": map[string]string{"setup.sh": strings.Repeat("c", 64)},
		})
	requirePreparationStatus(t, readOnlyAcceptance, http.StatusForbidden)
	_, err := f.server.Store.RequirementHead(t.Context(), preparationProject)
	require.ErrorIs(t, err, store.ErrNotFound)

	requirementID, _, _, _, _ := establishSourceDecisions(t, f)
	approvalBefore, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.NoError(t, err)
	f.admin = false
	bob := preparationSourceProxy(t, f, "bob", "read")
	approval := preparationSourcePostAs(t, bob, f.server.Config.ForgejoURL,
		"/api/environments/"+preparationProject+"/preparation/actions", "bob", map[string]any{
			"action": "approve-initial", "command_id": factory.NewID(), "decision_id": "d" + strings.Repeat("5", 24),
			"requirement": requirementID, "verified": true,
			"effects_evidence": "Reviewed effects.", "readiness_evidence": "Checked native readiness.",
		})
	requirePreparationStatus(t, approval, http.StatusForbidden)
	approvalAfter, err := f.server.Store.ApprovalHead(t.Context(), preparationProject)
	require.NoError(t, err)
	require.Equal(t, approvalBefore, approvalAfter)
}

func TestPreparationSourceReadOnlyNativeAdministratorCanStartCurrentWriterAcceptance(t *testing.T) {
	f := newPreparationSourceTest(t)
	requirementID, approvalID, paths, setupFiles, tools := establishSourceDecisions(t, f)
	bob := preparationSourceProxy(t, f, "bob", "read")
	response := preparationSourceStart(t, f, bob, "bob", requirementID, approvalID, paths, setupFiles, tools,
		"f"+strings.Repeat("a", 24), "f"+strings.Repeat("b", 24))
	requirePreparationStatus(t, response, http.StatusOK)
	require.Len(t, f.prepared, 2)
}

func TestPreparationSourceOrdinaryReadOnlyCallerCannotStart(t *testing.T) {
	f := newPreparationSourceTest(t)
	requirementID, approvalID, paths, setupFiles, tools := establishSourceDecisions(t, f)
	f.admin = false
	bob := preparationSourceProxy(t, f, "bob", "read")
	response := preparationSourceStart(t, f, bob, "bob", requirementID, approvalID, paths, setupFiles, tools,
		"f"+strings.Repeat("a", 24), "f"+strings.Repeat("b", 24))
	requirePreparationStatus(t, response, http.StatusForbidden)
	require.Empty(t, f.prepared)
}

func TestPreparationSourceStartRechecksInputsHeadsAndNativeWriters(t *testing.T) {
	for _, change := range []string{"source", "tools", "writer", "requirement-head"} {
		t.Run(change, func(t *testing.T) {
			f := newPreparationSourceTest(t)
			requirementID, approvalID, paths, setupFiles, tools := establishSourceDecisions(t, f)
			alice := preparationSourceProxy(t, f, "alice", "write")
			coder, reviewer := "f"+strings.Repeat("1", 24), "f"+strings.Repeat("2", 24)
			switch change {
			case "source":
				f.files["README.md"] = []byte("changed after native acceptance\n")
			case "tools":
				tools = []string{"git", "node"}
			case "writer":
				f.writable = false
			case "requirement-head":
				previous, err := f.server.Store.RequirementDecision(t.Context(), requirementID)
				require.NoError(t, err)
				previous.Predecessor = previous.ID
				previous.ID = "d" + strings.Repeat("6", 24)
				receipt, err := f.server.Coordinator.AdmitRequirement(t.Context(), factory.NewID(), "native:1", previous)
				require.NoError(t, err)
				require.Equal(t, previous.ID, receipt.Head)
			}
			response := preparationSourceStart(t, f, alice, "alice", requirementID, approvalID, paths, setupFiles, tools, coder, reviewer)
			requirePreparationStatus(t, response, http.StatusConflict)
			require.Empty(t, f.prepared)
			rows, err := f.server.Store.ProjectPreparations(t.Context(), preparationProject)
			require.NoError(t, err)
			require.Empty(t, rows)
			if change == "writer" {
				last := f.sourceRequests[len(f.sourceRequests)-1]
				require.Equal(t, []string{"1"}, last.SourceActors)
				require.False(t, f.writerResults["1"])
			}
		})
	}
}

func TestPreparationSourceStartAdoptsLostNativeReplyWithSameIdentities(t *testing.T) {
	f := newPreparationSourceTest(t)
	requirementID, approvalID, paths, setupFiles, tools := establishSourceDecisions(t, f)
	f.loseReply = true
	alice := preparationSourceProxy(t, f, "alice", "write")
	coder, reviewer := "f"+strings.Repeat("7", 24), "f"+strings.Repeat("8", 24)
	response := preparationSourceStart(t, f, alice, "alice", requirementID, approvalID, paths, setupFiles, tools, coder, reviewer)
	requirePreparationStatus(t, response, http.StatusOK)
	require.Equal(t, []string{coder, reviewer}, f.prepared)
	rows, err := f.server.Store.ProjectPreparations(t.Context(), preparationProject)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, []string{coder, reviewer}, []string{rows[0].Preparation.ID, rows[1].Preparation.ID})
}

func readResponse(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}
