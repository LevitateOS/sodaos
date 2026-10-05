package main

// Cockpit account gate coverage for the installed-probes suite: CLI
// refusal shape plus the behavior contracts owned by the Rust probe.
// The retired tests/installed/cockpit-account.py asserted the same
// refusals; success output is byte-identical between the two.

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCockpitAccountRefusesOutsideNativeRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters the PAM path")
	}
	probe := remoteProbeBinary(t)
	for _, env := range [][]string{nil, {"SODA_NATIVE_VALIDATE=wrong"}} {
		result := runProbe(t, env, 30*time.Second, probe, "cockpit-account")
		checkProbe(t, result.code == 1, "exit = %d", result.code)
		checkProbe(t, !strings.HasPrefix(result.stdout, "Cockpit PAM gate"), "stdout=%q", result.stdout)
		checkProbe(t, result.stderr != "", "empty stderr")
	}
}

func TestCockpitAccountContracts(t *testing.T) {
	source := readProbeSource(t, "rust/soda-acceptance/src/cockpit.rs")
	for _, want := range []string{
		"explicit native root target required",
		`account_uid("root")? != 0`,
		`account_uid("nobody")? == 0`,
		"pam_acct_mgmt",
		"no_prompt",
		`account_phase(&pam, "root", true)`,
		`account_phase(&pam, "nobody", false)`,
		"code == 6 || code == 7",
		"(pam.end)(handle, code)",
	} {
		checkProbe(t, strings.Contains(source, want), "missing contract %q", want)
	}
	driver := readProbeSource(t, "rust/soda-acceptance/src/bin/soda-acceptance-remote.rs")
	checkProbe(t, strings.Contains(driver, "SODA_NATIVE_VALIDATE"), "missing env contract")
}
