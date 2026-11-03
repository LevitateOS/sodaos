package web

import (
	"bytes"
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
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/identity"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
)

const factorySettingsProject = "p765432109876543210987654"

func factorySettingsServer(t *testing.T) *Server {
	t.Helper()
	s := apiTestServer(t)
	// Baseline journeys use the declared retained Fountain numeric namespace.
	// The mapped journey below deliberately allocates different local keys.
	for id := int64(1); id <= 7; id++ {
		if _, err := s.API.FountainIdentities.LocalRepository(t.Context(), id); err != nil {
			t.Fatal(err)
		}
	}
	for id := int64(3); id <= 16; id++ {
		if _, err := s.API.FountainIdentities.LocalActor(t.Context(), strconv.FormatInt(id, 10)); err != nil {
			t.Fatal(err)
		}
	}
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
	db := postgresFixture(t)
	for i, login := range []string{"alice", "bob"} {
		if err := db.UpsertUser(context.Background(), store.User{ID: int64(i + 1), Login: login, Name: login}); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewFactory(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
	if err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 7; id++ {
		if _, err := s.API.FountainIdentities.LocalRepository(t.Context(), id); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func factoryPolicyBody() string {
	return `{"expected_revision":0,"enabled":true,"paused":false,` +
		`"target_branch":"refs/heads/main",` +
		`"roles":{"soda-coder":{"harness":"muse","harness_version":"muse-1.4.9","model":"test"},"soda-reviewer":{"harness":"muse","harness_version":"muse-1.4.9","model":"test"}},` +
		`"required_checks":["native-ci/build"],"merge_method":"fast-forward-only",` +
		`"publish_actor":{"token_id":"11","actor_id":"12"},"create_actor":{"token_id":"11","actor_id":"12"},` +
		`"review_actor":{"token_id":"13","actor_id":"14"},"merge_actor":{"token_id":"15","actor_id":"16"},` +
		`"intent_minutes":2}`
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
	s.Config.FactoryNativeBindings = []config.FactoryNativeBinding{
		{TokenID: 11, ActorID: 12, RepositoryID: 7, Kind: factory.OpRefPublish},
		{TokenID: 21, ActorID: 22, RepositoryID: 8, Kind: factory.OpRefPublish},
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/repositories/7/factory", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var view struct {
		NativeBindings []factory.ActorBindingRef `json:"native_bindings"`
		Effective      struct {
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
	if len(view.NativeBindings) != 1 || view.NativeBindings[0] != (factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish}) {
		t.Fatalf("initial policy inventory crossed repository or omitted enrollment: %+v", view.NativeBindings)
	}
	assertNoSecrets(t, w.Body.String())
}

func TestFactoryPolicyJourney(t *testing.T) {
	t.Run("canonical_repository_and_native_authority", func(t *testing.T) {
		admin, ok := store.TestFixtureDSN()
		if !ok {
			t.Skip("SODA_PG_* fixture unavailable")
		}
		db, dsn, cleanup, err := store.OpenEphemeral(t.Context(), admin)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cleanup)
		inspection, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal("could not inspect the owned fixture")
		}
		t.Cleanup(func() { _ = inspection.Close() })
		c := config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1, FactoryNativeBindings: []config.FactoryNativeBinding{
			{RepositoryID: 7, TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
			{RepositoryID: 7, TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
			{RepositoryID: 7, TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
			{RepositoryID: 7, TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		}}
		s, err := NewFactory(c, db)
		if err != nil {
			t.Fatal(err)
		}
		github := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github.example.test"}
		// Use keys beyond JavaScript's exact-number range throughout the real
		// HTTP/store journey; native callback scope remains repository 7.
		for _, table := range []string{"collaboration_repositories", "collaboration_actors"} {
			if _, err := inspection.ExecContext(t.Context(), `SELECT setval(pg_get_serial_sequence($1,'id'),9007199254740991,true)`, table); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.PutCollaborationHost(t.Context(), github); err != nil {
			t.Fatal(err)
		}
		foreign, err := db.ObserveCollaborationRepository(t.Context(), github.ID, "7", "alice/repo")
		if err != nil {
			t.Fatal(err)
		}
		foreignActor, err := db.ObserveCollaborationActor(t.Context(), github.ID, "1", "alice")
		if err != nil {
			t.Fatal(err)
		}
		if err := db.UpsertUser(t.Context(), store.User{ID: 1, Login: "alice", Name: "alice"}); err != nil {
			t.Fatal(err)
		}
		if err := db.CreateProject(t.Context(), store.Project{ID: factorySettingsProject, Name: "Factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/repo"}); err != nil {
			t.Fatal(err)
		}
		if err := db.Join(t.Context(), factorySettingsProject, 1, "alice"); err != nil {
			t.Fatal(err)
		}
		repositoryCalls := 0
		issueContextCalls := 0
		intakeAuthorizations := 0
		callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
			if in.Operation == extensions.OperationOrganizationOwner {
				owner := false
				return extensions.CallbackResponse{Owner: &owner}
			}
			if in.Operation == extensions.OperationRepositoryIssues {
				issueContextCalls++
				if in.RepositoryID != "7" || in.Authority.Actor.ID != "1" {
					t.Errorf("canonical scope reached native issue context: repository=%s actor=%s", in.RepositoryID, in.Authority.Actor.ID)
				}
				return extensions.CallbackResponse{Issues: &extensions.RepositoryIssuePage{RepositoryID: "7", Items: []extensions.RepositoryIssue{}}}
			}
			if in.Operation == extensions.OperationIssueIntakeAuthorize {
				intakeAuthorizations++
				if in.RepositoryID != "7" || in.Authority.Actor.ID != "1" || in.IssueIntake == nil {
					t.Error("canonical scope reached the native proof issuer")
					return extensions.CallbackResponse{}
				}
				return extensions.CallbackResponse{IssueIntake: &extensions.IssueIntakeGrant{Token: "fixture-intake-" + in.IssueIntake.Source, Claims: extensions.IssueIntakeClaims{Installation: "fixture-native", Actor: "1", Repository: "7", Source: in.IssueIntake.Source, Revision: in.IssueIntake.Revision, Digest: in.IssueIntake.Digest, Expires: time.Now().Add(time.Hour).Unix()}}}
			}
			if in.Operation == extensions.OperationRepository {
				repositoryCalls++
				if in.RepositoryID != "7" {
					t.Errorf("canonical key reached native permission check: %s", in.RepositoryID)
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "admin"}}
			}
			return extensions.CallbackResponse{}
		}
		call := func(method, path, body string, want int) *httptest.ResponseRecorder {
			response := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, response, apiTestRequest(method, path, body, "alice"), callback)
			if response.Code != want {
				t.Fatalf("%s %s: %d want %d: %s", method, path, response.Code, want, response.Body.String())
			}
			return response
		}
		resolved := call(http.MethodPost, "/api/factory/native-repositories/resolve", `{"repository":"7"}`, 200)
		var resolvedIdentity struct {
			Repository string `json:"repository"`
			Actor      string `json:"actor"`
			Native     string `json:"native_repository"`
		}
		decodeBody(t, resolved, &resolvedIdentity)
		var viewerIdentity struct {
			Host   string `json:"host"`
			Actor  string `json:"actor"`
			Native string `json:"native_actor"`
		}
		decodeBody(t, call(http.MethodGet, "/api/factory/identity", "", 200), &viewerIdentity)
		if viewerIdentity.Host != string(s.FountainHost.ID) || viewerIdentity.Actor != resolvedIdentity.Actor || viewerIdentity.Native != "1" {
			t.Fatalf("host identity did not distinguish canonical and native actors: %+v", viewerIdentity)
		}
		if resolvedIdentity.Repository == "7" || resolvedIdentity.Repository == strconv.FormatInt(int64(foreign.ID), 10) || resolvedIdentity.Actor == "1" || resolvedIdentity.Actor == strconv.FormatInt(int64(foreignActor.ID), 10) || resolvedIdentity.Native != "7" {
			t.Fatalf("repository discovery conflated native/local/provider IDs: %+v", resolvedIdentity)
		}
		var reverse struct {
			Host, Repository, Actor, Name string
			Native                        string `json:"native_repository"`
		}
		reversePath := "/api/factory/repositories/" + resolvedIdentity.Repository + "/identity"
		decodeBody(t, call(http.MethodGet, reversePath, "", 200), &reverse)
		if reverse.Host != viewerIdentity.Host || reverse.Repository != resolvedIdentity.Repository || reverse.Actor != resolvedIdentity.Actor || reverse.Native != "7" || reverse.Name != "alice/repo" {
			t.Fatalf("canonical discovery lost native navigation/acting scope: %+v", reverse)
		}
		beforeReverse := repositoryCalls
		call(http.MethodGet, "/api/factory/repositories/"+strconv.FormatInt(int64(foreign.ID), 10)+"/identity", "", 404)
		call(http.MethodGet, "/api/factory/repositories/7/identity", "", 404)
		call(http.MethodGet, reversePath+"?native=7", "", 400)
		if repositoryCalls != beforeReverse {
			t.Fatal("foreign discovery reached native repository authority")
		}
		path := "/api/repositories/" + resolvedIdentity.Repository + "/factory"
		status := call(http.MethodGet, path, "", 200)
		var view struct {
			NativeBindings []factory.ActorBindingRef `json:"native_bindings"`
			Preparation    struct{ Project string }  `json:"preparation"`
		}
		decodeBody(t, status, &view)
		if view.Preparation.Project != factorySettingsProject || len(view.NativeBindings) != 4 {
			t.Fatalf("canonical status lost mapped Project or actor inventory: %+v", view)
		}
		actors := map[string]string{}
		for _, binding := range view.NativeBindings {
			native, err := s.API.FountainIdentities.NativeActor(t.Context(), collaboration.ActorID(binding.ActorID))
			if err != nil {
				t.Fatal(err)
			}
			actors[native] = strconv.FormatInt(binding.ActorID, 10)
		}
		body := factoryPolicyBody()
		for native, local := range actors {
			body = strings.ReplaceAll(body, `"actor_id":"`+native+`"`, `"actor_id":"`+local+`"`)
		}
		call(http.MethodPut, path+"/policy", body, 200)
		local, _ := strconv.ParseInt(resolvedIdentity.Repository, 10, 64)
		human, _ := strconv.ParseInt(resolvedIdentity.Actor, 10, 64)
		if human <= 9007199254740991 {
			t.Fatal("mapped journey must exercise canonical actors beyond JavaScript's exact range")
		}
		activityBody := `{"expected_revision":0,"settings":{"issue_created":false,"pr_created":true,"verified_merge":false}}`
		call(http.MethodPut, path+"/activity", activityBody, 200)
		var activityActor int64
		if err := inspection.QueryRowContext(t.Context(), `SELECT actor_id FROM factory_activity_settings WHERE repository=$1`, local).Scan(&activityActor); err != nil || activityActor != human {
			t.Fatalf("activity retained native actor attribution: %d %v", activityActor, err)
		}
		call(http.MethodPut, path+"/activity", activityBody, 412)
		repairID := factory.NewID()
		repairBody := `{"command_id":"` + repairID + `","expected_revision":0,"investigator_runs":1,"proposal_judgments":1,"active_minutes":1,"active":true}`
		call(http.MethodPut, path+"/repair", repairBody, 200)
		call(http.MethodPut, path+"/repair", repairBody, 200)
		if repair, err := db.RepairGrantCommand(t.Context(), local); err != nil || repair.Sponsor != human || repair.Repository != local || repair.ID != repairID {
			t.Fatalf("repair lost canonical sponsor/idempotency: %+v %v", repair, err)
		}
		if command, err := db.FactoryCommand(t.Context(), repairID); err != nil || command.Principal != "native:1" {
			t.Fatalf("repair rewrote its native admission principal: %+v %v", command, err)
		}
		policy, err := db.RepositoryPolicy(t.Context(), local)
		if err != nil || policy.Repository != local || policy.GrantedBy != human || policy.Revision != 1 {
			t.Fatalf("policy did not retain canonical scope/actor: %+v %v", policy, err)
		}
		call(http.MethodPut, path+"/operator-grant", `{"expected_revision":0,"active":true}`, 200)
		operator, err := db.OperatorGrant(t.Context(), local)
		if err != nil || operator.GrantedBy != human {
			t.Fatalf("native operator check did not retain canonical human attribution: %+v %v", operator, err)
		}
		call(http.MethodPut, "/api/factory/capacity", `{"expected_revision":0,"max_concurrent_runs":1}`, 200)
		capacity, err := db.Capacity(t.Context())
		if err != nil || capacity.UpdatedBy != human {
			t.Fatalf("native operator capacity update lost canonical human attribution: %+v %v", capacity, err)
		}
		s.API.Identity = &stubIdentityClient{
			connections: []identity.Connection{{ID: "mapped-conn", ProviderID: identity.Codex, OwnerID: 1, Generation: 1, State: identity.Ready}},
			grants:      []identity.Grant{{ID: "mapped-grant", ConnectionID: "mapped-conn", UserID: 1, ProjectID: factorySettingsProject}},
		}
		call(http.MethodPut, path+"/sponsorships/mapped-conn", `{"expected_revision":0,"roles":["soda-coder"],"allowance_minutes":60,"active":true}`, 200)
		sponsor, err := db.Sponsorship(t.Context(), local, "mapped-conn")
		if err != nil || sponsor.GrantedBy != human || sponsor.ActorID != 1 || sponsor.ProjectID != factorySettingsProject {
			t.Fatalf("canonical sponsor was conflated with native broker execution ownership: %+v %v", sponsor, err)
		}
		var sponsored struct {
			Sponsorships []factory.Sponsorship `json:"sponsorships"`
		}
		decodeBody(t, call(http.MethodGet, path, "", 200), &sponsored)
		if len(sponsored.Sponsorships) != 1 || sponsored.Sponsorships[0].AllowanceMinutes != 60 {
			t.Fatalf("canonical sponsor lost its own allowance visibility: %+v", sponsored)
		}
		var access struct {
			Viewer struct{ Member bool } `json:"viewer"`
		}
		decodeBody(t, call(http.MethodGet, path+"/access", "", 200), &access)
		if !access.Viewer.Member {
			t.Fatal("canonical API scope lost native Project membership")
		}
		if native, err := db.Project(t.Context(), factorySettingsProject); err != nil || native.OwnerID != 1 || native.RepositoryID != 7 {
			t.Fatalf("Factory settings rewrote native Project ownership: %+v %v", native, err)
		}
		var nativeUsers int
		if err := inspection.QueryRowContext(t.Context(), `SELECT count(*) FROM users WHERE id::bigint=$1`, human).Scan(&nativeUsers); err != nil || nativeUsers != 0 {
			t.Fatalf("canonical human was written into native user custody: %d %v", nativeUsers, err)
		}
		before := repositoryCalls
		call(http.MethodGet, "/api/repositories/"+strconv.FormatInt(int64(foreign.ID), 10)+"/factory", "", 404)
		call(http.MethodGet, "/api/repositories/7/factory", "", 404)
		if repositoryCalls != before {
			t.Fatal("foreign or missing local mapping reached native authority")
		}
		badActor := strings.Replace(body, `"actor_id":"`+actors["12"]+`"`, `"actor_id":"`+strconv.FormatInt(int64(foreignActor.ID), 10)+`"`, 1)
		call(http.MethodPut, path+"/policy", badActor, 400)
		if after, err := db.RepositoryPolicy(t.Context(), local); err != nil || after.Revision != 1 || after.GrantedBy != human {
			t.Fatalf("foreign actor changed stored policy: %+v %v", after, err)
		}
		// Request ownership is a collaboration actor; native session 1 remains
		// the authority at the private host/broker boundary, never the ledger owner.
		encode := func(value any) string {
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			return string(data)
		}
		department := factory.Department{ID: "d" + strings.Repeat("4", 24), Owner: int64(foreignActor.ID), Name: "Mapped requests", Description: "Canonical request ownership"}
		var savedDepartment factory.Department
		decodeBody(t, call(http.MethodPost, "/api/factory/departments", encode(department), 201), &savedDepartment)
		if savedDepartment.Owner != human {
			t.Fatalf("department retained native/foreign owner: %+v", savedDepartment)
		}
		var departments []factory.Department
		decodeBody(t, call(http.MethodGet, "/api/factory/departments", "", 200), &departments)
		if len(departments) != 1 || departments[0].Owner != human {
			t.Fatalf("canonical department read: %+v", departments)
		}
		idea := factory.ProductIdea{ID: "d" + strings.Repeat("5", 24), Description: "Track work", Consequence: "The owner can review progress", Decision: "accept", Source: "requested", Provenance: "human", Actor: human}
		draft := factory.IntentRevision{ID: "d" + strings.Repeat("3", 24), Intent: "d" + strings.Repeat("3", 24), Department: department.ID, Owner: int64(foreignActor.ID), Repository: local, Objective: "Track mapped work", State: "draft", MaxIssues: 1, Ideas: []factory.ProductIdea{idea}}
		var savedDraft factory.IntentRevision
		decodeBody(t, call(http.MethodPost, "/api/factory/intents", encode(draft), 201), &savedDraft)
		if savedDraft.Owner != human || savedDraft.Ideas[0].Actor != human {
			t.Fatalf("request lost canonical provenance: %+v", savedDraft)
		}
		intentPath := "/api/factory/intents/" + draft.Intent
		var readDraft factory.IntentRevision
		decodeBody(t, call(http.MethodGet, intentPath, "", 200), &readDraft)
		if readDraft.Owner != human || readDraft.ID != draft.ID {
			t.Fatalf("canonical request read: %+v", readDraft)
		}
		var listed struct {
			Intents []factory.IntentSummary `json:"intents"`
		}
		decodeBody(t, call(http.MethodGet, "/api/factory/intents?repository="+resolvedIdentity.Repository, "", 200), &listed)
		if len(listed.Intents) != 1 || listed.Intents[0].Intent != draft.Intent {
			t.Fatalf("canonical request list: %+v", listed)
		}
		call(http.MethodGet, intentPath+"/history", "", 200)
		call(http.MethodGet, intentPath+"/review", "", 200)
		call(http.MethodGet, intentPath+"/overview", "", 200)
		call(http.MethodGet, intentPath+"/revisions", "", 200)
		call(http.MethodGet, intentPath+"/plan", "", 200)
		call(http.MethodGet, intentPath+"/generation", "", 200)
		call(http.MethodPost, intentPath+"/plan", `{"revision":"d000000000000000000000000","operation":"decompose"}`, 412)
		call(http.MethodPost, intentPath+"/explore", `{"revision":"d000000000000000000000000"}`, 412)
		call(http.MethodPost, intentPath+"/generate", `{}`, 409)
		wrong := draft
		wrong.ID = "d" + strings.Repeat("9", 24)
		wrong.Predecessor = draft.ID
		wrong.Ideas = append([]factory.ProductIdea(nil), draft.Ideas...)
		wrong.Ideas[0].Actor = 1
		call(http.MethodPost, "/api/factory/intents", encode(wrong), 400)
		accepted := draft
		accepted.ID, accepted.Predecessor, accepted.State = "d"+strings.Repeat("6", 24), draft.ID, "accepted"
		accepted.Tasks = []factory.IntentTask{{Key: "d" + strings.Repeat("7", 24), Objective: "Show work", Outcomes: []string{"The owner sees progress"}, IdeaIDs: []string{idea.ID}}}
		accepted.EffectiveOutcomes = []string{idea.ID}
		call(http.MethodPost, "/api/factory/intents", encode(accepted), 201)
		call(http.MethodGet, intentPath+"/verification-allowance", "", 200)
		increase := factory.AppVerificationAllowanceChange{CommandID: factory.NewID(), Limits: factory.AppVerificationLimits{Runs: factory.DefaultAppVerificationRuns + 1, ActiveMinutes: factory.DefaultAppVerificationActiveMinutes + 1}}
		allowancePath := intentPath + "/verification-allowance"
		var allowance factory.AppVerificationAllowance
		decodeBody(t, call(http.MethodPost, allowancePath, encode(increase), 200), &allowance)
		command, err := db.FactoryCommand(t.Context(), increase.CommandID)
		if err != nil || command.Principal != "native:1" || command.Target != factory.AppVerificationAllowanceTarget(draft.Intent) || command.Finished == "" || allowance.Revision != increase.CommandID || allowance.Limits != increase.Limits {
			t.Fatalf("verification increase confused canonical owner and native command attribution: %+v %+v %v", command, allowance, err)
		}
		var replay factory.AppVerificationAllowance
		decodeBody(t, call(http.MethodPost, allowancePath, encode(increase), 200), &replay)
		if replay.Revision != allowance.Revision || replay.Limits != allowance.Limits || replay.Consumption != allowance.Consumption {
			t.Fatal("verification increase replay reset limits or usage", replay)
		}
		journal := func() string {
			t.Helper()
			var digest string
			if err := inspection.QueryRowContext(t.Context(), `SELECT md5(COALESCE(jsonb_agg(to_jsonb(c) ORDER BY id),'[]'::jsonb)::text) FROM factory_commands c`).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			return digest
		}
		beforeAllowanceJournal := journal()
		for _, wrongOwner := range []int64{1, int64(foreignActor.ID), human + 1} {
			if _, err := db.IncreaseAppVerificationAllowance(t.Context(), draft.Intent, wrongOwner, draft.Repository, "native:1", increase, time.Now()); !errors.Is(err, factory.ErrVerificationAllowanceOwner) {
				t.Fatal("native/foreign actor replayed the canonical owner's verification grant", err)
			}
		}
		for _, principal := range []string{"", strings.Repeat("p", 129)} {
			change := increase
			change.CommandID, change.ExpectedRevision = factory.NewID(), increase.CommandID
			change.Limits.Runs++
			if _, err := db.IncreaseAppVerificationAllowance(t.Context(), draft.Intent, human, draft.Repository, principal, change, time.Now()); err == nil {
				t.Fatal("invalid verification principal was recorded")
			}
		}
		for _, field := range []string{`"owner":"` + resolvedIdentity.Actor + `"`, `"actor":"1"`, `"principal":"native:` + resolvedIdentity.Actor + `"`} {
			call(http.MethodPost, allowancePath, strings.TrimSuffix(encode(increase), "}")+","+field+"}", 400)
		}
		changedIncrease := increase
		changedIncrease.Limits.Runs++
		call(http.MethodPost, allowancePath, encode(changedIncrease), 409)
		changedIncrease.CommandID = factory.NewID()
		call(http.MethodPost, allowancePath, encode(changedIncrease), 409)
		if journal() != beforeAllowanceJournal {
			t.Fatal("refused verification increases changed the command journal")
		}
		// Preserve an already captured command's attribution during replay.
		// This explicit retained fixture represents the earlier mislabeling;
		// replay authorizes the current canonical owner without rewriting it.
		retainedIncrease := increase
		retainedIncrease.CommandID, retainedIncrease.ExpectedRevision = factory.NewID(), increase.CommandID
		retainedIncrease.Limits.Runs++
		if _, err := db.IncreaseAppVerificationAllowance(t.Context(), draft.Intent, human, draft.Repository, "native:"+resolvedIdentity.Actor, retainedIncrease, time.Now()); err != nil {
			t.Fatal(err)
		}
		retainedCommand, err := db.FactoryCommand(t.Context(), retainedIncrease.CommandID)
		if err != nil {
			t.Fatal(err)
		}
		decodeBody(t, call(http.MethodPost, allowancePath, encode(retainedIncrease), 200), &replay)
		decodeBody(t, call(http.MethodPost, allowancePath, encode(increase), 200), &replay)
		if replay.Revision != retainedIncrease.CommandID || replay.Limits != retainedIncrease.Limits || replay.Consumption != allowance.Consumption || replay.CommandID != increase.CommandID {
			t.Fatal("old command replay rolled back a newer verification increase", replay)
		}
		if current, err := db.FactoryCommand(t.Context(), increase.CommandID); err != nil || current != command {
			t.Fatal("verification replay rewrote original native attribution", err)
		}
		if current, err := db.FactoryCommand(t.Context(), retainedIncrease.CommandID); err != nil || current != retainedCommand {
			t.Fatal("verification replay rewrote retained attribution", err)
		}
		foreignDraft := savedDraft
		foreignDraft.ID, foreignDraft.Intent, foreignDraft.Department = "d"+strings.Repeat("8", 24), "d"+strings.Repeat("8", 24), ""
		foreignDraft.Owner, foreignDraft.Repository, foreignDraft.Ideas = int64(foreignActor.ID), 0, nil
		if err := db.SaveIntentRevision(t.Context(), foreignDraft); err != nil {
			t.Fatal(err)
		}
		foreignPath := "/api/factory/intents/" + foreignDraft.Intent
		for _, suffix := range []string{"", "/history", "/review", "/overview", "/revisions", "/plan", "/generation"} {
			call(http.MethodGet, foreignPath+suffix, "", 404)
		}
		foreignEdit := foreignDraft
		foreignEdit.ID, foreignEdit.Predecessor, foreignEdit.Owner = "d"+strings.Repeat("a", 24), foreignDraft.ID, human
		call(http.MethodPost, "/api/factory/intents", encode(foreignEdit), 404)
		call(http.MethodGet, foreignPath+"/verification-allowance", "", 403)
		call(http.MethodPost, foreignPath+"/verification-allowance", encode(increase), 403)
		if retained, err := db.IntentRevision(t.Context(), foreignDraft.Intent, int64(foreignActor.ID)); err != nil || retained.ID != foreignDraft.ID {
			t.Fatalf("native session changed another provider actor's request: %+v %v", retained, err)
		}
		if _, err := db.IntentRevision(t.Context(), draft.Intent, human); err != nil {
			t.Fatal(err)
		}
		if _, err := db.IntentRevision(t.Context(), draft.Intent, 1); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("native actor acquired canonical request: %v", err)
		}
		{
			ctx := t.Context()
			foreignKey := strconv.FormatInt(int64(foreignActor.ID), 10)
			foreignSub := factory.FeedSubscription{Subscriber: foreignKey, Repositories: []int64{int64(foreign.ID)}, Revision: 1}
			if err := db.SetFeedSubscription(ctx, foreignKey, 0, foreignSub); err != nil {
				t.Fatal(err)
			}
			var subscription struct {
				Subscriber   string   `json:"subscriber"`
				Repositories []string `json:"repositories"`
				Revision     int64    `json:"revision"`
			}
			decodeBody(t, call(http.MethodGet, "/api/factory/feed/subscription", "", 200), &subscription)
			if subscription.Subscriber != resolvedIdentity.Actor || subscription.Revision != 0 {
				t.Fatalf("native session acquired a foreign subscription: %+v", subscription)
			}
			if local <= 9007199254740991 {
				t.Fatal("feed journey must exercise a repository beyond JavaScript's exact range")
			}
			for _, repositories := range []string{`[9007199254740993]`, `["9223372036854775808"]`, `["0"]`, `["07"]`, `["+7"]`} {
				call(http.MethodPut, "/api/factory/feed/subscription", `{"repositories":`+repositories+`,"expected_revision":0}`, 400)
			}
			call(http.MethodPut, "/api/factory/feed/subscription", encode(map[string]any{"repositories": []string{resolvedIdentity.Repository}, "expected_revision": 0, "subscriber": foreignKey}), 200)
			decodeBody(t, call(http.MethodGet, "/api/factory/feed/subscription", "", 200), &subscription)
			if subscription.Subscriber != resolvedIdentity.Actor || subscription.Revision != 1 || len(subscription.Repositories) != 1 || subscription.Repositories[0] != resolvedIdentity.Repository {
				t.Fatalf("subscription lost canonical owner/scope: %+v", subscription)
			}
			if retained, rev, err := db.FeedSubscription(ctx, foreignKey); err != nil || rev != 1 || len(retained.Repositories) != 1 || retained.Repositories[0] != int64(foreign.ID) {
				t.Fatalf("foreign subscription overwritten: %+v %d %v", retained, rev, err)
			}
			save := func(body, key, parent string) string {
				t.Helper()
				var result struct {
					ID string `json:"post_id"`
				}
				decodeBody(t, call(http.MethodPost, "/api/factory/feed/posts", encode(factory.PostRequest{Repository: local, Body: body, IdempotencyKey: key, ParentID: parent}), 200), &result)
				if result.ID == "" {
					t.Fatal("post identity missing")
				}
				return result.ID
			}
			call(http.MethodPost, "/api/factory/feed/posts", `{"repository":9007199254740993,"body":"Unsafe numeric scope","idempotency_key":"unsafe-numeric"}`, 400)
			post := save("Show the request status", "mapped-human-post", "")
			if repeated := save("Show the request status", "mapped-human-post", ""); repeated != post {
				t.Fatal("canonical author lost idempotent submission")
			}
			detail, err := db.FeedPost(ctx, post)
			if err != nil || detail.Record.Author != resolvedIdentity.Actor || detail.Record.Repository != local || detail.AuthorName != "alice" {
				t.Fatalf("post lost canonical author/native display metadata: %+v %v", detail, err)
			}
			if issueContextCalls == 0 {
				t.Fatal("post did not read native issue context")
			}
			if _, err := db.PostIssueContext(ctx, post, 0); err != nil {
				t.Fatalf("native context not captured for canonical post: %v", err)
			}
			proof, err := db.IssueIntakeGrant(ctx, post, 0)
			if err != nil {
				t.Fatal("canonical post did not retain its admitted proof", err)
			}
			var nativeProof extensions.IssueIntakeGrant
			if json.Unmarshal(proof, &nativeProof) != nil || nativeProof.Claims.Actor != "1" || nativeProof.Claims.Repository != "7" || nativeProof.Claims.Source != post {
				t.Fatal("canonical post rewrote native proof metadata")
			}
			if scope, err := s.API.FountainIdentities.DecodeIssueIntakeGrant(ctx, proof); err != nil || int64(scope.Actor) != human || int64(scope.Repository) != local || scope.Source != post || scope.Digest != factory.MCPInputDigest([]byte(detail.Body)) {
				t.Fatalf("retained proof did not resolve to canonical scope: %+v %v", scope, err)
			}
			if intakeAuthorizations != 1 {
				t.Fatalf("idempotent post requested %d native proofs", intakeAuthorizations)
			}
			var window struct {
				Entries []factory.FeedEntry `json:"entries"`
			}
			decodeBody(t, call(http.MethodGet, "/api/factory/feed?repositories="+resolvedIdentity.Repository, "", 200), &window)
			found := false
			for _, entry := range window.Entries {
				if entry.ID == post {
					found = entry.Actor == resolvedIdentity.Actor && entry.Repository == local
					joined, ok := entry.Post.(map[string]any)
					if !ok || joined["post_id"] != post || joined["repository"] != resolvedIdentity.Repository || joined["author"] != resolvedIdentity.Actor {
						t.Fatal("mapped feed lost shared post detail", entry.Post)
					}
				}
			}
			if !found {
				t.Fatal("combined feed did not use canonical subscription and author")
			}
			// Deliberately seed another provider's numeric actor at the same
			// local repository. Native actor 1 must not acquire that authorship.
			foreignPost, _, err := db.SaveFeedPost(ctx, factory.PostRequest{Repository: local, Body: "Other actor's request", IdempotencyKey: "foreign-human-post"}, foreignKey, "other alice", time.Now().Unix())
			if err != nil {
				t.Fatal(err)
			}
			if err := db.RecordIssueMakerIntake(ctx, foreignPost, 0); err != nil {
				t.Fatal(err)
			}
			foreignPipe, err := db.PostIntake(ctx, foreignPost)
			if err != nil {
				t.Fatal(err)
			}
			foreignRouted := foreignPipe
			foreignRouted.Judgment, foreignRouted.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
			if err := db.AdvancePostIntake(ctx, foreignPipe, foreignRouted); err != nil {
				t.Fatal(err)
			}
			if err := db.NotePostAuthoring(ctx, foreignPost, 0, factory.PostAuthoring{State: factory.PostAuthoringWaiting, Intent: foreignDraft.Intent, Detail: "Waiting for preparation"}, time.Now().Unix()); err != nil {
				t.Fatal(err)
			}
			var read struct {
				Repository string                   `json:"repository"`
				Author     string                   `json:"author"`
				CanRetry   bool                     `json:"can_retry"`
				Authoring  *factory.PostAuthoring   `json:"authoring"`
				Change     *store.PostProductChange `json:"product_change"`
			}
			decodeBody(t, call(http.MethodGet, "/api/factory/feed/posts/"+foreignPost, "", 200), &read)
			if read.Repository != resolvedIdentity.Repository || read.Author != foreignKey || read.CanRetry || read.Authoring == nil || read.Authoring.Intent != "" {
				t.Fatalf("native session acquired foreign post ownership: %+v", read)
			}
			call(http.MethodPost, "/api/factory/feed/posts/"+foreignPost+"/retry", `{"command_id":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","repository":"`+resolvedIdentity.Repository+`","expected_revision":0}`, 403)
			pipe, err := db.PostIntake(ctx, post)
			if err != nil {
				t.Fatal(err)
			}
			routed := pipe
			routed.Judgment, routed.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
			if err := db.AdvancePostIntake(ctx, pipe, routed); err != nil {
				t.Fatal(err)
			}
			// Declared authoring wait; this journey exercises real storage and
			// native HTTP admission without running a model or worker.
			waiting := factory.PostAuthoring{State: factory.PostAuthoringWaiting, Intent: draft.Intent, Reason: "preparation", Detail: "Waiting for preparation"}
			if err := db.NotePostAuthoring(ctx, post, 0, waiting, time.Now().Unix()); err != nil {
				t.Fatal(err)
			}
			postPath := "/api/factory/feed/posts/" + post
			decodeBody(t, call(http.MethodGet, postPath, "", 200), &read)
			if read.Author != resolvedIdentity.Actor || !read.CanRetry || read.Authoring == nil || read.Authoring.Intent != draft.Intent {
				t.Fatalf("canonical author lost private authoring/retry projection: %+v", read)
			}
			call(http.MethodPost, postPath+"/retry", `{"command_id":"ffffffffffffffffffffffffffffffff","repository":"`+resolvedIdentity.Repository+`","expected_revision":0,"body":"Show status and progress"}`, 200)
			if updated, err := db.FeedPost(ctx, post); err != nil || updated.Record.Revision != 1 || updated.Record.Author != resolvedIdentity.Actor || updated.Body != "Show status and progress" {
				t.Fatalf("canonical retry did not revise exact author: %+v %v", updated, err)
			}
			if renewed, err := db.IssueIntakeGrant(ctx, post, 1); err != nil || json.Unmarshal(renewed, &nativeProof) != nil || nativeProof.Claims.Revision != 1 || nativeProof.Claims.Digest != factory.MCPInputDigest([]byte("Show status and progress")) || intakeAuthorizations != 2 {
				t.Fatalf("retry did not renew the exact native revision: %v", err)
			}
			entries, _, _, err := db.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{local}, Limit: 50})
			if err != nil {
				t.Fatal(err)
			}
			parent := ""
			for _, entry := range entries {
				source, err := db.ResolvePostParent(ctx, local, entry.ID)
				if err == nil && source.Product != nil && source.Product.AcceptedRevision == accepted.ID {
					parent = entry.ID
					break
				}
			}
			if parent == "" {
				t.Fatal("accepted product source missing")
			}
			reply := save("Also show the current owner", "mapped-product-reply", parent)
			pipe, err = db.PostIntake(ctx, reply)
			if err != nil {
				t.Fatal(err)
			}
			routed = pipe
			routed.Judgment, routed.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
			if err := db.AdvancePostIntake(ctx, pipe, routed); err != nil {
				t.Fatal(err)
			}
			replyPath := "/api/factory/feed/posts/" + reply
			decodeBody(t, call(http.MethodGet, replyPath, "", 200), &read)
			if read.Change == nil || read.Change.State != "review_required" || !read.Change.CanReview {
				t.Fatalf("canonical product owner lost review: %+v", read)
			}
			var replies struct {
				Items []struct {
					Author     string `json:"author"`
					Repository string `json:"repository"`
				} `json:"items"`
			}
			decodeBody(t, call(http.MethodGet, "/api/factory/feed/posts/"+parent+"/replies?repository="+resolvedIdentity.Repository, "", 200), &replies)
			if len(replies.Items) != 1 || replies.Items[0].Author != resolvedIdentity.Actor || replies.Items[0].Repository != resolvedIdentity.Repository {
				t.Fatalf("reply lost canonical authorship: %+v", replies)
			}
			decodeBody(t, call(http.MethodPost, replyPath+"/review", `{"revision":0}`, 200), &read)
			if read.Change == nil || read.Change.State != "draft" || !read.Change.CanReview {
				t.Fatalf("canonical product review did not open draft: %+v", read)
			}
			reviewed, err := db.IntentRevision(ctx, draft.Intent, human)
			if err != nil || reviewed.Owner != human || reviewed.Predecessor != accepted.ID || len(reviewed.Ideas) != 2 || reviewed.Ideas[1].Actor != human || !strings.Contains(reviewed.Ideas[1].Evidence, "Factory actor "+resolvedIdentity.Actor) {
				t.Fatalf("review lost canonical provenance: %+v %v", reviewed, err)
			}
			continued := reviewed
			continued.ID, continued.Predecessor, continued.State = "d"+strings.Repeat("b", 24), reviewed.ID, "accepted"
			call(http.MethodPost, "/api/factory/intents", encode(continued), 201)
			decodeBody(t, call(http.MethodGet, replyPath, "", 200), &read)
			if read.Change == nil || read.Change.State != "accepted" || !read.Change.CanReview {
				t.Fatalf("accepted canonical product request cannot be opened: %+v", read)
			}
		}
		profile := `{"id":"rocky-headless","distribution":"rocky","version":"9.6","interface":"headless","architecture":"amd64","image":"sha256:` + strings.Repeat("b", 64) + `","revision":"` + strings.Repeat("c", 40) + `"}`
		environmentBody := `{"expected_revision":0,"profile":` + profile + `,"project":"` + factorySettingsProject + `","active":true}`
		environmentPath := path + "/environment-grant"
		call(http.MethodPut, environmentPath, environmentBody, 200)
		environment, err := db.EnvironmentGrant(t.Context(), local)
		if err != nil || environment.Repository != local || environment.Owner != human || environment.Project != factorySettingsProject || environment.Revision != 1 || !environment.Active {
			t.Fatalf("environment grant lost canonical owner/repository: %+v %v", environment, err)
		}
		var environmentStatus struct {
			Grant struct {
				Repository, Owner, Project string
			} `json:"environment_grant"`
		}
		decodeBody(t, call(http.MethodGet, path, "", 200), &environmentStatus)
		if environmentStatus.Grant.Owner != resolvedIdentity.Actor || environmentStatus.Grant.Repository != resolvedIdentity.Repository || environmentStatus.Grant.Project != factorySettingsProject {
			t.Fatalf("environment grant status conflated identity namespaces: %+v", environmentStatus)
		}
		beforeEnvironment := encode(environment)
		call(http.MethodPut, environmentPath, environmentBody, 412)
		for _, field := range []string{`"owner":"1"`, `"actor":"` + resolvedIdentity.Actor + `"`, `"principal":"native:` + resolvedIdentity.Actor + `"`} {
			call(http.MethodPut, environmentPath, strings.Replace(environmentBody, `"active":true`, `"active":true,`+field, 1), 400)
		}
		beforeEnvironmentCalls := repositoryCalls
		for _, wrong := range []int64{7, int64(foreign.ID)} {
			call(http.MethodPut, "/api/repositories/"+strconv.FormatInt(wrong, 10)+"/factory/environment-grant", environmentBody, 404)
			if _, err := db.EnvironmentGrant(t.Context(), wrong); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("environment grant leaked into a native/foreign repository", wrong, err)
			}
		}
		if repositoryCalls != beforeEnvironmentCalls {
			t.Fatal("foreign environment grant reached native repository authority")
		}
		if err := db.UpsertUser(t.Context(), store.User{ID: 2, Login: "bob", Name: "bob"}); err != nil {
			t.Fatal(err)
		}
		deniedEnvironment := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, deniedEnvironment, apiTestRequest(http.MethodPut, environmentPath, environmentBody, "bob"), callback)
		if deniedEnvironment.Code != 403 {
			t.Fatal("mapped environment grant admitted a non-owner", deniedEnvironment.Code, deniedEnvironment.Body.String())
		}
		unchanged, err := db.EnvironmentGrant(t.Context(), local)
		if err != nil || encode(unchanged) != beforeEnvironment {
			t.Fatalf("refused environment grant changed captured bytes: %+v %v", unchanged, err)
		}
		withdrawEnvironment := strings.Replace(environmentBody, `"expected_revision":0`, `"expected_revision":1`, 1)
		withdrawEnvironment = strings.Replace(withdrawEnvironment, `"active":true`, `"active":false`, 1)
		call(http.MethodPut, environmentPath, withdrawEnvironment, 200)
		open, _, withdrawal, err := db.DispatchState(t.Context(), local)
		if err != nil || open || withdrawal.Repository != local || withdrawal.Cause != "environment_grant_withdrawn" || withdrawal.ClosedBy != "native:1" {
			t.Fatalf("environment withdrawal confused canonical owner/native principal: %+v %v", withdrawal, err)
		}
		withdrawnEnvironment, err := db.EnvironmentGrant(t.Context(), local)
		if err != nil || withdrawnEnvironment.Owner != human || withdrawnEnvironment.Revision != 2 || withdrawnEnvironment.Active {
			t.Fatalf("withdrawal lost canonical environment ownership: %+v %v", withdrawnEnvironment, err)
		}
		nativeProject, err := db.Project(t.Context(), factorySettingsProject)
		if err != nil || nativeProject.OwnerID != 1 || nativeProject.RepositoryID != 7 {
			t.Fatalf("canonical environment grant rewrote native Project ownership: %+v %v", nativeProject, err)
		}
		registry := func() string {
			t.Helper()
			var digest string
			if err := inspection.QueryRowContext(t.Context(), `SELECT md5(jsonb_build_array(
			 (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM collaboration_repositories r),
			 (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM collaboration_actors a),
			 (SELECT last_value FROM collaboration_repositories_id_seq),
			 (SELECT last_value FROM collaboration_actors_id_seq))::text)`).Scan(&digest); err != nil {
				t.Fatal(err)
			}
			return digest
		}
		beforeRestartJournal, beforeRestartRegistry := journal(), registry()
		beforeRestartEnvironment, beforeRestartWithdrawal := encode(withdrawnEnvironment), encode(withdrawal)
		// Recompose against the retained mixed-provider store, rather than
		// interpreting today's canonical IDs as a fresh native-ledger import.
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		s, err = NewFactory(c, db)
		if err != nil {
			t.Fatal("recompose retained canonical Factory", err)
		}
		var resumedIdentity struct {
			Host, Actor string
			Native      string `json:"native_actor"`
		}
		decodeBody(t, call(http.MethodGet, "/api/factory/identity", "", 200), &resumedIdentity)
		if resumedIdentity.Host != viewerIdentity.Host || resumedIdentity.Actor != resolvedIdentity.Actor || resumedIdentity.Native != "1" {
			t.Fatalf("restart changed the host/canonical/native account binding: %+v", resumedIdentity)
		}
		call(http.MethodGet, path, "", 200)
		retainedEnvironment, err := db.EnvironmentGrant(t.Context(), local)
		if err != nil || encode(retainedEnvironment) != beforeRestartEnvironment {
			t.Fatal("restart changed the canonical environment grant", err)
		}
		open, _, withdrawal, err = db.DispatchState(t.Context(), local)
		if err != nil || open || encode(withdrawal) != beforeRestartWithdrawal {
			t.Fatal("restart reopened or reattributed withdrawn native execution", err)
		}
		decodeBody(t, call(http.MethodPost, allowancePath, encode(increase), 200), &replay)
		if replay.Revision != retainedIncrease.CommandID || replay.Limits != retainedIncrease.Limits || replay.Consumption != allowance.Consumption || replay.CommandID != increase.CommandID {
			t.Fatal("retained canonical command did not replay across recomposition", replay)
		}
		decodeBody(t, call(http.MethodPost, allowancePath, encode(retainedIncrease), 200), &replay)
		if current, err := db.FactoryCommand(t.Context(), increase.CommandID); err != nil || current != command {
			t.Fatal("restart rewrote original command attribution", err)
		}
		if current, err := db.FactoryCommand(t.Context(), retainedIncrease.CommandID); err != nil || current != retainedCommand {
			t.Fatal("restart rewrote captured retained attribution", err)
		}
		if journal() != beforeRestartJournal || registry() != beforeRestartRegistry {
			t.Fatal("restart/replay rewrote commands or reinterpreted canonical identities")
		}
		nativeProject, err = db.Project(t.Context(), factorySettingsProject)
		if err != nil || nativeProject.OwnerID != 1 || nativeProject.RepositoryID != 7 {
			t.Fatal("retained canonical grants acquired native Project custody", nativeProject, err)
		}
		if err := inspection.QueryRowContext(t.Context(), `SELECT count(*) FROM users WHERE id::bigint=$1`, human).Scan(&nativeUsers); err != nil || nativeUsers != 0 {
			t.Fatalf("request actor acquired native custody: %d %v", nativeUsers, err)
		}
	})
	s := factorySettingsServer(t)
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(), "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		CommandID string `json:"command_id"`
		Revision  int64  `json:"revision"`
		Withdrawn bool   `json:"withdrawn"`
	}
	decodeBody(t, w, &result)
	if result.CommandID != "" || result.Revision != 1 || result.Withdrawn {
		t.Fatalf("grant result: %+v", result)
	}
	policy, err := s.Store.RepositoryPolicy(t.Context(), 7)
	if err != nil || policy.AttemptLimits != factory.DefaultAttemptLimits() {
		t.Fatalf("omitted attempt limits were not recorded as defaults: %+v %v", policy.AttemptLimits, err)
	}
	assertNoSecrets(t, w.Body.String())
	replay := httptest.NewRecorder()
	nativeAPIServe(t, s, replay, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(), "alice"))
	if replay.Code != 412 {
		t.Fatal("stale revision saved twice", replay.Code, replay.Body.String())
	}
	changed := strings.Replace(factoryPolicyBody(), `"intent_minutes":2`, `"intent_minutes":3`, 1)
	conflict := httptest.NewRecorder()
	nativeAPIServe(t, s, conflict, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", changed, "alice"))
	if conflict.Code != 412 {
		t.Fatal("changed settings overwrote a stale revision", conflict.Code, conflict.Body.String())
	}
	stale := httptest.NewRecorder()
	nativeAPIServe(t, s, stale, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(), "alice"))
	if stale.Code != 412 {
		t.Fatal("stale revision applied", stale.Code, stale.Body.String())
	}
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(), "bob"))
	if denied.Code != 403 {
		t.Fatal("non-owner configured policy", denied.Code, denied.Body.String())
	}
	impersonate := strings.Replace(factoryPolicyBody(), `"enabled":true`, `"enabled":true,"granted_by":"2"`, 1)
	forged := httptest.NewRecorder()
	nativeAPIServe(t, s, forged, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", impersonate, "alice"))
	if forged.Code != 400 {
		t.Fatal("body-selected authorizer accepted", forged.Code, forged.Body.String())
	}
	invalidLimits := strings.Replace(factoryPolicyBody(), `"intent_minutes":2`, `"attempt_limits":{"active_minutes":0,"correction_cycles":3}`, 1)
	invalid := httptest.NewRecorder()
	nativeAPIServe(t, s, invalid, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", invalidLimits, "alice"))
	if invalid.Code != 400 {
		t.Fatal("zero active attempt time accepted", invalid.Code, invalid.Body.String())
	}
	customLimits := strings.Replace(factoryPolicyBody(), `"expected_revision":0`, `"expected_revision":1`, 1)
	customLimits = strings.Replace(customLimits, `"intent_minutes":2`, `"attempt_limits":{"active_minutes":45,"correction_cycles":1}`, 1)
	custom := httptest.NewRecorder()
	nativeAPIServe(t, s, custom, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", customLimits, "alice"))
	if custom.Code != 200 {
		t.Fatal("valid repository attempt limits rejected", custom.Code, custom.Body.String())
	}
	policy, err = s.Store.RepositoryPolicy(t.Context(), 7)
	// Creation records the frozen review/child defaults alongside the
	// posted minutes, cycles and the unfunded delivery posture.
	if err != nil || policy.AttemptLimits != (factory.AttemptLimits{ActiveMinutes: 45, CorrectionCycles: 1, ReviewRounds: 6, MaxChildren: 3}) {
		t.Fatalf("repository attempt limits were not recorded: %+v %v", policy.AttemptLimits, err)
	}
}

func TestSharedRepositoryPolicyAdmission(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("native PostgreSQL fixture is required")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	const principal = "profile:policy-owner"
	in := factoryservice.PolicyChange{
		Roles: map[string]factory.RoleSelection{
			"soda-coder":    {Harness: "muse", Model: "declared-fixture"},
			"soda-reviewer": {Harness: "muse", Model: "declared-fixture"},
		},
		PublishActor: factoryservice.PolicyActor{TokenID: 11, ActorID: 9007199254740995},
		CreateActor:  factoryservice.PolicyActor{TokenID: 11, ActorID: 9007199254740995},
		ReviewActor:  factoryservice.PolicyActor{TokenID: 13, ActorID: 9007199254740997},
		MergeActor:   factoryservice.PolicyActor{TokenID: 15, ActorID: 9007199254740999},
		TargetBranch: "refs/heads/main", MergeMethod: factory.MergeFastForward, RequiredChecks: []string{"native-ci/build"}, Enabled: true,
	}
	resolve := factoryservice.PolicyActor{TokenID: 17, ActorID: in.ReviewActor.ActorID}
	in.ResolveActor = &resolve
	allowed, revokeDuringBinding := true, false
	checks, bindings := 0, []factory.ActorBindingRef{}
	denied := errors.New("declared current repository administration withdrawn")
	var bindingErr error
	application := factoryservice.Policies{
		Writer: control.NewCoordinator(db, nil, nil),
		WriteAccess: func(ctx context.Context, owner collaboration.ActorID, repo int64) error {
			checks++
			if owner != actor || repo != int64(repository) {
				t.Fatalf("policy authority subject changed: %d %d", owner, repo)
			}
			if !allowed {
				return denied
			}
			return nil
		},
		BindingAccess: func(ctx context.Context, owner collaboration.ActorID, repo collaboration.RepositoryID, ref factory.ActorBindingRef) error {
			if owner != actor || repo != repository {
				t.Fatal("binding scope changed", owner, repo)
			}
			bindings = append(bindings, ref)
			if revokeDuringBinding {
				allowed = false
			}
			return bindingErr
		},
	}
	if _, err := (factoryservice.Policies{}).Apply(ctx, actor, principal, repository, in); !errors.Is(err, factoryservice.ErrPolicyUnavailable) {
		t.Fatal("unwired policy admission", err)
	}
	allowed = false
	if _, err := application.Apply(ctx, actor, principal, repository, in); !errors.Is(err, denied) || len(bindings) != 0 {
		t.Fatal("denied owner reached binding admission", err, bindings)
	}
	allowed, revokeDuringBinding = true, true
	if _, err := application.Apply(ctx, actor, principal, repository, in); !errors.Is(err, denied) {
		t.Fatal("binding reads outlived repository administration", err)
	}
	if _, err := db.RepositoryPolicy(ctx, int64(repository)); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("late denial recorded policy", err)
	}
	allowed, revokeDuringBinding = true, false
	bindingErr = factoryservice.ErrPolicyBindingInvalid
	if _, err := application.Apply(ctx, actor, principal, repository, in); !errors.Is(err, factoryservice.ErrPolicyBindingInvalid) {
		t.Fatal("foreign binding admitted", err)
	}
	bindingErr = factoryservice.ErrPolicyBindingUnavailable
	if _, err := application.Apply(ctx, actor, principal, repository, in); !errors.Is(err, factoryservice.ErrPolicyBindingUnavailable) {
		t.Fatal("unknown binding scope admitted", err)
	}
	bindingErr = nil
	wrong := in
	wrongResolve := resolve
	wrongResolve.ActorID = in.PublishActor.ActorID
	wrong.ResolveActor = &wrongResolve
	if _, err := application.Apply(ctx, actor, principal, repository, wrong); !errors.Is(err, factoryservice.ErrPolicyResolverActor) {
		t.Fatal("resolver switched away from independent reviewer", err)
	}
	wrong = in
	wrong.IntentMinutes = -1
	if _, err := application.Apply(ctx, actor, principal, repository, wrong); !errors.Is(err, factoryservice.ErrPolicyInvalid) {
		t.Fatal("invalid policy admitted", err)
	}
	bindings, checks = nil, 0
	result, err := application.Apply(ctx, actor, principal, repository, in)
	if err != nil || result.Revision != 1 || result.Withdrawn || checks != 2 || len(bindings) != 5 {
		t.Fatalf("shared policy admission: %+v %v checks=%d bindings=%v", result, err, checks, bindings)
	}
	for i, kind := range []string{factory.OpRefPublish, factory.OpPRCreate, factory.OpReviewSubmit, factory.OpMerge, factory.OpThreadResolve} {
		if bindings[i].Kind != kind {
			t.Fatalf("binding operation escaped its slot: %+v", bindings)
		}
	}
	policy, err := db.RepositoryPolicy(ctx, int64(repository))
	if err != nil || policy.GrantedBy != int64(actor) || policy.Repository != int64(repository) || policy.IntentMinutes != 0 || policy.AttemptLimits != factory.DefaultAttemptLimits() || policy.ReassessmentMinutes != factory.DefaultReassessmentMinutes || policy.Resolve.ActorID != policy.Review.ActorID {
		t.Fatalf("policy identity/defaults/unfunded posture: %+v %v", policy, err)
	}
	if _, err := application.Apply(ctx, actor, principal, repository, in); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("stale policy CAS admitted", err)
	}
	paused := in
	paused.ExpectedRevision, paused.Paused = 1, true
	paused.AttemptLimits = &factory.AttemptLimits{ActiveMinutes: 10, CorrectionCycles: 0}
	paused.ReassessmentMinutes, paused.IntentMinutes = 45, 2
	result, err = application.Apply(ctx, actor, principal, repository, paused)
	if err != nil || !result.Withdrawn || result.Revision != 2 {
		t.Fatalf("shared pause did not withdraw: %+v %v", result, err)
	}
	open, _, withdrawal, err := db.DispatchState(ctx, int64(repository))
	if err != nil || open || withdrawal.Repository != int64(repository) || withdrawal.ClosedBy != principal || withdrawal.Cause != "policy_paused" {
		t.Fatalf("canonical dispatch withdrawal: %+v %v open=%t", withdrawal, err, open)
	}
	policy, err = db.RepositoryPolicy(ctx, int64(repository))
	if err != nil || policy.AttemptLimits.ActiveMinutes != 10 || policy.AttemptLimits.CorrectionCycles != 0 || policy.AttemptLimits.ReviewRounds != factory.DefaultReviewRounds || policy.IntentMinutes != 2 || policy.ReassessmentMinutes != 45 {
		t.Fatalf("explicit policy limits/default fills lost: %+v %v", policy, err)
	}
	allowed = false
	before := len(bindings)
	if _, err := application.Apply(ctx, actor, principal, repository, paused); !errors.Is(err, denied) || len(bindings) != before {
		t.Fatal("known policy bypassed current administration", err)
	}
	allowed = true
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := application.Apply(cancelled, actor, principal, repository, paused); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled policy admission", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	application.Writer = control.NewCoordinator(reopened, nil, nil)
	retained, err := reopened.RepositoryPolicy(ctx, int64(repository))
	beforeJSON, _ := json.Marshal(policy)
	afterJSON, _ := json.Marshal(retained)
	if err != nil || string(beforeJSON) != string(afterJSON) {
		t.Fatal("reopened policy changed", retained, err)
	}
	if _, err := application.Apply(ctx, actor, principal, repository, paused); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("reopened policy lost CAS", err)
	}
	if runs, err := reopened.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
		t.Fatal("policy admission launched work", runs, err)
	}
	// A direct HTTP host binds the same canonical actor and independent command
	// principal to these real store/controller services, without a native session.
	var admitted, operator atomic.Bool
	admitted.Store(true)
	operator.Store(true)
	visibility := func(ctx context.Context, owner collaboration.ActorID, repo collaboration.RepositoryID) error {
		if owner != actor || repo != repository {
			return errors.New("declared repository visibility unavailable")
		}
		return ctx.Err()
	}
	transport := factoryhttpapi.Repositories{Store: reopened, Policies: application,
		Repair: factoryservice.RepairFunding{Store: reopened, Administration: application.WriteAccess},
		Runtime: factoryservice.RuntimeGrants{Writer: control.NewCoordinator(reopened, nil, nil), Visibility: visibility,
			OperatorAccess: func(ctx context.Context, owner collaboration.ActorID) error {
				if owner != actor || !operator.Load() {
					return factoryservice.ErrRuntimeGrantAccess
				}
				return ctx.Err()
			}},
		Host: collaboration.HostID(strings.Repeat("c", 32)), Actor: actor, Principal: principal, Visibility: visibility,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if !admitted.Load() || r.Header.Get("Authorization") != "Bearer declared-repository-client" {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		},
	}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	direct := func(method, path, body string, status int) []byte {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-repository-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status {
			t.Fatalf("repository HTTP %s %s: %d %s %v", method, path, response.StatusCode, data, err)
		}
		if status == 200 && response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("repository response was cacheable")
		}
		return data
	}
	path := "/api/repositories/9223372036854775807/factory/policy"
	paused.ExpectedRevision = 2
	policyBody, _ := json.Marshal(paused)
	direct(http.MethodPut, path, string(policyBody), 200)
	direct(http.MethodPut, path, string(policyBody), 412)
	direct(http.MethodPut, path, string(policyBody)+` {}`, 400)
	retained, err = reopened.RepositoryPolicy(ctx, int64(repository))
	if err != nil || retained.Revision != 3 || retained.GrantedBy != int64(actor) || !retained.Paused {
		t.Fatal("HTTP policy lost canonical attribution or CAS", retained, err)
	}
	capacity := `{"expected_revision":0,"max_concurrent_runs":2}`
	direct(http.MethodPut, "/api/factory/capacity", capacity, 200)
	direct(http.MethodPut, "/api/factory/capacity", capacity, 412)
	operator.Store(false)
	direct(http.MethodPut, "/api/factory/capacity", `{"expected_revision":1,"max_concurrent_runs":3}`, 503)
	operator.Store(true)
	repairPath := "/api/repositories/9223372036854775807/factory/repair"
	direct(http.MethodGet, repairPath, "", 200)
	repair := factoryservice.RepairFundingChange{CommandID: strings.Repeat("a", 32), InvestigatorRuns: 1, ProposalJudgments: 1, ActiveMinutes: 3, Active: true}
	repairBody, _ := json.Marshal(repair)
	firstRepair := direct(http.MethodPut, repairPath, string(repairBody), 200)
	secondRepair := direct(http.MethodPut, repairPath, string(repairBody), 200)
	if string(firstRepair) != string(secondRepair) {
		t.Fatal("HTTP repair command replay changed receipt", string(firstRepair), string(secondRepair))
	}
	admitted.Store(false)
	paused.ExpectedRevision = 3
	policyBody, _ = json.Marshal(paused)
	direct(http.MethodPut, path, string(policyBody), 401)
	admitted.Store(true)
	retained, err = reopened.RepositoryPolicy(ctx, int64(repository))
	if err != nil || retained.Revision != 3 {
		t.Fatal("refused HTTP admission changed policy", retained.Revision, err)
	}
	server.Close()
	transport.Principal = ""
	server = httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	direct(http.MethodPut, path, string(policyBody), 503)
	if runs, err := reopened.FactoryRuns(ctx, 10); err != nil || len(runs) != 0 {
		t.Fatal("HTTP configuration dispatched work", runs, err)
	}
}

