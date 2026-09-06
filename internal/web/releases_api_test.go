package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseCreateDuplicateAndPatchNativeSemantics(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong release actor")
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		switch r.Method {
		case "POST":
			if input["draft"] != true || input["tag_name"] != "v1" {
				t.Error("release publication defaults changed")
			}
			w.WriteHeader(409)
			fmt.Fprint(w, `{"message":"private-duplicate-tag-detail"}`)
		case "PATCH":
			if r.URL.Path != "/api/v1/repos/alice/demo/releases/7" || len(input) != 1 || input["name"] != "New title" {
				t.Error("unchanged fields overwritten")
			}
			fmt.Fprint(w, `{"id":7,"name":"New title","assets":[]}`)
		default:
			t.Error("unexpected operation")
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/releases", `{"tag_name":"v1","target_commitish":"main","draft":true}`, "bob"))
	if w.Code != 409 || calls != 1 || strings.Contains(w.Body.String(), "private-duplicate") {
		t.Fatal("duplicate hidden, leaked or retried")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/repos/alice/demo/releases/7", `{"name":"New title"}`, "bob"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/repos/alice/demo/releases/7", `{"body":""}`, "bob"))
	if w.Code != 422 || calls != 2 {
		t.Fatal("native ignored clear presented as success")
	}
}
func TestReleaseDownloadResolvesScopedMetadataAndRejectsRedirect(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong download actor")
		}
		switch r.URL.Path {
		case "/api/v1/repos/alice/demo/releases/7/assets/9":
			fmt.Fprint(w, `{"id":9,"uuid":"12345678-1234-1234-1234-123456789abc","name":"data.html","size":4,"type":"attachment","browser_download_url":"https://external.invalid/secret"}`)
		case "/attachments/12345678-1234-1234-1234-123456789abc":
			if calls == 2 {
				fmt.Fprint(w, "data")
			} else {
				w.Header().Set("Location", "https://external.invalid/secret")
				w.WriteHeader(302)
			}
		default:
			t.Error("unbounded download target", r.URL.Path)
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/releases/7/assets/9/download", "", "bob"))
	if w.Code != 200 || w.Body.String() != "data" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") || w.Header().Get("Content-Security-Policy") == "" || calls != 2 {
		t.Fatal("unsafe or missing download", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/releases/7/assets/9/download", "", "bob"))
	if w.Code < 400 || calls != 4 || strings.Contains(w.Body.String(), "external.invalid") {
		t.Fatal("redirect followed or forwarded", w.Code)
	}
}
func TestReleaseCrossRepositoryDownloadDeniedBeforeBinaryRequest(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(404) })
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/other/releases/7/assets/9/download", "", "bob"))
	if w.Code != 404 || calls != 1 {
		t.Fatal("native scope denial bypassed")
	}
}
func TestReleaseUploadIsBoundedNativeMultipart(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/api/v1/repos/alice/demo/releases/7/assets" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong native upload")
		}
		if err := r.ParseMultipartForm(65536); err != nil {
			t.Error(err)
		}
		defer r.MultipartForm.RemoveAll()
		file, header, err := r.FormFile("attachment")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		if header.Filename != "readme.txt" {
			t.Error("filename changed")
		}
		fmt.Fprint(w, `{"id":9007199254740993,"name":"readme.txt"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/releases/7/assets", `{"name":"readme.txt","content":"ZGF0YQ=="}`, "bob"))
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, body := range []string{`{"name":"../private","content":"ZGF0YQ=="}`, `{"name":"x","content":"!"}`, `{"name":"x","content":"","external_url":"https://external.invalid"}`} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/releases/7/assets", body, "bob"))
		if w.Code < 400 || calls != 1 {
			t.Fatal("invalid or external upload forwarded")
		}
	}
}
