package host

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tailnetexec "github.com/levitateos/sodaos/internal/host/tailnet"
	"github.com/levitateos/sodaos/internal/tailnet"
)

const preparationTestProject = "p0123456789abcdef01234567"

func preparationTestDaemon(t *testing.T, enabled func(context.Context, string, string) (bool, error), calls *int) *Daemon {
	t.Helper()
	cid := strings.Repeat("a", 64)
	exec := managementExec(func(_ context.Context, _ []byte, cmd string, args ...string) ([]byte, error) {
		*calls++
		if cmd != "/usr/bin/podman" {
			return nil, errors.New("synthetic-secret-unexpected-command-tskey-auth-synthetic")
		}
		joined := strings.Join(args, " ")
		// The lightweight project/container check succeeds so the Off
		// pre-check resolves; the full run observation below always
		// fails with secret-bearing native text that must never surface.
		if strings.Contains(joined, "org.soda.project") && strings.HasSuffix(joined, "soda-"+preparationTestProject) {
			return []byte(`{"id":` + `"` + cid + `"` + `,"running":true,"project":` + `"` + preparationTestProject + `"` + `,"owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`), nil
		}
		return nil, errors.New("synthetic-secret-podman-failure-tskey-auth-synthetic")
	})
	image := "sha256:" + strings.Repeat("d", 64)
	control := tailnet.NewProjectControl()
	d := testDaemonPtr(exec, Config{TailnetManagement: true, TailnetImage: image})
	d.Tailnet = control
	d.Companion = &tailnetexec.Companion{
		Exec:         exec,
		Tailnet:      control,
		Image:        image,
		EnabledCheck: enabled,
	}
	return d
}

func TestOffPolicyIsCleanNoOpWithoutRuntimeReadiness(t *testing.T) {
	calls := 0
	d := preparationTestDaemon(t, func(context.Context, string, string) (bool, error) { return false, nil }, &calls)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cid, e := d.StartTailnet(ctx, preparationTestProject)
	if e != nil || cid != "" {
		t.Fatal("Off policy was not a clean no-op", cid, e)
	}
	if calls != 1 {
		t.Fatal("Off policy required companion runtime readiness", calls)
	}
}

func TestMalformedPolicyFailsSafelyBeforeRuntime(t *testing.T) {
	calls := 0
	d := preparationTestDaemon(t, func(context.Context, string, string) (bool, error) { return false, tailnet.ErrUnavailable }, &calls)
	_, e := d.StartTailnet(t.Context(), preparationTestProject)
	if !errors.Is(e, tailnet.ErrUnavailable) || !strings.Contains(e.Error(), "Tailnet policy") {
		t.Fatal("malformed policy did not fail at its stage", e)
	}
	if e != nil && strings.Contains(e.Error(), "synthetic-secret") {
		t.Fatal("native text escaped stage diagnostics", e)
	}
	if calls != 1 {
		t.Fatal("malformed policy entered runtime preparation", calls)
	}
}

func TestEnabledProjectRunFailureIdentifiesItsStage(t *testing.T) {
	calls := 0
	d := preparationTestDaemon(t, func(context.Context, string, string) (bool, error) { return true, nil }, &calls)
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	_, e := d.StartTailnet(ctx, preparationTestProject)
	if e == nil {
		t.Fatal("incompatible runtime accepted")
	}
	if !errors.Is(e, tailnet.ErrUnavailable) || !strings.Contains(e.Error(), "project runtime") {
		t.Fatal("enabled failure did not identify its stage", e)
	}
	if strings.Contains(e.Error(), "synthetic-secret") || strings.Contains(e.Error(), "tskey-auth") {
		t.Fatal("native or credential text escaped stage diagnostics", e)
	}
	if calls < 2 {
		t.Fatal("enabled path skipped runtime validation", calls)
	}
}
