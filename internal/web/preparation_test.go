package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

const preparationProject = "p0123456789abcdef01234567"

func preparationFixture(t *testing.T) (*Server, *[]string) {
	t.Helper()
	s := apiTestServer(t)
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: preparationProject, Name: "prep", RepositoryID: 7, OwnerID: 1, Repository: "alice/prep"}); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		if r.URL.Path != "/prepare-hold" {
			return &http.Response{StatusCode: 404, Body: http.NoBody, Header: make(http.Header)}, nil
		}
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		hold, _ := body["hold"].(bool)
		revision, _ := body["revision"].(float64)
		payload := fmt.Sprintf(`{"active":%t,"revision":%d}`, hold, int64(revision))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(payload))), Header: make(http.Header)}, nil
	})}
	return s, &calls
}

func seedPreparationRecord(t *testing.T, s *Server, id, role, phase string, ready bool, output string) {
	t.Helper()
	digest := strings.Repeat("d", 64)
	commit := strings.Repeat("c", 40)
	record := project.StoredPreparation{
		Preparation: project.Preparation{
			ID: id, Project: preparationProject, Role: role,
			Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 1, SourceCommit: commit, Digest: digest},
			Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 1, EffectsDigest: digest},
			SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
		},
		State: project.PrepareState{ID: id, Project: preparationProject, Role: role, Phase: phase, Ready: ready, Output: output},
	}
	if _, _, err := s.Store.AdmitPreparation(t.Context(), record); err != nil {
		t.Fatal(err)
	}
}

func TestPreparationReadShowsHoldAndRoleReadiness(t *testing.T) {
	s, _ := preparationFixture(t)
	seedPreparationRecord(t, s, "f0123456789abcdef01234567", project.RoleCoder, project.PrepareReady, true, "secret-setup-output")
	if err := s.Store.SaveMaintenanceHold(t.Context(), project.MaintenanceHold{Project: preparationProject, Hold: true}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/environments/"+preparationProject+"/preparation", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var view struct {
		Hold         bool  `json:"hold"`
		HoldRevision int64 `json:"hold_revision"`
		Preparations []struct {
			Role  string `json:"role"`
			Phase string `json:"phase"`
			Ready bool   `json:"ready"`
		} `json:"preparations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if !view.Hold || view.HoldRevision != 1 || len(view.Preparations) != 1 || !view.Preparations[0].Ready {
		t.Fatalf("preparation view: %+v", view)
	}
	if strings.Contains(w.Body.String(), "secret-setup-output") {
		t.Fatal("setup logs leaked into the preparation view")
	}
}

func TestPreparationHoldRequiresAdministrator(t *testing.T) {
	s, calls := preparationFixture(t)
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+preparationProject+"/preparation", `{"action":"hold"}`, "bob"))
	if w.Code != 403 || len(*calls) != 0 {
		t.Fatal("non-administrator hold reached native state", w.Code)
	}
}

func TestPreparationHoldAndReleaseSyncMarker(t *testing.T) {
	s, calls := preparationFixture(t)
	hold := httptest.NewRecorder()
	nativeAPIServe(t, s, hold, apiTestRequest(http.MethodPost, "/api/environments/"+preparationProject+"/preparation", `{"action":"hold"}`, "alice"))
	if hold.Code != 200 {
		t.Fatal(hold.Code, hold.Body.String())
	}
	stored, err := s.Store.MaintenanceHold(t.Context(), preparationProject)
	if err != nil || !stored.Hold || stored.Revision != 1 {
		t.Fatalf("stored hold: %+v %v", stored, err)
	}
	release := httptest.NewRecorder()
	nativeAPIServe(t, s, release, apiTestRequest(http.MethodPost, "/api/environments/"+preparationProject+"/preparation", `{"action":"release"}`, "alice"))
	if release.Code != 200 {
		t.Fatal(release.Code, release.Body.String())
	}
	stored, err = s.Store.MaintenanceHold(t.Context(), preparationProject)
	if err != nil || stored.Hold || stored.Revision != 2 {
		t.Fatalf("released hold: %+v %v", stored, err)
	}
	if len(*calls) != 2 || (*calls)[0] != "/prepare-hold" || (*calls)[1] != "/prepare-hold" {
		t.Fatalf("marker sync calls: %q", *calls)
	}
}

func TestSpacesRowSummarizesPreparation(t *testing.T) {
	s, _ := preparationFixture(t)
	seedPreparationRecord(t, s, "f0123456789abcdef01234567", project.RoleCoder, project.PrepareReady, true, "")
	if err := s.Store.MarkReady(t.Context(), preparationProject, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.Join(t.Context(), preparationProject, 1, "original-alice"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest(http.MethodGet, "/api/spaces", "", "alice"), spacesCallback(0, new(atomic.Int32)))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result api.SpacesView
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Preparation == nil {
		t.Fatalf("spaces preparation summary: %+v", result.Items)
	}
	summary := result.Items[0].Preparation
	if summary.Hold || len(summary.Roles) != 1 || !summary.Roles[0].Ready {
		t.Fatalf("preparation summary: %+v", summary)
	}
}