// Exercise the shared admission commands with the real coordinator, transactions
// and retained PostgreSQL state. Authority ports declare a different host; no
// Fountain session, Project, native executor or provider credential is invented.
func TestSharedRuntimeGrantAdmission(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("native PostgreSQL fixture is required")
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
	const principal = "profile:runtime-owner"
	denied := errors.New("declared current host authority denied")
	operator, visible, owner := true, true, true
	revokeDuringVisibility := false
	var cancelInOperator, cancelInEnvironment context.CancelFunc
	application := factoryservice.RuntimeGrants{
		Writer: control.NewCoordinator(db, nil, nil),
		OperatorAccess: func(ctx context.Context, admitted collaboration.ActorID) error {
			if admitted != actor {
				t.Fatal("operator admission changed canonical actor", admitted)
			}
			if cancelInOperator != nil {
				cancelInOperator()
			}
			if !operator {
				return denied
			}
			return nil
		},
		Visibility: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			if admitted != actor || repo != repository || !visible {
				return denied
			}
			if revokeDuringVisibility {
				operator = false
			}
			return nil
		},
		EnvironmentAccess: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			if admitted != actor || repo != repository || !owner {
				return denied
			}
			if cancelInEnvironment != nil {
				cancelInEnvironment()
			}
			return nil
		},
	}
	capacity := factoryservice.CapacityChange{MaxConcurrentRuns: 2}
	permission := factoryservice.OperatorGrantChange{Active: true}
	environment := factoryservice.EnvironmentGrantChange{Profile: &project.Profile{
		ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless", Architecture: "amd64",
		Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}, Active: true}
	commands := []struct {
		name  string
		apply func(context.Context, factoryservice.RuntimeGrants) (factory.GrantResult, error)
	}{
		{"capacity", func(ctx context.Context, app factoryservice.RuntimeGrants) (factory.GrantResult, error) {
			return app.ApplyCapacity(ctx, actor, principal, capacity)
		}},
		{"operator", func(ctx context.Context, app factoryservice.RuntimeGrants) (factory.GrantResult, error) {
			return app.ApplyOperatorGrant(ctx, actor, principal, repository, permission)
		}},
		{"environment", func(ctx context.Context, app factoryservice.RuntimeGrants) (factory.GrantResult, error) {
			return app.ApplyEnvironmentGrant(ctx, actor, principal, repository, environment)
		}},
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(data ORDER BY id) FROM factory_capacity),
		 (SELECT jsonb_agg(data ORDER BY repository) FROM factory_operator_grants),
		 (SELECT jsonb_agg(data ORDER BY repository) FROM project_environment_grants),
		 (SELECT jsonb_agg(data ORDER BY repository) FROM factory_dispatch))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	for _, command := range commands {
		result, err := command.apply(ctx, application)
		if err != nil || result.Revision != 1 || result.Withdrawn {
			t.Fatal(command.name, "shared grant save", result, err)
		}
	}
	storedCapacity, err := db.Capacity(ctx)
	if err != nil || storedCapacity.UpdatedBy != int64(actor) || storedCapacity.MaxConcurrentRuns != 2 {
		t.Fatal("capacity lost canonical attribution", storedCapacity, err)
	}
	storedOperator, err := db.OperatorGrant(ctx, int64(repository))
	if err != nil || storedOperator.GrantedBy != int64(actor) || storedOperator.Repository != int64(repository) {
		t.Fatal("operator grant lost canonical scope", storedOperator, err)
	}
	storedEnvironment, err := db.EnvironmentGrant(ctx, int64(repository))
	if err != nil || storedEnvironment.Owner != int64(actor) || storedEnvironment.Repository != int64(repository) || storedEnvironment.Project != "" {
		t.Fatal("environment grant invented native custody", storedEnvironment, err)
	}
	before := snapshot()
	for _, command := range commands {
		if _, err := command.apply(ctx, application); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal(command.name, "lost grant CAS", err)
		}
		unavailable := application
		unavailable.Writer = nil
		if _, err := command.apply(ctx, unavailable); !errors.Is(err, factoryservice.ErrRuntimeGrantUnavailable) {
			t.Fatal(command.name, "missing writer accepted", err)
		}
		unavailable = factoryservice.RuntimeGrants{Writer: application.Writer}
		if _, err := command.apply(ctx, unavailable); !errors.Is(err, factoryservice.ErrRuntimeGrantUnavailable) {
			t.Fatal(command.name, "missing authority accepted", err)
		}
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := command.apply(cancelled, application); !errors.Is(err, context.Canceled) {
			t.Fatal(command.name, "cancelled command reached save", err)
		}
		operator, owner = false, false
		if _, err := command.apply(ctx, application); !errors.Is(err, denied) || !errors.Is(err, factoryservice.ErrRuntimeGrantAccess) {
			t.Fatal(command.name, "retained grant bypassed current authority", err)
		}
		operator, owner = true, true
	}
	visible = false
	if _, err := application.ApplyOperatorGrant(ctx, actor, principal, repository, permission); !errors.Is(err, denied) {
		t.Fatal("operator bypassed repository visibility", err)
	}
	visible = true
	revokeDuringVisibility = true
	if _, err := application.ApplyOperatorGrant(ctx, actor, principal, repository, permission); !errors.Is(err, denied) {
		t.Fatal("operator authority was retained across repository observation", err)
	}
	operator, revokeDuringVisibility = true, false
	if _, err := application.ApplyEnvironmentGrant(ctx, actor, principal, repository-1, environment); !errors.Is(err, denied) {
		t.Fatal("environment authority crossed repository", err)
	}
	invalidCapacity := capacity
	invalidCapacity.MaxConcurrentRuns = 65
	if _, err := application.ApplyCapacity(ctx, actor, principal, invalidCapacity); !errors.Is(err, factoryservice.ErrRuntimeCapacityInvalid) {
		t.Fatal("invalid capacity accepted", err)
	}
	if _, err := application.ApplyOperatorGrant(ctx, 0, principal, repository, permission); !errors.Is(err, factoryservice.ErrRuntimeOperatorInvalid) {
		t.Fatal("invalid canonical operator accepted", err)
	}
	if _, err := application.ApplyEnvironmentGrant(ctx, actor, "", repository, environment); !errors.Is(err, factoryservice.ErrRuntimeEnvironmentInvalid) {
		t.Fatal("missing authenticated withdrawal principal accepted", err)
	}
	late, cancel := context.WithCancel(ctx)
	cancelInOperator = cancel
	if _, err := application.ApplyCapacity(late, actor, principal, capacity); !errors.Is(err, context.Canceled) {
		t.Fatal("late capacity cancellation saved", err)
	}
	cancelInOperator = nil
	late, cancel = context.WithCancel(ctx)
	cancelInEnvironment = cancel
	if _, err := application.ApplyEnvironmentGrant(late, actor, principal, repository, environment); !errors.Is(err, context.Canceled) {
		t.Fatal("late environment cancellation saved", err)
	}
	cancelInEnvironment = nil
	if snapshot() != before {
		t.Fatal("refused grant commands changed durable grants or dispatch")
	}
	// An environment owner need not be the runtime operator, or vice versa.
	operator = false
	environment.ExpectedRevision = 1
	if result, err := commands[2].apply(ctx, application); err != nil || result.Revision != 2 {
		t.Fatal("environment ownership incorrectly requires runtime operation", result, err)
	}
	operator, owner = true, false
	permission.ExpectedRevision = 1
	if result, err := commands[1].apply(ctx, application); err != nil || result.Revision != 2 {
		t.Fatal("runtime operation incorrectly requires environment ownership", result, err)
	}
	owner = true
	environment.ExpectedRevision, environment.Active = 2, false
	if result, err := commands[2].apply(ctx, application); err != nil || !result.Withdrawn || result.Revision != 3 {
		t.Fatal("environment withdrawal did not settle", result, err)
	}
	permission.ExpectedRevision, permission.Active = 2, false
	if result, err := commands[1].apply(ctx, application); err != nil || !result.Withdrawn || result.Revision != 3 {
		t.Fatal("operator withdrawal did not settle", result, err)
	}
	open, _, withdrawal, err := db.DispatchState(ctx, int64(repository))
	if err != nil || open || withdrawal.ClosedBy != principal || withdrawal.Repository != int64(repository) || len(withdrawal.ActiveCauses) != 2 {
		t.Fatal("withdrawal lost scope, principal or accumulated causes", withdrawal, err)
	}
	retained := snapshot()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	application.Writer = control.NewCoordinator(db, nil, nil)
	if snapshot() != retained {
		t.Fatal("restart changed grant/dispatch records")
	}
	for _, command := range commands {
		if _, err := command.apply(ctx, application); !errors.Is(err, store.ErrStaleRevision) {
			t.Fatal(command.name, "restart lost grant CAS", err)
		}
	}
	capacity.ExpectedRevision = 1
	if result, err := commands[0].apply(ctx, application); err != nil || result.Revision != 2 || result.Withdrawn {
		t.Fatal("reopened capacity save", result, err)
	}
	open, _, after, err := db.DispatchState(ctx, int64(repository))
	beforeJSON, _ := json.Marshal(withdrawal)
	afterJSON, _ := json.Marshal(after)
	if err != nil || open || string(beforeJSON) != string(afterJSON) {
		t.Fatal("capacity change reopened withdrawn repository", after, err)
	}
	var projects, runs, reservations int
	if err := raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM projects),(SELECT count(*) FROM factory_runs),(SELECT count(*) FROM factory_reservations)`).Scan(&projects, &runs, &reservations); err != nil || projects != 0 || runs != 0 || reservations != 0 {
		t.Fatal("grant admission provisioned or launched work", projects, runs, reservations, err)
	}
}

func TestFactoryPolicyPauseWithdrawsDispatch(t *testing.T) {
	s := factorySettingsServer(t)
	enable := httptest.NewRecorder()
	nativeAPIServe(t, s, enable, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/policy", factoryPolicyBody(), "alice"))
	if enable.Code != 200 {
		t.Fatal(enable.Code, enable.Body.String())
	}
	paused := strings.Replace(factoryPolicyBody(), `"expected_revision":0`, `"expected_revision":1`, 1)
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
	body := `{"expected_revision":0,"max_concurrent_runs":2}`
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
	grant := `{"expected_revision":0,"active":true}`
	allowed := httptest.NewRecorder()
	nativeAPIServe(t, s, allowed, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/operator-grant", grant, "alice"))
	if allowed.Code != 200 {
		t.Fatal(allowed.Code, allowed.Body.String())
	}
	for _, in := range []string{
		`{"expected_revision":1,"max_concurrent_runs":65}`,
		`{"expected_revision":1,"max_concurrent_runs":2,"updated_by":"2"}`,
	} {
		invalid := httptest.NewRecorder()
		nativeAPIServe(t, s, invalid, apiTestRequest(http.MethodPut, "/api/factory/capacity", in, "alice"))
		if invalid.Code != 400 {
			t.Fatal("invalid capacity/body attribution admitted", invalid.Code, invalid.Body.String())
		}
	}
	s.API.Coordinator = nil
	for _, command := range []struct{ path, body string }{
		{"/api/factory/capacity", `{"expected_revision":1,"max_concurrent_runs":3}`},
		{"/api/repositories/7/factory/operator-grant", `{"expected_revision":1,"active":false}`},
	} {
		unavailable := httptest.NewRecorder()
		nativeAPIServe(t, s, unavailable, apiTestRequest(http.MethodPut, command.path, command.body, "alice"))
		if unavailable.Code != 503 {
			t.Fatal("missing grant coordinator", command.path, unavailable.Code, unavailable.Body.String())
		}
	}
	capacity, err := s.Store.Capacity(t.Context())
	if err != nil || capacity.Revision != 1 || capacity.MaxConcurrentRuns != 2 {
		t.Fatal("refused HTTP capacity changed the grant", capacity, err)
	}
	permission, err := s.Store.OperatorGrant(t.Context(), 7)
	if err != nil || permission.Revision != 1 || !permission.Active {
		t.Fatal("unavailable HTTP withdrawal changed the grant", permission, err)
	}
}

func TestFactoryEnvironmentGrantIsOwnerOnly(t *testing.T) {
	s := factorySettingsServer(t)
	profile := `{"id":"rocky-headless","distribution":"rocky","version":"9.6","interface":"headless","architecture":"amd64","image":"sha256:` + strings.Repeat("b", 64) + `","revision":"` + strings.Repeat("c", 40) + `"}`
	body := `{"expected_revision":0,"profile":` + profile + `,"active":true}`
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
	s.API.Coordinator = nil
	unavailable := httptest.NewRecorder()
	nativeAPIServe(t, s, unavailable, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/environment-grant", strings.ReplaceAll(strings.ReplaceAll(body, `"expected_revision":0`, `"expected_revision":1`), `"active":true`, `"active":false`), "alice"))
	if unavailable.Code != 503 {
		t.Fatal("missing environment grant coordinator", unavailable.Code, unavailable.Body.String())
	}
	grant, err := s.Store.EnvironmentGrant(t.Context(), 7)
	if err != nil || grant.Revision != 1 || !grant.Active {
		t.Fatal("unavailable environment withdrawal changed the grant", grant, err)
	}
}

func TestFactorySponsorshipNeedsConnectionOwner(t *testing.T) {
	s := factoryEncryptedServer(t)
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: factorySettingsProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	s.API.Identity = &stubIdentityClient{
		connections: []identity.Connection{
			{ID: "conn-1", ProviderID: identity.Codex, OwnerID: 1, Label: "test", Generation: 1, State: identity.Ready},
			{ID: "conn-2", ProviderID: identity.Codex, OwnerID: 1, Label: "delegated", Generation: 1, State: identity.Ready},
		},
		grants: []identity.Grant{
			{ID: "grant-1", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject},
			{ID: "grant-2", ConnectionID: "conn-1", UserID: 1, ProjectID: factorySettingsProject, Revoked: true},
			{ID: "grant-3", ConnectionID: "conn-2", UserID: 2, ProjectID: factorySettingsProject},
		},
	}
	sponsor := func(revision int64) string {
		return `{"expected_revision":` + strconv.FormatInt(revision, 10) +
			`,"roles":["soda-coder"],"allowance_minutes":60,"active":true}`
	}
	put := func(connection, body, login string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(http.MethodPut, "/api/repositories/7/factory/sponsorships/"+connection, body, login))
		return w
	}
	w := put("conn-1", sponsor(0), "alice")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	stored, err := s.Store.Sponsorship(t.Context(), 7, "conn-1")
	if err != nil || stored.GrantID != "grant-1" || stored.Generation != 1 || stored.ActorID != 1 || stored.ProjectID != factorySettingsProject {
		t.Fatalf("sponsorship did not resolve the live broker grant: %+v %v", stored, err)
	}
	delegated := put("conn-2", sponsor(0), "alice")
	if delegated.Code != http.StatusOK {
		t.Fatal("connection owner could not sponsor through a delegated grant", delegated.Code, delegated.Body.String())
	}
	stored, err = s.Store.Sponsorship(t.Context(), 7, "conn-2")
	if err != nil || stored.GrantID != "grant-3" || stored.GrantedBy != 1 || stored.ActorID != 2 || stored.ProjectID != factorySettingsProject {
		t.Fatalf("delegated sponsorship did not separate sponsor from broker actor: %+v %v", stored, err)
	}
	assertNoSecrets(t, w.Body.String())
	foreign := put("conn-1", sponsor(1), "bob")
	if foreign.Code != 404 {
		t.Fatal("non-owner saw the connection", foreign.Code, foreign.Body.String())
	}
	stub := s.API.Identity.(*stubIdentityClient)
	stub.connections[0].Generation = 2
	if rotated := put("conn-1", sponsor(1), "alice"); rotated.Code != 200 {
		t.Fatal("a rotated credential needed a re-pick", rotated.Code, rotated.Body.String())
	}
	if stored, err = s.Store.Sponsorship(t.Context(), 7, "conn-1"); err != nil || stored.Generation != 2 || stored.GrantID != "grant-1" {
		t.Fatalf("sponsorship did not record the current generation: %+v %v", stored, err)
	}
	stub.grants[0].Revoked = true
	if revoked := put("conn-1", sponsor(2), "alice"); revoked.Code != 409 {
		t.Fatal("sponsorship without a live grant accepted", revoked.Code, revoked.Body.String())
	}
	stub.grants[0].Revoked = false
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
	if len(view.Sponsorships) != 2 || view.Sponsorships[0].Allowance != 0 || view.Sponsorships[1].Allowance != 0 {
		t.Fatalf("allowance leaked to non-owner: %+v", view.Sponsorships)
	}
	started := time.Unix(1700000000, 0)
	for index, id := range []string{"conn-1", "conn-2"} {
		minutes := index + 2
		if err := s.Store.RecordRunUsage(t.Context(), factory.Usage{RunID: strings.Repeat(strconv.Itoa(index+1), 32), Repository: 7, Connection: id,
			Minutes: minutes, StartedAt: started, EndedAt: started.Add(time.Duration(minutes) * time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	readAccess := func(login string) factoryservice.RepositoryAccessView {
		t.Helper()
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/repositories/7/factory/access", "", login))
		if w.Code != 200 {
			t.Fatal("native settings access", w.Code, w.Body.String())
		}
		var access factoryservice.RepositoryAccessView
		decodeBody(t, w, &access)
		return access
	}
	ownerAccess, otherAccess := readAccess("alice"), readAccess("bob")
	if len(ownerAccess.Usage) != 2 || len(otherAccess.Usage) != 2 {
		t.Fatal("missing native usage", ownerAccess, otherAccess)
	}
	for index, usage := range ownerAccess.Usage {
		if usage.Status != "" || usage.UsedMinutes == nil || *usage.UsedMinutes != index+2 || usage.BudgetMinutes != 60 || !ownerAccess.Editable.Sponsorships[index].Editable {
			t.Fatal("native connection owner lost private usage", usage)
		}
		other := otherAccess.Usage[index]
		if other.Status != "connection_owner_only" || other.UsedMinutes != nil || other.BudgetMinutes != 0 || otherAccess.Editable.Sponsorships[index].Editable {
			t.Fatal("native non-owner saw private usage", other)
		}
	}
	stub.grantsErr = identity.ErrDenied
	if w := put("conn-1", sponsor(2), "alice"); w.Code != 403 {
		t.Fatal("denied broker grant accepted", w.Code, w.Body.String())
	}
	stub.grantsErr = errStubIdentity
	if w := put("conn-1", sponsor(2), "alice"); w.Code != 503 {
		t.Fatal("broken broker grant accepted", w.Code, w.Body.String())
	}
	stub.grantsErr = nil
	stub.connErr = errStubIdentity
	if w := put("conn-1", sponsor(2), "alice"); w.Code != 503 {
		t.Fatal("broken broker connection accepted", w.Code, w.Body.String())
	}
	for _, usage := range readAccess("alice").Usage {
		if usage.Status != "identity_unavailable" || usage.UsedMinutes != nil || usage.BudgetMinutes != 0 {
			t.Fatal("unknown native ownership exposed usage", usage)
		}
	}
	stub.connErr = nil
	s.API.Identity = nil
	if w := put("conn-1", sponsor(2), "alice"); w.Code != 503 {
		t.Fatal("missing identity service accepted", w.Code, w.Body.String())
	}
	s.API.Identity = stub
	for _, field := range []string{`"grant_id":"chosen"`, `"generation":99`, `"actor_id":"2"`, `"granted_by":"2"`, `"project_id":"chosen"`} {
		if w := put("conn-1", strings.Replace(sponsor(2), `"active":true`, `"active":true,`+field, 1), "alice"); w.Code != 400 {
			t.Fatal("body-selected sponsorship custody accepted", field, w.Code, w.Body.String())
		}
	}
	coordinator := s.API.Coordinator
	s.API.Coordinator = nil
	if w := put("conn-1", sponsor(2), "alice"); w.Code != 503 {
		t.Fatal("missing sponsorship coordinator accepted", w.Code, w.Body.String())
	}
	s.API.Coordinator = coordinator
	stored, err = s.Store.Sponsorship(t.Context(), 7, "conn-1")
	if err != nil || stored.Revision != 2 || stored.Generation != 2 || !stored.Active {
		t.Fatal("refused native sponsorship changed durable grant", stored, err)
	}
	for index, id := range []string{"conn-1", "conn-2"} {
		revision := int64(2 - index)
		w := put(id, strings.Replace(sponsor(revision), `"active":true`, `"active":false`, 1), "alice")
		if w.Code != 200 {
			t.Fatal("native sponsorship withdrawal", id, w.Code, w.Body.String())
		}
		var result factory.GrantResult
		decodeBody(t, w, &result)
		if result.Withdrawn != (index == 1) {
			t.Fatal("withdrawal ignored the other active sponsorship", id, result)
		}
	}
	open, _, withdrawal, err := s.Store.DispatchState(t.Context(), 7)
	if err != nil || open || withdrawal.ClosedBy != "native:1" || withdrawal.Cause != "sponsorship_withdrawn" {
		t.Fatal("native sponsorship withdrawal attribution", withdrawal, err)
	}
}

// Shared sponsorship runs against the real grant journal and coordinator. The
// custody port declares nonsecret facts of a different host; it creates no native
// Project and neither enrolls credentials nor invokes a model/provider.
func TestSharedSponsorshipAdmission(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("native PostgreSQL fixture is required")
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
	const connection, principal = "owned-model", "profile:sponsor"
	current := factoryservice.SponsorshipCustody{Repository: repository, Connection: connection, Project: factorySettingsProject, Generation: 1,
		Grants: []identity.Grant{
			{ID: "foreign-connection", ConnectionID: "other", ProjectID: factorySettingsProject, UserID: 8, Revision: 90},
			{ID: "foreign-runtime", ConnectionID: connection, ProjectID: "p" + strings.Repeat("8", 24), UserID: 8, Revision: 90},
			{ID: "revoked", ConnectionID: connection, ProjectID: factorySettingsProject, UserID: 8, Revision: 90, Revoked: true},
			{ID: "missing-actor", ConnectionID: connection, ProjectID: factorySettingsProject, Revision: 90},
			{ID: "old", ConnectionID: connection, ProjectID: factorySettingsProject, UserID: 8, Revision: 2},
			{ID: "live", ConnectionID: connection, ProjectID: factorySettingsProject, UserID: 9, Revision: 5},
		}}
	visible, custodyCalls, visibilityCalls, denyAt := true, 0, 0, 0
	denied := errors.New("declared current sponsorship access denied")
	var custodyErr error
	var cancelInCustody context.CancelFunc
	application := factoryservice.Sponsorships{Store: db, Writer: control.NewCoordinator(db, nil, nil),
		Visibility: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			if admitted != actor || repo != repository {
				t.Fatal("sponsorship visibility changed canonical subject", admitted, repo)
			}
			visibilityCalls++
			if !visible || visibilityCalls == denyAt {
				return denied
			}
			return nil
		},
		Custody: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID, id string) (factoryservice.SponsorshipCustody, error) {
			custodyCalls++
			if admitted != actor || repo != repository || id != connection {
				t.Fatal("sponsorship custody changed canonical subject", admitted, repo, id)
			}
			if cancelInCustody != nil {
				cancelInCustody()
			}
			return current, custodyErr
		},
	}
	in := factoryservice.SponsorshipChange{Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, Active: true}
	apply := func(ctx context.Context) (factory.GrantResult, error) {
		visibilityCalls = 0
		return application.Apply(ctx, actor, principal, repository, connection, in)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(s) ORDER BY repository,connection) FROM factory_sponsorships s),
		 (SELECT jsonb_agg(to_jsonb(d) ORDER BY repository) FROM factory_dispatch d))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	if result, err := apply(ctx); err != nil || result.Revision != 1 || result.Withdrawn {
		t.Fatal("initial shared sponsorship", result, err)
	}
	stored, err := db.Sponsorship(ctx, int64(repository), connection)
	if err != nil || stored.GrantID != "live" || stored.GrantedBy != int64(actor) || stored.ActorID != 9 || stored.Generation != 1 {
		t.Fatal("grant selection conflated sponsor, broker actor or scope", stored, err)
	}
	current.Grants = append(current.Grants, identity.Grant{ID: "newest", ConnectionID: connection, ProjectID: factorySettingsProject, UserID: 10, Revision: 8})
	current.Generation, in.ExpectedRevision = 2, 1
	if result, err := apply(ctx); err != nil || result.Revision != 2 {
		t.Fatal("current generation sponsorship", result, err)
	}
	stored, err = db.Sponsorship(ctx, int64(repository), connection)
	if err != nil || stored.GrantID != "live" || stored.ActorID != 9 || stored.Generation != 2 {
		t.Fatal("new grant replaced a still-live recorded grant", stored, err)
	}
	current.Grants[5].Revoked = true
	in.ExpectedRevision = 2
	if result, err := apply(ctx); err != nil || result.Revision != 3 {
		t.Fatal("revoked selected grant replacement", result, err)
	}
	stored, err = db.Sponsorship(ctx, int64(repository), connection)
	if err != nil || stored.GrantID != "newest" || stored.ActorID != 10 || stored.Revision != 3 {
		t.Fatal("revoked grant did not select newest live scoped grant", stored, err)
	}
	in.ExpectedRevision = 3
	before := snapshot()
	for _, failure := range []error{factoryservice.ErrSponsorshipOwnerRequired, factoryservice.ErrSponsorshipConnectionMissing, errors.New("declared unknown broker read")} {
		custodyErr = failure
		if _, err := apply(ctx); !errors.Is(err, failure) {
			t.Fatal("custody denial or unknown read was ignored", failure, err)
		}
	}
	custodyErr = nil
	liveGrants := current.Grants
	current.Grants = nil
	if _, err := apply(ctx); !errors.Is(err, factoryservice.ErrSponsorshipGrantUnavailable) {
		t.Fatal("known absent broker grant accepted", err)
	}
	current.Grants = liveGrants
	current.Repository--
	if _, err := apply(ctx); !errors.Is(err, factoryservice.ErrSponsorshipBrokerUnavailable) {
		t.Fatal("foreign custody scope accepted", err)
	}
	current.Repository = repository
	visible = false
	calls := custodyCalls
	if _, err := apply(ctx); !errors.Is(err, denied) || custodyCalls != calls {
		t.Fatal("invisible sponsorship reached custody", err)
	}
	visible, denyAt = true, 2
	if _, err := apply(ctx); !errors.Is(err, denied) {
		t.Fatal("late visibility revocation saved sponsorship", err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := apply(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled sponsorship saved", err)
	}
	cancelled, cancel = context.WithCancel(ctx)
	cancelInCustody = cancel
	if _, err := apply(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal("late custody cancellation saved", err)
	}
	cancelInCustody = nil
	for _, missing := range []string{"store", "writer", "visibility", "custody"} {
		unavailable := application
		switch missing {
		case "store":
			unavailable.Store = nil
		case "writer":
			unavailable.Writer = nil
		case "visibility":
			unavailable.Visibility = nil
		case "custody":
			unavailable.Custody = nil
		}
		if _, err := unavailable.Apply(ctx, actor, principal, repository, connection, in); !errors.Is(err, factoryservice.ErrSponsorshipUnavailable) {
			t.Fatal("missing sponsorship port accepted", missing, err)
		}
	}
	in.Roles = []string{project.RoleCoder, project.RoleCoder}
	if _, err := apply(ctx); !errors.Is(err, factoryservice.ErrSponsorshipInvalid) {
		t.Fatal("duplicate sponsorship roles accepted", err)
	}
	in.Roles = []string{project.RoleCoder}
	if snapshot() != before {
		t.Fatal("refused sponsorship changed grant or dispatch records")
	}
	// A real PostgreSQL row whose JSON cannot be decoded is unknown, not absent.
	// Refusal must precede custody selection and preserve that unknown record.
	if _, err := raw.ExecContext(ctx, `UPDATE factory_sponsorships SET data=jsonb_set(data,'{revision}','"unknown"') WHERE repository=$1 AND connection=$2`, int64(repository), connection); err != nil {
		t.Fatal(err)
	}
	unknown, calls := snapshot(), custodyCalls
	if _, err := apply(ctx); err == nil || errors.Is(err, store.ErrNotFound) || custodyCalls != calls || snapshot() != unknown {
		t.Fatal("unreadable prior sponsorship was treated as absent", err)
	}
	encoded, _ := json.Marshal(stored)
	if _, err := raw.ExecContext(ctx, `UPDATE factory_sponsorships SET data=$3 WHERE repository=$1 AND connection=$2`, int64(repository), connection, encoded); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before {
		t.Fatal("restored fixture changed sponsorship")
	}
	in.Active = false
	if result, err := apply(ctx); err != nil || !result.Withdrawn || result.Revision != 4 {
		t.Fatal("last sponsorship withdrawal", result, err)
	}
	open, _, withdrawal, err := db.DispatchState(ctx, int64(repository))
	if err != nil || open || withdrawal.ClosedBy != principal || withdrawal.Cause != "sponsorship_withdrawn" || withdrawal.Repository != int64(repository) {
		t.Fatal("sponsor withdrawal lost authenticated principal or scope", withdrawal, err)
	}
	before = snapshot()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	application.Store, application.Writer = db, control.NewCoordinator(db, nil, nil)
	if snapshot() != before {
		t.Fatal("store reopen changed sponsorship or withdrawal")
	}
	if _, err := apply(ctx); !errors.Is(err, store.ErrStaleRevision) {
		t.Fatal("store reopen lost sponsorship CAS", err)
	}
	visible = false
	if _, err := apply(ctx); !errors.Is(err, denied) {
		t.Fatal("retained sponsorship bypassed current admission", err)
	}
	var projects, runs, reservations int
	if err := raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM projects),(SELECT count(*) FROM factory_runs),(SELECT count(*) FROM factory_reservations)`).Scan(&projects, &runs, &reservations); err != nil || projects != 0 || runs != 0 || reservations != 0 {
		t.Fatal("sponsorship provisioned or launched work", projects, runs, reservations, err)
	}
}

