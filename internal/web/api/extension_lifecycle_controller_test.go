package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
)

const (
	controllerTestToken = "test-controller-token"
	controllerTestSHA   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type controllerTestManager struct {
	mu      sync.Mutex
	calls   int
	status  int
	desired bool
}

func (m *controllerTestManager) served() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *controllerTestManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if r.Header.Get(extensions.LifecycleAdmissionHeader) != controllerTestToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if m.status != 0 {
		http.Error(w, "refused", m.status)
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

type controllerTestService struct {
	mu       sync.Mutex
	hits     int
	token    string
	status   int
	raw      []byte
	bodies   [][]byte
	redirect string
}

func (s *controllerTestService) served() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

func (s *controllerTestService) lastBody() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.bodies) == 0 {
		return nil
	}
	return s.bodies[len(s.bodies)-1]
}

func (s *controllerTestService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.hits++
	s.mu.Unlock()
	if r.URL.Path != extensionLifecycleRoute || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if s.redirect != "" {
		http.Redirect(w, r, s.redirect, http.StatusFound)
		return
	}
	if r.Header.Get(extensions.LifecycleAdmissionHeader) != s.token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.bodies = append(s.bodies, body)
	s.mu.Unlock()
	if s.status != 0 {
		if s.raw != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(s.status)
			_, _ = w.Write(s.raw)
			return
		}
		http.Error(w, "refused", s.status)
		return
	}
	var call extensions.LifecycleCall
	call, err = extensions.DecodeLifecycleCall(body)
	if err != nil {
		http.Error(w, "malformed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(extensions.LifecycleObservation{
		LifecycleCall: call,
		State:         "stopped",
		Detail:        "controller test observation",
	})
}

func startControllerTestSockets(t *testing.T, manager *controllerTestManager, service *controllerTestService) (managerSocket, serviceSocket string) {
	t.Helper()
	dir, err := os.MkdirTemp(os.TempDir(), "t01ctl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	managerSocket = dir + "/manager.sock"
	managerListener, err := net.Listen("unix", managerSocket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = managerListener.Close() })
	go func() { _ = http.Serve(managerListener, manager) }()
	serviceSocket = dir + "/soda.sock"
	serviceListener, err := net.Listen("unix", serviceSocket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serviceListener.Close() })
	go func() { _ = http.Serve(serviceListener, service) }()
	t.Setenv(extensions.CallbackEnv, managerSocket)
	t.Setenv(extensions.ServiceCallbackEnv, "")
	t.Setenv(extensions.ServiceCallbackPeerEnv, "")
	return managerSocket, serviceSocket
}

func controllerTestCall() extensions.LifecycleCall {
	return extensions.LifecycleCall{
		ControllerID:         "soda",
		ControllerInstanceID: "dddddddddddddddddddddddddddddddd",
		PackageID:            "tailnet",
		PackageVersion:       "1.0.0",
		PackageSHA256:        controllerTestSHA,
		InstallationID:       "550e8400-e29b-41d4-a716-446655440000",
		Generation:           "7",
		Operation:            extensions.LifecycleObserve,
		DeadlineUnix:         time.Now().Add(5 * time.Minute).Unix(),
	}
}

func controllerTestAuthority(t *testing.T, token string) extensions.LifecycleAuthority {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, "http://extension-host"+extensions.LifecyclePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		r.Header.Set(extensions.LifecycleAdmissionHeader, token)
	}
	authority, err := extensions.LifecycleContext(r)
	if err != nil {
		t.Fatal(err)
	}
	return authority
}

func TestLifecycleControllerForwardsVerifiedCall(t *testing.T) {
	manager := &controllerTestManager{desired: true}
	service := &controllerTestService{token: controllerTestToken}
	_, serviceSocket := startControllerTestSockets(t, manager, service)
	handler := LifecycleController(serviceSocket, uint32(os.Geteuid()))
	call := controllerTestCall()
	observation, err := handler(context.Background(), controllerTestAuthority(t, controllerTestToken), call)
	if err != nil {
		t.Fatalf("controller failed: %v", err)
	}
	if !observation.Echoes(call) {
		t.Fatalf("observation does not echo its call: %+v", observation)
	}
	if manager.served() != 1 || service.served() != 1 {
		t.Fatalf("manager calls = %d service hits = %d, want 1 and 1", manager.served(), service.served())
	}
	forwarded, err := extensions.DecodeLifecycleCall(service.lastBody())
	if err != nil {
		t.Fatalf("forwarded body is not a strict call: %v", err)
	}
	if forwarded != call {
		t.Fatalf("forwarded call diverged: %+v", forwarded)
	}
}

func TestLifecycleControllerRefusalCausesZeroForwards(t *testing.T) {
	call := controllerTestCall()
	wrongPackage := call
	wrongPackage.PackageID = "other"
	expired := call
	expired.DeadlineUnix = time.Now().Add(-time.Minute).Unix()

	cases := []struct {
		name         string
		mutate       func(*extensions.LifecycleCall)
		manager      *controllerTestManager
		token        string
		service      *controllerTestService
		wantForwards int
	}{
		{"manager refuses", nil, &controllerTestManager{status: http.StatusForbidden}, controllerTestToken, &controllerTestService{token: controllerTestToken}, 0},
		{"wrong package", func(c *extensions.LifecycleCall) { c.PackageID = "other" }, &controllerTestManager{desired: true}, controllerTestToken, &controllerTestService{token: controllerTestToken}, 0},
		{"wrong controller", func(c *extensions.LifecycleCall) { c.ControllerID = "other" }, &controllerTestManager{desired: true}, controllerTestToken, &controllerTestService{token: controllerTestToken}, 0},
		{"expired deadline", func(c *extensions.LifecycleCall) { c.DeadlineUnix = expired.DeadlineUnix }, &controllerTestManager{desired: true}, controllerTestToken, &controllerTestService{token: controllerTestToken}, 0},
		{"service refuses", nil, &controllerTestManager{desired: true}, controllerTestToken, &controllerTestService{token: controllerTestToken, status: http.StatusForbidden}, 1},
		{"service echo mismatch", nil, &controllerTestManager{desired: true}, controllerTestToken, func() *controllerTestService {
			other := wrongPackage
			raw, _ := json.Marshal(extensions.LifecycleObservation{LifecycleCall: other, State: "stopped", Detail: "mismatch"})
			return &controllerTestService{token: controllerTestToken, status: http.StatusOK, raw: raw}
		}(), 1},
		{"service redirect refused", nil, &controllerTestManager{desired: true}, controllerTestToken, &controllerTestService{token: controllerTestToken, redirect: "/elsewhere"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, serviceSocket := startControllerTestSockets(t, tc.manager, tc.service)
			handler := LifecycleController(serviceSocket, uint32(os.Geteuid()))
			mutated := call
			if tc.mutate != nil {
				tc.mutate(&mutated)
			}
			if _, err := handler(context.Background(), controllerTestAuthority(t, tc.token), mutated); err == nil {
				t.Fatal("controller admitted a refused call")
			}
			if tc.service.served() != tc.wantForwards {
				t.Fatalf("service hits = %d, want %d", tc.service.served(), tc.wantForwards)
			}
		})
	}
}
