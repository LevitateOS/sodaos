package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
}

func newPreparationPreviewFixture(t *testing.T) *preparationPreviewFixture {
	t.Helper()
	s, _ := preparationFixture(t)
	s.Config.OperatorID = 99 // Require each actor's actual native Project privilege.
	for i, login := range []string{"alice", "bob"} {
		require.NoError(t, s.Store.Join(t.Context(), preparationProject, int64(i+1), login))
	}
	f := &preparationPreviewFixture{
		server: s, states: map[string]project.PrepareState{},
		files: map[string][]byte{
			"setup.sh": []byte("echo private-setup-content\n"),
			"check.sh": []byte("echo private-check-content\n"),
		},
	}
	f.requirement = project.RequirementDecision{
		ID: "d111111111111111111111111", Project: preparationProject, Approver: 1,
		SourceCommit: strings.Repeat("c", 40), SetupDigest: project.SetupDigestOf(f.files),
		InputsDigest: strings.Repeat("d", 64), SourceInputs: map[string]string{},
	}
	for path, content := range f.files {
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
			require.LessOrEqual(t, time.Until(in.NotAfter), 5*time.Second)
			f.contexts = append(f.contexts, in)
			material := project.FactoryPreparationContext{
				Project: in.Project, ID: in.ID, Role: state.Role, SourceCommit: in.SourceCommit,
				ApprovedBase: in.ApprovedBase, DiffBase: in.DiffBase, Candidate: in.Candidate,
				Setup: f.files["setup.sh"], Check: f.files["check.sh"],
			}
			for _, path := range in.Paths {
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
	require.Len(t, f.contexts, 2)
	for _, request := range f.contexts {
		require.Equal(t, paths, request.Paths)
	}
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
