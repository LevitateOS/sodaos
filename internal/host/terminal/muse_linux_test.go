//go:build linux

package terminal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestMuseKernelIdentityContracts(t *testing.T) {
	id := strings.Repeat("a", 64)
	got, err := museProjectCgroup("0::/machine.slice/libpod-" + id + ".scope/container/system.slice/service")
	if err != nil || got != id {
		t.Fatalf("exact scope: %q %v", got, err)
	}
	for _, cg := range []string{"0::/prefix-libpod-" + id + ".scope", "0::/libpod-" + id + ".scope-extra"} {
		if _, err = museProjectCgroup(cg); err == nil {
			t.Fatal("substring admitted")
		}
	}
	if uid, err := museMappedUID("0 1000000 262144\n", 1001000); err != nil || uid != 1000 {
		t.Fatalf("mapped uid: %d %v", uid, err)
	}
	if _, err = museMappedUID("0 1000000 262144", 1000); err == nil {
		t.Fatal("unmapped UID admitted")
	}
}

func TestMuseProvisionedAccountMarker(t *testing.T) {
	valid := "0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:755:directory\n0:0:600:regular file\n"
	if !museAccountModes(valid) {
		t.Fatal("provisioned marker rejected")
	}
	if museAccountModes(strings.Replace(valid, "600:regular file", "600:symbolic link", 1)) || museAccountModes(strings.Replace(valid, "755:directory", "777:directory", 1)) {
		t.Fatal("unsafe account record admitted")
	}
}

func TestMuseCommandPreservesArgumentsAndRestrictsAuth(t *testing.T) {
	c := museCaller{Container: strings.Repeat("a", 64), Login: "soda-tester", Home: "/home/soda-tester", UID: 1000, GID: 1000}
	cmd := museCommand(context.Background(), c, identity.LaunchRequest{CWD: "/work with spaces", Args: []string{"prompt with spaces", "$(not-a-shell)"}}, "soda-muse-test.service", "/run/soda-muse/test")
	if cmd.Args[len(cmd.Args)-1] != "$(not-a-shell)" || cmd.Args[len(cmd.Args)-2] != "prompt with spaces" {
		t.Fatal("arguments transformed")
	}
	joined := strings.Join(cmd.Args, "\n")
	for _, part := range []string{"--pipe", "--working-directory=/work with spaces", "KillMode=control-group", "BindReadOnlyPaths=", "TBH_CREDENTIAL_BACKEND=file"} {
		if !strings.Contains(joined, part) {
			t.Fatal("missing contract", part)
		}
	}
	if strings.Contains(strings.Join(cmd.Env, "\n"), "META_API_KEY") {
		t.Fatal("auth override admitted")
	}
	if strings.Contains(joined, "container stop") {
		t.Fatal("project stop in execution launcher")
	}
}

func TestMuseStateCleanupBlocksUnavailableContainer(t *testing.T) {
	executor := &identityExecutor{existsErr: errors.New("native container unavailable")}
	runtime := &MuseRuntime{Exec: executor}
	binding := identity.Binding{Scope: "muse-project", ID: strings.Repeat("a", 32), Project: strings.Repeat("b", 64)}
	if err := runtime.cleanupExecutionState(context.Background(), binding); err != identity.ErrUncertain {
		t.Fatalf("cleanup uncertainty lost: %v", err)
	}
	if len(executor.calls) != 1 {
		t.Fatal("cleanup continued after uncertain container observation")
	}
}
