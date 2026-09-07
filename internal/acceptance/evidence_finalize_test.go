package acceptance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructuredEvidenceEscapesAndNumericIdentity(t *testing.T) {
	secret := "synthetic-\"credential\\with\nnewline"
	e := fixtureEvidence(t, []byte(secret))
	input := map[string]any{"secret": secret, "url": "https://example.test/path?code=hidden\"", "id": json.Number("9223372036854775807")}
	if err := e.WriteJSON("metadata.json", input); err != nil {
		t.Fatal(err)
	}
	raw, err := e.root.ReadFile("metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("hidden")) {
		t.Fatal("redirect query retained")
	}
	if string(result["id"]) != "9223372036854775807" {
		t.Fatal("integer identity changed")
	}
	var value string
	if err := json.Unmarshal(result["secret"], &value); err != nil || value != "[REDACTED]" {
		t.Fatal("escaped credential retained")
	}
	if err := e.CheckSecrets(); err != nil {
		t.Fatal(err)
	}
}

func TestEscapedCredentialsInRawSplitWrites(t *testing.T) {
	secret := "synthetic-\"credential\\line\nend"
	e := fixtureEvidence(t, []byte(secret))
	encoded, _ := json.Marshal(secret)
	w, err := e.Writer("raw")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range encoded {
		if _, err := w.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := e.root.ReadFile("raw")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, encoded[1:len(encoded)-1]) || !bytes.Contains(raw, []byte("[REDACTED]")) {
		t.Fatal("encoded secret not redacted")
	}
}

func TestFinalizationDoesNotPublishFailedOrOccupiedAttempts(t *testing.T) {
	for _, mode := range []string{"pending-collision", "leak", "final-collision"} {
		t.Run(mode, func(t *testing.T) {
			e := fixtureEvidence(t, []byte("synthetic-private-marker"))
			switch mode {
			case "pending-collision":
				if err := e.root.Mkdir("observation.pending.json", 0700); err != nil {
					t.Fatal(err)
				}
			case "leak":
				if err := e.root.WriteFile("unredacted", []byte("synthetic-private-marker"), 0600); err != nil {
					t.Fatal(err)
				}
			case "final-collision":
				if err := e.root.WriteFile("observation.json", []byte("earlier bytes"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := e.PublishObservation(Observation{Outcome: "completed"}); err == nil {
				t.Fatal("finalized failed attempt")
			}
			raw, err := e.root.ReadFile("observation.json")
			if mode == "final-collision" {
				if err != nil || string(raw) != "earlier bytes" {
					t.Fatal("overwrote previous record")
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("published success-shaped record")
			}
		})
	}
}

func TestEvidenceScansRetainOpenDirectoryAfterRename(t *testing.T) {
	e := fixtureEvidence(t, []byte("synthetic-secret"))
	if err := e.Write("safe", []byte("original")); err != nil {
		t.Fatal(err)
	}
	before, err := e.Hashes()
	if err != nil {
		t.Fatal(err)
	}
	old := e.Path()
	if err := os.Rename(old, old+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(old, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "safe"), []byte("synthetic-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.CheckSecrets(); err != nil {
		t.Fatal("scan followed replaced root")
	}
	after, err := e.Hashes()
	if err != nil || after["safe"] != before["safe"] {
		t.Fatal("hash followed replaced root")
	}
}

func TestHandoffShowsCleanupAndRejectsPendingRecord(t *testing.T) {
	e := fixtureEvidence(t)
	if err := e.Write("check", []byte("synthetic observation")); err != nil {
		t.Fatal(err)
	}
	files, err := e.Hashes()
	if err != nil {
		t.Fatal(err)
	}
	rev := strings.Repeat("a", 40)
	o := Observation{Owner: "P05", RequestedRevision: rev, RequestedArchitecture: "x86_64", Outcome: "failed", Execution: "failed", Evidence: "completed", Cleanup: "remote cleanup unknown", Invocation: []string{"transfer"}, Files: files, Artifacts: map[string]string{"bundle/build-info.json": strings.Repeat("b", 64)}}
	if err := e.PublishObservation(o); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Handoff(filepath.Join(dir, "pending.md"), "x86_64", rev, []string{filepath.Join(e.Path(), "observation.pending.json")}); err == nil {
		t.Fatal("accepted pending record")
	}
	out := filepath.Join(dir, "final.md")
	if err := Handoff(out, "x86_64", rev, []string{filepath.Join(e.Path(), "observation.json")}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "remote cleanup unknown") || !strings.Contains(string(raw), "bundle/build-info.json") {
		t.Fatal("missing cleanup/artifact context")
	}
}
