// Port of test_project_foundation.py: native development foundation guards.
package build

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestFoundationRecipeDeclaresNativeDevelopmentFoundation(t *testing.T) {
	recipe := strings.ReplaceAll(ReadFile(t, "project-os/Containerfile"), "\\\n", " ")
	match := regexp.MustCompile(`RUN dnf -y --enablerepo=crb install (.*?) && dnf clean all`).FindStringSubmatch(recipe)
	Require(t, match != nil, "install RUN not found")
	packages := map[string]bool{}
	for _, pkg := range strings.Fields(match[1]) {
		packages[pkg] = true
	}
	for _, want := range []string{
		"gcc", "gcc-c++", "glibc-devel", "libstdc++-devel", "make", "cmake",
		"ninja-build", "pkgconf-pkg-config", "binutils", "gdb", "strace",
		"openssl-devel", "zlib-devel", "rsync", "iproute", "iputils",
		"bind-utils", "lsof", "jq",
	} {
		Check(t, packages[want], "package %s missing", want)
	}
	Check(t, !strings.Contains(recipe, "--nogpgcheck"), "nogpgcheck present")
	Check(t, !strings.Contains(recipe, "dnf upgrade"), "dnf upgrade present")
}

func TestFoundationInstalledProbeRequiresScopeBeforeWrites(t *testing.T) {
	script := filepath.Join(RepoRoot, "tests/installed/project-foundation.sh")
	result := Run(t, RunOpt{}, "/bin/sh", "-n", script)
	Require(t, result.Code == 0, "sh -n failed: %s", result.Stderr)
	defpath := Run(t, RunOpt{}, "getconf", "PATH")
	Require(t, defpath.Code == 0, "cannot read default PATH")
	temporary := TempDir(t)
	run := Run(t, RunOpt{Env: []string{
		"PATH=" + strings.TrimSpace(defpath.Stdout),
		"TMPDIR=" + temporary,
	}}, "/bin/sh", script)
	Check(t, run.Code != 0, "probe succeeded without scope")
	Check(t, strings.Contains(run.Stderr, "SODA_NATIVE_VALIDATE"), "stderr=%q", run.Stderr)
	entries, err := os.ReadDir(temporary)
	Require(t, err == nil, "read tempdir: %v", err)
	Check(t, len(entries) == 0, "probe wrote %d entries", len(entries))
	source := ReadFile(t, "tests/installed/project-foundation.sh")
	Check(t, strings.Contains(source, `test "$(id -u)" != 0`), "missing root refusal")
	Check(t, strings.Contains(source, `HOME="$work/home"`), "missing HOME sandbox")
	Check(t, strings.Contains(source, "OpenSSL::Crypto ZLIB::ZLIB"), "missing link check")
	Check(t, strings.Contains(source, "--return-child-result"), "missing child result")
	Check(t, !strings.Contains(source, "sudo "), "sudo present")
	Check(t, !strings.Contains(source, "rm -"), "rm present")
}
