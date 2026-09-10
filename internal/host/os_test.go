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
	if !reflect.DeepEqual(args, []string{"exec", "soda-p0123456789abcdef01234567", "/usr/bin/python3", "-I", "-S", "-B", "-c", projectOSProgram}) {
		return nil, errors.New("unexpected execution")
	}
	return []byte(e.response), nil
}
func TestOSObservationIsSeparateAndNeverStartsOrInfersAProfile(t *testing.T) {
	for _, tc := range []struct {
		running bool
		raw     string
		valid   bool
	}{
		{false, "", false}, {true, `{"id":"rocky","version":"9.7","name":"Rocky Linux 9.7"}`, true},
		{true, `{"id":"rocky"}`, false}, {true, `{"id":"rocky","version":"9","name":"unsafe\u202e"}`, false},
		{true, strings.Repeat("x", 2049), false},
	} {
		e := &osInspection{running: tc.running, response: tc.raw}
		d := Daemon{Exec: e, Config: Config{Image: "not-an-existing-root-identity"}}
		out, err := d.observeOS(t.Context(), "p0123456789abcdef01234567")
		if err != nil || out.Environment.Profile != nil || out.Unavailable == tc.valid || (out.Release != nil) != tc.valid || out.Environment.Image != "sha256:"+strings.Repeat("a", 64) {
			t.Fatal(out, err)
		}
		want := 1
		if tc.running {
			want = 2
		}
		if len(e.calls) != want {
			t.Fatal(e.calls)
		}
	}
	e := &noExec{}
	d := Daemon{Exec: e}
	if _, err := d.observeOS(t.Context(), "../other"); err == nil || e.called {
		t.Fatal("untrusted ID reached executor")
	}
}
