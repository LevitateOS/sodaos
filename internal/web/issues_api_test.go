package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIssueCreationAndCommentUseActingIdentity(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong issue actor")
		}
		switch r.URL.Path {
		case "/api/v1/repos/alice/demo/issues":
			if r.Method != "POST" {
				t.Error("wrong native verb")
			}
			var input struct {
				Title     string   `json:"title"`
				Assignees []string `json:"assignees"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.Title != "Bug" || len(input.Assignees) != 1 || input.Assignees[0] != "alice" {
				t.Error("native issue input changed")
			}
			fmt.Fprint(w, `{"id":9007199254740993,"number":12,"title":"Bug","state":"open","user":{"id":2,"login":"bob"}}`)
		case "/api/v1/repos/alice/demo/issues/12/comments":
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"private native denial"}`)
		default:
			t.Error("unexpected issue endpoint")
			w.WriteHeader(404)
		}
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		t.Error("issue operation touched native environment")
		return nil, fmt.Errorf("not permitted")
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/issues", `{"title":"Bug","body":"Details","assignees":["alice"]}`, "bob"))
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) || !strings.Contains(w.Body.String(), `"number":"12"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/issues/12/comments", `{"body":"Comment"}`, "bob"))
	if w.Code != 403 || calls != 2 || strings.Contains(w.Body.String(), "private native") {
		t.Fatal("comment denial hidden or retried", w.Code)
	}
}
func TestSubscriptionTargetIsAlwaysTheActingUser(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/alice/demo/issues/12/subscriptions/bob" || r.Method != "PUT" {
			t.Error("caller selected subscription actor")
		}
		w.WriteHeader(201)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/repos/alice/demo/issues/12/subscription", `{"user":"alice"}`, "bob"))
	if w.Code != 400 || calls != 0 {
		t.Fatal("forged subscription actor accepted")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/repos/alice/demo/issues/12/subscription", `{}`, "bob"))
	if w.Code != 204 || calls != 1 {
		t.Fatal(w.Code)
	}
}
func TestAttachmentConvertsOnlyBoundedBytesToNativeMultipart(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/alice/demo/issues/12/assets" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong attachment target")
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		data, err := io.ReadAll(part)
		if err != nil || part.FormName() != "attachment" || part.FileName() != "note.txt" || string(data) != "hello" {
			t.Error("multipart bytes changed")
		}
		fmt.Fprint(w, `{"id":22,"name":"note.txt","uuid":"00000000-0000-0000-0000-000000000000","size":5}`)
	})
	path := "/api/forgejo/repos/alice/demo/issues/12/attachments"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, `{"name":"../private","content":"aGVsbG8="}`, "bob"))
	if w.Code != 422 || calls != 0 {
		t.Fatal("unsafe filename accepted")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, `{"name":"note.txt","content":"aGVsbG8="}`, "bob"))
	if w.Code != 201 || calls != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
}
