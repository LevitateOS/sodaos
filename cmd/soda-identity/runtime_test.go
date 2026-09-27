package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestForgejoCannotUseFactoryWorkspaceRuntime(t *testing.T) {
	lease, log := fakeFactoryRuntime(t, "still-running")
	lease.ProviderID = identity.Forgejo
	if err := (nativeRuntime{}).Validate(t.Context(), lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("workspace admitted as Git execution", err)
	}
	if err := (nativeRuntime{}).Stop(t.Context(), lease); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("workspace accepted as stopped Git execution", err)
	}
	if _, err := os.Stat(log); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Git operation touched factory workspace", err)
	}
}

func TestFactoryStopRejectsUncertainOrForeignBoundary(t *testing.T) {
	for _, scenario := range []string{"engine-failed", "foreign-label", "still-running"} {
		t.Run(scenario, func(t *testing.T) {
			lease, log := fakeFactoryRuntime(t, scenario)
			if err := (nativeRuntime{}).Stop(context.Background(), lease); err == nil {
				t.Fatal("unconfirmed or foreign execution accepted as stopped")
			}
			if scenario != "still-running" && strings.Contains(readRuntimeLog(t, log), "kill") {
				t.Fatal("unattested execution was killed")
			}
		})
	}
}

func TestFactoryStopConfirmsExactAbsence(t *testing.T) {
	lease, log := fakeFactoryRuntime(t, "absent")
	if err := (nativeRuntime{}).Stop(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readRuntimeLog(t, log), "kill") {
		t.Fatal("absence caused a mutation")
	}
	if err := (nativeRuntime{}).Validate(context.Background(), lease); err == nil {
		t.Fatal("absent execution admitted")
	}
}

func fakeFactoryRuntime(t *testing.T, scenario string) (identity.Lease, string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "commands")
	id := strings.Repeat("a", 64)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SODA_TEST_SCENARIO", scenario)
	t.Setenv("SODA_TEST_COMMANDS", log)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$SODA_TEST_COMMANDS"
shift
case "$1" in
container)
  case "$SODA_TEST_SCENARIO" in absent) exit 1;; engine-failed) exit 125;; esac
  exit 0;;
inspect)
  owner=execution
  test "$SODA_TEST_SCENARIO" != foreign-label || owner=other-execution
  printf '{"id":"%s","owner":"%s","running":true}\n' "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" "$owner"
  exit 0;;
kill) exit 0;;
esac
exit 125
`
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return identity.Lease{Kind: identity.Factory, ExecutionID: "execution", Binding: &identity.Binding{Kind: identity.Factory, ID: id}}, log
}

func readRuntimeLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
