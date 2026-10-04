// Port of test_project_runtime.py: native project service/secret contracts.
package build

import (
	"regexp"
	"strings"
	"testing"
)

// parseUnit reads a systemd unit like configparser (case-insensitive keys,
// no interpolation): sections, key=value pairs, # and ; comments.
func parseUnit(t *testing.T, name string) map[string]map[string]string {
	t.Helper()
	units := map[string]map[string]string{}
	section := ""
	for _, raw := range strings.Split(ReadFile(t, "project-os/rootfs/etc/systemd/system/"+name), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			units[section] = map[string]string{}
			continue
		}
		Require(t, section != "", "%s: option outside section: %q", name, line)
		eq, colon := strings.Index(line, "="), strings.Index(line, ":")
		sep := eq
		if eq < 0 || (colon >= 0 && colon < eq) {
			sep = colon
		}
		Require(t, sep > 0, "%s: bad option line: %q", name, line)
		units[section][strings.ToLower(strings.TrimSpace(line[:sep]))] = strings.TrimSpace(line[sep+1:])
	}
	return units
}

func TestRuntimeCreationImageIdentityHasRecipeAndOSGuard(t *testing.T) {
	recipe := ReadFile(t, "project-os/Containerfile")
	Check(t, strings.Contains(recipe, `org.soda.profile="rocky-headless"`), "missing profile label")
	Check(t, strings.Contains(recipe, `org.soda.interface="headless"`), "missing interface label")
	Check(t, strings.Contains(recipe, `RUN . /etc/os-release && test "$ID:$VERSION_ID" = "rocky:10.2"`), "missing os guard")
	build := strings.ReplaceAll(ReadFile(t, "internal/release/build/production.go"), " ", "")
	Check(t, strings.Contains(build, `"--label=org.opencontainers.image.revision="+p.Revision`), "missing revision label")
	Check(t, !strings.Contains(recipe, "fedora-kde"), "fedora-kde present")
}

func TestRuntimeServiceUsesLocalEngineAndActivationFD(t *testing.T) {
	unit := parseUnit(t, "soda-podman.service")
	service := unit["Service"]
	Check(t, service["unsetenvironment"] == "CONTAINER_HOST", "UnsetEnvironment=%q", service["unsetenvironment"])
	Check(t, service["user"] == "root", "User=%q", service["user"])
	Check(t, service["group"] == "root", "Group=%q", service["group"])
	_, present := service["environment"]
	Check(t, !present, "Environment present")
	Check(t, service["execstart"] == "/usr/bin/podman system service --time=0", "ExecStart=%q", service["execstart"])
	requires := strings.Fields(unit["Unit"]["requires"])
	found := false
	for _, req := range requires {
		found = found || req == "soda-podman.socket"
	}
	Check(t, found, "Requires=%q", unit["Unit"]["requires"])
	_, present = service["runtimedirectory"]
	Check(t, present == false, "RuntimeDirectory present")
}

func TestRuntimeSocketIsProjectAdminOnly(t *testing.T) {
	unit := parseUnit(t, "soda-podman.socket")
	socket := unit["Socket"]
	Check(t, socket["listenstream"] == "/run/soda-podman/podman.sock", "ListenStream=%q", socket["listenstream"])
	Check(t, socket["socketuser"] == "root", "SocketUser=%q", socket["socketuser"])
	Check(t, socket["socketgroup"] == "wheel", "SocketGroup=%q", socket["socketgroup"])
	Check(t, socket["socketmode"] == "0660", "SocketMode=%q", socket["socketmode"])
	Check(t, !strings.Contains(unit["Unit"]["after"], "soda-project-init.service"), "socket After init")
	Check(t, !strings.Contains(unit["Unit"]["requires"], "soda-project-init.service"), "socket Requires init")
	service := parseUnit(t, "soda-podman.service")["Unit"]
	for _, field := range []string{"requires", "after"} {
		found := false
		for _, entry := range strings.Fields(service[field]) {
			found = found || entry == "soda-project-init.service"
		}
		Check(t, found, "service %s=%q", field, service[field])
	}
	Check(t, socket["directorymode"] == "0750", "DirectoryMode=%q", socket["directorymode"])
	init := ReadFile(t, "project-os/rootfs/usr/libexec/soda/project-init")
	Check(t, strings.Contains(init, "install -d -m 0750 -o root -g wheel /run/soda-podman"), "missing runtime dir install")
	recipe := ReadFile(t, "project-os/Containerfile")
	Check(t, strings.Contains(recipe, "systemctl enable sshd.service soda-project-init.service soda-podman.socket"), "missing enable")
}

func TestRuntimeOnlyNetworkSysctlSubtreeIsRebound(t *testing.T) {
	init := ReadFile(t, "project-os/rootfs/usr/libexec/soda/project-init")
	Check(t, strings.Contains(init, "mount -t proc -o nosuid,nodev,noexec proc /run/soda-net-proc"), "missing proc mount")
	Check(t, strings.Contains(init, "mount --bind /run/soda-net-proc/sys/net /proc/sys/net"), "missing bind")
	Check(t, !strings.Contains(init, "remount,rw /proc/sys"), "remount present")
	Check(t, strings.Index(init, "mount --bind") < strings.Index(init, "touch /run/soda-project-ready"), "bind after ready")
}

func TestRuntimeSharedToolsUsesLiveIsolationAddress(t *testing.T) {
	probe := ReadFile(t, "tests/installed/shared-tools.sh")
	Check(t, strings.Contains(probe, "${ISOLATION_IP:?Live second-project address required}"), "missing isolation guard")
	Check(t, strings.Contains(probe, `"$BOB@$ISOLATION_IP"`), "missing isolation ssh")
	Check(t, !strings.Contains(probe, "$BOB@10.89.0.3"), "static address present")
}

func TestRuntimeComposeDatabaseImageIsDigestPinned(t *testing.T) {
	var images []string
	for _, line := range strings.Split(ReadFile(t, "tests/fixtures/workload/compose.yaml"), "\n") {
		if strings.Contains(line, "image:") {
			images = append(images, strings.TrimSpace(strings.SplitN(line, "image:", 2)[1]))
		}
	}
	Require(t, len(images) == 1, "image entries = %d", len(images))
	matched, err := regexp.MatchString(`^docker\.io/library/postgres:17@sha256:[0-9a-f]{64}$`, images[0])
	Require(t, err == nil, "regex: %v", err)
	Check(t, matched, "compose image %q is not digest-pinned", images[0])
}

func TestRuntimeComposeUsesNativeSecretNotPasswordArgv(t *testing.T) {
	compose := ReadFile(t, "tests/fixtures/workload/compose.yaml")
	Check(t, strings.Contains(compose, "POSTGRES_PASSWORD_FILE: /run/secrets/soda-example-db"), "missing password file")
	Check(t, !strings.Contains(compose, "POSTGRES_PASSWORD:"), "argv password present")
	Check(t, !strings.Contains(compose, "${EXAMPLE_DB_PASSWORD"), "password interpolation present")
	Check(t, strings.Contains(compose, "external: true"), "secret not external")
	Check(t, strings.Contains(compose, `mode: "0400"`), "secret mode wrong")
}
