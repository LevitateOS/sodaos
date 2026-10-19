package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// scriptedBackgroundServer speaks the background dispatcher paths with a
// rotating admission: each bootstrap revokes the previous one like the
// native host, and unknown admissions get 401.
type scriptedBackgroundServer struct {
	mu               sync.Mutex
	admissions       []string
	revision         int64
	ops              map[string]extensions.OperationRecord
	conflict         map[string]bool
	submits          int
	bootstrapStarted chan struct{}
	bootstrapRelease chan struct{}
	beforeRequest    func(context.Context, string, string)
}

type backgroundRequest struct {
	operationID string
	admission   string
	attempt     int
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
			if f.bootstrapStarted != nil {
				select {
				case f.bootstrapStarted <- struct{}{}:
				default:
				}
				<-f.bootstrapRelease
			}
			f.mu.Lock()
			token := strings.Repeat(string(rune('a'+len(f.admissions))), 43)
			f.admissions = append(f.admissions, token)
			f.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{
				"admission": token, "installation_id": "install-1",
			})
		case extensions.BackgroundRevisionPath,
			extensions.BackgroundSubmitPath, extensions.BackgroundGetPath, extensions.BackgroundCancelPath:
			var operationID string
			if r.URL.Path == extensions.BackgroundGetPath && f.beforeRequest != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "invalid background request", http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
				var request struct {
					OperationID string `json:"operation_id"`
				}
				if json.Unmarshal(body, &request) == nil {
					operationID = request.OperationID
				}
			}
			admission := r.Header.Get(extensions.AdmissionHeader)
			if operationID != "" {
				f.beforeRequest(r.Context(), operationID, admission)
			}
			f.mu.Lock()
			current := f.admissions[len(f.admissions)-1]
			f.mu.Unlock()
			if admission != current {
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
	dir, err := os.MkdirTemp("", "soda-forgejo-*")
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

func TestServiceBackgroundRefusesRedirects(t *testing.T) {
	for _, redirectedPath := range []string{"bootstrap", "revision"} {
		t.Run(redirectedPath, func(t *testing.T) {
			const admission = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			var mu sync.Mutex
			counts := map[string]int{}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				counts[r.URL.Path]++
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case extensions.BackgroundBootstrapPath:
					if redirectedPath == "bootstrap" {
						http.Redirect(w, r, "/redirect-target", http.StatusFound)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
				case extensions.BackgroundRevisionPath:
					if redirectedPath == "revision" {
						http.Redirect(w, r, "/redirect-target", http.StatusFound)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"revision": 12, "idle": true})
				case "/redirect-target":
					if redirectedPath == "bootstrap" {
						_ = json.NewEncoder(w).Encode(map[string]string{"admission": admission, "installation_id": "install-1"})
					} else {
						_ = json.NewEncoder(w).Encode(map[string]any{"revision": 12, "idle": true})
					}
				default:
					http.NotFound(w, r)
				}
			})
			socket := shortSocketPath(t, "redirect.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			server := &http.Server{Handler: handler}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })

			background := NewServiceBackground(socket, uint32(os.Getuid()), "")
			if _, err = background.ReadNativeRevision(context.Background()); err == nil {
				t.Fatal("background client accepted a redirected response")
			}
			mu.Lock()
			defer mu.Unlock()
			if counts["/redirect-target"] != 0 {
				t.Fatalf("redirect target received %d requests", counts["/redirect-target"])
			}
			if redirectedPath == "bootstrap" && counts[extensions.BackgroundRevisionPath] != 0 {
				t.Fatalf("revision path received a request after bootstrap redirect: %d", counts[extensions.BackgroundRevisionPath])
			}
		})
	}
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

