package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func approvalProjectAccessHost(s *Server, calls *int, administrator *bool, statusCode *int, malformed *bool) {
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/project-access" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(`{"active":true,"revision":1}`))), Header: make(http.Header)}, nil
		}
		(*calls)++
		var in struct {
			Project  string `json:"project"`
			Login    string `json:"login"`
			Identity int64  `json:"identity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return nil, err
		}
		code := *statusCode
		if code == 0 {
			code = http.StatusOK
		}
		response := struct {
			Project       string `json:"project"`
			Login         string `json:"login"`
			Identity      int64  `json:"identity"`
			Administrator bool   `json:"administrator"`
		}{in.Project, in.Login, in.Identity, *administrator}
		if *malformed {
			response.Identity++
		}
		body, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func factoryEncryptedServer(t *testing.T) *Server {
	t.Helper()
	db := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	for i, login := range []string{"alice", "bob"} {
		if err := db.UpsertUser(context.Background(), store.User{ID: int64(i + 1), Login: login, Name: login}); err != nil {
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
	s.API.Identity = &stubIdentityClient{
		connections: []identity.Connection{{ID: "conn-1", ProviderID: identity.Codex, OwnerID: 1, Label: "test", Generation: 1, State: identity.Ready}},
		grants: []identity.Grant{
			{ID: "grant-1", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject},
			{ID: "grant-2", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject, Revoked: true},
		},
	}
	sponsor := func(command, grant string, generation int64) string {
		return `{"command_id":"` + command + `","expected_revision":0,"grant_id":"` + grant + `","generation":` + strconv.FormatInt(generation, 10) +
			`,"roles":["soda-coder"],"allowance_minutes":60,"max_concurrent":1,"active":true}`
	}
	put := func(connection, body, login string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/"+connection, body, login))
		return w
	}
	w := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "alice")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	assertNoSecrets(t, w.Body.String())
	foreign := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "bob")
	if foreign.Code != 404 {
		t.Fatal("non-owner saw the connection", foreign.Code, foreign.Body.String())
	}
	rotated := put("conn-1", sponsor(factory.NewID(), "grant-1", 2), "alice")
	if rotated.Code != 409 {
		t.Fatal("rotated credential generation accepted", rotated.Code, rotated.Body.String())
	}
	missing := put("conn-1", sponsor(factory.NewID(), "grant-9", 1), "alice")
	if missing.Code != 404 {
		t.Fatal("unknown grant accepted", missing.Code, missing.Body.String())
	}
	revoked := put("conn-1", sponsor(factory.NewID(), "grant-2", 1), "alice")
	if revoked.Code != 409 {
		t.Fatal("revoked grant accepted", revoked.Code, revoked.Body.String())
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
	stub := s.API.Identity.(*stubIdentityClient)
	stub.grantsErr = identity.ErrDenied
	if w := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "alice"); w.Code != 403 {
		t.Fatal("denied broker grant accepted", w.Code, w.Body.String())
	}
	stub.grantsErr = errStubIdentity
	if w := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "alice"); w.Code != 503 {
		t.Fatal("broken broker grant accepted", w.Code, w.Body.String())
	}
	stub.grantsErr = nil
	stub.connErr = errStubIdentity
	if w := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "alice"); w.Code != 503 {
		t.Fatal("broken broker connection accepted", w.Code, w.Body.String())
	}
	stub.connErr = nil
	s.API.Identity = nil
	if w := put("conn-1", sponsor(factory.NewID(), "grant-1", 1), "alice"); w.Code != 503 {
		t.Fatal("missing identity service accepted", w.Code, w.Body.String())
	}
}

func TestFactoryConnectionUsageBudgetUsesBrokerOwnerAndReplays(t *testing.T) {
	s := factoryEncryptedServer(t)
	s.API.Identity = &stubIdentityClient{
		connections: []identity.Connection{{ID: "conn-1", ProviderID: identity.Codex, OwnerID: 1, Label: "test", Generation: 1, State: identity.Ready}},
		grants:      []identity.Grant{{ID: "grant-1", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject}},
	}
	sponsor := `{"command_id":"` + factory.NewID() + `","expected_revision":0,"grant_id":"grant-1","generation":1,"roles":["soda-coder"],"allowance_minutes":60,"max_concurrent":1,"active":true}`
	created := httptest.NewRecorder()
	nativeAPIServe(t, s, created, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/conn-1", sponsor, "alice"))
	if created.Code != http.StatusOK {
		t.Fatal(created.Code, created.Body.String())
	}
	path := "/api/factory/connections/conn-1/usage-budget"
	read := httptest.NewRecorder()
	nativeAPIServe(t, s, read, apiTestRequest(http.MethodGet, path, "", "alice"))
	var initial factory.ConnectionUsageBudget
	decodeBody(t, read, &initial)
	if read.Code != http.StatusOK || initial.RollingMinutes != factory.DefaultConnectionUsageBudgetMinutes || initial.Revision != 0 {
		t.Fatalf("initial budget = %+v, status %d", initial, read.Code)
	}
	command := factory.NewID()
	body := `{"command_id":"` + command + `","expected_revision":0,"rolling_minutes":720}`
	updated := httptest.NewRecorder()
	nativeAPIServe(t, s, updated, apiTestRequest(http.MethodPut, path, body, "alice"))
	var result factory.ConnectionUsageBudget
	decodeBody(t, updated, &result)
	if updated.Code != http.StatusOK || result.Revision != 1 || result.RollingMinutes != 720 {
		t.Fatalf("updated budget = %+v, status %d", result, updated.Code)
	}
	replay := httptest.NewRecorder()
	nativeAPIServe(t, s, replay, apiTestRequest(http.MethodPut, path, body, "alice"))
	if replay.Code != http.StatusOK || replay.Body.String() != updated.Body.String() {
		t.Fatalf("budget replay = %d %s", replay.Code, replay.Body.String())
	}
	changed := httptest.NewRecorder()
	nativeAPIServe(t, s, changed, apiTestRequest(http.MethodPut, path, strings.Replace(body, "720", "721", 1), "alice"))
	if changed.Code != http.StatusConflict {
		t.Fatalf("changed budget payload reused command: %d %s", changed.Code, changed.Body.String())
	}
	foreign := httptest.NewRecorder()
	nativeAPIServe(t, s, foreign, apiTestRequest(http.MethodGet, path, "", "bob"))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign connection budget was disclosed: %d %s", foreign.Code, foreign.Body.String())
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
	s.Config.OperatorID = 99 // exercise current native member authority, not the operator exception
	if err := s.Store.Join(t.Context(), factorySettingsProject, 1, "alice"); err != nil {
		t.Fatal(err)
	}
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
	accessCalls, statusCode := 0, 0
	administrator, malformed := true, false
	approvalProjectAccessHost(s, &accessCalls, &administrator, &statusCode, &malformed)
	wrong := httptest.NewRecorder()
	nativeAPIServe(t, s, wrong, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d999999999999999999999999"), "alice"))
	if wrong.Code != 409 {
		t.Fatal("approval bound a superseded requirement", wrong.Code, wrong.Body.String())
	}
	if accessCalls != 1 {
		t.Fatalf("member approval did not use exact native project observation: calls=%d", accessCalls)
	}
	// Bob has a current native session but no Project membership, so no host
	// privilege observation or approval mutation is allowed.
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "bob"))
	if denied.Code != 403 {
		t.Fatal("nonmember reached privileged-effect approval", denied.Code, denied.Body.String())
	}
	if accessCalls != 1 {
		t.Fatalf("nonmember triggered native authority lookup: calls=%d", accessCalls)
	}
	// Repository owner Alice is not a Project administrator: repository
	// ownership cannot substitute for the current guest-native result.
	administrator = false
	ownerDenied := httptest.NewRecorder()
	nativeAPIServe(t, s, ownerDenied, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "alice"))
	if ownerDenied.Code != 403 {
		t.Fatal("repository owner without native Project privilege was admitted", ownerDenied.Code, ownerDenied.Body.String())
	}
	inspectBefore := httptest.NewRecorder()
	nativeAPIServe(t, s, inspectBefore, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", `{"action":"inspect"}`, "alice"))
	var before struct {
		Approval string `json:"approval"`
	}
	decodeBody(t, inspectBefore, &before)
	if before.Approval != "" {
		t.Fatalf("denied request changed approval state: %+v", before)
	}
	// Native errors and mismatched echoes remain unconfirmed, not authority.
	statusCode = http.StatusServiceUnavailable
	uncertain := httptest.NewRecorder()
	nativeAPIServe(t, s, uncertain, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "alice"))
	if uncertain.Code != 502 {
		t.Fatal("unavailable native authority was admitted", uncertain.Code, uncertain.Body.String())
	}
	statusCode = 0
	malformed = true
	badEcho := httptest.NewRecorder()
	nativeAPIServe(t, s, badEcho, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "alice"))
	if badEcho.Code != 502 {
		t.Fatal("mismatched native authority echo was admitted", badEcho.Code, badEcho.Body.String())
	}
	malformed = false
	inspectUnconfirmed := httptest.NewRecorder()
	nativeAPIServe(t, s, inspectUnconfirmed, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", `{"action":"inspect"}`, "alice"))
	var unconfirmedView struct {
		Approval string `json:"approval"`
	}
	decodeBody(t, inspectUnconfirmed, &unconfirmedView)
	if unconfirmedView.Approval != "" {
		t.Fatalf("uncertain native result changed approval state: %+v", unconfirmedView)
	}
	if err := s.Store.Join(t.Context(), factorySettingsProject, 2, "bob"); err != nil {
		t.Fatal(err)
	}
	// Bob is a non-owner Project member with the positive native stub result.
	administrator = true
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve(factory.NewID(), "d123456789012345678901234"), "bob"))
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

func TestPreparationApprovalFixedOperatorSkipsNativeLookup(t *testing.T) {
	s := factorySettingsServer(t)
	digest := strings.Repeat("d", 64)
	accept := `{"command_id":"` + factory.NewID() + `","decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + strings.Repeat("e", 40) + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	accepted := httptest.NewRecorder()
	nativeAPIServe(t, s, accepted, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", accept, "alice"))
	if accepted.Code != 201 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	calls, statusCode := 0, http.StatusServiceUnavailable
	administrator, malformed := false, true
	approvalProjectAccessHost(s, &calls, &administrator, &statusCode, &malformed)
	approve := `{"command_id":"` + factory.NewID() + `","action":"approve","decision_id":"d223456789012345678901234",` +
		`"requirement":"d123456789012345678901234","effects_digest":"` + digest + `","readiness_digest":"` + digest + `","verified":true}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve, "alice"))
	if w.Code != 201 {
		t.Fatal("fixed operator exception failed", w.Code, w.Body.String())
	}
	if calls != 0 {
		t.Fatalf("fixed operator queried Project access: calls=%d", calls)
	}
}
