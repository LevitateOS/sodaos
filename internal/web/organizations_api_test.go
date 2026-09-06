package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrganizationEditRetainsOmittedNativeProfile(t *testing.T) {
	writes := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token acting-bob" || r.URL.Path != "/api/v1/orgs/team-org" {
			t.Error("wrong native organization actor")
		}
		if r.Method == "GET" {
			fmt.Fprint(w, `{"id":9007199254740993,"name":"team-org","full_name":"Team","description":"Old","website":"https://example.test/","location":"Office","visibility":"private"}`)
			return
		}
		writes++
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["full_name"]) != `"Team"` || string(body["location"]) != `"Office"` || string(body["website"]) != `"https://example.test/"` || string(body["description"]) != `"Updated"` {
			t.Error("omitted native profile field cleared")
		}
		if _, present := body["email"]; present {
			t.Error("email changed without explicit intent")
		}
		fmt.Fprint(w, `{"id":9007199254740993,"name":"team-org","description":"Updated"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/organizations/team-org", `{"description":"Updated"}`, "bob"))
	if w.Code != 200 || writes != 1 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) {
		t.Fatal("organization patch failed", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/organizations/team-org", `{"username":"renamed"}`, "bob"))
	if w.Code != 400 || writes != 1 {
		t.Fatal("organization identity change accepted")
	}
}
func TestTeamRepositoryGrantIsNativeScopedAndNeverProvisions(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/teams/9/repos/other/demo" || r.Header.Get("Authorization") != "token acting-alice" {
			t.Error("team request actor changed")
		}
		w.WriteHeader(403)
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		t.Error("team operation acquired environment authority")
		return nil, fmt.Errorf("forbidden")
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/teams/9/repositories/other/demo", `{}`, "alice"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("operator overrode native team/repository denial", w.Code)
	}
}
func TestTeamPolicyRejectsIgnoredOrUnknownNativeFields(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(403) })
	for _, body := range []string{`{"includes_all_repositories":true}`, `{"permission":"admin","units_map":{"repo.code":"none"}}`, `{"permission":"write","units_map":{"host.root":"write"}}`, `{"soda_operator":true}`} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/teams/9", body, "bob"))
		if w.Code != 400 && w.Code != 422 {
			t.Fatal("unsupported team policy accepted", w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid team policy reached provider")
	}
}
