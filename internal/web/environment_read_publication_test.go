package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/web/api"
	"github.com/stretchr/testify/require"
)

// Read handlers must reject a late result if the native page authority changes
// while repository callbacks or the read-only host observation are in flight.
func TestEnvironmentReadPublicationRechecksNativeAuthority(t *testing.T) {
	for _, read := range []struct {
		name, suffix, changeAt                            string
		operator, authorityUnavailable, nativeUnavailable bool
	}{
		{name: "detail/organization", changeAt: "organization"},
		{name: "detail/helper", changeAt: "host"},
		{name: "detail/native-unavailable", changeAt: "host", nativeUnavailable: true},
		{name: "detail/degraded-member", changeAt: "host", authorityUnavailable: true},
		{name: "detail/operator", changeAt: "host", operator: true},
		{name: "members/organization", suffix: "/members", changeAt: "organization"},
		{name: "members/degraded-member", suffix: "/members", changeAt: "repository", authorityUnavailable: true},
		{name: "connection/helper", suffix: "/connection", changeAt: "host"},
	} {
		for _, change := range []string{"unchanged", "generation", "actor", "instance"} {
			t.Run(read.name+"/"+change, func(t *testing.T) {
				s, _ := managementWebFixture(t)
				if read.operator {
					s.Config.OperatorID = 1
				}
				stored, err := s.Store.Project(t.Context(), webTerminalProject)
				require.NoError(t, err)
				members, err := s.Store.Members(t.Context(), webTerminalProject)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := apiTestRequest(http.MethodGet, "/api/environments/"+webTerminalProject+read.suffix, "", "alice").WithContext(ctx)
				changed := false
				changeAuthority := func(at string) {
					if at != read.changeAt {
						return
					}
					require.False(t, changed, "read boundary reached more than once")
					changed = true
					if change == "unchanged" {
						return
					}
					authority := r.Header.Get(extensions.ContextHeader)
					switch change {
					case "generation":
						r.Header.Set(extensions.ContextHeader, strings.Replace(authority, nativeProductGeneration, "expired", 1))
					case "actor":
						r.Header.Set(extensions.ContextHeader, strings.Replace(authority, `"id":"1"`, `"id":"2"`, 1))
					case "instance":
						r.Header.Set(extensions.ContextHeader, strings.Replace(authority, "native-test-instance", "other-instance", 1))
					}
				}
				callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
					switch in.Operation {
					case extensions.OperationRepository:
						changeAuthority("repository")
						if read.authorityUnavailable {
							return extensions.CallbackResponse{ErrorCode: "unavailable"}
						}
						return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "team", Name: "repo", Permission: "write"}}
					case extensions.OperationOrganizationOwner:
						changeAuthority("organization")
						owner := true
						return extensions.CallbackResponse{Owner: &owner}
					default:
						t.Error("unexpected native callback", in.Operation)
						return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
					}
				}
				nativeCalls := 0
				s.Host.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
					nativeCalls++
					require.Equal(t, http.MethodPost, req.Method)
					wantPath := "/inspect"
					if read.suffix == "/connection" {
						wantPath = "/connection"
					}
					require.Equal(t, wantPath, req.URL.Path, "read must not mutate native state")
					var input project.Create
					require.NoError(t, json.NewDecoder(req.Body).Decode(&input))
					require.Equal(t, project.Create{ID: webTerminalProject}, input)
					changeAuthority("host")
					if read.nativeUnavailable {
						return nil, errors.New("synthetic-private-helper-error")
					}
					w := httptest.NewRecorder()
					observed := project.Environment{ID: webTerminalProject, IP: "10.89.0.2", Running: true}
					if read.suffix == "/connection" {
						require.NoError(t, json.NewEncoder(w).Encode(project.Connection{Environment: observed, HostKey: "public-fixture", Fingerprint: "SHA256:fixture"}))
					} else {
						require.NoError(t, json.NewEncoder(w).Encode(observed))
					}
					return w.Result(), nil
				})}
				w := httptest.NewRecorder()
				nativeAPIServeWithCallback(t, s, w, r, callback)
				require.True(t, changed, "request never reached selected I/O boundary")
				wantNative := 1
				if read.suffix == "/members" {
					wantNative = 0
				}
				require.Equal(t, wantNative, nativeCalls)
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
				require.NotContains(t, w.Body.String(), "synthetic-private-helper-error")
				if change != "unchanged" {
					require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
					var result map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
					require.Len(t, result, 1, "refusal must not include protected data")
					require.Contains(t, result, "error")
				} else {
					require.Equal(t, http.StatusOK, w.Code, w.Body.String())
					var result struct {
						Environment          api.EnvironmentView  `json:"environment"`
						Observed             *project.Environment `json:"observed"`
						Administrator        bool                 `json:"environment_administrator"`
						AuthorityUnavailable bool                 `json:"authority_unavailable"`
						NativeUnavailable    bool                 `json:"native_unavailable"`
						Login                string               `json:"login"`
						Items                []struct {
							UserID string `json:"user_id"`
							Login  string `json:"login"`
						} `json:"items"`
						Connection      project.Connection `json:"connection"`
						RoutingVerified bool               `json:"routing_verified"`
					}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
					require.Equal(t, read.authorityUnavailable, result.AuthorityUnavailable)
					switch read.suffix {
					case "":
						require.Equal(t, api.EnvironmentDTO(stored), result.Environment)
						require.Equal(t, "original-alice", result.Login)
						require.Equal(t, !read.authorityUnavailable, result.Administrator)
						require.Equal(t, read.nativeUnavailable, result.NativeUnavailable)
						if read.nativeUnavailable {
							require.Nil(t, result.Observed)
						} else {
							require.Equal(t, &project.Environment{ID: webTerminalProject, IP: "10.89.0.2", Running: true}, result.Observed)
						}
					case "/members":
						if read.authorityUnavailable {
							require.Len(t, result.Items, 1)
							require.Equal(t, "1", result.Items[0].UserID)
						} else {
							require.Len(t, result.Items, 2)
						}
					case "/connection":
						require.Equal(t, "original-alice", result.Login)
						require.Equal(t, webTerminalProject, result.Connection.Environment.ID)
						require.Equal(t, "public-fixture", result.Connection.HostKey)
						require.Equal(t, "SHA256:fixture", result.Connection.Fingerprint)
						require.False(t, result.RoutingVerified)
					}
				}
				retained, err := s.Store.Project(t.Context(), webTerminalProject)
				require.NoError(t, err)
				require.Equal(t, stored, retained)
				retainedMembers, err := s.Store.Members(t.Context(), webTerminalProject)
				require.NoError(t, err)
				require.Equal(t, members, retainedMembers)
				otherActor, err := s.Store.User(t.Context(), 2)
				require.NoError(t, err, "read must preserve the other actor profile")
				require.Equal(t, "bob", otherActor.Login)
			})
		}
	}
}
