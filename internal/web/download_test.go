package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepositoryHTMLDownloadsAreAttachments(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/alice/demo/raw/dir/index.html" || r.URL.Query().Get("ref") != "feature/one" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("invalid native download target")
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<script>privateRepositoryContent()</script>`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/download?path=dir%2Findex.html&ref=feature%2Fone", "", "bob"))
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unsafe download headers", w.Code)
	}
}
func TestDownloadRejectsOversizeBeforeResponseCommit(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "99999999")
		w.WriteHeader(200)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/download?path=huge.bin", "", "alice"))
	if w.Code != 413 || !strings.Contains(w.Header().Get("Content-Type"), "application/json") || w.Header().Get("Content-Disposition") != "" {
		t.Fatal("oversized response committed as download", w.Code)
	}
}
func TestDownloadTraversalDoesNotReachProvider(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("traversal reached provider") })
	for _, path := range []string{"..%2Fother", "%2Fetc%2Fshadow", "a%2F%2Fb", "a%5Cb", "%00"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/download?path="+path, "", "alice"))
		if w.Code != 400 {
			t.Fatal("invalid path accepted", w.Code)
		}
	}
}
