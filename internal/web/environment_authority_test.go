package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/store"
)

func TestEnvironmentAuthorityUsesCurrentNativeIdentity(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		owner, permission            string
		organizationOwner            bool
		unavailable, operator, admin bool
	}{
		{name: "current human owner", owner: "alice", permission: "write", admin: true},
		{name: "transferred former owner", owner: "current", permission: "write"},
		{name: "organization owner", owner: "current", permission: "write", organizationOwner: true, admin: true},
		{name: "organization admin is not owner", owner: "current", permission: "admin"},
		{name: "malformed owner capability", owner: "current", permission: "write", unavailable: true},
		{name: "repository unavailable", unavailable: true},
		{name: "repository deleted or hidden", unavailable: true},
		{name: "repository wrong identity", unavailable: true},
		{name: "explicit Soda operator", owner: "current", permission: "write", operator: true, admin: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := apiTestServer(t)
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
			calls := 0
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				calls++
				switch in.Operation {
				case extensions.OperationRepository:
					if in.RepositoryID != "7" {
						t.Error("unexpected repository", in.RepositoryID)
					}
					if strings.HasPrefix(tc.name, "repository ") {
						if tc.name == "repository wrong identity" {
							return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "8", Owner: "alice", Name: "wrong", Permission: "write"}}
						}
						return extensions.CallbackResponse{ErrorCode: "unavailable"}
					}
					return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: tc.owner, Name: "renamed", Permission: tc.permission}}
				case extensions.OperationOrganizationOwner:
					if in.Organization != tc.owner {
						t.Error("unexpected owner", in.Organization)
					}
					if tc.name == "malformed owner capability" {
						return extensions.CallbackResponse{ErrorCode: "invalid_result"}
					}
					return extensions.CallbackResponse{Owner: &tc.organizationOwner}
				default:
					t.Error("unexpected native callback", in.Operation)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
			}
			for _, suffix := range []string{"", "/members"} {
				w := httptest.NewRecorder()
				nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments/"+id+suffix, "", "alice"), callback)
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
			if calls < 2 {
				t.Fatal("native repository authority was not checked")
			}
			retained, err := s.Store.Project(t.Context(), id)
			login, e := s.Store.MemberLogin(t.Context(), id, 1)
			if err != nil || e != nil || retained.OwnerID != 1 || retained.Repository != "alice/old" || login != "linux-alice" {
				t.Fatal("authority check remapped retained state")
			}
		})
	}
}
