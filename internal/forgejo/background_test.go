package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	extensions "forgejo.org/extension-sdk"

	hostpublish "github.com/levitateos/sodaos/internal/host/publish"
)

// scriptedBackgroundServer speaks the background dispatcher paths with a
// rotating admission: each bootstrap revokes the previous one like the
// native host, and unknown admissions get 401.
type scriptedBackgroundServer struct {
	mu         sync.Mutex
	admissions []string
	revision   int64
	ops        map[string]extensions.OperationRecord
	conflict   map[string]bool
	submits    int
}

func (f *scriptedBackgroundServer) current() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.admissions[len(f.admissions)-1]
}

func (f *scriptedBackgroundServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case extensions.BackgroundBootstrapPath:
			f.mu.Lock()
			token := strings.Repeat(string(rune('a'+len(f.admissions))), 43)
			f.admissions = append(f.admissions, token)
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{
				"admission": token, "installation_id": "install-1",
			})
		case extensions.BackgroundRevisionPath,
			extensions.BackgroundSubmitPath, extensions.BackgroundGetPath, extensions.BackgroundCancelPath:
			if r.Header.Get(extensions.AdmissionHeader) != f.current() {
				http.Error(w, "background admission expired", http.StatusUnauthorized)
				return
			}
			f.serveOp(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func (f *scriptedBackgroundServer) serveOp(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case extensions.BackgroundRevisionPath:
		_ = json.NewEncoder(w).Encode(map[string]any{"revision": f.revision, "idle": true})
	case extensions.BackgroundSubmitPath:
		var in struct {
			extensions.OperationIntent
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background intent", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.submits++
		if in.Token != "test-pat" {
			http.Error(w, "invalid_credential", http.StatusUnauthorized)
			return
		}
		if existing, ok := f.ops[in.OperationID]; ok {
			_ = json.NewEncoder(w).Encode(existing)
			return
		}
		if f.conflict[in.OperationID] {
			http.Error(w, "intent_conflict", http.StatusConflict)
			return
		}
		record := extensions.OperationRecord{
			InstallationID: "install-1", OperationID: in.OperationID, Outcome: "pending",
			ActorID: in.ActorID, RepositoryID: in.RepositoryID, Kind: in.Kind,
			EffectState: "pending", CancellationStatus: "none", CompletionState: "pending",
		}
		f.ops[in.OperationID] = record
		_ = json.NewEncoder(w).Encode(record)
	case extensions.BackgroundGetPath:
		var in struct {
			OperationID string `json:"operation_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if record, ok := f.ops[in.OperationID]; ok {
			_ = json.NewEncoder(w).Encode(extensions.OperationLookup{
				InstallationID: "install-1", OperationID: in.OperationID,
				Status: record.EffectState, Record: &record,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(extensions.OperationLookup{
			InstallationID: "install-1", OperationID: in.OperationID,
			Status: extensions.BackgroundOutcomeNotObserved,
		})
	case extensions.BackgroundCancelPath:
		var in struct {
			OperationID string `json:"operation_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid background request", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		record, ok := f.ops[in.OperationID]
		if !ok {
			record = extensions.OperationRecord{
				InstallationID: "install-1", OperationID: in.OperationID,
				Outcome: "not_committed", EffectState: "not_committed",
				CancellationStatus: "cancelled",
			}
		} else {
			record.CancellationStatus = "cancelled"
			record.EffectState, record.Outcome = "not_committed", "not_committed"
		}
		f.ops[in.OperationID] = record
		_ = json.NewEncoder(w).Encode(record)
	}
}

func TestServiceBackgroundVerifiesPeerBeforeSendingCredential(t *testing.T) {
	fake := &scriptedBackgroundServer{revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	if _, err := background.ReadNativeRevision(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Change the configured UID after successful bootstrap so this exercises
	// the next request's fresh connection, not the bootstrap check.
	background.hostUID++
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	if _, err := background.SubmitOperation(context.Background(), credential, extensions.OperationIntent{OperationID: "test"}); err == nil {
		t.Fatal("request sent to an unverified peer")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.submits != 0 {
		t.Fatal("credential-bearing request reached the wrong peer")
	}
}

func TestServiceBackgroundRejectsForeignNestedRecord(t *testing.T) {
	background := &ServiceBackground{pinned: "installation-a"}
	lookup := extensions.OperationLookup{
		InstallationID: "installation-a", OperationID: "op-1", Status: "committed",
		Record: &extensions.OperationRecord{InstallationID: "installation-b", OperationID: "op-1", Outcome: "committed"},
	}
	if err := checkBackgroundLookup(background, "op-1", lookup); err == nil {
		t.Fatal("foreign installation's nested record adopted")
	}
}

// shortSocketPath returns a unix socket path under a short fixed-parent
// directory. t.TempDir inherits TMPDIR, and a long temp parent plus the
// test-name suffix exceeds the 107-byte unix socket path limit, so fixtures
// that bind unix sockets must not build their path from it.
func shortSocketPath(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "soda-forgejo-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, name)
}

func serveScriptedBackground(t *testing.T, fake *scriptedBackgroundServer) string {
	t.Helper()
	socket := shortSocketPath(t, "background.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server := &http.Server{Handler: fake.handler()}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestServiceBackgroundSharesOneAdmission(t *testing.T) {
	fake := &scriptedBackgroundServer{admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	observation, err := background.ReadNativeRevision(ctx)
	if err != nil || observation.Revision != 12 || !observation.Idle {
		t.Fatalf("revision: %+v %v", observation, err)
	}
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	intent := extensions.OperationIntent{
		OperationID: "soda-test-publish-1", ActorID: "5", RepositoryID: "7",
		Kind: "git.ref.publish", AuthorizationRevision: "rev",
		ExpectedNativeRevision: 12, NotAfter: 9999999999,
	}
	record, err := background.SubmitOperation(ctx, credential, intent)
	if err != nil || record.EffectState != "pending" {
		t.Fatalf("submit: %+v %v", record, err)
	}
	lookup, err := background.GetOperation(ctx, "soda-test-publish-1")
	if err != nil || lookup.Status != "pending" {
		t.Fatalf("lookup: %+v %v", lookup, err)
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 1 {
		t.Fatalf("bootstraps: %d", bootstraps)
	}
	env, err := background.PublishPushEnv(ctx, "soda-test-publish-1", false)
	if err != nil {
		t.Fatalf("push env: %v", err)
	}
	var operation, admission string
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_0="); ok {
			operation, _ = strings.CutPrefix(value, extensions.PublishOperationHeader+": ")
		}
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_1="); ok {
			admission, _ = strings.CutPrefix(value, extensions.AdmissionHeader+": ")
		}
	}
	if operation != "soda-test-publish-1" || admission != fake.current() {
		t.Fatalf("env: %q", env)
	}
}

func TestServiceBackgroundRebindsAfterRevocation(t *testing.T) {
	fake := &scriptedBackgroundServer{admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{}}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	if _, err := background.ReadNativeRevision(ctx); err != nil {
		t.Fatal(err)
	}
	first := fake.current()
	fake.mu.Lock()
	fake.admissions = append(fake.admissions, strings.Repeat("z", 43))
	fake.mu.Unlock()
	observation, err := background.ReadNativeRevision(ctx)
	if err != nil || observation.Revision != 12 {
		t.Fatalf("rebind: %+v %v", observation, err)
	}
	if fake.current() == first {
		t.Fatal("revoked admission still current")
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 3 {
		t.Fatalf("bootstraps: %d", bootstraps)
	}
}

func TestServiceBackgroundMapsDispatchStatuses(t *testing.T) {
	fake := &scriptedBackgroundServer{
		admissions: []string{}, revision: 12,
		ops:      map[string]extensions.OperationRecord{},
		conflict: map[string]bool{"soda-test-clash-1": true},
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx := context.Background()
	credential := extensions.CredentialFile(observationCredential(t, "test-pat"))
	intent := extensions.OperationIntent{
		OperationID: "soda-test-clash-1", ActorID: "5", RepositoryID: "7",
		Kind: "git.ref.publish", AuthorizationRevision: "rev",
		ExpectedNativeRevision: 12, NotAfter: 9999999999,
	}
	_, err := background.SubmitOperation(ctx, credential, intent)
	var status *hostpublish.StatusError
	if !errors.As(err, &status) || status.Status != 409 || status.Body != "intent_conflict" {
		t.Fatalf("conflict: %v", err)
	}
	wrong := extensions.CredentialFile(observationCredential(t, "wrong-pat"))
	intent.OperationID = "soda-test-pat-1"
	_, err = background.SubmitOperation(ctx, wrong, intent)
	if !errors.As(err, &status) || status.Status != 401 {
		t.Fatalf("credential: %v", err)
	}
	if _, err := background.GetOperation(ctx, "bad id"); err == nil {
		t.Fatal("malformed lookup accepted")
	}
	missing := NewServiceBackground(filepath.Join(t.TempDir(), "missing.sock"), uint32(os.Getuid()), "")
	if _, err := missing.ReadNativeRevision(ctx); err == nil {
		t.Fatal("missing socket observed")
	}
	if _, err := missing.PublishPushEnv(ctx, "soda-test-publish-1", false); err == nil {
		t.Fatal("missing socket bound push headers")
	}
	pinned := NewServiceBackground(socket, uint32(os.Getuid()), "install-2")
	if _, err := pinned.ReadNativeRevision(ctx); err == nil {
		t.Fatal("installation mismatch accepted")
	}
}
