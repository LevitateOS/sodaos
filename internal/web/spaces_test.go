package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

func spacesFixture(t *testing.T, status int, member bool) (*Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	providerCalls, nativeCalls := new(atomic.Int32), new(atomic.Int32)
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		providerCalls.Add(1)
		if r.URL.Path == "/api/v1/user" {
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
			return
		}
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		var id int64
		_, _ = fmt.Sscanf(r.URL.Path, "/api/v1/repositories/%d", &id)
		fmt.Fprintf(w, `{"id":%d,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}`, id)
	})
	s.Config.OperatorID = 999
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nativeCalls.Add(1)
		var input host.Create
		_ = json.NewDecoder(r.Body).Decode(&input)
		_ = json.NewEncoder(w).Encode(host.Environment{ID: input.ID, Running: true})
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
	return s, providerCalls, nativeCalls
}
func readSpaces(t *testing.T, s *Server) spacesView {
	t.Helper()
	w := terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/spaces", nil)
	var result spacesView
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
				s, _, native := spacesFixture(t, status, member)
				result := readSpaces(t, s)
				if status != 0 && !member {
					if len(result.Items) != 0 || native.Load() != 0 {
						t.Fatal("unauthorized row/helper inspection")
					}
					if result.Complete != (status == 403 || status == 404) {
						t.Fatal("unavailable disguised as complete empty")
					}
					return
				}
				if len(result.Items) != 1 || native.Load() != 1 {
					t.Fatal("legitimate observation missing")
				}
				row := result.Items[0]
				if status != 0 && (result.Complete || !row.AuthorityUnavailable || row.Administrator || len(row.Terminals) != 0 || row.Login != "original-alice") {
					t.Fatal("degraded data elevated", row)
				}
			})
		}
	}
}
func TestSpacesProviderIdentityDenialIsNotCompleteEmpty(t *testing.T) {
	s, _, native := spacesFixture(t, 0, false)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer provider.Close()
	s.Forgejo = forgejo.New(provider.URL)
	result := readSpaces(t, s)
	if result.Complete || len(result.Items) != 0 || native.Load() != 0 {
		t.Fatal("unavailable actor authority disguised as a complete empty collection")
	}
}

func TestSpacesBoundsAreIncompleteNotCompleteEmpty(t *testing.T) {
	for _, status := range []int{0, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			s, provider, native := spacesFixture(t, status, false)
			for i := 0; i < 130; i++ {
				if err := s.Store.CreateProject(t.Context(), store.Project{ID: fmt.Sprintf("p%024x", i+100), RepositoryID: int64(i + 100), OwnerID: 1, Name: "associated"}); err != nil {
					t.Fatal(err)
				}
			}
			result := readSpaces(t, s)
			if result.Complete || len(result.Items) > 32 || provider.Load() > 256 || native.Load() > 32 {
				t.Fatal("unbounded or falsely complete", len(result.Items), provider.Load(), native.Load())
			}
		})
	}
}
func TestSpacesResponseByteLimitAndOversizedStoreLabel(t *testing.T) {
	s, _, _ := spacesFixture(t, 0, false)
	for i := 0; i < 32; i++ {
		if err := s.Store.CreateProject(t.Context(), store.Project{ID: fmt.Sprintf("p%024x", i+100), RepositoryID: int64(i + 100), OwnerID: 1, Name: strings.Repeat("<", 256), Repository: strings.Repeat("<", 512)}); err != nil {
			t.Fatal(err)
		}
	}
	result := readSpaces(t, s)
	if result.Complete || len(result.Items) >= 32 {
		t.Fatal("response bound not applied")
	}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: "p000000000000000000000001", RepositoryID: 9999, OwnerID: 1, Name: strings.Repeat("x", 2000)}); err != nil {
		t.Fatal(err)
	}
	if terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/spaces", nil).Code != 503 {
		t.Fatal("oversized DB metadata silently truncated")
	}
}
func TestSpacesAdmissionActorAndQueryBounds(t *testing.T) {
	s, provider, native := spacesFixture(t, 0, true)
	for range cap(s.spacesSlots) {
		s.spacesSlots <- struct{}{}
	}
	if terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/spaces", nil).Code != 503 {
		t.Fatal("request gate")
	}
	for range cap(s.spacesSlots) {
		<-s.spacesSlots
	}
	for _, query := range []string{"?", "?repository_id=7", "?after=1"} {
		if terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/spaces"+query, nil).Code != 400 {
			t.Fatal("query alias")
		}
	}
	r := apiTestRequest("GET", "/api/spaces", "", "alice")
	r.Header.Set(expectedUserHeader, "2")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 || provider.Load() != 0 || native.Load() != 0 {
		t.Fatal("actor guard did not precede collection")
	}
}
func TestSpacesSlowInspectionCannotBlockLogoutOrPublishAfterIt(t *testing.T) {
	s, _, _ := spacesFixture(t, 0, true)
	entered, release := make(chan struct{}), make(chan struct{})
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release }))
	defer helper.Close()
	defer close(release)
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, helper.Listener.Addr().String())
	}}}
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() { result <- terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/spaces", nil) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("inspection not entered")
	}
	start := time.Now()
	w := terminalAPI(t, s, s.Config.ForgejoURL, "POST", "/api/session/logout", map[string]any{})
	if w.Code != 204 || time.Since(start) > time.Second {
		t.Fatal("logout blocked behind inspection")
	}
	select {
	case w := <-result:
		if w.Code != 401 {
			t.Fatal("published after logout", w.Code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("unbounded inspection")
	}
}
