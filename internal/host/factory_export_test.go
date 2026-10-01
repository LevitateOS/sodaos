package host

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryExportClientPreservesConfirmedRefusals(t *testing.T) {
	in := project.FactoryExport{Project: "p0123456789abcdef01234567", ID: strings.Repeat("a", 32), Role: project.RoleCoder, Preparation: "f0123456789abcdef01234567", Candidate: strings.Repeat("b", 40)}
	for _, tc := range []struct {
		status int
		want   error
	}{
		{http.StatusNotFound, ErrRunNotFound},
		{http.StatusConflict, ErrRunStale},
		{http.StatusUnprocessableEntity, project.ErrFactoryExportCandidate},
		{http.StatusRequestEntityTooLarge, project.ErrFactoryExportBounds},
		{http.StatusInternalServerError, nil},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/factory-export" || r.Method != http.MethodPost {
					t.Fatal("export used unexpected transport route")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader("private daemon error")), Header: make(http.Header)}, nil
			})
			_, err := c.FactoryExport(context.Background(), in)
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatalf("confirmed refusal lost: %v", err)
				}
			} else if err == nil || errors.Is(err, project.ErrFactoryExportCandidate) || errors.Is(err, project.ErrFactoryExportBounds) {
				t.Fatal("unconfirmed server failure became a candidate verdict")
			}
		})
	}
}

func TestFactoryExportRouteUsesFactoryRuntime(t *testing.T) {
	d := &Daemon{}
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/factory-export", strings.NewReader(`{}`)))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("export route without runtime returned %d", w.Code)
	}
}
