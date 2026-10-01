package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

const factorySettingsProject = "p765432109876543210987654"

func factorySettingsServer(t *testing.T) *Server {
	t.Helper()
	s := apiTestServer(t)
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: factorySettingsProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(`{"active":true,"revision":1}`))), Header: make(http.Header)}, nil
	})}
	return s
}

func factoryEncryptedServer(t *testing.T) *Server {
	t.Helper()
	db, err := store.OpenEncrypted(filepath.Join(t.TempDir(), "soda.db"), bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for i, login := range []string{"alice", "bob"} {
		if err = db.UpsertUser(context.Background(), store.User{ID: int64(i + 1), Login: login, Name: login}); err != nil {
			t.Fatal(err)
		}
	}
	return New(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
}

func factoryPolicyBody(commandID string) string {
	return `{"command_id":"` + commandID + `","expected_revision":0,"enabled":true,"paused":false,` +
		`"target_branch":"refs/heads/main",` +
		`"roles":{"soda-coder":{"harness":"codex-0.157.1","model":"test"},"soda-reviewer":{"harness":"codex-0.157.1","model":"test"}},` +
		`"required_checks":["native-ci/build"],"merge_method":"fast-forward-only",` +
		`"publish_actor":{"token_id":"11","actor_id":"12"},"create_actor":{"token_id":"11","actor_id":"12"},` +
		`"review_actor":{"token_id":"13","actor_id":"14"},"merge_actor":{"token_id":"15","actor_id":"16"},` +
		`"max_concurrent":2}`
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("body %q: %v", w.Body.String(), err)
	}
}

func assertNoSecrets(t *testing.T, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	for _, leaked := range []string{"secret", "credential", "private_key", "bearer"} {
		if strings.Contains(lower, leaked) {
			t.Fatalf("response leaks %q: %s", leaked, body)
		}
	}
}

func TestFactoryStatusShowsMissingAuthority(t *testing.T) {
	s := factorySettingsServer(t)
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/repositories/7/factory", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var view struct {
		Effective struct {
			Effective bool     `json:"effective"`
			Missing   []string `json:"missing"`
		} `json:"effective"`
		Queue struct {
			Queued int `json:"queued"`
			Active int `json:"active"`
		} `json:"queue"`
	}
	decodeBody(t, w, &view)
	if view.Effective.Effective || len(view.Effective.Missing) == 0 {
		t.Fatalf("empty authority: %+v", view.Effective)
	}
	assertNoSecrets(t, w.Body.String())
}

func TestFactoryPolicyJourney(t *testing.T) {
	s := factorySettingsServer(t)
	command := factory.NewID()
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(command), "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		CommandID string `json:"command_id"`
		Revision  int64  `json:"revision"`
		Withdrawn bool   `json:"withdrawn"`
	}
	decodeBody(t, w, &receipt)
	if receipt.CommandID != command || receipt.Revision != 1 || receipt.Withdrawn {
		t.Fatalf("receipt: %+v", receipt)
	}
	assertNoSecrets(t, w.Body.String())
	replay := httptest.NewRecorder()
	nativeAPIServe(t, s, replay, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(command), "alice"))
	if replay.Code != 200 || replay.Body.String() != w.Body.String() {
		t.Fatal("identical command did not replay its receipt", replay.Code, replay.Body.String())
	}
	changed := strings.Replace(factoryPolicyBody(command), `"max_concurrent":2`, `"max_concurrent":3`, 1)
	conflict := httptest.NewRecorder()
	nativeAPIServe(t, s, conflict, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", changed, "alice"))
	if conflict.Code != 409 {
		t.Fatal("changed payload reused its command identity", conflict.Code, conflict.Body.String())
	}
	stale := httptest.NewRecorder()
	nativeAPIServe(t, s, stale, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(factory.NewID()), "alice"))
	if stale.Code != 412 {
		t.Fatal("stale revision applied", stale.Code, stale.Body.String())
	}
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(factory.NewID()), "bob"))
	if denied.Code != 403 {
		t.Fatal("non-owner configured policy", denied.Code, denied.Body.String())
	}
	impersonate := strings.Replace(factoryPolicyBody(factory.NewID()), `"enabled":true`, `"enabled":true,"granted_by":"2"`, 1)
	forged := httptest.NewRecorder()
	nativeAPIServe(t, s, forged, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", impersonate, "alice"))
	if forged.Code != 400 {
		t.Fatal("body-selected authorizer accepted", forged.Code, forged.Body.String())
	}
}

