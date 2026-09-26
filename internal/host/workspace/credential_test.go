package workspace

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCredentialCaptureFreezesBeforeReadingAndKillsBeforeReturn(t *testing.T) {
	r := testRun()
	var calls []string
	state := []byte(`{"tokens":{"refresh_token":"synthetic"}}`)
	w := Runtime{Exec: executorFunc(func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		switch {
		case args[0] == "container" && args[1] == "inspect":
			return json.Marshal(map[string]string{"id": r.Resources[0].ID, "name": r.Resources[0].Name, "owner": r.ID})
		case args[0] == "inspect":
			return []byte("1234"), nil
		case args[0] == "unshare":
			return state, nil
		default:
			return nil, nil
		}
	})}
	got, err := w.CaptureCredential(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(state) {
		t.Fatal("changed credential bytes")
	}
	if len(calls) != 6 || calls[2] != "pause "+r.Resources[0].ID || !strings.Contains(calls[4], "/proc/1234/root/run/codex/auth.json") || calls[5] != "kill --signal=KILL "+r.Resources[0].ID {
		t.Fatalf("unsafe credential return order: %v", calls)
	}
}
