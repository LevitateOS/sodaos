package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type applicationBoundary struct {
	store         *store.Store
	call          func(actor int64, method, path string, body any) (int, []byte)
	effects       atomic.Int64
	fail          atomic.Bool
	changeSession atomic.Bool
	last          application.Request
	bundleDigest  string
}

func applicationHTTPFixture(t *testing.T) *applicationBoundary {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, _, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	fixture := &applicationBoundary{store: db}
	var actor atomic.Int64
	var callbacks atomic.Int64
	path := "@soda-application-callback-" + strconv.Itoa(os.Getpid())
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			w.WriteHeader(403)
			return
		}
		if in.Operation == extensions.OperationRepository {
			if in.RepositoryID != "7" || actor.Load() == 43 {
				w.WriteHeader(404)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "northstar", Name: "equipment", Permission: "read"}})
			return
		}
		if in.Operation != extensions.OperationCurrentActor {
			w.WriteHeader(403)
			return
		}
		id := actor.Load()
		if fixture.changeSession.Load() && callbacks.Add(1) > 1 {
			id++
		}
		_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: strconv.FormatInt(id, 10), Username: "soda-tester"}})
	}))
	_ = callback.Listener.Close()
	callback.Listener = listener
	callback.Start()
	t.Cleanup(callback.Close)
	t.Setenv(extensions.CallbackEnv, path)
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/application/snapshots" {
			var request application.SnapshotRequest
			if json.NewDecoder(r.Body).Decode(&request) != nil || request.Validate() != nil {
				w.WriteHeader(400)
				return
			}
			id := request.ID
			if id == "" {
				id = "a" + strings.Repeat("9", 24)
			}
			_ = json.NewEncoder(w).Encode(application.SnapshotInventory{Snapshots: []application.Snapshot{{ID: id, CreatedAt: 1, Application: request.Application}}, Complete: true})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/application/setup" {
			var setup application.SetupRequest
			if json.NewDecoder(r.Body).Decode(&setup) != nil || setup.Validate() != nil {
				w.WriteHeader(400)
				return
			}
			fixture.effects.Add(1)
			if fixture.fail.Load() {
				w.WriteHeader(502)
				return
			}
			container := setup.ExpectedContainer
			if container == "" {
				container = strings.Repeat("7", 64)
			}
			_ = json.NewEncoder(w).Encode(application.SetupResult{Operation: setup.Setup.ID, Observation: application.SetupObservation{Container: container, Running: true, Ready: true}})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/application" || json.NewDecoder(r.Body).Decode(&fixture.last) != nil {
			w.WriteHeader(400)
			return
		}
		fixture.effects.Add(1)
		if fixture.fail.Load() {
			w.WriteHeader(502)
			return
		}
		in := fixture.last
		container := in.ExpectedContainer
		if container == "" {
			container = strings.Repeat("a", 64)
		}
		if in.Action == "replace" || in.Action == "restore" {
			container = strings.Repeat("b", 64)
			if container == in.ExpectedContainer {
				container = strings.Repeat("c", 64)
			}
		}
		running := in.Action == "start"
		_ = json.NewEncoder(w).Encode(application.Observation{ID: in.Application.ID, Revision: in.Application.Revision, Container: container, Image: in.Application.Image, Origin: in.Application.Origin, Running: running, Ready: running, BundleFileSHA256: fixture.bundleDigest})
	}))
	t.Cleanup(native.Close)
	api := &API{Store: db, Config: &config.Config{OperatorID: 1, ForgejoURL: "https://forgejo.example.test"}, Host: &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}}
	mux := http.NewServeMux()
	api.registerExtensionProductRoutes(mux)
	fixture.call = func(id int64, method, path string, body any) (int, []byte) {
		t.Helper()
		actor.Store(id)
		callbacks.Store(0)
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		authority, err := json.Marshal(extensions.Authority{ExtensionID: "soda", InstanceID: "application-test", SessionGeneration: "generation-1", Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)}, Actor: extensions.Actor{ID: strconv.FormatInt(id, 10), Username: "soda-tester"}})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		if method != http.MethodGet {
			r.Header.Set("Origin", "https://forgejo.example.test")
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code, w.Body.Bytes()
	}
	return fixture
}