func TestFactoryPolicyPauseWithdrawsDispatch(t *testing.T) {
	s := factorySettingsServer(t)
	enable := httptest.NewRecorder()
	nativeAPIServe(t, s, enable, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(factory.NewID()), "alice"))
	if enable.Code != 200 {
		t.Fatal(enable.Code, enable.Body.String())
	}
	paused := strings.Replace(factoryPolicyBody(factory.NewID()), `"expected_revision":0`, `"expected_revision":1`, 1)
	paused = strings.Replace(paused, `"paused":false`, `"paused":true`, 1)
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", paused, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		Withdrawn bool `json:"withdrawn"`
		Effective struct {
			Effective bool     `json:"effective"`
			Missing   []string `json:"missing"`
		} `json:"effective"`
	}
	decodeBody(t, w, &receipt)
	if !receipt.Withdrawn || receipt.Effective.Effective {
		t.Fatalf("pause receipt: %+v", receipt)
	}
	found := map[string]bool{}
	for _, reason := range receipt.Effective.Missing {
		found[reason] = true
	}
	if !found["policy_paused"] || !found["dispatch_closed"] {
		t.Fatalf("pause missing reasons: %+v", receipt.Effective.Missing)
	}
}

func TestFactoryCapacityIsOperatorOnly(t *testing.T) {
	s := factorySettingsServer(t)
	body := `{"command_id":"` + factory.NewID() + `","expected_revision":0,"max_concurrent_runs":2,"max_queued":4}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/factory/capacity", body, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPut, "/api/factory/capacity", body, "bob"))
	if denied.Code != 403 {
		t.Fatal("non-operator changed capacity", denied.Code, denied.Body.String())
	}
	grant := `{"command_id":"` + factory.NewID() + `","expected_revision":0,"active":true,"max_concurrent":2}`
	allowed := httptest.NewRecorder()
	nativeAPIServe(t, s, allowed, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/operator-grant", grant, "alice"))
	if allowed.Code != 200 {
		t.Fatal(allowed.Code, allowed.Body.String())
	}
}

func TestFactoryEnvironmentGrantIsOwnerOnly(t *testing.T) {
	s := factorySettingsServer(t)
	profile := `{"id":"rocky-headless","distribution":"rocky","version":"9.6","interface":"headless","architecture":"amd64","image":"sha256:` + strings.Repeat("b", 64) + `","revision":"` + strings.Repeat("c", 40) + `"}`
	body := `{"command_id":"` + factory.NewID() + `","expected_revision":0,"profile":` + profile + `,"active":true}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/environment-grant", body, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/environment-grant", body, "bob"))
	if denied.Code != 403 {
		t.Fatal("non-owner permitted environment setup", denied.Code, denied.Body.String())
	}
}

