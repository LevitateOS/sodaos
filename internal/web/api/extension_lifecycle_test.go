package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/host"
)

const (
	lifecycleTestToken    = "test-lifecycle-token"
	lifecycleTestInstance = "dddddddddddddddddddddddddddddddd"
	lifecycleTestInstall  = "550e8400-e29b-41d4-a716-446655440000"
	lifecycleTestSHA      = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// lifecycleTestManager is a fake Fountain authorize callback. It counts
// hits and answers strictly from its configured behavior.
type lifecycleTestManager struct {
	mu        sync.Mutex
	calls     int
	token     string
	desired   bool
	status    int
	raw       []byte
	failAfter int
}

func (m *lifecycleTestManager) served() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *lifecycleTestManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.calls++
	n := m.calls
	m.mu.Unlock()
	if r.URL.Path != extensions.LifecycleAuthorizePath || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get(extensions.LifecycleAdmissionHeader) != m.token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if m.failAfter > 0 && n > m.failAfter {
		http.Error(w, "revoked", http.StatusForbidden)
		return
	}
	if m.status != 0 {
		if m.raw != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(m.status)
			_, _ = w.Write(m.raw)
			return
		}
		http.Error(w, "manager refused", m.status)
		return
	}
	var call extensions.LifecycleCall
	if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
		http.Error(w, "malformed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(extensions.LifecycleVerification{Call: call, DesiredEnabled: m.desired})
}

// lifecycleTestRust fakes the Rust host socket: it counts prepare/lifecycle
// hits and returns canned wire bytes.
type lifecycleTestRust struct {
	mu             sync.Mutex
	prepareCalls   int
	lifecycleCalls int
	prepareStatus  int
	prepareBody    []byte
	status         int
	body           []byte
}

func (f *lifecycleTestRust) counts() (prepare, lifecycle int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prepareCalls, f.lifecycleCalls
}

func (f *lifecycleTestRust) RoundTrip(r *http.Request) (*http.Response, error) {
	_, _ = io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	status, body := f.status, f.body
	if strings.HasSuffix(r.URL.Path, "/prepare") {
		f.prepareCalls++
		status, body = f.prepareStatus, f.prepareBody
	} else {
		f.lifecycleCalls++
	}
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func lifecycleTestCall(t *testing.T, operation string, deadline time.Time) (extensions.LifecycleCall, []byte) {
	t.Helper()
	call := extensions.LifecycleCall{
		ControllerID:         "soda",
		ControllerInstanceID: lifecycleTestInstance,
		PackageID:            "tailnet",
		PackageVersion:       "1.0.0",
		PackageSHA256:        lifecycleTestSHA,
		InstallationID:       lifecycleTestInstall,
		Generation:           "7",
		Operation:            operation,
		DeadlineUnix:         deadline.Unix(),
	}
	body, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	return call, body
}

func lifecycleTestObservation(call extensions.LifecycleCall, state string, effect *string) []byte {
	observation := extensions.LifecycleObservation{
		LifecycleCall:    call,
		State:            state,
		EffectGeneration: effect,
		AdmissionReady:   false,
		Detail:           "test observation",
	}
	body, err := json.Marshal(observation)
	if err != nil {
		panic(err)
	}
	return body
}

// lifecycleTestSocketDir returns a short socket directory: unix socket
// paths are capped near 108 bytes, which test-name-nested TempDirs exceed.
func lifecycleTestSocketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(os.TempDir(), "t01sock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func startLifecycleTestManager(t *testing.T, manager *lifecycleTestManager) {
	t.Helper()
	path := lifecycleTestSocketDir(t) + "/manager.sock"
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() { _ = http.Serve(listener, manager) }()
	t.Setenv(extensions.CallbackEnv, path)
	t.Setenv(extensions.ServiceCallbackEnv, "")
	t.Setenv(extensions.ServiceCallbackPeerEnv, "")
}

type lifecycleTestFixture struct {
	api     *API
	manager *lifecycleTestManager
	rust    *lifecycleTestRust
}

func newLifecycleTestFixture(t *testing.T, manager *lifecycleTestManager, rust *lifecycleTestRust) *lifecycleTestFixture {
	t.Helper()
	startLifecycleTestManager(t, manager)
	if rust.status == 0 && rust.body == nil {
		rust.status = http.StatusOK
	}
	return &lifecycleTestFixture{
		api:     &API{Host: &host.Client{HTTP: &http.Client{Transport: rust}}},
		manager: manager,
		rust:    rust,
	}
}

func (f *lifecycleTestFixture) serve(t *testing.T, method, target string, body []byte, token string) *httptest.ResponseRecorder {
	return f.serveHeaders(t, method, target, body, token, nil)
}

func (f *lifecycleTestFixture) serveHeaders(t *testing.T, method, target string, body []byte, token string, extra map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, target, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set(extensions.LifecycleAdmissionHeader, token)
	}
	for name, value := range extra {
		r.Header.Set(name, value)
	}
	w := httptest.NewRecorder()
	f.api.ExtensionHandler().ServeHTTP(w, r)
	return w
}

