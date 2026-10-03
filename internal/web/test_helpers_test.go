package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func apiTestServer(t *testing.T) *Server {
	t.Helper()
	db := postgresFixture(t, nil)
	for i, login := range []string{"alice", "bob"} {
		id := int64(i + 1)
		if err := db.UpsertUser(context.Background(), store.User{ID: id, Login: login, Name: login}); err != nil {
			t.Fatal(err)
		}
	}
	return New(config.Config{ForgejoURL: "https://forgejo.example.test", OperatorID: 1}, db)
}

func apiTestRequest(method, path, body, login string) *http.Request {
	r := httptest.NewRequest(method, config.SodaPath+path, strings.NewReader(body))
	r.Header.Set("X-Test-Native-Actor", login)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://forgejo.example.test")
	return r
}

const nativeProductGeneration = "native-test-generation"

func nativeProductProxy(t *testing.T, s *Server, contribution extensions.Contribution) *httptest.Server {
	return nativeProductProxyForActor(t, s, contribution, "alice")
}

func nativeProductProxyForActor(t *testing.T, s *Server, contribution extensions.Contribution, login string) *httptest.Server {
	return nativeProductProxyForActorWithCallback(t, s, contribution, login, nil)
}

func nativeProductProxyForActorWithCallback(t *testing.T, s *Server, contribution extensions.Contribution, login string, nativeCallback func(extensions.CallbackRequest) extensions.CallbackResponse) *httptest.Server {
	return nativeProductProxyForActorWithCallbacks(t, s, contribution, login, nativeCallback, nil)
}

