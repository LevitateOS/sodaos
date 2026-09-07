package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureEvidence(t *testing.T, secrets ...[]byte) *Evidence {
	t.Helper()
	e, err := CreateEvidence(filepath.Join(t.TempDir(), "evidence"), secrets)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}
func TestEvidenceSplitSecretsAndRedirectQueries(t *testing.T) {
	e := fixtureEvidence(t, []byte("synthetic-password"))
	w, err := e.Writer("out")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"synthetic-", "pass", "word\nhttps://example.test/callback?co", "de=unknown-code&state=unknown-state\n"} {
		if _, err = w.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(e.Path(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, bad := range []string{"synthetic-password", "unknown-code", "unknown-state"} {
		if strings.Contains(text, bad) {
			t.Fatalf("retained unsafe output: %q", text)
		}
	}
	if !strings.Contains(text, "https://example.test/callback") {
		t.Fatal(text)
	}
	if err = e.CheckSecrets(); err != nil {
		t.Fatal(err)
	}
}
func TestEvidenceExclusiveAndConfined(t *testing.T) {
	e := fixtureEvidence(t)
	if err := e.Write("once", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := e.Write("once", []byte("second")); !errors.Is(err, os.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	for _, name := range []string{"../escape", "/absolute", "a/../b", "."} {
		if _, err := e.Writer(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(e.Path(), "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Writer("link/out"); err == nil {
		t.Fatal("followed evidence symlink")
	}
	st, err := os.Stat(e.Path())
	if err != nil || st.Mode().Perm() != 0700 {
		t.Fatalf("private root: %v", err)
	}
	st, err = os.Stat(filepath.Join(e.Path(), "once"))
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("private file: %v", err)
	}
}
func TestRedactedErrorRetainsIdentity(t *testing.T) {
	e := fixtureEvidence(t, []byte("synthetic-password"))
	sentinel := errors.New("underlying")
	err := e.RedactError(fmt.Errorf("synthetic-password https://example.test/?code=hidden: %w", sentinel))
	if !errors.Is(err, sentinel) || strings.Contains(err.Error(), "synthetic-password") || strings.Contains(err.Error(), "code=") {
		t.Fatal(err)
	}
}
func TestCommandAndEvidenceFailuresAreSeparate(t *testing.T) {
	if err := ownedGroupsSupported(); err != nil {
		t.Skip(err)
	}
	e := fixtureEvidence(t, []byte("synthetic-password"))
	result, err := Execute(context.Background(), e, "denied", Command{Name: "/bin/sh", Args: []string{"-c", `read -r value; printf '%s\n' "$value"; printf 'expected denial\n' >&2; exit 23`}, Stdin: strings.NewReader("synthetic-password\n")})
	var exit *exec.ExitError
	if err != nil || !result.Started || !errors.As(result.Err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("execution=%v evidence=%v", result.Err, err)
	}
	if strings.Contains(string(result.Stdout), "synthetic-password") {
		t.Fatal("returned raw diagnostic")
	}
	if err = e.Write("collision.stdout", nil); err != nil {
		t.Fatal(err)
	}
	result, err = Execute(context.Background(), e, "collision", Command{Name: "/bin/sh", Args: []string{"-c", "exit 0"}})
	if err == nil || result.Started || result.Err != nil {
		t.Fatalf("evidence failure became an execution result: %#v %v", result, err)
	}
}
func TestCancelledCommandIsNotDenialOrSuccess(t *testing.T) {
	e := fixtureEvidence(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := Execute(ctx, e, "cancelled", Command{Name: "/bin/sh", Args: []string{"-c", "exit 0"}})
	if err != nil || r.Started || !errors.Is(r.Err, context.Canceled) {
		t.Fatalf("%#v %v", r, err)
	}
}
func TestEvidenceOutputBound(t *testing.T) {
	e := fixtureEvidence(t)
	w, err := e.Writer("large")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(make([]byte, evidenceLimit+1))
	if err == nil {
		t.Fatal("unbounded output accepted")
	}
	if err = w.Close(); err == nil {
		t.Fatal("retention failure discarded")
	}
}
