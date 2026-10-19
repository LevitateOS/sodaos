package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

// Native repository facts are rechecked for each mutation. A page that changes
// actor, instance, or generation during callback I/O cannot dispatch to the host.
func TestMutationAdmissionAfterNativeCallbackIO(t *testing.T) {
	for _, operation := range []string{"join", "start", "stop", "apply"} {
		for _, change := range []string{"unchanged", "generation", "actor", "instance", "callback-denied", "cancel"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				s, _ := managementWebFixture(t)
				projectID := webTerminalProject
				path, body := "/lifecycle", `{"action":"`+operation+`"}`
				if operation == "stop" {
					body = `{"action":"stop","confirm_stop":true}`
				}
				if operation == "join" {
					projectID = "pabcdef0123456789abcdef01"
					if err := s.Store.CreateProject(t.Context(), store.Project{ID: projectID, RepositoryID: 8, OwnerID: 1}); err != nil {
						t.Fatal(err)
					}
					if err := s.Store.MarkReady(t.Context(), projectID, "10.89.0.3"); err != nil {
						t.Fatal(err)
					}
					path, body = "/join", `{}`
				} else if operation == "apply" {
					path, body = "/access-keys", `{"revision":"`+strings.Repeat("a", 64)+`","saved_fingerprints":[],"confirm_empty":true}`
				}
				r := apiTestRequest("POST", "/api/environments/"+projectID+path, body, "alice")
				ctx, cancel := context.WithCancel(r.Context())
				defer cancel()
				r = r.WithContext(ctx)
				peerCtx, endPeer := context.WithCancel(t.Context())
				defer endPeer()
				s.API.TerminalPeers = map[*http.Request]*api.TerminalPeer{r: {ContextID: nativeProductGeneration, Project: projectID, Cancel: endPeer}}
				nativeCalls := 0
				s.Host.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
					nativeCalls++
					var in map[string]any
					if err := json.NewDecoder(req.Body).Decode(&in); err != nil || in["project"] != projectID {
						t.Error("untrusted native target", err)
					}
					w := httptest.NewRecorder()
					switch operation {
					case "join":
						if req.URL.Path != "/account" || in["identity"] != float64(1) || in["login"] != "alice" {
							t.Error("Join lost native actor")
						}
						_, _ = w.WriteString(`{"ok":true}`)
					case "apply":
						if req.URL.Path != "/access-keys" || in["identity"] != float64(1) || in["login"] != "original-alice" || in["apply"] != true {
							t.Error("Apply remapped membership")
						}
						_, _ = w.WriteString(`{"revision":"` + strings.Repeat("a", 64) + `","keys":[]}`)
					default:
						if req.URL.Path == "/prepare-hold" {
							if operation != "stop" || in["hold"] != true {
								t.Error("hold dispatched outside coordinated stop")
							}
							_, _ = w.WriteString(`{"active":true,"revision":1}`)
							break
						}
						if req.URL.Path != "/lifecycle" || in["action"] != operation {
							t.Error("lifecycle action changed")
						}
						if operation == "stop" && (!s.API.TerminalStopping[projectID] || peerCtx.Err() == nil) {
							t.Error("admitted Stop lost terminal coordination")
						}
						_ = json.NewEncoder(w).Encode(project.LifecycleState{Environment: project.Environment{ID: projectID, Running: operation == "start"}, BootEnabled: operation == "start"})
					}
					return w.Result(), nil
				})}
				callbacks := 0
				callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
					if in.Operation != extensions.OperationRepository {
						t.Error("unexpected native callback", in.Operation)
						return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
					}
					callbacks++
					if change == "callback-denied" {
						return extensions.CallbackResponse{ErrorCode: "not_found"}
					}
					if callbacks == 1 {
						switch change {
						case "generation":
							r.Header.Set(extensions.ContextHeader, strings.Replace(r.Header.Get(extensions.ContextHeader), nativeProductGeneration, "expired", 1))
						case "actor":
							r.Header.Set(extensions.ContextHeader, strings.Replace(r.Header.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
						case "instance":
							r.Header.Set(extensions.ContextHeader, strings.Replace(r.Header.Get(extensions.ContextHeader), "native-test-instance", "other-instance", 1))
						case "cancel":
							cancel()
						}
					}
					return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}}
				}
				w := httptest.NewRecorder()
				nativeAPIServeWithCallback(t, s, w, r, callback)
				if change == "unchanged" {
					// Coordinated stop dispatches the maintenance hold before
					// the lifecycle call; every other mutation keeps one call.
					want := 1
					if operation == "stop" {
						want = 2
					}
					if w.Code != 200 || nativeCalls != want {
						t.Errorf("authorized mutation: status=%d native=%d body=%s", w.Code, nativeCalls, w.Body.String())
					}
				} else if w.Code < 400 || nativeCalls != 0 {
					t.Errorf("stale authority dispatched: status=%d native=%d body=%s", w.Code, nativeCalls, w.Body.String())
				}
				shouldEnd := operation == "stop" && change == "unchanged"
				if (peerCtx.Err() != nil) != shouldEnd || s.API.TerminalStopping[projectID] {
					t.Error("denied mutation disturbed terminals or leaked Stop admission")
				}
				if operation == "join" {
					login, err := s.Store.MemberLogin(t.Context(), projectID, 1)
					if change == "unchanged" {
						if err != nil || login != "alice" {
							t.Error("confirmed Join did not record native login", err)
						}
					} else if !errors.Is(err, store.ErrNotFound) {
						t.Error("denied Join recorded membership", err)
					}
				}
			})
		}
	}
}