func TestServiceBackgroundBootstrapWaiterContextDoesNotCancelSharedBootstrap(t *testing.T) {
	fake := &scriptedBackgroundServer{
		admissions: []string{}, revision: 12, ops: map[string]extensions.OperationRecord{},
		bootstrapStarted: make(chan struct{}, 1), bootstrapRelease: make(chan struct{}),
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	first := make(chan error, 1)
	go func() {
		_, err := background.ReadNativeRevision(context.Background())
		first <- err
	}()
	<-fake.bootstrapStarted

	waiterCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	waiter := make(chan error, 1)
	go func() {
		_, err := background.ReadNativeRevision(waiterCtx)
		waiter <- err
	}()
	var waiterErr error
	select {
	case waiterErr = <-waiter:
	case <-time.After(time.Second):
		close(fake.bootstrapRelease)
		t.Fatal("waiting caller ignored its deadline")
	}
	if !errors.Is(waiterErr, context.DeadlineExceeded) {
		t.Fatalf("waiting caller did not honor its deadline: %v", waiterErr)
	}
	close(fake.bootstrapRelease)
	if err := <-first; err != nil {
		t.Fatalf("shared bootstrap was cancelled by waiter: %v", err)
	}
	if _, err := background.ReadNativeRevision(context.Background()); err != nil {
		t.Fatalf("shared admission was not published: %v", err)
	}
	fake.mu.Lock()
	bootstraps := len(fake.admissions)
	fake.mu.Unlock()
	if bootstraps != 1 {
		t.Fatalf("bootstrap count: %d", bootstraps)
	}
}

func TestServiceBackgroundStaleRejectionWaitsForForcedBootstrap(t *testing.T) {
	rejected, cached := strings.Repeat("a", 43), strings.Repeat("b", 43)
	fake := &scriptedBackgroundServer{
		admissions: []string{rejected, cached}, revision: 12,
		ops:              map[string]extensions.OperationRecord{},
		bootstrapStarted: make(chan struct{}, 1), bootstrapRelease: make(chan struct{}),
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	background.mu.Lock()
	background.admission = cached
	background.pinned = "install-1"
	background.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	forceResult := make(chan struct {
		env []string
		err error
	}, 1)
	joined := make(chan struct{})
	waiterDone := make(chan struct{})
	var cancelWaiter context.CancelFunc
	waiterStarted := false
	var releaseOnce sync.Once
	releaseBootstrap := func() { releaseOnce.Do(func() { close(fake.bootstrapRelease) }) }
	t.Cleanup(func() {
		releaseBootstrap()
		cancel()
		if cancelWaiter != nil {
			cancelWaiter()
		}
		select {
		case <-joined:
		case <-time.After(time.Second):
			t.Error("forced bootstrap caller did not stop")
		}
		if waiterStarted {
			select {
			case <-waiterDone:
			case <-time.After(time.Second):
				t.Error("stale waiter did not stop")
			}
		}
	})
	go func() {
		defer close(joined)
		env, err := background.PublishPushEnv(ctx, "soda-test-stale-force", true)
		forceResult <- struct {
			env []string
			err error
		}{env, err}
	}()
	select {
	case <-fake.bootstrapStarted:
	case <-ctx.Done():
		t.Fatalf("forced bootstrap did not start: %v", ctx.Err())
	}

	waiterCtx, cancelStaleWaiter := context.WithTimeout(context.Background(), 50*time.Millisecond)
	cancelWaiter = cancelStaleWaiter
	waiterResult := make(chan struct {
		admission string
		err       error
	}, 1)
	waiterStarted = true
	go func() {
		defer close(waiterDone)
		admission, err := background.bootstrap(waiterCtx, true, rejected)
		waiterResult <- struct {
			admission string
			err       error
		}{admission, err}
	}()
	var staleResult struct {
		admission string
		err       error
	}
	select {
	case staleResult = <-waiterResult:
	case <-time.After(time.Second):
		t.Fatal("stale rejection ignored its deadline while waiting for forced bootstrap")
	}
	select {
	case <-forceResult:
		t.Fatal("forced bootstrap completed before its gate was released")
	default:
	}
	releaseBootstrap()
	var forced struct {
		env []string
		err error
	}
	select {
	case forced = <-forceResult:
	case <-ctx.Done():
		t.Fatalf("forced bootstrap did not finish: %v", ctx.Err())
	}
	if forced.err != nil {
		t.Fatalf("explicit forced rebind: %v", forced.err)
	}
	var envAdmission string
	for _, entry := range forced.env {
		if value, ok := strings.CutPrefix(entry, "GIT_CONFIG_VALUE_1="); ok {
			envAdmission, _ = strings.CutPrefix(value, extensions.AdmissionHeader+": ")
		}
	}
	if envAdmission == "" || envAdmission != fake.current() {
		t.Fatalf("forced rebind environment admission %q, current %q", envAdmission, fake.current())
	}
	if !errors.Is(staleResult.err, context.DeadlineExceeded) || staleResult.admission != "" {
		t.Fatalf("stale waiter returned admission %q and error %v; want its deadline", staleResult.admission, staleResult.err)
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

func TestServiceBackgroundDelayedUnauthorizedReusesReplacementAdmission(t *testing.T) {
	firstID, secondID := "soda-test-stale-401-first", "soda-test-stale-401-second"
	firstInitialRelease, secondInitialRelease := make(chan struct{}), make(chan struct{})
	firstRetryRelease := make(chan struct{})
	firstGate, secondGate, retryGate := new(sync.Once), new(sync.Once), new(sync.Once)
	requestSeen := make(chan backgroundRequest, 8)
	retryStarted := make(chan struct{}, 1)
	var countMu sync.Mutex
	requestCounts := map[string]int{}
	fake := &scriptedBackgroundServer{
		admissions: []string{},
		revision:   12,
		ops:        map[string]extensions.OperationRecord{},
		beforeRequest: func(ctx context.Context, operationID, admission string) {
			countMu.Lock()
			requestCounts[operationID]++
			attempt := requestCounts[operationID]
			countMu.Unlock()
			select {
			case requestSeen <- backgroundRequest{operationID: operationID, admission: admission, attempt: attempt}:
			case <-ctx.Done():
				return
			}
			switch {
			case operationID == firstID && attempt == 1:
				select {
				case <-firstInitialRelease:
				case <-ctx.Done():
				}
			case operationID == secondID && attempt == 1:
				select {
				case <-secondInitialRelease:
				case <-ctx.Done():
				}
			case operationID == firstID && attempt == 2:
				select {
				case retryStarted <- struct{}{}:
				case <-ctx.Done():
					return
				}
				select {
				case <-firstRetryRelease:
				case <-ctx.Done():
				}
			}
		},
	}
	socket := serveScriptedBackground(t, fake)
	background := NewServiceBackground(socket, uint32(os.Getuid()), "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if _, err := background.ReadNativeRevision(ctx); err != nil {
		t.Fatalf("initial admission: %v", err)
	}
	results := make(chan struct {
		id  string
		err error
	}, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		_, err := background.GetOperation(ctx, firstID)
		results <- struct {
			id  string
			err error
		}{firstID, err}
	}()
	go func() {
		defer workers.Done()
		_, err := background.GetOperation(ctx, secondID)
		results <- struct {
			id  string
			err error
		}{secondID, err}
	}()
	joined := make(chan struct{})
	go func() {
		workers.Wait()
		close(joined)
	}()
	releaseFirstInitial := func() { firstGate.Do(func() { close(firstInitialRelease) }) }
	releaseSecondInitial := func() { secondGate.Do(func() { close(secondInitialRelease) }) }
	releaseFirstRetry := func() { retryGate.Do(func() { close(firstRetryRelease) }) }
	t.Cleanup(func() {
		releaseFirstInitial()
		releaseSecondInitial()
		releaseFirstRetry()
		cancel()
		select {
		case <-joined:
		case <-time.After(time.Second):
			t.Error("background request workers did not stop")
		}
	})

	waitRequest := func(wantID string, wantAttempt int) backgroundRequest {
		t.Helper()
		for {
			select {
			case request := <-requestSeen:
				if request.operationID == wantID && request.attempt == wantAttempt {
					return request
				}
			case <-ctx.Done():
				t.Fatalf("waiting for %s attempt %d: %v", wantID, wantAttempt, ctx.Err())
				return backgroundRequest{}
			}
		}
	}
	waitResult := func(wantID string) error {
		t.Helper()
		for {
			select {
			case result := <-results:
				if result.id == wantID {
					return result.err
				}
				t.Fatalf("unexpected request completed before %s: %s (%v)", wantID, result.id, result.err)
				return errors.New("unexpected request completion")
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", wantID, ctx.Err())
				return ctx.Err()
			}
		}
	}
	initial := make(map[string]backgroundRequest, 2)
	for len(initial) < 2 {
		select {
		case request := <-requestSeen:
			if request.attempt == 1 {
				initial[request.operationID] = request
			}
		case <-ctx.Done():
			t.Fatalf("waiting for both initial requests: %v", ctx.Err())
		}
	}
	first, second := initial[firstID], initial[secondID]
	if first.admission != second.admission {
		t.Fatalf("initial admissions differ: %q and %q", first.admission, second.admission)
	}
	// Model the host revoking A outside these requests, as in
	// TestServiceBackgroundRebindsAfterRevocation.
	fake.mu.Lock()
	fake.admissions = append(fake.admissions, strings.Repeat("z", 43))
	fake.mu.Unlock()
	releaseFirstInitial()
	select {
	case <-retryStarted:
	case <-ctx.Done():
		t.Fatalf("first replacement retry did not reach the server: %v", ctx.Err())
	}
	firstRetry := waitRequest(firstID, 2)
	releaseSecondInitial()
	secondRetry := waitRequest(secondID, 2)
	secondErr := waitResult(secondID)
	releaseFirstRetry()
	firstErr := waitResult(firstID)
	if firstErr != nil || secondErr != nil {
		t.Fatalf("calls failed: first=%v second=%v", firstErr, secondErr)
	}
	if secondRetry.admission != firstRetry.admission {
		t.Fatalf("delayed stale response replaced the shared admission: %q then %q", firstRetry.admission, secondRetry.admission)
	}

	fake.mu.Lock()
	// One publication is the explicit host-side revocation marker above.
	bootstraps := len(fake.admissions) - 1
	fake.mu.Unlock()
	countMu.Lock()
	counts := map[string]int{firstID: requestCounts[firstID], secondID: requestCounts[secondID]}
	countMu.Unlock()
	if bootstraps != 2 {
		t.Fatalf("bootstraps: got %d, want initial admission plus one replacement", bootstraps)
	}
	if counts[firstID] != 2 || counts[secondID] != 2 {
		t.Fatalf("request counts: first=%d second=%d, want two each", counts[firstID], counts[secondID])
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
	var status *forgejopublish.StatusError
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
