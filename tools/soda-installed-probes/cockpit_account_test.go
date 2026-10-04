package main

// Port of tests/installed/cockpit-account.py into the installed-probes
// suite: the real PAM gate probe, executed (refusal paths) and pinned.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCockpitAccountRefusesWithoutNativeRootTarget(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root passes the operator check")
	}
	script := filepath.Join(probeRoot, "tests/installed/cockpit-account.py")
	hostname, err := os.Hostname()
	requireProbe(t, err == nil, "hostname: %v", err)
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"default env", nil},
		{"declared native target", append(os.Environ(), "SODA_NATIVE_VALIDATE="+hostname)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := runProbe(t, tc.env, 30*time.Second, "python3", script)
			checkProbe(t, result.code == 1, "exit = %d", result.code)
			checkProbe(t, result.stdout == "", "stdout=%q", result.stdout)
			checkProbe(t, result.stderr == "explicit native root target required\n", "stderr=%q", result.stderr)
		})
	}
}

func TestCockpitAccountPAMGateContracts(t *testing.T) {
	source := readProbeSource(t, "tests/installed/cockpit-account.py")
	for _, want := range []string{
		"SODA_NATIVE_VALIDATE",
		"platform.node()",
		"pwd.getpwnam('root').pw_uid == 0",
		"pwd.getpwnam('nobody').pw_uid != 0",
		"pam_acct_mgmt",
		"PAM_CONV_ERR",
		"[('root', True), ('nobody', False)]",
		"code in (6, 7)",
		"pam_end(handle, code)",
	} {
		checkProbe(t, strings.Contains(source, want), "missing contract %q", want)
	}
}