type applicationNativeTransport struct{ target string }

func (t applicationNativeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	copy.URL = &u
	copy.URL.Scheme = "http"
	copy.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(copy)
}

func applicationAPISpec() application.Spec {
	return application.Spec{ID: "a" + strings.Repeat("1", 24), Revision: 1, Name: "Example", ArtifactSHA256: strings.Repeat("2", 64), BundleSHA256: strings.Repeat("3", 64), ToolsSHA256: strings.Repeat("4", 64), RuntimeImage: "sha256:" + strings.Repeat("5", 64), Image: "sha256:" + strings.Repeat("6", 64), Data: "a" + strings.Repeat("7", 24), OwnerTeam: "12345678-1234-1234-1234-123456789abc", Origin: "https://app.example.test", Port: 20001}
}

func TestApplicationRestoreHTTPFencesUncertainEffectsAndReplays(t *testing.T) {
	f := applicationHTTPFixture(t)
	spec := applicationAPISpec()
	base := "/api/applications/" + spec.ID
	if status, body := f.call(1, "POST", "/api/applications", applicationRegistration{Application: spec, AdministratorID: 41}); status != 201 {
		t.Fatalf("register %d %s", status, body)
	}
	if status, body := f.call(1, "POST", base+"/actions", applicationAction{Action: "install", Revision: 1}); status != 200 {
		t.Fatalf("install %d %s", status, body)
	}
	if status, _ := f.call(41, "GET", base+"/snapshots", nil); status != 403 {
		t.Fatalf("administrator read operator snapshots: %d", status)
	}
	if status, body := f.call(1, "GET", base+"/snapshots", nil); status != 200 {
		t.Fatalf("snapshot inventory %d %s", status, body)
	}
	input := applicationRestoreInput{Action: "restore", ID: "a" + strings.Repeat("8", 24), Snapshot: "a" + strings.Repeat("9", 24), Revision: 1, ExpectedContainer: strings.Repeat("a", 64), Confirm: true}
	if status, _ := f.call(41, "POST", base+"/restores", input); status != 403 {
		t.Fatalf("administrator restored: %d", status)
	}
	unconfirmed := input
	unconfirmed.Confirm = false
	if status, _ := f.call(1, "POST", base+"/restores", unconfirmed); status != 400 {
		t.Fatalf("unconfirmed restore: %d", status)
	}
	f.fail.Store(true)
	if status, body := f.call(1, "POST", base+"/restores", input); status != 502 {
		t.Fatalf("failed native restore %d %s", status, body)
	}
	pending, err := f.store.PendingApplicationRestore(context.Background(), spec.ID)
	if err != nil || pending == nil || pending.ID != input.ID {
		t.Fatalf("uncertain restore not retained: %+v %v", pending, err)
	}
	effects := f.effects.Load()
	if status, _ := f.call(1, "POST", base+"/actions", applicationAction{Action: "start", Revision: 1, ExpectedContainer: input.ExpectedContainer}); status != 409 {
		t.Fatalf("start bypassed pending restore: %d", status)
	}
	if f.effects.Load() != effects {
		t.Fatal("pending restore launched another native action")
	}
	if status, body := f.call(1, "GET", base, nil); status != 200 || !strings.Contains(string(body), `"restore"`) {
		t.Fatalf("pending readback %d %s", status, body)
	}
	f.fail.Store(false)
	if status, body := f.call(1, "POST", base+"/restores", input); status != 200 {
		t.Fatalf("reconcile %d %s", status, body)
	}
	current, err := f.store.Application(context.Background(), spec.ID, 41)
	if err != nil || current.Application.Revision != 2 || current.Application.Data == spec.Data || current.Application.Origin != spec.Origin {
		t.Fatalf("restore binding %+v %v", current, err)
	}
	old, err := f.store.ApplicationRevision(context.Background(), spec.ID, 41, 1)
	if err != nil || old != spec {
		t.Fatalf("old reference lost: %+v %v", old, err)
	}
	effects = f.effects.Load()
	if status, body := f.call(1, "POST", base+"/restores", input); status != 200 {
		t.Fatalf("replay %d %s", status, body)
	}
	if f.effects.Load() != effects {
		t.Fatal("completed restore replayed a native effect")
	}
	input.Action = "abort"
	if status, _ := f.call(1, "POST", base+"/restores", input); status != 409 {
		t.Fatalf("completed restore appeared aborted: %d", status)
	}
	input = applicationRestoreInput{Action: "restore", ID: "a" + strings.Repeat("f", 24), Snapshot: input.Snapshot, Revision: 2, ExpectedContainer: current.Observation.Container, Confirm: true}
	f.fail.Store(true)
	if status, _ := f.call(1, "POST", base+"/restores", input); status != 502 {
		t.Fatalf("second pending restore: %d", status)
	}
	f.fail.Store(false)
	input.Action = "abort"
	if status, body := f.call(1, "POST", base+"/restores", input); status != 200 {
		t.Fatalf("abort %d %s", status, body)
	}
	after, err := f.store.Application(context.Background(), spec.ID, 41)
	if err != nil || after.Application != current.Application {
		t.Fatalf("abort changed admitted data: %+v %v", after, err)
	}
}