func TestFactorySponsorshipNeedsConnectionOwner(t *testing.T) {
	s := factoryEncryptedServer(t)
	ctx := context.Background()
	connection := identity.Connection{ID: "conn-1", ProviderID: identity.Codex, OwnerID: 1, Label: "test", Generation: 1, State: identity.Ready}
	if err := s.Store.IdentitySaveConnection(ctx, connection, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.IdentitySaveGrant(ctx, identity.Grant{ID: "grant-1", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject}); err != nil {
		t.Fatal(err)
	}
	sponsor := func(command string, generation int64) string {
		return `{"command_id":"` + command + `","expected_revision":0,"grant_id":"grant-1","generation":` + strconv.FormatInt(generation, 10) +
			`,"roles":["soda-coder"],"allowance_minutes":60,"max_concurrent":1,"active":true}`
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/conn-1", sponsor(factory.NewID(), 1), "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertNoSecrets(t, w.Body.String())
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/conn-1", sponsor(factory.NewID(), 1), "bob"))
	if denied.Code != 403 {
		t.Fatal("non-owner sponsored the connection", denied.Code, denied.Body.String())
	}
	rotated := httptest.NewRecorder()
	nativeAPIServe(t, s, rotated, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/conn-1", sponsor(factory.NewID(), 2), "alice"))
	if rotated.Code != 409 {
		t.Fatal("rotated credential generation accepted", rotated.Code, rotated.Body.String())
	}
	status := httptest.NewRecorder()
	nativeAPIServe(t, s, status, apiTestRequest(http.MethodGet, "/api/repositories/7/factory", "", "bob"))
	if status.Code != 200 {
		t.Fatal(status.Code, status.Body.String())
	}
	var view struct {
		Sponsorships []struct {
			Allowance int `json:"allowance_minutes"`
		} `json:"sponsorships"`
	}
	decodeBody(t, status, &view)
	if len(view.Sponsorships) != 1 || view.Sponsorships[0].Allowance != 0 {
		t.Fatalf("allowance leaked to non-owner: %+v", view.Sponsorships)
	}
}

func TestPreparationAcceptanceJourney(t *testing.T) {
	s := factorySettingsServer(t)
	digest := strings.Repeat("d", 64)
	commit := strings.Repeat("e", 40)
	body := `{"command_id":"` + factory.NewID() + `","decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + commit + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", body, "alice"))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	read := httptest.NewRecorder()
	nativeAPIServe(t, s, read, apiTestRequest(http.MethodGet, "/api/environments/"+factorySettingsProject+"/preparation", "", "alice"))
	if read.Code != 200 {
		t.Fatal(read.Code, read.Body.String())
	}
	var view struct {
		Requirements string `json:"requirements"`
	}
	decodeBody(t, read, &view)
	if view.Requirements != "d123456789012345678901234" {
		t.Fatalf("preparation view: %+v", view)
	}
	second := `{"command_id":"` + factory.NewID() + `","decision_id":"d223456789012345678901234",` +
		`"source_commit":"` + commit + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	stale := httptest.NewRecorder()
	nativeAPIServe(t, s, stale, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", second, "alice"))
	if stale.Code != 412 {
		t.Fatal("acceptance skipped its predecessor", stale.Code, stale.Body.String())
	}
}

func TestPreparationApprovalJourney(t *testing.T) {
	s := factorySettingsServer(t)
	digest := strings.Repeat("d", 64)
	commit := strings.Repeat("e", 40)
	accept := `{"command_id":"` + factory.NewID() + `","decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + commit + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	accepted := httptest.NewRecorder()
	nativeAPIServe(t, s, accepted, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", accept, "alice"))
	if accepted.Code != 201 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	approve := func(command, requirement string) string {
		return `{"command_id":"` + command + `","action":"approve","decision_id":"d223456789012345678901234",` +
			`"requirement":"` + requirement + `","effects_digest":"` + digest + `","readiness_digest":"` + digest + `","verified":true}`
	}
	wrong := httptest.NewRecorder()
	nativeAPIServe(t, s, wrong, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d999999999999999999999999"), "alice"))
	if wrong.Code != 409 {
		t.Fatal("approval bound a superseded requirement", wrong.Code, wrong.Body.String())
	}
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "bob"))
	if denied.Code != 403 {
		t.Fatal("non-administrator approved privileged effects", denied.Code, denied.Body.String())
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "alice"))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	inspect := httptest.NewRecorder()
	nativeAPIServe(t, s, inspect, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", `{"action":"inspect"}`, "alice"))
	if inspect.Code != 200 {
		t.Fatal(inspect.Code, inspect.Body.String())
	}
	var view struct {
		Requirements string `json:"requirements"`
		Approval     string `json:"approval"`
	}
	decodeBody(t, inspect, &view)
	if view.Requirements != "d123456789012345678901234" || view.Approval != "d223456789012345678901234" {
		t.Fatalf("inspect view: %+v", view)
	}
}