func TestExtensionLifecycleRejectsWithoutEffects(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	_, observeBody := lifecycleTestCall(t, extensions.LifecycleObserve, future)
	_, expiredBody := lifecycleTestCall(t, extensions.LifecycleObserve, time.Now().Add(-time.Minute))
	oversize := append([]byte(`{"controller_id":"soda",`), bytes.Repeat([]byte("x"), extensions.MaximumLifecycleBodyBytes)...)

	cases := []struct {
		name   string
		method string
		target string
		body   []byte
		token  string
		mutate func(*extensions.LifecycleCall) []byte
		want   int
	}{
		{"get rejected", http.MethodGet, extensionLifecycleRoute, observeBody, lifecycleTestToken, nil, http.StatusMethodNotAllowed},
		{"query rejected", http.MethodPost, extensionLifecycleRoute + "?x=1", observeBody, lifecycleTestToken, nil, http.StatusBadRequest},
		{"missing token", http.MethodPost, extensionLifecycleRoute, observeBody, "", nil, http.StatusForbidden},
		{"expired deadline", http.MethodPost, extensionLifecycleRoute, expiredBody, lifecycleTestToken, nil, http.StatusForbidden},
		{"oversize body", http.MethodPost, extensionLifecycleRoute, oversize, lifecycleTestToken, nil, http.StatusBadRequest},
		{"unknown field", http.MethodPost, extensionLifecycleRoute, nil, lifecycleTestToken, func(c *extensions.LifecycleCall) []byte {
			raw, _ := json.Marshal(c)
			return append(raw[:len(raw)-1], []byte(`,"extra":1}`)...)
		}, http.StatusBadRequest},
		{"trailing data", http.MethodPost, extensionLifecycleRoute, nil, lifecycleTestToken, func(c *extensions.LifecycleCall) []byte {
			raw, _ := json.Marshal(c)
			return append(raw, 'x')
		}, http.StatusBadRequest},
		{"wrong controller", http.MethodPost, extensionLifecycleRoute, nil, lifecycleTestToken, func(c *extensions.LifecycleCall) []byte {
			c.ControllerID = "other"
			raw, _ := json.Marshal(c)
			return raw
		}, http.StatusForbidden},
		{"wrong package", http.MethodPost, extensionLifecycleRoute, nil, lifecycleTestToken, func(c *extensions.LifecycleCall) []byte {
			c.PackageID = "other"
			raw, _ := json.Marshal(c)
			return raw
		}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
			call, base := lifecycleTestCall(t, extensions.LifecycleObserve, future)
			body := tc.body
			if tc.mutate != nil {
				body = tc.mutate(&call)
			}
			if body == nil {
				body = base
			}
			rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
			f := newLifecycleTestFixture(t, manager, rust)
			w := f.serve(t, tc.method, tc.target, body, tc.token)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d, body = %q", w.Code, tc.want, w.Body.String())
			}
			if manager.served() != 0 {
				t.Fatalf("authorize calls = %d, want 0", manager.served())
			}
			if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 0 {
				t.Fatalf("host calls = prepare %d lifecycle %d, want zero effects", prepare, lifecycle)
			}
		})
	}
}

func TestExtensionLifecycleActorTokenIsNotAuthority(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	call, body := lifecycleTestCall(t, extensions.LifecycleObserve, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serveHeaders(t, http.MethodPost, extensionLifecycleRoute, body, "", map[string]string{
		extensions.AdmissionHeader: "ordinary-actor-token",
		extensions.ContextHeader:   `{"extension_id":"soda"}`,
		"Origin":                   "https://soda.test",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if manager.served() != 0 {
		t.Fatalf("authorize calls = %d, want 0", manager.served())
	}
	if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 0 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want zero effects", prepare, lifecycle)
	}
}

func TestExtensionLifecycleManagerRefusalCausesZeroEffects(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	for _, status := range []int{http.StatusForbidden, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true, status: status}
			call, body := lifecycleTestCall(t, extensions.LifecycleObserve, future)
			rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
			f := newLifecycleTestFixture(t, manager, rust)
			w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
			if w.Code != status {
				t.Fatalf("status = %d, want %d", w.Code, status)
			}
			if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 0 {
				t.Fatalf("host calls = prepare %d lifecycle %d, want zero effects", prepare, lifecycle)
			}
		})
	}
}