func TestApplicationAuthorityAndLifecycleHTTP(t *testing.T) {
	f := applicationHTTPFixture(t)
	spec := applicationAPISpec()
	base := "/api/applications/" + spec.ID
	registration := applicationRegistration{Application: spec, AdministratorID: 41}
	if status, _ := f.call(41, "POST", "/api/applications", registration); status != 403 {
		t.Fatalf("administrator registered native references: %d", status)
	}
	bad := registration
	bad.Application.Origin = "https://forgejo.example.test:8443"
	if status, _ := f.call(1, "POST", "/api/applications", bad); status != 409 {
		t.Fatalf("same cookie hostname: %d", status)
	}
	if status, body := f.call(1, "POST", "/api/applications", registration); status != 201 {
		t.Fatalf("registration %d %s", status, body)
	}
	if f.effects.Load() != 0 {
		t.Fatal("registration launched application")
	}
	if status, _ := f.call(42, "GET", base, nil); status != 404 {
		t.Fatalf("unrelated Soda/app identity read: %d", status)
	}
	if status, body := f.call(42, "GET", "/api/applications", nil); status != 200 || !strings.Contains(string(body), `"applications":[]`) {
		t.Fatalf("inventory grant %d %s", status, body)
	}
	if status, _ := f.call(41, "GET", base, nil); status != 200 || f.effects.Load() != 0 {
		t.Fatal("navigation performed native operation")
	}
	install := applicationAction{Action: "install", Revision: 1}
	if status, _ := f.call(41, "POST", base+"/actions", install); status != 403 {
		t.Fatalf("admin installed %d", status)
	}
	if status, body := f.call(1, "POST", base+"/actions", install); status != 200 {
		t.Fatalf("install %d %s", status, body)
	}
	container := strings.Repeat("a", 64)
	start := applicationAction{Action: "start", Revision: 1, ExpectedContainer: container}
	if status, _ := f.call(42, "POST", base+"/actions", start); status != 404 {
		t.Fatalf("unrelated user start %d", status)
	}
	stale := start
	stale.ExpectedContainer = strings.Repeat("c", 64)
	effects := f.effects.Load()
	if status, _ := f.call(41, "POST", base+"/actions", stale); status != 409 || f.effects.Load() != effects {
		t.Fatal("stale native generation effected")
	}
	f.changeSession.Store(true)
	if status, _ := f.call(41, "POST", base+"/actions", start); status != 401 || f.effects.Load() != effects {
		t.Fatal("changed session effected")
	}
	f.changeSession.Store(false)
	f.bundleDigest = strings.Repeat("d", 64)
	if status, body := f.call(41, "POST", base+"/actions", start); status != 200 {
		t.Fatalf("start %d %s", status, body)
	}
	if err := f.store.SetApplicationProject(context.Background(), spec.ID, 41, 7); err != nil {
		t.Fatal(err)
	}
	beforeLaunch := f.effects.Load()
	status, body := f.call(42, "GET", "/api/applications", nil)
	var launch struct {
		Applications []json.RawMessage            `json:"applications"`
		Launchable   []map[string]json.RawMessage `json:"launchable"`
		CanSetup     bool                         `json:"can_setup"`
	}
	if status != 200 || json.Unmarshal(body, &launch) != nil || len(launch.Applications) != 0 || len(launch.Launchable) != 1 || launch.CanSetup {
		t.Fatalf("ordinary launch projection %d %s", status, body)
	}
	row := launch.Launchable[0]
	var reference map[string]json.RawMessage
	if len(row) != 8 || json.Unmarshal(row["application"], &reference) != nil || len(reference) != 4 {
		t.Fatalf("ordinary launch exposed private fields: %s", body)
	}
	for _, key := range []string{"id", "revision", "name", "origin"} {
		if _, ok := reference[key]; !ok {
			t.Fatalf("launch missing %s", key)
		}
	}
	if _, ok := row["delivery_id"]; ok {
		t.Fatal("image readback without verified F10 supplied a delivery identity")
	}
	if status, _ := f.call(42, "GET", "/api/applications?delivery_id="+strings.Repeat("e", 32), nil); status != 400 {
		t.Fatal("caller-supplied delivery filter was admitted")
	}
	status, body = f.call(43, "GET", "/api/applications", nil)
	if status != 200 || json.Unmarshal(body, &launch) != nil || len(launch.Launchable) != 0 || f.effects.Load() != beforeLaunch {
		t.Fatalf("unrelated project reader or launch performed native effect: %d %s", status, body)
	}
	stop := applicationAction{Action: "stop", Revision: 1, ExpectedContainer: container}
	if status, _ := f.call(41, "POST", base+"/actions", stop); status != 409 {
		t.Fatal("stop lacked explicit confirmation")
	}
	stop.ConfirmStop = true
	f.fail.Store(true)
	if status, _ := f.call(41, "POST", base+"/actions", stop); status != 502 {
		t.Fatalf("unknown stop %d", status)
	}
	record, err := f.store.Application(context.Background(), spec.ID, 41)
	if err != nil || record.Observation != nil {
		t.Fatalf("uncertainty reported as success %+v %v", record, err)
	}
	f.fail.Store(false)
	if status, _ := f.call(41, "POST", base+"/actions", applicationAction{Action: "inspect", Revision: 1}); status != 200 {
		t.Fatalf("explicit inspect %d", status)
	}
}