func TestStartAdmissionRefusesOverlappingStopAndKeepsPeer(t *testing.T) {
	s, _ := managementWebFixture(t)
	peerCtx, cancelPeer := context.WithCancel(t.Context())
	defer cancelPeer()
	peerRequest := apiTestRequest("POST", "/unused", "", "alice")
	s.API.TerminalPeers = map[*http.Request]*api.TerminalPeer{
		peerRequest: {Project: webTerminalProject, Cancel: cancelPeer},
	}
	defer func() { delete(s.API.TerminalPeers, peerRequest) }()
	entered := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	var hostCalls atomic.Int32
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		hostCalls.Add(1)
		if r.URL.Path != "/lifecycle" {
			return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected helper call"))}, nil
		}
		var body struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Action != "start" {
			return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected lifecycle action"))}, nil
		}
		close(entered)
		<-release
		response := `{"environment":{"id":"` + webTerminalProject + `","running":true},"boot_enabled":true}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	startResponse := httptest.NewRecorder()
	startDone := make(chan struct{})
	go func() {
		nativeAPIServe(t, s, startResponse, apiTestRequest("POST", "/api/environments/"+webTerminalProject+"/lifecycle", `{"action":"start"}`, "alice"))
		close(startDone)
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not reach the host while holding lifecycle admission")
	}
	stop := httptest.NewRecorder()
	nativeAPIServe(t, s, stop, apiTestRequest("POST", "/api/environments/"+webTerminalProject+"/lifecycle", `{"action":"stop","confirm_stop":true}`, "alice"))
	if stop.Code != http.StatusConflict || !strings.Contains(stop.Body.String(), "lifecycle_pending") {
		t.Fatalf("overlapping Stop was not reported as lifecycle busy: %d %s", stop.Code, stop.Body.String())
	}
	if peerCtx.Err() != nil {
		t.Fatal("Start cancelled an existing human terminal peer")
	}
	close(release)
	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not finish after host response")
	}
	if startResponse.Code != http.StatusOK || hostCalls.Load() != 1 {
		t.Fatalf("Start failed or overlapping Stop reached the host: status=%d host calls=%d body=%s", startResponse.Code, hostCalls.Load(), startResponse.Body.String())
	}
}

type mutationAdmissionBody struct {
	io.Reader
	beforeRead func()
}

func (b *mutationAdmissionBody) Read(p []byte) (int, error) {
	if b.beforeRead != nil {
		f := b.beforeRead
		b.beforeRead = nil
		f()
	}
	return b.Reader.Read(p)
}

func TestMutationAdmissionAfterDecode(t *testing.T) {
	for _, operation := range []string{"start", "stop", "apply"} {
		for _, change := range []string{"generation", "actor", "instance", "cancel"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				s, calls := managementWebFixture(t)
				s.Config.OperatorID = 1
				path, body := "/lifecycle", `{"action":"`+operation+`"}`
				if operation == "stop" {
					body = `{"action":"stop","confirm_stop":true}`
				} else if operation == "apply" {
					path, body = "/access-keys", `{"revision":"`+strings.Repeat("a", 64)+`","saved_fingerprints":[],"confirm_empty":true}`
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := apiTestRequest("POST", "/api/environments/"+webTerminalProject+path, body, "alice").WithContext(ctx)
				r.Body = io.NopCloser(&mutationAdmissionBody{Reader: strings.NewReader(body), beforeRead: func() {
					context := r.Header.Get(extensions.ContextHeader)
					switch change {
					case "generation":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, nativeProductGeneration, "expired", 1))
					case "actor":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, `"id":"1"`, `"id":"2"`, 1))
					case "instance":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, "native-test-instance", "other-instance", 1))
					case "cancel":
						cancel()
					}
				}})
				w := httptest.NewRecorder()
				nativeAPIServe(t, s, w, r)
				if w.Code < 400 || len(*calls) != 0 {
					t.Fatal("post-decode stale native authority reached helper", w.Code, *calls, w.Body.String())
				}
			})
		}
	}
}