func nativeProductProxyForActorWithCallbacks(t *testing.T, s *Server, contribution extensions.Contribution, login string, nativeCallback, actorCallback func(extensions.CallbackRequest) extensions.CallbackResponse) *httptest.Server {
	t.Helper()
	root := filepath.Join("..", "..", ".artifacts")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "native-api-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	callbackPath := filepath.Join(dir, "callback.sock")
	callbackListener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "native-test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			response := extensions.CallbackResponse{Actor: &extensions.Actor{ID: nativeTestActorID(login), Username: login}}
			if actorCallback != nil {
				response = actorCallback(in)
			}
			_ = json.NewEncoder(w).Encode(response)
		default:
			if nativeCallback != nil {
				response := nativeCallback(in)
				if in.Operation == extensions.OperationRepository {
					switch response.ErrorCode {
					case "not_found":
						w.WriteHeader(http.StatusNotFound)
						return
					case "forbidden":
						w.WriteHeader(http.StatusForbidden)
						return
					}
				}
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			response := extensions.CallbackResponse{}
			switch in.Operation {
			case extensions.OperationRepository:
				response.Repository = &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}
			case extensions.OperationOwnedRepositories:
				response.Page = &extensions.RepositoryPage{Items: []extensions.Repository{}}
			case extensions.OperationOrganizationOwner:
				owner := false
				response.Owner = &owner
			case extensions.OperationPublicSSHKeys:
				response.PublicKeys = []extensions.PublicKey{}
			default:
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	callback.Listener.Close()
	callback.Listener = callbackListener
	callback.Start()
	t.Cleanup(callback.Close)
	t.Setenv(extensions.ServiceCallbackEnv, callbackPath)

	servicePath := filepath.Join(dir, "service.sock")
	serviceListener, err := net.Listen("unix", servicePath)
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewUnstartedServer(s.ExtensionHandler())
	service.Listener.Close()
	service.Listener = serviceListener
	service.Start()
	t.Cleanup(service.Close)
	application, closeTransport := Extension(servicePath)
	t.Cleanup(closeTransport)
	proxy := httptest.NewTLSServer(application.HTTP)
	t.Cleanup(proxy.Close)
	return proxy
}

func nativeProductHeaders(method, origin string, contribution extensions.Contribution) http.Header {
	return nativeProductHeadersForActor(method, origin, contribution, "alice")
}

func nativeTestActorID(login string) string {
	if login == "bob" {
		return "2"
	}
	return "1"
}

func nativeProductHeadersForActor(method, origin string, contribution extensions.Contribution, login string) http.Header {
	contribution.Action = strings.ToLower(method)
	context, _ := json.Marshal(extensions.Authority{
		ExtensionID: "soda", InstanceID: "native-test-instance", SessionGeneration: nativeProductGeneration,
		Contribution: contribution, Actor: extensions.Actor{ID: nativeTestActorID(login), Username: login},
	})
	return http.Header{
		extensions.ContextHeader:           {string(context)},
		extensions.AdmissionHeader:         {"native-test-admission"},
		extensions.SessionGenerationHeader: {nativeProductGeneration},
		"Origin":                           {origin},
		"Sec-Fetch-Site":                   {"same-origin"},
	}
}

// nativeAPIServe keeps request-body and context hooks in process while exercising
// the private extension route with the same native authority as the proxy tests.
func nativeAPIServe(t *testing.T, s *Server, w http.ResponseWriter, r *http.Request) {
	nativeAPIServeWithOptions(t, s, w, r, nil, nil)
}

func nativeAPIServeWithCallback(t *testing.T, s *Server, w http.ResponseWriter, r *http.Request, callback func(extensions.CallbackRequest) extensions.CallbackResponse) {
	nativeAPIServeWithOptions(t, s, w, r, callback, nil)
}

func nativeAPIServeWithOptions(t *testing.T, s *Server, w http.ResponseWriter, r *http.Request, callback func(extensions.CallbackRequest) extensions.CallbackResponse, headers func(http.Header)) {
	t.Helper()
	login := r.Header.Get("X-Test-Native-Actor")
	if login == "" {
		login = "alice"
	}
	r.Header.Del("X-Test-Native-Actor")
	_ = nativeProductProxyForActorWithCallback(t, s, extensions.Contribution{}, login, callback)
	path := strings.TrimPrefix(r.URL.Path, config.SodaPath)
	contribution := extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}
	if strings.HasPrefix(path, "/api/settings/tailnet") {
		contribution.ID, contribution.Scope = "tailnet", "admin"
	}
	for name, values := range nativeProductHeadersForActor(r.Method, s.Config.ForgejoURL, contribution, login) {
		if name == "Origin" {
			continue
		}
		r.Header.Del(name)
		for _, value := range values {
			r.Header.Add(name, value)
		}
	}
	if headers != nil {
		headers(r.Header)
	}
	r.URL.Path = path
	s.ExtensionHandler().ServeHTTP(w, r)
}

func nativeProductRequest(t *testing.T, proxy *httptest.Server, origin, method, path string, body []byte) *http.Response {
	t.Helper()
	path = strings.TrimPrefix(path, "/api")
	r, err := http.NewRequestWithContext(t.Context(), method, proxy.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	contribution := extensions.Contribution{Kind: "page", ID: "spaces", Scope: "global"}
	if strings.HasPrefix(path, "/settings/tailnet") {
		contribution.ID, contribution.Scope = "tailnet", "admin"
	} else if !strings.HasPrefix(path, "/spaces") && !strings.HasPrefix(path, "/environments") && !strings.HasPrefix(path, "/repositories") {
		contribution = extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel"}
	}
	r.Header = nativeProductHeaders(method, origin, contribution)
	if len(body) > 0 {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := proxy.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func stubbedHostWebFixture(t *testing.T, native http.HandlerFunc) *Server {
	t.Helper()
	s := apiTestServer(t)
	peer := httptest.NewServer(native)
	t.Cleanup(peer.Close)
	s.SetHost(&host.Client{HTTP: peer.Client()})
	transport := peer.Client().Transport
	s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = strings.TrimPrefix(peer.URL, "http://")
		return transport.RoundTrip(r)
	})
	return s
}
