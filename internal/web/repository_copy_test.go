package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepositoryCopiesRejectUnconfirmedDestinationWithoutRetry(t *testing.T) {
	for _, kind := range []string{"fork", "import"} {
		for _, body := range []string{`{}`, `{"id":9,"name":"copy","owner":{"id":1,"login":"alice"},"fork":true}`, `{"id":9,"name":"copy","owner":{"id":2,"login":"bob"},"fork":true,"mirror":true}`} {
			t.Run(kind+body, func(t *testing.T) {
				calls := 0
				s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(202); fmt.Fprint(w, body) })
				path, input := "/api/forgejo/repos/alice/demo/fork", `{"name":"copy"}`
				if kind == "import" {
					path, input = "/api/forgejo/repositories/import", `{"url":"https://git.example/repo.git","name":"copy","private":true}`
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest("POST", path, input, "bob"))
				if w.Code != 503 || calls != 1 {
					t.Fatalf("unconfirmed destination accepted/retried: %d %d", w.Code, calls)
				}
			})
		}
	}
}

func TestNativeForkAcceptedResponseMustDescribeAPersonalFork(t *testing.T) {
	for _, fork := range []bool{true, false} {
		t.Run(fmt.Sprint(fork), func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/api/v1/repos/alice/demo/forks" || r.Header.Get("Authorization") != "token acting-bob" {
					t.Error("wrong native fork actor/path")
				}
				w.WriteHeader(202)
				fmt.Fprintf(w, `{"id":9,"name":"copy","owner":{"id":2,"login":"bob"},"fork":%t}`, fork)
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/fork", `{"name":"copy"}`, "bob"))
			want := 503
			if fork {
				want = 201
			}
			if w.Code != want || calls != 1 {
				t.Fatalf("fork result: %d %d", w.Code, calls)
			}
		})
	}
}

func TestImportFailureNeverEchoesCredentialsOrRetries(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(422)
		fmt.Fprint(w, `{"message":"synthetic-import-password synthetic-import-token"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repositories/import", `{"url":"https://git.example/repo.git","name":"copy","password":"synthetic-import-password","token":"synthetic-import-token"}`, "bob"))
	if w.Code != 422 || calls != 1 || strings.Contains(w.Body.String(), "synthetic-import") {
		t.Fatal("credential-bearing native failure escaped/retried")
	}
}
