package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffPreservesMissingAndFailedScopes(t *testing.T) {
	e := fixtureEvidence(t)
	if err := e.Write("check.stdout", []byte("synthetic failed observation\n")); err != nil {
		t.Fatal(err)
	}
	hashes, err := e.Hashes()
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	o := Observation{Owner: "P06", RequestedRevision: revision, RequestedArchitecture: "x86_64", Target: "synthetic", Outcome: "failed", Execution: "failed", Evidence: "completed", Files: hashes}
	data, _ := json.Marshal(o)
	if err = e.Write("observation.json", data); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(parent, "handoff.md")
	record := filepath.Join(e.Path(), "observation.json")
	if err = Handoff(out, "x86_64", revision, []string{record}); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"failed", "U20 owns it", "U08: no observation supplied", "P09/P10 remain not selected"} {
		if !strings.Contains(string(text), required) {
			t.Fatalf("missing %q", required)
		}
	}
	if err = Handoff(filepath.Join(parent, "wrong.md"), "aarch64", revision, []string{record}); err == nil {
		t.Fatal("mixed sibling evidence")
	}
	if err = os.WriteFile(filepath.Join(e.Path(), "check.stdout"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = Handoff(filepath.Join(parent, "changed.md"), "x86_64", revision, []string{record}); err == nil {
		t.Fatal("accepted changed evidence")
	}
}
