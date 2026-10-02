package terminal

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func factoryLeaseFixture() identity.Lease {
	run := strings.Repeat("a", 32)
	container := strings.Repeat("b", 64)
	prep := "f" + strings.Repeat("c", 24)
	_, runDir, _, _ := domain.FactoryRunPaths(domain.RoleCoder, prep, run)
	return identity.Lease{
		ProviderID: identity.Codex, ID: "lease", ConnectionID: "subscription",
		Generation: 3, ActorID: 1001, ProjectID: "p" + strings.Repeat("d", 24),
		ExecutionID: run, Kind: identity.Factory,
		Binding: &identity.Binding{
			Kind: identity.Factory, ID: run, Project: container,
			Login: domain.RoleCoder, UID: 2001, GID: 2001, Scope: domain.FactoryScopeCodex,
			InvocationID: strings.Repeat("e", 32), CredentialRoot: runDir, Generation: 3, ChildID: prep,
		},
	}
}

func TestFactoryCodexBindingRequiresExactFields(t *testing.T) {
	if _, err := factoryCodexBinding(factoryLeaseFixture()); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*identity.Lease){
		"provider":   func(l *identity.Lease) { l.ProviderID = identity.Muse },
		"kind":       func(l *identity.Lease) { l.Binding.Kind = identity.Terminal },
		"scope":      func(l *identity.Lease) { l.Binding.Scope = "other" },
		"run":        func(l *identity.Lease) { l.Binding.ID = "short" },
		"container":  func(l *identity.Lease) { l.Binding.Project = "short" },
		"login":      func(l *identity.Lease) { l.Binding.Login = "root" },
		"uid":        func(l *identity.Lease) { l.Binding.UID = 0 },
		"invocation": func(l *identity.Lease) { l.Binding.InvocationID = "short" },
		"generation": func(l *identity.Lease) { l.Binding.Generation = 2 },
		"prep":       func(l *identity.Lease) { l.Binding.ChildID = "bad" },
		"root":       func(l *identity.Lease) { l.Binding.CredentialRoot = "/home/soda-coder/checkouts/other" },
	}
	for name, mutate := range mutations {
		l := factoryLeaseFixture()
		mutate(&l)
		if _, err := factoryCodexBinding(l); err == nil {
			t.Fatalf("binding with bad %s admitted", name)
		}
	}
	l := factoryLeaseFixture()
	l.Binding = nil
	if _, err := factoryCodexBinding(l); err == nil {
		t.Fatal("unbound lease admitted")
	}
}

func TestFactorySupervisorIsFixedEntrypoint(t *testing.T) {
	p := FactoryCodexPaths{
		RunDir: "/home/soda-coder/checkouts/f00/.soda-home/runs/a00",
		Prompt: "/home/soda-coder/checkouts/f00/.soda-home/runs/a00/prompt",
		Output: "/home/soda-coder/checkouts/f00/.soda-home/runs/a00/last-message.txt",
	}
	script := factorySupervisor(p, "/usr/local/bin/codex-factory-0.157.1", "")
	for _, want := range []string{
		"supervisor.pid", "marker", "started", "stop", "fail 42", "fail 43", "fail 44",
		"codex-factory-0.157.1", "exec --color never", "--sandbox danger-full-access",
		"--skip-git-repo-check", `model_reasoning_effort="low"`, "--output-last-message",
		"last-message.txt", "<", "/prompt",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("supervisor lost %q", want)
		}
	}
	if strings.Contains(script, "sudo") || strings.Contains(script, "podman") {
		t.Fatal("supervisor reaches outside its fixed entrypoint")
	}
}

func TestSystemdEscapeDoublesEveryDollar(t *testing.T) {
	if got := systemdEscape("echo $$ ${20} $X"); got != "echo $$$$ $${20} $$X" {
		t.Fatal("systemd escaping changed", got)
	}
	p := FactoryCodexPaths{RunDir: "/r", Prompt: "/r/prompt", Output: "/r/out"}
	escaped := systemdEscape(factorySupervisor(p, "/bin/codex", ""))
	for _, want := range []string{"$$$$", "$${20}", "$$RUNDIR", "$$?"} {
		if !strings.Contains(escaped, want) {
			t.Fatalf("escaped supervisor lost %q", want)
		}
	}
	if strings.Contains(factoryRetire(p), "$$") {
		t.Fatal("direct-exec retire script must keep single dollars")
	}
}

func TestFactoryRetireVerifiesBeforeKilling(t *testing.T) {
	p := FactoryCodexPaths{RunDir: "/home/soda-coder/checkouts/f00/.soda-home/runs/a00", PIDFile: "/home/soda-coder/checkouts/f00/.soda-home/runs/a00/supervisor.pid"}
	script := factoryRetire(p)
	for _, want := range []string{
		"stop", "supervisor.pid", "/proc/$PID/stat", "/proc/$PID/cmdline",
		"kill -KILL", "/proc/[0-9]*/stat", "lingering",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("retire script lost %q", want)
		}
	}
	if strings.Contains(script, "kill -KILL -- -1") || strings.Contains(script, "pkill") {
		t.Fatal("retire script kills outside the recorded group")
	}
}

func TestFactoryUnitShowParsing(t *testing.T) {
	show := parseFactoryUnitShow([]byte("ActiveState=active\nInvocationID=" + strings.Repeat("e", 32) + "\n"))
	if !show.active || show.invocation != strings.Repeat("e", 32) {
		t.Fatal("active unit misparsed", show)
	}
	show = parseFactoryUnitShow([]byte("ActiveState=inactive\nInvocationID=\n"))
	if show.active || show.invocation != "" {
		t.Fatal("inactive unit misparsed", show)
	}
}

func TestFactoryRoleID(t *testing.T) {
	id, err := factoryRoleID([]byte("2001\n"))
	if err != nil || id != 2001 {
		t.Fatal(err)
	}
	for _, bad := range []string{"0\n", "2001 2001\n", "a\n", ""} {
		if _, err = factoryRoleID([]byte(bad)); err == nil {
			t.Fatalf("role id %q admitted", bad)
		}
	}
}

func TestFactoryUnitNameMatchesDomain(t *testing.T) {
	unit, err := factoryUnitName(strings.Repeat("a", 32))
	if err != nil || unit != domain.FactoryUnitName(strings.Repeat("a", 32)) {
		t.Fatal("unit name diverged from domain", unit, err)
	}
	if _, err = factoryUnitName("short"); err == nil {
		t.Fatal("invalid run admitted a unit")
	}
}
