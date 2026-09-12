package tailnet

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailedEnrollmentCanBeExplicitlyRetried(t *testing.T) {
	for _, failure := range []string{"token", "key", "consume"} {
		t.Run(failure, func(t *testing.T) {
			m, target, calls, dir := runFixture(t)
			provider, keys := m.provider, m.keyHTTP
			before, err := os.ReadFile(filepath.Join(dir, "soda-tailnet", "project-"+target.Project+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if failure == "token" {
				m.provider = &http.Client{Transport: managementRoundTrip(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}
			}
			if failure == "key" {
				m.keyHTTP = managementRoundTrip(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, io.ErrUnexpectedEOF })
			}
			validate := func(context.Context) error { return nil }
			consume := func(context.Context, string) error { return io.ErrUnexpectedEOF }
			if err := m.EnrollRun(t.Context(), target, validate, consume); err == nil {
				t.Fatal("failure concealed")
			}
			// Passive reads neither replay the attempt nor change the saved policy.
			count := calls.Load()
			if _, err := m.RunBinding(t.Context(), target); err != nil || calls.Load() != count {
				t.Fatal("read replayed enrollment", err)
			}
			m.provider, m.keyHTTP = provider, keys
			consumed := 0
			if err := m.EnrollRun(t.Context(), target, validate, func(_ context.Context, key string) error {
				consumed++
				if !strings.HasPrefix(key, "tskey-auth-") {
					t.Fatal("wrong input")
				}
				return nil
			}); err != nil || consumed != 1 || calls.Load() != count+1 {
				t.Fatal("explicit same-run recovery refused", err)
			}
			after, err := os.ReadFile(filepath.Join(dir, "soda-tailnet", "project-"+target.Project+".json"))
			if err != nil || string(after) != string(before) {
				t.Fatal("enrollment wrote a permanent attempt marker", err)
			}
			entries, err := os.ReadDir(filepath.Join(dir, "soda-tailnet"))
			if err != nil || len(entries) != 2 {
				t.Fatal("unexpected journal/credential archive", err)
			}
		})
	}
}