func TestPreparationAcceptanceJourney(t *testing.T) {
	s := factorySettingsServer(t)
	digest := strings.Repeat("d", 64)
	commit := strings.Repeat("e", 40)
	body := `{"decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + commit + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", body, "alice"))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var saved map[string]any
	decodeBody(t, w, &saved)
	if saved["decision_id"] != "d123456789012345678901234" || saved["command_id"] != nil {
		t.Fatalf("acceptance result did not use its decision identity: %+v", saved)
	}
	repeated := httptest.NewRecorder()
	nativeAPIServe(t, s, repeated, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", body, "alice"))
	if repeated.Code != 201 {
		t.Fatal("identical acceptance decision refused", repeated.Code, repeated.Body.String())
	}
	if depth, err := s.Store.RequirementDepth(t.Context(), factorySettingsProject); err != nil || depth != 1 {
		t.Fatal("repeated acceptance advanced its decision chain", depth, err)
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
	second := `{"decision_id":"d223456789012345678901234",` +
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
	accept := `{"decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + commit + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	accepted := httptest.NewRecorder()
	nativeAPIServe(t, s, accepted, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", accept, "alice"))
	if accepted.Code != 201 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	approve := func(requirement string) string {
		return `{"action":"approve","decision_id":"d223456789012345678901234",` +
			`"requirement":"` + requirement + `","effects_digest":"` + digest + `","readiness_digest":"` + digest + `","verified":true}`
	}
	accessCalls, statusCode := 0, 0
	administrator, malformed := true, false
	approvalProjectAccessHost(s, &accessCalls, &administrator, &statusCode, &malformed)
	wrong := httptest.NewRecorder()
	nativeAPIServe(t, s, wrong, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d999999999999999999999999"), "alice"))
	if wrong.Code != 409 {
		t.Fatal("approval bound a superseded requirement", wrong.Code, wrong.Body.String())
	}
	if accessCalls != 1 {
		t.Fatalf("member approval did not use exact native project observation: calls=%d", accessCalls)
	}
	// Bob has a current native session but no Project membership, so no host
	// privilege observation or approval mutation is allowed.
	denied := httptest.NewRecorder()
	nativeAPIServe(t, s, denied, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d123456789012345678901234"), "bob"))
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
	nativeAPIServe(t, s, ownerDenied, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d123456789012345678901234"), "alice"))
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
	nativeAPIServe(t, s, uncertain, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d123456789012345678901234"), "alice"))
	if uncertain.Code != 502 {
		t.Fatal("unavailable native authority was admitted", uncertain.Code, uncertain.Body.String())
	}
	statusCode = 0
	malformed = true
	badEcho := httptest.NewRecorder()
	nativeAPIServe(t, s, badEcho, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d123456789012345678901234"), "alice"))
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
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/actions", approve("d123456789012345678901234"), "bob"))
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
	accept := `{"decision_id":"d123456789012345678901234",` +
		`"source_commit":"` + strings.Repeat("e", 40) + `","setup_digest":"` + digest + `","inputs_digest":"` + digest + `"}`
	accepted := httptest.NewRecorder()
	nativeAPIServe(t, s, accepted, apiTestRequest(http.MethodPost, "/api/environments/"+factorySettingsProject+"/preparation/acceptances", accept, "alice"))
	if accepted.Code != 201 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	calls, statusCode := 0, http.StatusServiceUnavailable
	administrator, malformed := false, true
	approvalProjectAccessHost(s, &calls, &administrator, &statusCode, &malformed)
	approve := `{"action":"approve","decision_id":"d223456789012345678901234",` +
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
