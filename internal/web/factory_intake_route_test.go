package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/forgejo/observation"
)

type intakeRouteSource struct {
	evidence control.AcceptanceEvidence
}

// Prove Fountain uses Factory directly, including a persisted mutation, stale
// authority rejection and continued Factory access while Forge is unavailable.
func TestFactoryProductNativeBridge(t *testing.T) {
	factoryServer := apiTestServer(t)
	if configured, err := factoryServer.Store.CollaborationHostByAPIBase(t.Context(), collaboration.Fountain, factoryServer.Config.ForgejoURL); err != nil || configured != factoryServer.FountainHost {
		t.Fatalf("Factory construction did not bind configured Fountain: %+v %v", configured, err)
	}
	if repeated, err := NewFactory(factoryServer.Config, factoryServer.Store); err != nil || repeated.FountainHost != factoryServer.FountainHost {
		t.Fatalf("repeated Factory construction changed the provider binding: %v", err)
	}
	wrong := factoryServer.Config
	wrong.ForgejoURL = "https://different-fountain.example.test"
	if rejected, err := NewFactory(wrong, factoryServer.Store); !errors.Is(err, collaboration.ErrIdentityConflict) || rejected != nil {
		t.Fatalf("Factory constructed a coordinator for another source: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(home, "factory-bridge-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "factory.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	backend := &http.Server{Handler: factoryServer.ExtensionHandler()}
	go func() { _ = backend.Serve(listener) }()
	t.Cleanup(func() { _ = backend.Close() })
	// Reuse the native callback fixture; requests below enter the real product
	// handler rather than its shared, in-process API integration fixture.
	_ = nativeProductProxy(t, factoryServer, extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"})
	c := factoryServer.Config
	forge := NewForge(c, factoryServer.Store)
	if forge.Coordinator != nil || forge.API.Coordinator != nil {
		t.Fatal("Forge constructed a coordinator")
	}
	application, closeFrontend := Extension(filepath.Join(dir, "absent-forge.sock"), socket)
	t.Cleanup(closeFrontend)
	front := httptest.NewServer(application.HTTP)
	t.Cleanup(front.Close)
	requestPath := "/factory/departments"
	call := func(method, body string, headers http.Header, want int) []byte {
		r, err := http.NewRequest(method, front.URL+requestPath, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Header = headers
		r.Header.Set("Content-Type", "application/json")
		res, err := front.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("native bridge: %d, want %d: %s", res.StatusCode, want, data)
		}
		return data
	}
	contribution := extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}
	body := `{"id":"d111111111111111111111111","name":"Platform","description":"Platform engineering"}`
	call("POST", body, nativeProductHeaders("POST", c.ForgejoURL, contribution), 201)
	data := call("GET", "", nativeProductHeaders("GET", c.ForgejoURL, contribution), 200)
	if !strings.Contains(string(data), "Platform") {
		t.Fatalf("Factory mutation was not persisted: %s", data)
	}
	call("GET", "", make(http.Header), 403)
	stale := nativeProductHeaders("POST", c.ForgejoURL, contribution)
	stale.Set(extensions.SessionGenerationHeader, "expired-generation")
	call("POST", `{"id":"d222222222222222222222222","name":"Stale","description":"Rejected stale request"}`, stale, 409)
	departments, err := factoryServer.Store.Departments(t.Context(), 1)
	if err != nil || len(departments) != 1 {
		t.Fatalf("rejected request changed Factory data: %d %v", len(departments), err)
	}
	forgeListener, err := net.Listen("unix", filepath.Join(dir, "absent-forge.sock"))
	if err != nil {
		t.Fatal(err)
	}
	forgeHTTP := &http.Server{Handler: forge.ExtensionHandler()}
	go func() { _ = forgeHTTP.Serve(forgeListener) }()
	t.Cleanup(func() { _ = forgeHTTP.Close() })
	requestPath = "/session"
	call("GET", "", nativeProductHeaders("GET", c.ForgejoURL, contribution), 200)
	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
	// Forge account operations still work; Factory requests never fall back
	// to that live Forge listener when Factory is unavailable.
	call("GET", "", nativeProductHeaders("GET", c.ForgejoURL, contribution), 200)
	requestPath = "/factory/departments"
	call("GET", "", nativeProductHeaders("GET", c.ForgejoURL, contribution), 503)
	restartedListener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	restarted := &http.Server{Handler: factoryServer.ExtensionHandler()}
	go func() { _ = restarted.Serve(restartedListener) }()
	t.Cleanup(func() { _ = restarted.Close() })
	call("GET", "", nativeProductHeaders("GET", c.ForgejoURL, contribution), 200)
}

func (s intakeRouteSource) ReadAcceptanceEvidence(context.Context, string, string, []string) (control.AcceptanceEvidence, error) {
	return s.evidence, nil
}

// Declared provider metadata for the PostgreSQL ingress/controller integration.
// The production bracket validator still owns request/revision/content checks.
type intakeScopedSnapshotReader struct {
	snapshot forgejo.NativeSnapshot
	requests []forgejo.SnapshotRequest
	failure  error
}

func (s *intakeScopedSnapshotReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	return extensions.NativeRevisionObservation{Revision: 12, Idle: true}, nil
}

func (s *intakeScopedSnapshotReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, request forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	s.requests = append(s.requests, request)
	if request.RepositoryID != "7" || request.IssueIndex != "3" {
		return forgejo.NativeSnapshot{}, forgejo.ErrInvalidResponse
	}
	if s.failure != nil {
		return forgejo.NativeSnapshot{}, s.failure
	}
	return s.snapshot, nil
}

func intakeRouteServer(t *testing.T, secret string) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	db := postgresFixture(t)
	// The original route cases exercise retained, provider-numbered history.
	if err := db.UpsertUser(t.Context(), store.User{ID: 5, Login: "retained"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProject(t.Context(), store.Project{ID: "p" + strings.Repeat("a", 24), Name: "Retained", RepositoryID: 7, OwnerID: 5, Repository: "team/retained"}); err != nil {
		t.Fatal(err)
	}
	c := config.Config{
		Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/",
		ForgejoInternalURL: "http://127.0.0.1:3000",
		DatabaseDSNFile:    filepath.Join(dir, "soda.dsn"),
		HostSocket:         filepath.Join(dir, "host.sock"),
		IdentitySocket:     filepath.Join(dir, "identity.sock"),
		OperatorID:         1,
	}
	if secret != "" {
		path := filepath.Join(dir, "intake-secret")
		if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
			t.Fatal(err)
		}
		c.FactoryIntakeSecretFile = path
	}
	server, err := NewFactory(c, db)
	if err != nil {
		t.Fatal(err)
	}
	return server, func() {}
}

func TestFactoryIntakeRouteServesVerifiedDeliveries(t *testing.T) {
	server, done := intakeRouteServer(t, "intake-secret")
	defer done()
	t.Run("mapped_signed_creation_and_readiness", func(t *testing.T) {
		admin, ok := store.TestFixtureDSN()
		if !ok {
			t.Skip("SODA_PG_* fixture unavailable")
		}
		db, dsn, cleanup, err := store.OpenEphemeral(t.Context(), admin)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cleanup)
		mapped, err := NewFactory(server.Config, db)
		if err != nil {
			t.Fatal(err)
		}
		github := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github.example.test"}
		if _, err := db.PutCollaborationHost(t.Context(), github); err != nil {
			t.Fatal(err)
		}
		foreignRepo, err := db.ObserveCollaborationRepository(t.Context(), github.ID, "7", "same/name")
		if err != nil {
			t.Fatal(err)
		}
		foreignActor, err := db.ObserveCollaborationActor(t.Context(), github.ID, "5", "same-login")
		if err != nil {
			t.Fatal(err)
		}
		reader := &intakeScopedSnapshotReader{snapshot: forgejo.NativeSnapshot{
			RepositoryID: "7", Issue: &forgejo.IssueEvidence{
				ID: "101", Index: "3", Title: "Title", Content: "Body", Visible: true, Complete: true,
				TitleDigest: forgejo.ContentDigest("Title"), ContentDigest: forgejo.ContentDigest("Body"), CreatedUnix: 1700000000,
				Provenance: forgejo.CreationProvenance{PosterID: "5", CreatedUnix: 1700000000, FirstCreated: true, Verified: true},
			}, Dependencies: &forgejo.DependencyPage{IssueID: "101", Complete: true, Items: []forgejo.DependencyEvidence{}},
			Refs: []forgejo.RefEvidence{{Ref: "refs/heads/main", OID: strings.Repeat("c", 40), Exists: true, Visible: true, Complete: true}},
		}}
		source := &observation.ServiceReadinessSource{
			Identities: mapped.API.FountainIdentities,
			Evidence:   observation.AcceptanceSnapshotSource{Reader: reader}, Dispatch: observation.DispatchSnapshotSource{Reader: reader},
		}
		mapped.Coordinator.AcceptanceReads = source
		body := `{"action":"opened","number":3,"repository":{"id":7,"permissions":{"push":true}},"sender":{"id":5},"issue":{"id":101,"number":3,"user":{"id":5},"title":"Title","body":"Body","created_at":"2023-11-14T22:13:20Z"}}`
		deliver := func(product *Server, delivery string, signed bool) *httptest.ResponseRecorder {
			request := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
			request.Header.Set("X-Forgejo-Event", "issues")
			request.Header.Set("X-Forgejo-Delivery", delivery)
			if signed {
				sum := hmac.New(sha256.New, []byte("intake-secret"))
				_, _ = sum.Write([]byte(body))
				request.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
			}
			response := httptest.NewRecorder()
			product.ServeHTTP(response, request)
			return response
		}
		if response := deliver(mapped, "unsigned-mapped", false); response.Code != http.StatusUnauthorized || len(reader.requests) != 0 {
			t.Fatalf("unsigned event reached provider readiness: %d", response.Code)
		}
		if _, err := db.CollaborationRepositoryByExternalID(t.Context(), mapped.FountainHost.ID, "7"); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("unsigned event allocated a Fountain identity: %v", err)
		}
		response := deliver(mapped, "mapped-creation", true)
		if response.Code != http.StatusOK {
			t.Fatalf("mapped creation refused: %d %s", response.Code, response.Body.String())
		}
		repository, err := db.CollaborationRepositoryByExternalID(t.Context(), mapped.FountainHost.ID, "7")
		if err != nil || repository.ID == 7 || repository.ID == foreignRepo.ID {
			t.Fatalf("native/Fountain/GitHub repository identities collided: %+v %v", repository, err)
		}
		actor, err := db.CollaborationActorByExternalID(t.Context(), mapped.FountainHost.ID, "5")
		if err != nil || actor.ID == 5 || actor.ID == foreignActor.ID {
			t.Fatalf("native/Fountain/GitHub actors collided: %+v %v", actor, err)
		}
		local := strconv.FormatInt(int64(repository.ID), 10)
		var outcome struct{ Repository string }
		if err := json.Unmarshal(response.Body.Bytes(), &outcome); err != nil || outcome.Repository != local || len(reader.requests) == 0 {
			t.Fatalf("intake did not retain the local repository and read native evidence: %+v %v", outcome, err)
		}
		evidence, err := source.ReadAcceptanceEvidence(t.Context(), local, "3", nil)
		if err != nil || evidence.Issue.PosterID != strconv.FormatInt(int64(actor.ID), 10) || reader.snapshot.Issue.Provenance.PosterID != "5" || reader.snapshot.RepositoryID != "7" {
			t.Fatalf("creator translation changed original provider evidence: %+v %v", evidence, err)
		}
		inputs, err := source.ReadDispatchInputs(t.Context(), local, "3", nil, "refs/heads/main")
		if err != nil || inputs.Tip != strings.Repeat("c", 40) {
			t.Fatalf("dispatch did not read the native repository: %+v %v", inputs, err)
		}
		before := len(reader.requests)
		for _, missing := range []string{strconv.FormatInt(int64(foreignRepo.ID), 10), "999", "0" + local} {
			if _, err := source.ReadAcceptanceEvidence(t.Context(), missing, "3", nil); err == nil || len(reader.requests) != before {
				t.Fatalf("foreign/missing/noncanonical local identity reached provider: %s %v", missing, err)
			}
		}
		if _, err := forgejo.NewFountainIdentities(t.Context(), db, github.ID); !errors.Is(err, collaboration.ErrIdentityConflict) {
			t.Fatalf("Fountain adapter accepted another namespace: %v", err)
		}
		if _, err := mapped.API.FountainIdentities.NativeActor(t.Context(), foreignActor.ID); err == nil {
			t.Fatal("foreign local actor reached the native namespace")
		}
		if native, err := mapped.API.FountainIdentities.NativeActor(t.Context(), actor.ID); err != nil || native != "5" {
			t.Fatalf("local actor did not resolve its exact native identity: %q %v", native, err)
		}
		reader.failure = forgejo.ErrUnavailable
		if _, err := source.ReadDispatchInputs(t.Context(), local, "3", nil, "refs/heads/main"); !errors.Is(err, forgejo.ErrUnavailable) {
			t.Fatalf("provider failure became usable/absent evidence: %v", err)
		}
		if failed := deliver(mapped, "mapped-pending", true); failed.Code != http.StatusServiceUnavailable {
			t.Fatalf("failed provider observation acknowledged pending creation: %d", failed.Code)
		}
		pendingDigest := sha256.Sum256([]byte("mapped-pending"))
		pendingID := "delivery:" + hex.EncodeToString(pendingDigest[:])
		pending, err := db.ReadinessWork(t.Context(), pendingID)
		if err != nil || pending.Root.Repository != int64(repository.ID) || pending.CreatorID != int64(actor.ID) || !pending.CreationVerified || !pending.CreatorWrite || pending.NativeIssueID != 101 {
			t.Fatalf("pending creation did not retain canonical creator/repository and native issue facts: %+v %v", pending, err)
		}
		reader.failure = nil
		renamed, err := db.ObserveCollaborationRepository(t.Context(), mapped.FountainHost.ID, "7", "renamed/team")
		if err != nil {
			t.Fatal(err)
		}
		if got, err := db.ObserveCollaborationRepository(t.Context(), mapped.FountainHost.ID, "7", ""); err != nil || got != renamed {
			t.Fatalf("identity-only observation overwrote metadata: %+v %v", got, err)
		}
		renamedActor, err := db.ObserveCollaborationActor(t.Context(), mapped.FountainHost.ID, "5", "renamed-account")
		if err != nil {
			t.Fatal(err)
		}
		if got, err := db.ObserveCollaborationActor(t.Context(), mapped.FountainHost.ID, "5", ""); err != nil || got != renamedActor {
			t.Fatalf("identity-only observation overwrote actor metadata: %+v %v", got, err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := store.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		restarted, err := NewFactory(server.Config, reopened)
		if err != nil {
			t.Fatal(err)
		}
		source.Identities = restarted.API.FountainIdentities
		restarted.Coordinator.AcceptanceReads = source
		if again := deliver(restarted, "mapped-pending", true); again.Code != http.StatusOK {
			t.Fatalf("mapped redelivery after reopen failed: %d %s", again.Code, again.Body.String())
		}
		if _, err := reopened.ReadinessWork(t.Context(), pendingID); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("mapped pending work did not settle after reopen/redelivery: %v", err)
		}
		if got, err := reopened.CollaborationRepositoryByExternalID(t.Context(), mapped.FountainHost.ID, "7"); err != nil || got != renamed {
			t.Fatalf("redelivery/rename/reopen changed local identity: %+v %v", got, err)
		}
	})
	digest := strings.Repeat("d", 64)
	server.Coordinator.AcceptanceReads = intakeRouteSource{evidence: control.AcceptanceEvidence{
		Issue:    control.AcceptanceIssueView{Index: "3", TitleDigest: digest, ContentDigest: digest, Visible: true},
		Revision: 12,
	}}
	body := `{"action":"edited","number":3,"repository":{"id":7},"sender":{"id":5}}`
	sum := hmac.New(sha256.New, []byte("intake-secret"))
	_, _ = sum.Write([]byte(body))
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatal("verified intake refused:", w.Code, w.Body.String())
	}
	var outcome struct {
		Repository string `json:"repository"`
		Issue      string `json:"issue"`
		Readiness  string `json:"readiness"`
		Changed    bool   `json:"changed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if outcome.Repository != "7" || outcome.Issue != "3" ||
		outcome.Readiness != "not_authorized" || !outcome.Changed {
		t.Fatal("intake outcome wrong:", outcome)
	}
	again := httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
	server.ServeHTTP(again, r)
	if again.Code != http.StatusOK {
		t.Fatal("duplicate intake refused:", again.Code)
	}
	t.Run("signed delivery reaches Factory without Forge", func(t *testing.T) {
		factoryHTTP := httptest.NewServer(server)
		defer factoryHTTP.Close()
		request, err := http.NewRequest(http.MethodPost, factoryHTTP.URL+"/-/soda/api/factory/intake", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("X-Forgejo-Event", "issues")
		request.Header.Set("X-Forgejo-Delivery", "delivery-1")
		request.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
		response, err := factoryHTTP.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(response.Body)
			t.Fatalf("signed intake did not cross the product boundary: %d %s", response.StatusCode, data)
		}
	})
	t.Run("repository creation publishes one question without starting intake", func(t *testing.T) {
		deliver := func(action, delivery string, signed bool) int {
			body := `{"action":"` + action + `","repository":{"id":17,"created_at":"2026-10-29T20:00:00Z"}}`
			r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
			r.Header.Set("X-Forgejo-Event", "repository")
			r.Header.Set("X-Forgejo-Delivery", delivery)
			if signed {
				sum := hmac.New(sha256.New, []byte("intake-secret"))
				_, _ = sum.Write([]byte(body))
				r.Header.Set("X-Forgejo-Signature", hex.EncodeToString(sum.Sum(nil)))
			}
			w := httptest.NewRecorder()
			server.ServeHTTP(w, r)
			return w.Code
		}
		if deliver("created", "unsigned", false) != http.StatusUnauthorized {
			t.Fatal("unsigned repository event was admitted")
		}
		if deliver("deleted", "deleted", true) != http.StatusOK {
			t.Fatal("unrelated repository action was not ignored")
		}
		if _, err := server.Store.CollaborationRepositoryByExternalID(t.Context(), server.FountainHost.ID, "17"); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("unadmitted repository event allocated a mapping: %v", err)
		}
		query := factory.FeedQuery{Repositories: []int64{17}, Limit: 20}
		before, _, _, err := server.Store.SubscribedFeed(context.Background(), query)
		if err != nil || len(before) != 0 {
			t.Fatalf("unadmitted creation published a post: %+v %v", before, err)
		}
		for _, delivery := range []string{"created", "created", "redelivery"} {
			if deliver("created", delivery, true) != http.StatusOK {
				t.Fatal("signed repository event was refused")
			}
		}
		repository, err := server.Store.CollaborationRepositoryByExternalID(t.Context(), server.FountainHost.ID, "17")
		if err != nil {
			t.Fatal(err)
		}
		query.Repositories = []int64{int64(repository.ID)}
		entries, _, _, err := server.Store.SubscribedFeed(context.Background(), query)
		if err != nil || len(entries) != 1 {
			t.Fatalf("opening question missing or duplicated: %+v %v", entries, err)
		}
		post, err := server.Store.FeedPost(context.Background(), entries[0].ID)
		if err != nil || post.Actor != "factory" || post.AuthorName != "Factory" || post.Body != store.RepositoryOpeningQuestion ||
			post.Payload.Communication.Kind != "ask" || post.Payload.Communication.QuestionState != "open" || post.Record.ParentID != "" {
			t.Fatalf("opening post attribution or content wrong: %+v %v", post, err)
		}
		pending, err := server.Store.PendingPostIntakes(context.Background(), 20)
		if err != nil || len(pending) != 0 {
			t.Fatalf("Factory question started human intake: %+v %v", pending, err)
		}
	})
}

// The private peer declares Fountain wire evidence. PostgreSQL, Unix transport,
// UID/admission/credential checks and the relocated source adapters are real.
func TestFactoryObservationAdaptersUsePrivateNativeSources(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := t.Context()
	db, _, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	host, err := db.BindConfiguredFountainLedger(ctx, "https://observation-fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	foreign := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	foreignRepo, err := db.ObserveCollaborationRepository(ctx, foreign.ID, "7", "same/name")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ObserveCollaborationActor(ctx, foreign.ID, "5", "same-login"); err != nil {
		t.Fatal(err)
	}
	identities, err := forgejo.NewFountainIdentities(ctx, db, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := identities.LocalRepository(ctx, 7)
	if err != nil || repository == 7 || repository == foreignRepo.ID {
		t.Fatal("fixture namespaces did not differ", repository, err)
	}
	actor, err := identities.LocalActor(ctx, "5")
	if err != nil || actor == 5 {
		t.Fatal("fixture actor namespaces did not differ", actor, err)
	}
	credential := filepath.Join(t.TempDir(), "observation-pat")
	if err := os.WriteFile(credential, []byte("declared-source-pat"), 0600); err != nil {
		t.Fatal(err)
	}
	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" || r.Header.Get("Authorization") != "token declared-source-pat" {
			http.Error(w, "bad actor", 403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(forgejo.User{ID: 5, Login: "fixture-observer"})
	}))
	defer rest.Close()
	var mode, calls, revision atomic.Int64
	revision.Store(12)
	const admission = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// Keep the owned private socket below Linux's sockaddr_un path bound; the
	// Go runner's test-name directory can already exceed that bound.
	socketDir, err := os.MkdirTemp("/home/vince", "source-observation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	socket := filepath.Join(socketDir, "s")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	peer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == extensions.BackgroundBootstrapPath {
			_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "declared-observation-peer"})
			return
		}
		if r.Header.Get(extensions.AdmissionHeader) != admission {
			http.Error(w, "bad admission", 401)
			return
		}
		if r.URL.Path == extensions.BackgroundRevisionPath {
			_ = json.NewEncoder(w).Encode(extensions.NativeRevisionObservation{Revision: revision.Load(), Idle: true})
			return
		}
		if r.URL.Path != extensions.BackgroundSnapshotPath {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		var in struct {
			extensions.SnapshotRequest
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Token != "declared-source-pat" || in.RepositoryID != "7" || in.ActorID != "5" {
			t.Error("native credential or scope changed")
			http.Error(w, "bad scope", 400)
			return
		}
		if len(in.Families) == 1 && string(in.Families[0]) == "repository_issues" {
			if in.IssueIndex != "" || in.Limit != 50 || in.Cursor != "" {
				t.Error("completion request changed")
				http.Error(w, "bad page", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.NativeSnapshot{RepositoryID: "7", RepositoryIssues: &forgejo.RepositoryIssuesPage{Complete: true, Items: []forgejo.RepositoryIssueItem{{IssueID: "101", IssueIndex: "1"}}}})
			return
		}
		if in.IssueIndex != "1" {
			t.Error("issue locator changed")
			http.Error(w, "bad issue", 400)
			return
		}
		snapshot := extensions.NativeSnapshot{RepositoryID: "7", Issue: &forgejo.IssueEvidence{ID: "101", Index: "1", Title: "Current title", Content: "Exact body\n", TitleDigest: forgejo.ContentDigest("Current title"), ContentDigest: forgejo.ContentDigest("Exact body\n"), Visible: true, Complete: true, Provenance: forgejo.CreationProvenance{PosterID: "5", CreatedUnix: 1700000000, Verified: true, FirstCreated: true}, Lifecycle: []forgejo.LifecycleEvent{{Kind: "retitled", OccurrenceID: "50"}}},
			Dependencies: &forgejo.DependencyPage{IssueID: "101", Total: 2, Complete: true, Items: []forgejo.DependencyEvidence{{IssueID: "101", OccurrenceID: "60", DependencyID: "102", Visible: true, Complete: true}, {IssueID: "101", OccurrenceID: "61", Complete: true}}},
			Comments:     &forgejo.CommentPage{Complete: true, Total: 2, Items: []forgejo.CommentEvidence{{ID: "71", Content: "Chosen source", ContentDigest: forgejo.ContentDigest("Chosen source"), Visible: true, Complete: true}, {ID: "72", Content: "Unselected discussion", ContentDigest: forgejo.ContentDigest("Unselected discussion"), Visible: true, Complete: true}}},
			Refs:         []forgejo.RefEvidence{{Ref: "refs/heads/main", OID: strings.Repeat("c", 40), Exists: true, Visible: true, Complete: true}},
		}
		switch mode.Load() {
		case 2:
			snapshot.Issue.Content = "Changed after digest capture"
		case 7:
			snapshot.Dependencies.Items[1].Visible = true
			snapshot.Dependencies.Items[1].DependencyID = "103"
		case 8:
			snapshot.Dependencies.Complete = false
		case 9:
			revision.Add(1)
		case 10:
			snapshot.Issue.Lifecycle = append(snapshot.Issue.Lifecycle, forgejo.LifecycleEvent{Kind: "retitled", OccurrenceID: "49"})
		case 11:
			snapshot.RepositoryID = "8"
		case 12:
			snapshot.Comments.Complete = false
		case 13:
			snapshot.Refs[0].OID = ""
			snapshot.Refs[0].Exists = false
			snapshot.Refs[0].Visible = false
		}
		_ = json.NewEncoder(w).Encode(snapshot)
	})}
	go func() { _ = peer.Serve(listener) }()
	defer peer.Close()
	observer := forgejo.NewServiceObserver(socket, uint32(os.Getuid()), credential, forgejo.New(rest.URL))
	observer.Identities = identities
	source := observation.NewServiceReadinessSource(observer, identities)
	repo := strconv.FormatInt(int64(repository), 10)
	evidence, err := source.ReadAcceptanceEvidence(ctx, repo, "1", []string{"71"})
	if err != nil || evidence.Revision != 12 || evidence.Issue.PosterID != strconv.FormatInt(int64(actor), 10) || evidence.Issue.TitleOccurrence != "50" || len(evidence.Comments) != 1 || evidence.Comments[0].ID != "71" || evidence.Comments[0].ContentVer != 0 || len(evidence.Dependencies) != 2 || evidence.Dependencies[1].Visible {
		t.Fatal("readiness lost exact scope or redaction", evidence, err)
	}
	inputs, err := source.ReadDispatchInputs(ctx, repo, "1", []string{"71"}, "refs/heads/main")
	if err != nil || inputs.Revision != 12 || inputs.Tip != strings.Repeat("c", 40) || inputs.Issue.Body != "Exact body\n" || len(inputs.Comments) != 1 || inputs.Comments[0].Content != "Chosen source" {
		t.Fatal("dispatch input changed", inputs, err)
	}
	if got, idle, err := source.ObserveNativeRevision(ctx); err != nil || got != 12 || !idle {
		t.Fatal("revision changed", got, idle, err)
	}
	before := calls.Load()
	for _, foreign := range []int64{int64(foreignRepo.ID), 7} {
		if _, err := source.ReadAcceptanceEvidence(ctx, strconv.FormatInt(foreign, 10), "1", nil); err == nil || calls.Load() != before {
			t.Fatal("foreign readiness reached native peer", foreign, err)
		}
		if _, err := source.ReadDispatchInputs(ctx, strconv.FormatInt(foreign, 10), "1", nil, "refs/heads/main"); err == nil || calls.Load() != before {
			t.Fatal("foreign dispatch reached native peer", foreign, err)
		}
	}
	for _, bad := range []struct {
		mode   int64
		reason string
	}{{2, control.RefusalSnapshotUnavailable}, {8, control.RefusalIncompleteEvidence}, {9, control.RefusalStaleEvidence}, {10, control.RefusalIncompleteEvidence}, {12, control.RefusalIncompleteEvidence}} {
		mode.Store(bad.mode)
		_, err := source.ReadAcceptanceEvidence(ctx, repo, "1", []string{"71"})
		var refusal *control.AcceptanceRefusal
		if !errors.As(err, &refusal) || refusal.Reason != bad.reason {
			t.Fatal("refusal changed", bad.mode, err)
		}
		revision.Store(12)
	}
	mode.Store(11)
	foreignEvidence, err := source.ReadAcceptanceEvidence(ctx, repo, "1", nil)
	if err == nil || foreignEvidence.Revision != 0 || foreignEvidence.Issue.ID != "" {
		t.Fatal("transport accepted foreign snapshot", foreignEvidence, err)
	}
	mode.Store(13)
	inputs, err = source.ReadDispatchInputs(ctx, repo, "1", nil, "refs/heads/main")
	if err != nil || inputs.Tip != "" {
		t.Fatal("hidden ref invented a source tip", inputs, err)
	}
	completion := observation.NewCompletionSnapshotSource(observer, identities)
	mode.Store(7)
	inventory, err := completion.EnumerateIssues(ctx, int64(repository), "")
	if err != nil || !inventory.Complete || !inventory.Idle || inventory.Revision != 12 || len(inventory.Issues) != 1 || inventory.Issues[0].Index != 1 {
		t.Fatal("completion bracket changed", inventory, err)
	}
	var body factory.CompletionIssueEvidence
	if err := json.Unmarshal([]byte(inventory.Issues[0].Data), &body); err != nil || body.Title != "Current title" || body.Body != "Exact body\n" || len(body.Dependencies) != 2 || body.Dependencies[0] != "102" || body.Dependencies[1] != "103" {
		t.Fatal("completion body changed", body, err)
	}
	mode.Store(0)
	if _, err := completion.EnumerateIssues(ctx, int64(repository), ""); err == nil {
		t.Fatal("completion sealed hidden prerequisite subset")
	}
	before = calls.Load()
	for _, foreign := range []int64{int64(foreignRepo.ID), 7} {
		if _, err := completion.EnumerateIssues(ctx, foreign, ""); err == nil || calls.Load() != before {
			t.Fatal("foreign completion reached native peer", foreign, err)
		}
	}
	missing := observation.NewServiceReadinessSource(nil, identities)
	if _, err := missing.ReadAcceptanceEvidence(ctx, repo, "1", nil); err == nil {
		t.Fatal("missing observer returned evidence")
	}
	if _, err := missing.ReadDispatchInputs(ctx, repo, "1", nil, "refs/heads/main"); err == nil {
		t.Fatal("missing observer returned inputs")
	}
	if _, _, err := missing.ObserveNativeRevision(ctx); err == nil {
		t.Fatal("missing observer returned revision")
	}
	missingCompletion := observation.NewCompletionSnapshotSource(nil, identities)
	if _, err := missingCompletion.EnumerateIssues(ctx, int64(repository), ""); err == nil {
		t.Fatal("missing completion observer returned page")
	}
}

func TestFactoryIntakeRouteRefusesUnauthenticated(t *testing.T) {
	server, done := intakeRouteServer(t, "intake-secret")
	defer done()
	body := `{"action":"edited","number":3,"repository":{"id":7}}`
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("unsigned intake served:", w.Code)
	}
}

func TestFactoryIntakeRouteUnavailableWithoutSecret(t *testing.T) {
	server, done := intakeRouteServer(t, "")
	defer done()
	body := `{"action":"edited","number":3,"repository":{"id":7}}`
	r := httptest.NewRequest(http.MethodPost, "/-/soda/api/factory/intake", strings.NewReader(body))
	r.Header.Set("X-Forgejo-Event", "issues")
	r.Header.Set("X-Forgejo-Delivery", "delivery-1")
	r.Header.Set("X-Forgejo-Signature", strings.Repeat("a", 64))
	w := httptest.NewRecorder()
	server.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("secretless intake served:", w.Code)
	}
}
