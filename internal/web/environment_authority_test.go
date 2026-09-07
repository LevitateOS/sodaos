package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

func TestEnvironmentAuthorityUsesCurrentNativeIdentity(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		owner                        int64
		status                       int
		permission                   string
		admin, unavailable, operator bool
	}{
		{name: "current human owner", owner: 1, admin: true},
		{name: "transferred former owner", owner: 2, permission: `{"is_owner":false}`},
		{name: "organization owner", owner: 99, permission: `{"is_owner":true}`, admin: true},
		{name: "organization admin is not owner", owner: 99, permission: `{"is_admin":true,"is_owner":false}`},
		{name: "malformed owner capability", owner: 99, permission: `{"is_admin":true}`, unavailable: true},
		{name: "repository unavailable", status: 503, unavailable: true},
		{name: "repository deleted or hidden", status: 404, unavailable: true},
		{name: "repository wrong identity", status: 200, unavailable: true},
		{name: "explicit Soda operator", operator: true, admin: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "token acting-alice" {
					t.Error("wrong actor")
				}
				switch r.URL.Path {
				case "/api/v1/repositories/7":
					if tc.status != 0 {
						w.WriteHeader(tc.status)
						fmt.Fprint(w, `{"id":8,"name":"wrong","full_name":"wrong/repo","owner":{"id":1,"login":"alice"}}`)
						return
					}
					fmt.Fprintf(w, `{"id":7,"name":"renamed","full_name":"current/renamed","owner":{"id":%d,"login":"current"}}`, tc.owner)
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":1,"login":"alice-now"}`)
				case "/api/v1/users/alice-now/orgs/current/permissions":
					fmt.Fprint(w, tc.permission)
				default:
					t.Error("unexpected provider path", r.URL.Path)
					w.WriteHeader(500)
				}
			})
			if !tc.operator {
				s.Config.OperatorID = 777
			}
			id := "p0123456789abcdef01234567"
			p := store.Project{ID: id, RepositoryID: 7, OwnerID: 1, Name: "old", Repository: "alice/old"}
			if err := s.Store.CreateProject(t.Context(), p); err != nil {
				t.Fatal(err)
			}
			for _, u := range []struct {
				id    int64
				login string
			}{{1, "linux-alice"}, {2, "linux-bob"}} {
				if err := s.Store.Join(t.Context(), id, u.id, u.login); err != nil {
					t.Fatal(err)
				}
			}
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/inspect" {
					t.Fatal("authority changed native state")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":%q,"ip":"10.89.0.2","running":true}`, id)))}, nil
			})}
			for _, suffix := range []string{"", "/members"} {
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+id+suffix, "", "alice"))
				var result struct {
					Administrator bool   `json:"environment_administrator"`
					Unavailable   bool   `json:"authority_unavailable"`
					Login         string `json:"login"`
					Items         []struct {
						Login string `json:"login"`
					} `json:"items"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if w.Code != 200 || result.Unavailable != tc.unavailable {
					t.Fatal(w.Code, w.Body.String())
				}
				if suffix == "" {
					if result.Administrator != tc.admin || result.Login != "linux-alice" {
						t.Fatal(w.Body.String())
					}
				} else {
					want := 1
					if tc.admin {
						want = 2
					}
					if len(result.Items) != want {
						t.Fatal("stale or lost membership authority", w.Body.String())
					}
				}
			}
			if tc.operator && calls != 0 {
				t.Fatal("Soda operator conflated with native owner")
			}
			retained, err := s.Store.Project(t.Context(), id)
			login, e := s.Store.MemberLogin(t.Context(), id, 1)
			if err != nil || e != nil || retained.OwnerID != 1 || retained.Repository != "alice/old" || login != "linux-alice" {
				t.Fatal("authority check remapped retained state")
			}
		})
	}
}
