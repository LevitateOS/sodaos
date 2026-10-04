package host

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type osInspection struct {
	running  bool
	missing  bool
	response string
	calls    [][]string
}

func (e *osInspection) Run(_ context.Context, _ []byte, command string, args ...string) ([]byte, error) {
	e.calls = append(e.calls, append([]string{command}, args...))
	if command != "/usr/bin/podman" {
		return nil, errors.New("unexpected command")
	}
	if reflect.DeepEqual(args, []string{"inspect", "soda-p0123456789abcdef01234567"}) {
		return json.Marshal([]any{map[string]any{"Image": strings.Repeat("a", 64), "Config": map[string]any{"Labels": map[string]string{"org.soda.project": "p0123456789abcdef01234567", "org.soda.owner": "1"}}, "State": map[string]bool{"Running": e.running}}})
	}
	if reflect.DeepEqual(args, []string{"exec", "soda-p0123456789abcdef01234567", "/usr/bin/test", "-f", "/etc/os-release"}) {
		if e.missing {
			return nil, errors.New("exit status 1")
		}
		return nil, nil
	}
	if reflect.DeepEqual(args, []string{"exec", "soda-p0123456789abcdef01234567", "/usr/bin/head", "-c", "4097", "/etc/os-release"}) {
		return []byte(e.response), nil
	}
	return nil, errors.New("unexpected execution")
}

func TestOSObservationIsSeparateAndNeverStartsOrInfersAProfile(t *testing.T) {
	padded := "ID=rocky\nVERSION_ID=9\n"
	padded += "#" + strings.Repeat("p", 4096-len(padded)-2) + "\n"
	if len(padded) != 4096 {
		t.Fatal("truncation vector has wrong size", len(padded))
	}
	for _, tc := range []struct {
		name             string
		running          bool
		missing          bool
		raw              string
		valid            bool
		id, version, rel string
	}{
		{name: "stopped", running: false, valid: false},
		{name: "missing file", running: true, missing: true, valid: false},
		{name: "quoted", running: true, raw: "ID=\"rocky\"\nVERSION_ID=\"9.7\"\nPRETTY_NAME=\"Rocky Linux 9.7\"\n", valid: true, id: "rocky", version: "9.7", rel: "Rocky Linux 9.7"},
		{name: "unquoted", running: true, raw: "ID=rocky\nVERSION_ID=9\n", valid: true, id: "rocky", version: "9", rel: "rocky"},
		{name: "single quotes fall back to id", running: true, raw: "ID='rocky'\nVERSION_ID='9'\n", valid: true, id: "rocky", version: "9", rel: "rocky"},
		{name: "comments and unknown keys ignored", running: true, raw: "# soda\n\nID=rocky\nHOME_URL=https://example.invalid\nVERSION_ID=9\n", valid: true, id: "rocky", version: "9", rel: "rocky"},
		{name: "exact limit", running: true, raw: padded, valid: true, id: "rocky", version: "9", rel: "rocky"},
		{name: "missing version", running: true, raw: "ID=rocky\n", valid: false},
		{name: "duplicate id", running: true, raw: "ID=rocky\nID=rocky\nVERSION_ID=9\n", valid: false},
		{name: "kept key without value", running: true, raw: "ID\nVERSION_ID=9\n", valid: false},
		{name: "mismatched quotes kept literally", running: true, raw: "ID=\"rocky'\nVERSION_ID=9\n", valid: false},
		{name: "no escape processing", running: true, raw: "ID=\"roc\\\"ky\"\nVERSION_ID=9\n", valid: false},
		{name: "keys are not trimmed", running: true, raw: " ID=rocky\nVERSION_ID=9\n", valid: false},
		{name: "empty id", running: true, raw: "ID=\nVERSION_ID=9\n", valid: false},
		{name: "unsafe name", running: true, raw: "ID=rocky\nVERSION_ID=9\nPRETTY_NAME=\"unsafe‮\"\n", valid: false},
		{name: "oversized", running: true, raw: padded + "x", valid: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &osInspection{running: tc.running, missing: tc.missing, response: tc.raw}
			d := testDaemon(e, Config{Image: "not-an-existing-root-identity"})
			out, err := d.Project.ObserveOS(t.Context(), "p0123456789abcdef01234567")
			if err != nil || out.Environment.Profile != nil || out.Unavailable == tc.valid || (out.Release != nil) != tc.valid || out.Environment.Image != "sha256:"+strings.Repeat("a", 64) {
				t.Fatal(out, err)
			}
			if tc.valid && (out.Release.ID != tc.id || out.Release.Version != tc.version || out.Release.Name != tc.rel) {
				t.Fatalf("wrong release: %+v", out.Release)
			}
			want := 1
			if tc.running {
				want = 3
				if tc.missing {
					want = 2
				}
			}
			if len(e.calls) != want {
				t.Fatal(e.calls)
			}
		})
	}
	e := &noExec{}
	d := testDaemon(e, Config{})
	if _, err := d.Project.ObserveOS(t.Context(), "../other"); err == nil || e.called {
		t.Fatal("untrusted ID reached executor")
	}
}