func TestApplicationImageReplacementHTTP(t *testing.T) {
	f := applicationHTTPFixture(t)
	spec := applicationAPISpec()
	base := "/api/applications/" + spec.ID
	if status, _ := f.call(1, "POST", "/api/applications", applicationRegistration{Application: spec, AdministratorID: 41}); status != 201 {
		t.Fatal(status)
	}
	if status, _ := f.call(1, "POST", base+"/actions", applicationAction{Action: "install", Revision: 1}); status != 200 {
		t.Fatal(status)
	}
	// Prepared builds are operator-only and resolved from native receipts. A
	// browser-supplied build ID alone cannot stop or replace the current app.
	buildRequest := applicationAction{Action: "replace", Revision: 1, ExpectedContainer: strings.Repeat("a", 64), ConfirmStop: true, ReplacementBuild: strings.Repeat("e", 32)}
	beforeBuild := f.effects.Load()
	if status, _ := f.call(41, "POST", base+"/actions", buildRequest); status != 403 || f.effects.Load() != beforeBuild {
		t.Fatal("administrator used build replacement")
	}
	if status, _ := f.call(1, "POST", base+"/actions", buildRequest); status != 409 || f.effects.Load() != beforeBuild {
		t.Fatal("unknown build affected native application")
	}
	if status, _ := f.call(41, "GET", "/api/repositories/7/application-builds", nil); status != 403 {
		t.Fatal("employee read operator build inventory")
	}
	next := spec
	next.Revision = 2
	next.Image = "sha256:" + strings.Repeat("b", 64)
	next.BundleSHA256 = strings.Repeat("d", 64)
	request := applicationAction{Action: "replace", Revision: 1, ExpectedContainer: strings.Repeat("a", 64), ConfirmStop: true, Replacement: &next}
	if status, _ := f.call(41, "POST", base+"/actions", request); status != 403 {
		t.Fatal("administrator replaced image")
	}
	bad := next
	bad.Data = "a" + strings.Repeat("9", 24)
	request.Replacement = &bad
	effects := f.effects.Load()
	if status, _ := f.call(1, "POST", base+"/actions", request); status != 409 || f.effects.Load() != effects {
		t.Fatal("replacement changed retained data identity")
	}
	request.Replacement = &next
	f.fail.Store(true)
	if status, _ := f.call(1, "POST", base+"/actions", request); status != 502 {
		t.Fatal("native refusal was not reported")
	}
	unchanged, err := f.store.Application(context.Background(), spec.ID, 41)
	if err != nil || unchanged.Application != spec || unchanged.Observation != nil {
		t.Fatal("failed replacement advanced app references", unchanged, err)
	}
	f.fail.Store(false)
	// Retry binds the exact old container even though the interrupted call
	// cleared its cached observation. Native admission checks it again.
	if status, body := f.call(1, "POST", base+"/actions", request); status != 200 {
		t.Fatalf("replace %d %s", status, body)
	}
	record, err := f.store.Application(context.Background(), spec.ID, 41)
	if err != nil || record.Application != next || record.Observation == nil || record.Observation.Running || record.Observation.Ready || record.Observation.Container == request.ExpectedContainer {
		t.Fatalf("replacement not stopped exact new generation %+v %v", record, err)
	}
	old, err := f.store.ApplicationRevision(context.Background(), spec.ID, 41, 1)
	if err != nil || old != spec {
		t.Fatal("previous admitted references lost")
	}
	if status, _ := f.call(41, "POST", base+"/actions", applicationAction{Action: "start", Revision: 1, ExpectedContainer: request.ExpectedContainer}); status != 409 {
		t.Fatal("old revision could start replaced app")
	}
}

