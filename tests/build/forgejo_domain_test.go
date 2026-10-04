// Port of test_forgejo_domain.py: soda-forgejo-domain CLI surface.
package build

import (
	"os"
	"strings"
	"testing"
)

func domainBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-forgejo-domain", "soda-forgejo-domain")
}

func runDomain(t *testing.T, argv ...string) ProcResult {
	t.Helper()
	return Run(t, RunOpt{}, domainBinary(t), argv...)
}

func TestForgejoDomainHelpReportsRecoveryVerbs(t *testing.T) {
	proc := runDomain(t, "--help")
	Check(t, proc.Code == 0, "exit = %d", proc.Code)
	Check(t, strings.Contains(proc.Stdout, "{stop,inhibit,status,lift,start}"), "stdout=%q", proc.Stdout)
}

func TestForgejoDomainMissingVerbRejected(t *testing.T) {
	proc := runDomain(t)
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	Check(t, strings.Contains(proc.Stderr, "the following arguments are required: verb"), "stderr=%q", proc.Stderr)
}

func TestForgejoDomainInvalidVerbRejected(t *testing.T) {
	proc := runDomain(t, "freeze")
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	Check(t, strings.Contains(proc.Stderr, "argument verb: invalid choice: 'freeze' (choose from stop, inhibit, status, lift, start)"),
		"stderr=%q", proc.Stderr)
}

func TestForgejoDomainNonrootRefusedWithoutEffects(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root passes the operator check")
	}
	proc := runDomain(t, "status")
	Check(t, proc.Code == 2, "exit = %d", proc.Code)
	lines := strings.Split(strings.TrimSpace(proc.Stderr), "\n")
	Check(t, lines[len(lines)-1] == "soda-forgejo-domain: error: native host operator/root required",
		"stderr=%q", proc.Stderr)
}
