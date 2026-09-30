package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/stretchr/testify/require"
)

// The active stop/remove mutations recheck native authority after decoding and
// before dispatch. Changing presentation authority during the body read cannot
// run a native command under the original actor or page generation.
func TestRunnerMutationAdmissionAfterDecode(t *testing.T) {
	for _, operation := range runnerAPIRequests {
		if operation.name != "stop" && operation.name != "remove" {
			continue
		}
		for _, change := range []string{"unchanged", "actor", "generation", "instance", "cancel"} {
			t.Run(operation.name+"/"+change, func(t *testing.T) {
				s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != "/runners/"+operation.name {
						t.Error("unexpected helper operation", r.Method, r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"ok":true}`))
				})
				var calls atomic.Int32
				transport := s.Host.HTTP.Transport
				s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					return transport.RoundTrip(r)
				})
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				r := apiTestRequest(operation.method, operation.path, operation.body, "alice").WithContext(ctx)
				read := false
				r.Body = io.NopCloser(&mutationAdmissionBody{Reader: strings.NewReader(operation.body), beforeRead: func() {
					read = true
					context := r.Header.Get(extensions.ContextHeader)
					switch change {
					case "actor":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, `"id":"1"`, `"id":"2"`, 1))
					case "generation":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, nativeProductGeneration, "expired", 1))
					case "instance":
						r.Header.Set(extensions.ContextHeader, strings.Replace(context, "native-test-instance", "other-instance", 1))
					case "cancel":
						cancel()
					}
				}})
				w := httptest.NewRecorder()
				nativeAPIServe(t, s, w, r)
				require.True(t, read, "request never reached the post-authorization body read")
				wantStatus, wantCalls := http.StatusConflict, int32(0)
				if change == "unchanged" {
					wantStatus, wantCalls = http.StatusOK, 1
				}
				if w.Code != wantStatus || calls.Load() != wantCalls {
					t.Fatalf("status=%d want=%d helper_dispatches=%d want=%d", w.Code, wantStatus, calls.Load(), wantCalls)
				}
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
				require.NotContains(t, w.Body.String(), "synthetic-runner-secret")
				if change == "unchanged" {
					require.JSONEq(t, `{"ok":true}`, w.Body.String())
				}
			})
		}
	}
}