func setupRenewalSpec(seed string, port int) application.SetupSpec {
	return application.SetupSpec{
		ID: "a" + strings.Repeat(seed, 24), Name: "Example",
		ArtifactSHA256: strings.Repeat("2", 64), BundleSHA256: strings.Repeat("3", 64), ToolsSHA256: strings.Repeat("4", 64),
		RuntimeImage: "sha256:" + strings.Repeat("5", 64), Image: "sha256:" + strings.Repeat("6", 64),
		Data: "a" + strings.Repeat(seed, 24), Origin: "https://renew-" + seed + ".example.test", Port: port,
	}
}

func TestApplicationSetupRenewalHTTP(t *testing.T) {
	f := applicationHTTPFixture(t)
	ctx := context.Background()
	act := func(actor int64, id, action string, confirm bool) (int, []byte) {
		return f.call(actor, "POST", "/api/applications/setups/"+id+"/actions", map[string]any{"action": action, "confirm": confirm})
	}

	ref := setupRenewalSpec("b", 20021)
	status, body := f.call(1, "POST", "/api/applications/setups", map[string]any{"application": ref, "administrator_id": "41"})
	if status != 201 {
		t.Fatalf("setup reservation %d %s", status, body)
	}
	var created application.SetupRecord
	// The reservation mints a fresh setup identity (application.NewIdentity);
	// the spec ID survives as the application reference, not the record ID.
	if err := json.Unmarshal(body, &created); err != nil || created.Application.ID != ref.ID || created.State != "pending" {
		t.Fatalf("setup record %s %v", body, err)
	}
	if f.effects.Load() != 0 {
		t.Fatal("reservation performed a native effect")
	}
	// A fresh window needs no renewal, and renewal stays operator-confirmed.
	if status, body := act(1, created.ID, "renew", true); status != 409 || !strings.Contains(string(body), "setup_renewal_unneeded") {
		t.Fatalf("unexpired renew %d %s", status, body)
	}
	if status, _ := act(1, created.ID, "renew", false); status != 400 {
		t.Fatal("unconfirmed renew admitted")
	}
	if status, _ := act(41, created.ID, "renew", true); status != 403 {
		t.Fatal("administrator renewed")
	}
	if f.effects.Load() != 0 {
		t.Fatal("refused renew performed a native effect")
	}

	// Interim fail-closed: a Soda-side "unclaimed" read cannot prove the
	// durable owner claim alone, so an expired pending setup refuses
	// rotation before any renewal-audit mutation and before any host call.
	// The old token is kept, no renewal is consumed, and the retained
	// reservation still blocks fresh setups at the same address.
	expiredRef := setupRenewalSpec("c", 20022)
	expired := application.SetupRecord{ID: expiredRef.ID, Application: expiredRef, AdministratorID: 41, ExpiresAt: time.Now().Add(-time.Minute).Unix(), State: "pending"}
	if err := f.store.CreateApplicationSetup(ctx, expired); err != nil {
		t.Fatal(err)
	}
	refusalEffects := f.effects.Load()
	status, body = act(1, expired.ID, "renew", true)
	if status != 409 || !strings.Contains(string(body), "setup_renewal_unavailable") {
		t.Fatalf("Soda-unclaimed renew %d %s", status, body)
	}
	if f.effects.Load() != refusalEffects {
		t.Fatal("refused renew performed a native effect")
	}
	stored, err := f.store.ApplicationSetup(ctx, expired.ID)
	if err != nil || stored.ExpiresAt != expired.ExpiresAt || stored.Observation != nil {
		t.Fatalf("refused renew mutated the retained reservation: %+v %v", stored, err)
	}
	// The refusal consumes no renewal: retrying refuses identically, and
	// the address is still reserved — no fresh setup can claim it.
	if status, _ := act(1, expired.ID, "renew", true); status != 409 {
		t.Fatal("refused setup renewed on retry")
	}
	clash := expiredRef
	clash.ID = "a" + strings.Repeat("d", 24)
	if status, _ := f.call(1, "POST", "/api/applications/setups", map[string]any{"application": clash, "administrator_id": "41"}); status != 409 {
		t.Fatal("refused reservation released its address")
	}

	// A malformed re-begin never consumes the bounded renewal: 400 before
	// any expiry or audit mutation, with no native effect, and the same
	// setup still refuses without rotation afterward.
	malformedRef := setupRenewalSpec("1", 20025)
	malformed := application.SetupRecord{ID: malformedRef.ID, Application: malformedRef, AdministratorID: 41, ExpiresAt: time.Now().Add(-time.Minute).Unix(), State: "pending"}
	if err := f.store.CreateApplicationSetup(ctx, malformed); err != nil {
		t.Fatal(err)
	}
	beforeMalformed := f.effects.Load()
	if status, body := f.call(1, "POST", "/api/applications/setups/"+malformed.ID+"/actions", map[string]any{"action": "renew", "confirm": true, "expected_container": "not-a-generation"}); status != 400 || !strings.Contains(string(body), "invalid_setup_action") {
		t.Fatalf("malformed renew %d %s", status, body)
	}
	if f.effects.Load() != beforeMalformed {
		t.Fatal("malformed renew performed a native effect")
	}
	storedMalformed, err := f.store.ApplicationSetup(ctx, malformed.ID)
	if err != nil || storedMalformed.ExpiresAt != malformed.ExpiresAt {
		t.Fatalf("malformed renew moved expiry: %+v %v", storedMalformed, err)
	}
	// Interim fail-closed: the valid re-begin after a malformed attempt
	// meets the same claim gate — refused, no rotation, no native effect.
	if status, body := act(1, malformed.ID, "renew", true); status != 409 || !strings.Contains(string(body), "setup_renewal_unavailable") {
		t.Fatalf("renew after malformed rotated: %d %s", status, body)
	}
	if f.effects.Load() != beforeMalformed {
		t.Fatal("renew after malformed performed a native effect")
	}

	// Past the grace window the locally provable closed gate fires before
	// the claim check: still no rotation, still no native effects.
	staleRef := setupRenewalSpec("e", 20023)
	stale := application.SetupRecord{ID: staleRef.ID, Application: staleRef, AdministratorID: 41, ExpiresAt: time.Now().Add(-time.Duration(store.SetupRenewGraceSeconds+60) * time.Second).Unix(), State: "pending"}
	if err := f.store.CreateApplicationSetup(ctx, stale); err != nil {
		t.Fatal(err)
	}
	effects := f.effects.Load()
	if status, body := act(1, stale.ID, "renew", true); status != 410 || !strings.Contains(string(body), "setup_renewal_closed") {
		t.Fatalf("past-grace renew %d %s", status, body)
	}
	if f.effects.Load() != effects {
		t.Fatal("closed renew performed a native effect")
	}
	if status, _ := act(1, "a"+strings.Repeat("9", 24), "renew", true); status != 404 {
		t.Fatal("unknown setup renewed")
	}

	// Settled setups answer without replaying native effects.
	settledRef := setupRenewalSpec("f", 20024)
	settled := application.SetupRecord{ID: settledRef.ID, Application: settledRef, AdministratorID: 41, ExpiresAt: time.Now().Add(-time.Minute).Unix(), State: "pending"}
	if err := f.store.CreateApplicationSetup(ctx, settled); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SettleApplicationSetup(ctx, settled, application.SetupResult{Operation: settled.ID}, true); err != nil {
		t.Fatal(err)
	}
	effects = f.effects.Load()
	status, body = act(1, settled.ID, "renew", true)
	if status != 200 || !strings.Contains(string(body), `"aborted"`) || f.effects.Load() != effects {
		t.Fatalf("settled renew replayed: %d %s", status, body)
	}
}