func TestExtensionLifecycleObserveRoundTrip(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	call, body := lifecycleTestCall(t, extensions.LifecycleObserve, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
	observation, err := extensions.DecodeLifecycleObservation(w.Body.Bytes())
	if err != nil {
		t.Fatalf("response is not a strict observation: %v", err)
	}
	if !observation.Echoes(call) {
		t.Fatalf("response does not echo its call: %+v", observation)
	}
	if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 1 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want 0 and 1", prepare, lifecycle)
	}
	if manager.served() != 1 {
		t.Fatalf("authorize calls = %d, want 1", manager.served())
	}
}

func TestExtensionLifecycleActivatePreparesThenRevalidates(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	call, body := lifecycleTestCall(t, extensions.LifecycleActivate, future)
	rust := &lifecycleTestRust{
		prepareStatus: http.StatusOK,
		prepareBody:   lifecycleTestObservation(call, "stopped", nil),
		status:        http.StatusOK,
		body:          lifecycleTestObservation(call, "stopped", nil),
	}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
	if prepare, lifecycle := rust.counts(); prepare != 1 || lifecycle != 1 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want 1 and 1", prepare, lifecycle)
	}
	if manager.served() != 2 {
		t.Fatalf("authorize calls = %d, want verify plus revalidation", manager.served())
	}
}

func TestExtensionLifecycleActivateRequiresDesired(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: false}
	call, body := lifecycleTestCall(t, extensions.LifecycleActivate, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 0 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want zero effects", prepare, lifecycle)
	}
}

func TestExtensionLifecycleDeactivateAdmitsCleanupDesired(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	call, body := lifecycleTestCall(t, extensions.LifecycleDeactivate, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
	if prepare, lifecycle := rust.counts(); prepare != 0 || lifecycle != 1 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want 0 and 1", prepare, lifecycle)
	}
}

func TestExtensionLifecycleFailedPrepareStopsFlow(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	call, body := lifecycleTestCall(t, extensions.LifecycleActivate, future)
	rust := &lifecycleTestRust{
		prepareStatus: http.StatusServiceUnavailable,
		prepareBody:   []byte("unavailable"),
		status:        http.StatusOK,
		body:          lifecycleTestObservation(call, "stopped", nil),
	}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if prepare, lifecycle := rust.counts(); prepare != 1 || lifecycle != 0 {
		t.Fatalf("host calls = prepare %d lifecycle %d, want 1 and 0", prepare, lifecycle)
	}
	if manager.served() != 1 {
		t.Fatalf("authorize calls = %d, want 1 without revalidation", manager.served())
	}
}

func TestExtensionLifecycleLostAuthorityAfterPrepareReportsUnknown(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true, failAfter: 1}
	call, body := lifecycleTestCall(t, extensions.LifecycleActivate, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(call, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if _, lifecycle := rust.counts(); lifecycle != 0 {
		t.Fatalf("lifecycle calls = %d, want 0 after lost authority", lifecycle)
	}
}

func TestExtensionLifecycleRustIdentityChangeIsConflict(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	_, body := lifecycleTestCall(t, extensions.LifecycleObserve, future)
	rust := &lifecycleTestRust{status: http.StatusConflict, body: []byte("identity changed")}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestExtensionLifecycleRustEchoMismatchIsUnconfirmed(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	manager := &lifecycleTestManager{token: lifecycleTestToken, desired: true}
	_, body := lifecycleTestCall(t, extensions.LifecycleObserve, future)
	other, _ := lifecycleTestCall(t, extensions.LifecycleDeactivate, future)
	rust := &lifecycleTestRust{status: http.StatusOK, body: lifecycleTestObservation(other, "stopped", nil)}
	f := newLifecycleTestFixture(t, manager, rust)
	w := f.serve(t, http.MethodPost, extensionLifecycleRoute, body, lifecycleTestToken)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 for mismatched echo", w.Code)
	}
}
