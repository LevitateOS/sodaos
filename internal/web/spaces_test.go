package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

func spacesFixture(t *testing.T, member bool) (*Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	callbackCalls, nativeCalls := new(atomic.Int32), new(atomic.Int32)
	s := apiTestServer(t)
	s.Config.OperatorID = 999
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nativeCalls.Add(1)
		if r.URL.Path == "/terminal" {
			c, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer c.CloseNow()
			_, _, _ = c.Read(r.Context())
			_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"type":"metadata","terminals":[]}`))
			return
		}
		var input project.Create
		_ = json.NewDecoder(r.Body).Decode(&input)
		_ = json.NewEncoder(w).Encode(project.Environment{ID: input.ID, Running: true})
	}))
	t.Cleanup(helper.Close)
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, helper.Listener.Addr().String())
	}}}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, Name: "private-name", RepositoryID: 7, OwnerID: 1, Repository: "private-owner/private-name"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, ""); err != nil {
		t.Fatal(err)
	}
	if member {
		if err := s.Store.Join(t.Context(), webTerminalProject, 1, "original-alice"); err != nil {
			t.Fatal(err)
		}
	}
	return s, callbackCalls, nativeCalls
}

func spacesCallback(status int, calls *atomic.Int32) func(extensions.CallbackRequest) extensions.CallbackResponse {
	return func(in extensions.CallbackRequest) extensions.CallbackResponse {
		calls.Add(1)
		if in.Operation != extensions.OperationRepository {
			return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
		}
		switch status {
		case http.StatusForbidden:
			return extensions.CallbackResponse{ErrorCode: "forbidden"}
		case http.StatusNotFound:
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		case http.StatusServiceUnavailable:
			return extensions.CallbackResponse{ErrorCode: "unavailable"}
		default:
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "write"}}
		}
	}
}

func spacesAPI(t *testing.T, s *Server, path string, callback func(extensions.CallbackRequest) extensions.CallbackResponse) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest(http.MethodGet, path, "", "alice"), callback)
	return w
}

func readSpaces(t *testing.T, s *Server, callback func(extensions.CallbackRequest) extensions.CallbackResponse) api.SpacesView {
	t.Helper()
	w := spacesAPI(t, s, "/api/spaces", callback)
	var result api.SpacesView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("collection", w.Code, w.Body.String())
	}
	if w.Body.Len() > 65536 {
		t.Fatal("unbounded collection")
	}
	return result
}

func TestSpacesDeniedUnavailableAndDegradedOwnMembership(t *testing.T) {
	for _, status := range []int{0, 403, 404, 503} {
		for _, member := range []bool{false, true} {
			t.Run(fmt.Sprint(status, member), func(t *testing.T) {
				s, calls, native := spacesFixture(t, member)
				result := readSpaces(t, s, spacesCallback(status, calls))
				if status != 0 && !member {
					if len(result.Items) != 0 || native.Load() != 0 {
						t.Fatal("unauthorized row/helper inspection")
					}
					if result.Complete != (status == 404) {
						t.Fatal("unavailable disguised as complete empty")
					}
					return
				}
				expectedNative := int32(1)
				if status == 0 {
					expectedNative++
				} // Optional authorized Tailnet observation.
				if status == 0 && member {
					expectedNative++
				} // Actual native terminal inventory.
				if len(result.Items) != 1 || native.Load() != expectedNative {
					t.Fatal("legitimate observation missing")
				}
				row := result.Items[0]
				if (status == 0 && row.TailnetState != "unavailable") || (status != 0 && row.TailnetState != "") {
					t.Fatal("private network observation escaped fresh authority", row.TailnetState)
				}
				if status != 0 && (result.Complete || !row.AuthorityUnavailable || row.Administrator || len(row.Terminals) != 0 || row.Login != "original-alice") {
					t.Fatal("degraded data elevated", row)
				}
			})
		}
	}
}

func TestSpacesNativeActorDenialDoesNotPublishCollection(t *testing.T) {
	s, _, native := spacesFixture(t, false)
	w := httptest.NewRecorder()
	nativeAPIServeWithOptions(t, s, w, apiTestRequest(http.MethodGet, "/api/spaces", "", "alice"), nil, func(h http.Header) {
		h.Set(extensions.ContextHeader, "invalid-authority")
	})
	if w.Code != http.StatusForbidden || native.Load() != 0 {
		t.Fatal("invalid native actor reached a Spaces collection", w.Code)
	}
}

func TestSpacesBoundsAreIncompleteNotCompleteEmpty(t *testing.T) {
	for _, status := range []int{0, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			s, callbackCalls, native := spacesFixture(t, false)
			for i := 0; i < 130; i++ {
				if err := s.Store.CreateProject(t.Context(), store.Project{ID: fmt.Sprintf("p%024x", i+100), RepositoryID: int64(i + 100), OwnerID: 1, Name: "associated"}); err != nil {
					t.Fatal(err)
				}
			}
			result := readSpaces(t, s, spacesCallback(status, callbackCalls))
			if result.Complete || len(result.Items) > 32 || callbackCalls.Load() > 256 || native.Load() > 64 {
				t.Fatal("unbounded or falsely complete", len(result.Items), callbackCalls.Load(), native.Load())
			}
		})
	}
}

func TestSpacesResponseByteLimitAndOversizedStoreLabel(t *testing.T) {
	s, _, _ := spacesFixture(t, false)
	for i := 0; i < 32; i++ {
		if err := s.Store.CreateProject(t.Context(), store.Project{ID: fmt.Sprintf("p%024x", i+100), RepositoryID: int64(i + 100), OwnerID: 1, Name: strings.Repeat("<", 256), Repository: strings.Repeat("<", 512)}); err != nil {
			t.Fatal(err)
		}
	}
	result := readSpaces(t, s, spacesCallback(0, new(atomic.Int32)))
	if result.Complete || len(result.Items) >= 32 {
		t.Fatal("response bound not applied")
	}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: "p000000000000000000000001", RepositoryID: 9999, OwnerID: 1, Name: strings.Repeat("x", 2000)}); err != nil {
		t.Fatal(err)
	}
	if spacesAPI(t, s, "/api/spaces", spacesCallback(0, new(atomic.Int32))).Code != 503 {
		t.Fatal("oversized DB metadata silently truncated")
	}
}

func TestSpacesAdmissionActorAndQueryBounds(t *testing.T) {
	s, callbackCalls, native := spacesFixture(t, true)
	for range cap(s.API.SpacesSlots) {
		s.API.SpacesSlots <- struct{}{}
	}
	if spacesAPI(t, s, "/api/spaces", spacesCallback(0, callbackCalls)).Code != 503 {
		t.Fatal("request gate")
	}
	for range cap(s.API.SpacesSlots) {
		<-s.API.SpacesSlots
	}
	for _, query := range []string{"?", "?repository_id=7", "?after=1"} {
		want := http.StatusBadRequest
		if query == "?" {
			want = http.StatusNotFound // The extension rejects noncanonical empty queries at its boundary.
		}
		if w := spacesAPI(t, s, "/api/spaces"+query, spacesCallback(0, callbackCalls)); w.Code != want {
			t.Fatal("query alias", query, w.Code, w.Body.String())
		}
	}
	if spacesAPI(t, s, "/api/spaces?actor=2", spacesCallback(0, callbackCalls)).Code != 400 || callbackCalls.Load() != 0 || native.Load() != 0 {
		t.Fatal("query aliases are rejected before collection")
	}
}

func TestSpacesSlowInspectionCannotPublishAfterNativeRevocation(t *testing.T) {
	s, calls, _ := spacesFixture(t, true)
	entered, release := make(chan struct{}), make(chan struct{})
	var enteredOnce sync.Once
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { enteredOnce.Do(func() { close(entered) }); <-release }))
	defer helper.Close()
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, helper.Listener.Addr().String())
	}}}
	var revoked atomic.Bool
	proxy := nativeProductProxyForActorWithCallbacks(t, s, extensions.Contribution{}, "alice", spacesCallback(0, calls), func(extensions.CallbackRequest) extensions.CallbackResponse {
		if revoked.Load() {
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		}
		return extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}}
	})
	result := make(chan *http.Response, 1)
	go func() {
		result <- nativeProductRequest(t, proxy, s.Config.ForgejoURL, http.MethodGet, "/api/spaces", nil)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("inspection not entered")
	}
	revoked.Store(true)
	releaseOnce.Do(func() { close(release) })
	select {
	case response := <-result:
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatal("published after native revocation", response.StatusCode)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("unbounded inspection")
	}
}