// FJ-v3-812: a 502 setup_unconfirmed does not imply no rotation — the
// store renewal may already be committed while the native outcome is
// unknown. The setup is not stranded: a plain begin retries to
// convergence without consuming another renewal, and the observation is
// recorded against the actual generation.
func TestApplicationSetupBeginRetriesAfterUnconfirmed(t *testing.T) {
	f := applicationHTTPFixture(t)
	ctx := context.Background()
	ref := setupRenewalSpec("7", 20026)
	expired := application.SetupRecord{ID: ref.ID, Application: ref, AdministratorID: 41, ExpiresAt: time.Now().Add(-time.Minute).Unix(), State: "pending"}
	if err := f.store.CreateApplicationSetup(ctx, expired); err != nil {
		t.Fatal(err)
	}
	// Interim fail-closed stages the committed renewal below the HTTP
	// refusal: Soda-unclaimed renew refuses rotation over HTTP, so the
	// committed-renewal premise this convergence test needs is staged
	// through the store primitive directly (same audit row, same expiry
	// advance, no host effect). The re-begin below runs the ordinary
	// begin flow against the re-armed record.
	f.fail.Store(false)
	rearmedSetup, err := f.store.RenewApplicationSetup(ctx, expired, 1)
	if err != nil || rearmedSetup.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("renewal did not re-arm: %v", err)
	}
	rearmed := struct {
		Setup application.SetupRecord `json:"setup"`
	}{Setup: rearmedSetup}
	// The re-begin outcome is lost: 502 carries no claim about whether
	// the native side rotated, only that the outcome is unknown.
	f.fail.Store(true)
	status, body := f.call(1, "POST", "/api/applications/setups/"+expired.ID+"/actions", map[string]any{"action": "begin", "expected_container": strings.Repeat("7", 64)})
	if status != 502 || !strings.Contains(string(body), "setup_unconfirmed") {
		t.Fatalf("lost begin %d %s", status, body)
	}
	// A plain begin retries to convergence: 200 with the observation
	// recorded, exactly one renewal consumed, and no further renewal
	// needed or admitted.
	f.fail.Store(false)
	status, body = f.call(1, "POST", "/api/applications/setups/"+expired.ID+"/actions", map[string]any{"action": "begin"})
	if status != 200 {
		t.Fatalf("retry begin %d %s", status, body)
	}
	stored, err := f.store.ApplicationSetup(ctx, expired.ID)
	if err != nil || stored.Observation == nil || stored.Observation.Container != strings.Repeat("7", 64) {
		t.Fatalf("retry observation not recorded: %+v %v", stored, err)
	}
	if stored.ExpiresAt != rearmed.Setup.ExpiresAt {
		t.Fatalf("retry consumed another renewal: %d vs %d", stored.ExpiresAt, rearmed.Setup.ExpiresAt)
	}
	if status, _ := f.call(1, "POST", "/api/applications/setups/"+expired.ID+"/actions", map[string]any{"action": "renew", "confirm": true}); status != 409 {
		t.Fatalf("re-armed setup renewed again: %d", status)
	}
}
