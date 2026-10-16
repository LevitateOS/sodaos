package build

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	dockerInstallerTag      = "v0.2.0-docker-fixture"
	dockerInstallerSource   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	dockerInstallerManifest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type dockerInstallerFixture struct {
	dir, install, log string
	metadata          map[string]any
	env               []string
	noTTY             bool
}

func newDockerInstallerFixture(t *testing.T) *dockerInstallerFixture {
	t.Helper()
	dir := t.TempDir()
	f := &dockerInstallerFixture{
		dir: dir, install: filepath.Join(dir, "installation"), log: filepath.Join(dir, "docker.log"),
		env: os.Environ(),
	}
	compose := []byte(ReadFile(t, "system/docker/compose.yaml"))
	identity := func(data []byte) map[string]any {
		return map[string]any{"SHA256": fmt.Sprintf("%x", sha256.Sum256(data)), "Bytes": len(data)}
	}
	hash := strings.Repeat("c", 64)
	base := "registry.fedoraproject.org/fedora@sha256:" + hash
	f.metadata = map[string]any{
		"Format": 1, "Distribution": "docker", "Purpose": "production", "ReleaseTag": dockerInstallerTag,
		"Architecture": "x86_64", "Platform": "linux/amd64", "SodaRevision": dockerInstallerSource,
		"ForgejoRevision": strings.Repeat("d", 40), "CandidateSHA256": hash, "PayloadSHA256": hash,
		"HostManifest": "sha256:" + hash, "HostArchiveSHA256": hash, "FedoraBase": base,
		"SodaSource": identity([]byte("source")), "ForgejoSource": identity([]byte("fountain")),
		"Recipe": identity([]byte("recipe")), "ImageArchive": identity([]byte("archive")),
		"Packages": identity([]byte("packages")), "Compose": identity(compose),
		"Image": map[string]any{
			"Manifest": dockerInstallerManifest, "Config": "sha256:" + hash, "Architecture": "amd64",
			"Revision": dockerInstallerSource, "Source": "https://github.com/LevitateOS/sodaos",
			"BaseName": base, "BaseDigest": "sha256:" + hash,
		},
	}
	WriteFile(t, filepath.Join(dir, "sodaos-docker-compose.yaml"), compose, 0o600)
	f.writeMetadata(t)
	f.writeRelease(t, false, true)
	WriteFile(t, filepath.Join(dir, "docker-info.json"), []byte(`{"OSType":"linux","Architecture":"x86_64","CgroupVersion":"2","OperatingSystem":"Linux","SecurityOptions":["name=seccomp"]}`), 0o600)
	for _, name := range []string{"curl", "docker", "uname", "ip", "sysctl", "ss", "install", "sudo"} {
		WriteFile(t, filepath.Join(dir, "bin", name), []byte("#!/bin/bash\nset -euo pipefail\n"+dockerInstallerCommands), 0o700)
	}
	for key, value := range map[string]string{
		"PATH":   filepath.Join(dir, "bin") + ":" + os.Getenv("PATH"),
		"TMPDIR": dir, "SODA_INSTALLER_FIXTURE": dir,
	} {
		f.env = SetEnv(f.env, key, value)
	}
	return f
}

// Only external host/download observations are replaced. JSON decoding, hashes,
// permissions, installation files and streamed Bash execution remain real.
const dockerInstallerCommands = `root=$SODA_INSTALLER_FIXTURE
case ${0##*/} in
  uname)
    case $1 in -s) echo "${FIXTURE_OS:-Linux}";; -m) echo "${FIXTURE_ARCH:-x86_64}";; -r) echo "${FIXTURE_KERNEL:-6.13-fixture}";; *) exit 91;; esac ;;
  curl)
    url= out= ca=
    printf '%s\n' "$*" >> "$root/curl.log"
    while (( $# )); do
      case $1 in
        https://*) url=$1; shift ;;
        -o) out=$2; shift 2 ;;
        --cacert) ca=$2; shift 2 ;;
        --proto|--proto-redir|--connect-timeout|--max-time|--max-filesize) shift 2 ;;
        --fail|--silent|--show-error|--location) shift ;;
        *) echo "Unexpected curl argument $1" >&2; exit 91 ;;
      esac
    done
    case $url in
      https://192.168.50.10/api/healthz)
        [[ -n $ca && -f $ca ]] && cmp "$ca" "$root/root.crt" || exit 91
        [[ ${FIXTURE_HTTPS_FAILURE:-} != true ]] || exit 60
        exit 0 ;;
      https://api.github.com/repos/LevitateOS/sodaos/releases/tags/"$FIXTURE_RELEASE_TAG") asset=release.json ;;
      https://github.com/LevitateOS/sodaos/releases/download/"$FIXTURE_RELEASE_TAG"/docker.json) asset=docker.json ;;
      https://github.com/LevitateOS/sodaos/releases/download/"$FIXTURE_RELEASE_TAG"/sodaos-docker-compose.yaml) asset=sodaos-docker-compose.yaml ;;
      *) echo "Unexpected download $url" >&2; exit 91 ;;
    esac
    printf '%s\n' "$url" >> "$root/downloads.log"
    if [[ ${FIXTURE_FAILED_DOWNLOAD:-} == "$asset" ]]; then printf truncated > "$out"; exit 22; fi
    cp "$root/$asset" "$out" ;;
  docker)
    anonymous=false
    if [[ $1 == --config ]]; then
      [[ -d $2 && ! -e $2/config.json ]] || exit 91
      anonymous=true
      shift 2
    fi
    [[ $1 == --host && $2 == unix:///var/run/docker.sock ]] || exit 91
    shift 2
    printf 'docker %s\n' "$*" >> "$root/docker.log"
    case "$*" in
      'info --format {{json .}}')
        [[ ${FIXTURE_NEEDS_SUDO:-} != true || ${FIXTURE_USING_SUDO:-} == true ]] || exit 1
        cat "$root/docker-info.json" ;;
      'compose version --short') echo "${FIXTURE_COMPOSE_VERSION:-2.39.1}" ;;
      'ps -aq --filter label=com.docker.compose.project=sodaos')
        if [[ ${FIXTURE_RETAINED_CONTAINER:-} == true || -f $root/runtime-active ]]; then echo retained; fi ;;
      'volume inspect '*)
        [[ ${FIXTURE_RETAINED_VOLUME:-} == true || -f $root/runtime-active ]] || exit 1
        [[ ${FIXTURE_MISSING_VOLUME:-} != all && ${FIXTURE_MISSING_VOLUME:-} != "${3#sodaos_}" ]] || exit 1
        jq -n --arg volume "${3#sodaos_}" --arg project "${FIXTURE_VOLUME_OWNER:-sodaos}" '[{Labels:{"com.docker.compose.project":$project,"com.docker.compose.volume":$volume}}]' ;;
      'container inspect retained')
        jq --arg image "${FIXTURE_CONTAINER_IMAGE:-}" '[{
          Config:{Image:(if $image == "" then "ghcr.io/levitateos/sodaos@" + .Image.Manifest else $image end),
            Labels:{"com.docker.compose.project":"sodaos","com.docker.compose.service":"soda"}},
          Image:.Image.Config, HostConfig:{Privileged:true,NetworkMode:"host"},
          Mounts:[{Type:"volume",Name:"sodaos_config",Destination:"/etc"},
            {Type:"volume",Name:"sodaos_home",Destination:"/home"},{Type:"volume",Name:"sodaos_state",Destination:"/var"}]
        }]' "$root/docker.json" ;;
      'network ls --quiet') : ;;
      'pull --platform linux/amd64 ghcr.io/levitateos/sodaos@sha256:'*)
        [[ $anonymous == true && ${FIXTURE_PULL_FAILURE:-} != true ]] ;;
      'image inspect '*)
        jq '[{Id: .Image.Config, Os: "linux", Architecture: "amd64", RepoDigests: [("ghcr.io/levitateos/sodaos@" + .Image.Manifest)], Config: {Labels: {"org.opencontainers.image.revision": .SodaRevision}}}]' "$root/docker.json" ;;
      compose*)
        case " $* " in
          *' config --quiet '*) [[ ${FIXTURE_BAD_COMPOSE:-} != true ]] ;;
          *' pull '*) [[ ${FIXTURE_PULL_FAILURE:-} != true ]] ;;
          *' up -d --no-recreate --wait --wait-timeout 1800 '*)
            [[ ${FIXTURE_NOT_READY:-} != true ]] || exit 1
            touch "$root/runtime-active" ;;
          *' ps '*) : ;;
          *' exec -T soda bash -c '*)
            if [[ -f $root/activated ]]; then echo activated; else echo new; fi ;;
          *' exec -T soda systemctl is-active --quiet '*) [[ ${FIXTURE_FAILED_BROWSER_UNIT:-} != "${!#}" ]] ;;
          *' cp soda:/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt '*) cp "$root/root.crt" "${!#}" ;;
          *) echo "Unexpected Compose command $*" >&2; exit 91 ;;
        esac ;;
      *) echo "Unexpected Docker command $*" >&2; exit 91 ;;
    esac ;;
  ip)
    case "$*" in
      '-j -4 address show') printf '[{"addr_info":[{"local":"192.168.50.10"}]}]\n' ;;
      '-j -4 route show table all') printf '[{"dst":"%s"}]\n' "${FIXTURE_ROUTE:-192.168.50.0/24}" ;;
      *) exit 91 ;;
    esac ;;
  sysctl) echo "${FIXTURE_FORWARDING:-1}" ;;
  ss) if [[ ${FIXTURE_PORT_BUSY:-} == true ]]; then echo 'LISTEN 0 128 0.0.0.0:443 0.0.0.0:*'; fi ;;
  install)
    if [[ ${FIXTURE_INSTALL_FAILURE:-} == true && ${!#} == */compose.yaml ]]; then
      printf partial > "${!#}"
      exit 73
    fi
    command -p install "$@" ;;
  sudo)
    printf '%s\n' "$*" >> "$root/sudo.log"
    if [[ $* == '-n -v' ]]; then exit 0; fi
    [[ $1 == --preserve-env=SODA_IMAGE,SODA_PROJECT_SUBNET && $2 == docker ]] || exit 91
    shift
    FIXTURE_USING_SUDO=true exec "$@" ;;
  *) exit 91 ;;
esac
`

func (f *dockerInstallerFixture) writeRelease(t *testing.T, prerelease, qualified bool) {
	t.Helper()
	body := ""
	if qualified {
		body = fmt.Sprintf("<!-- soda-docker-qualified: %s %s %s -->", f.metadata["ReleaseTag"], dockerInstallerSource, dockerInstallerManifest)
	}
	release := map[string]any{
		"tag_name": f.metadata["ReleaseTag"], "draft": false, "immutable": true, "prerelease": prerelease, "body": body,
		"assets": []any{map[string]string{"name": "docker.json"}, map[string]string{"name": "sodaos-docker-compose.yaml"}, map[string]string{"name": "install-docker.sh"}},
	}
	data, err := json.Marshal(release)
	Require(t, err == nil, "marshal release: %v", err)
	WriteFile(t, filepath.Join(f.dir, "release.json"), data, 0o600)
}

func (f *dockerInstallerFixture) writeMetadata(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(f.metadata)
	Require(t, err == nil, "marshal Docker metadata: %v", err)
	WriteFile(t, filepath.Join(f.dir, "docker.json"), data, 0o600)
}

func (f *dockerInstallerFixture) script(t *testing.T) string {
	t.Helper()
	f.env = SetEnv(f.env, "FIXTURE_RELEASE_TAG", f.metadata["ReleaseTag"].(string))
	metadata, err := os.ReadFile(filepath.Join(f.dir, "docker.json"))
	Require(t, err == nil, "read Docker metadata: %v", err)
	script := strings.NewReplacer(
		"@RELEASE_TAG@", f.metadata["ReleaseTag"].(string),
		"@SOURCE_REVISION@", dockerInstallerSource,
		"@METADATA_SHA256@", fmt.Sprintf("%x", sha256.Sum256(metadata)),
		"@METADATA_BYTES@", fmt.Sprint(len(metadata)),
	).Replace(ReadFile(t, "system/docker/install-docker.sh"))
	path := filepath.Join(f.dir, "install-docker.sh")
	WriteFile(t, path, []byte(script), 0o600)
	return path
}

func (f *dockerInstallerFixture) run(t *testing.T, script string, args ...string) ProcResult {
	t.Helper()
	command := []string{
		"-o", "pipefail", "-c", `cat "$1" | /bin/bash -s -- "${@:2}"`, "installer-fixture", script,
		"--yes", "--bind-ip", "192.168.50.10", "--project-subnet", "10.89.0.0/24", "--install-dir", f.install,
	}
	name := "/bin/bash"
	if f.noTTY {
		name = "setsid"
		command = append([]string{"--wait", "/bin/bash"}, command...)
	}
	return Run(t, RunOpt{Cwd: f.dir, Env: f.env, Timeout: 10 * time.Second}, name, append(command, args...)...)
}

func (f *dockerInstallerFixture) requireNoInstall(t *testing.T, result ProcResult) {
	t.Helper()
	Require(t, result.Code != 0, "installer accepted unsafe input: %s", result.Stdout)
	_, err := os.Lstat(f.install)
	Check(t, os.IsNotExist(err), "failed admission created installation: %v", err)
	log, err := os.ReadFile(f.log)
	Require(t, err == nil || os.IsNotExist(err), "read Docker command log: %v", err)
	for _, mutation := range []string{" pull", " up", " create", " rm", " down", " prune"} {
		Check(t, !strings.Contains(string(log), mutation), "failed admission ran %q: %s", mutation, log)
	}
}

func TestDockerInstallerRejectsBadAssetsBeforeMutation(t *testing.T) {
	for _, fault := range []string{"release-download", "metadata-download", "compose-download", "missing-assets", "unqualified", "bad-json", "missing-image", "wrong-platform", "metadata-checksum", "metadata-size", "compose-checksum", "compose-size", "invalid-compose"} {
		t.Run(fault, func(t *testing.T) {
			f := newDockerInstallerFixture(t)
			script := f.script(t)
			switch fault {
			case "release-download", "metadata-download", "compose-download":
				asset := map[string]string{"release-download": "release.json", "metadata-download": "docker.json", "compose-download": "sodaos-docker-compose.yaml"}[fault]
				f.env = SetEnv(f.env, "FIXTURE_FAILED_DOWNLOAD", asset)
			case "missing-assets":
				WriteFile(t, filepath.Join(f.dir, "release.json"), []byte(`{"tag_name":"`+dockerInstallerTag+`","draft":false,"immutable":true,"prerelease":false,"assets":[]}`), 0o600)
			case "unqualified":
				f.writeRelease(t, false, false)
			case "bad-json":
				WriteFile(t, filepath.Join(f.dir, "docker.json"), []byte("{"), 0o600)
				script = f.script(t)
			case "missing-image", "wrong-platform", "compose-size":
				if fault == "missing-image" {
					delete(f.metadata, "Image")
				} else if fault == "wrong-platform" {
					f.metadata["Platform"] = "linux/arm64"
				} else {
					f.metadata["Compose"].(map[string]any)["Bytes"] = 1
				}
				f.writeMetadata(t)
				script = f.script(t)
			case "metadata-checksum":
				f.metadata["Purpose"] = "invalidxxx"
				f.writeMetadata(t)
			case "metadata-size":
				WriteFile(t, filepath.Join(f.dir, "docker.json"), []byte("{}"), 0o600)
			case "compose-checksum":
				path := filepath.Join(f.dir, "sodaos-docker-compose.yaml")
				data, err := os.ReadFile(path)
				Require(t, err == nil, "read Compose: %v", err)
				data[0] = '#'
				WriteFile(t, path, data, 0o600)
			case "invalid-compose":
				f.env = SetEnv(f.env, "FIXTURE_BAD_COMPOSE", "true")
			}
			result := f.run(t, script)
			f.requireNoInstall(t, result)
			if diagnostic := map[string]string{
				"release-download": "Download failed", "metadata-download": "Download failed", "compose-download": "Download failed",
				"missing-assets": "lacks a complete Docker", "unqualified": "not qualified",
				"bad-json": "Docker metadata", "missing-image": "Docker metadata", "wrong-platform": "Docker metadata",
				"metadata-checksum": "Checksum mismatch", "metadata-size": "Incorrect byte size",
				"compose-checksum": "Checksum mismatch", "compose-size": "Incorrect byte size",
			}[fault]; diagnostic != "" {
				Check(t, strings.Contains(result.Stderr, diagnostic), "wrong refusal for %s: %s", fault, result.Stderr)
			}
		})
	}
}

func TestDockerInstallerRejectsUnsupportedHosts(t *testing.T) {
	for _, fault := range []string{"macos", "arm", "wsl", "rootless", "desktop", "cgroup-v1", "old-compose", "forwarding", "busy-port", "overlap", "retained-volume", "retained-container"} {
		t.Run(fault, func(t *testing.T) {
			f := newDockerInstallerFixture(t)
			if settings, ok := map[string][2]string{
				"macos": {"FIXTURE_OS", "Darwin"}, "arm": {"FIXTURE_ARCH", "aarch64"},
				"wsl":         {"FIXTURE_KERNEL", "6.6.87.2-microsoft-standard-WSL2"},
				"old-compose": {"FIXTURE_COMPOSE_VERSION", "2.16.0"}, "forwarding": {"FIXTURE_FORWARDING", "0"},
				"busy-port": {"FIXTURE_PORT_BUSY", "true"}, "overlap": {"FIXTURE_ROUTE", "10.89.0.0/16"},
				"retained-volume": {"FIXTURE_RETAINED_VOLUME", "true"}, "retained-container": {"FIXTURE_RETAINED_CONTAINER", "true"},
			}[fault]; ok {
				f.env = SetEnv(f.env, settings[0], settings[1])
			} else {
				info := map[string]any{"OSType": "linux", "Architecture": "x86_64", "CgroupVersion": "2", "OperatingSystem": "Linux", "SecurityOptions": []string{}}
				if fault == "rootless" {
					info["SecurityOptions"] = []string{"name=rootless"}
				} else if fault == "desktop" {
					info["OperatingSystem"] = "Docker Desktop"
				} else {
					info["CgroupVersion"] = "1"
				}
				data, err := json.Marshal(info)
				Require(t, err == nil, "marshal host info: %v", err)
				WriteFile(t, filepath.Join(f.dir, "docker-info.json"), data, 0o600)
			}
			f.requireNoInstall(t, f.run(t, f.script(t)))
		})
	}
}

func TestDockerInstallerStreamedRerunPreservesIdentityAndRefusesUpgrade(t *testing.T) {
	f := newDockerInstallerFixture(t)
	script := f.script(t)
	var retained os.FileInfo
	for attempt := 0; attempt < 2; attempt++ {
		result := f.run(t, script)
		if result.Code != 0 {
			log, err := os.ReadFile(f.log)
			t.Logf("Docker observations: %s (read error: %v)", log, err)
		}
		Require(t, result.Code == 0, "streamed install attempt %d: %s\n%s", attempt+1, result.Stdout, result.Stderr)
		Check(t, strings.Contains(result.Stdout, "Browser setup is pending"), "unattended install must report pending browser setup")
		identity, err := os.Stat(filepath.Join(f.install, "identity.json"))
		Require(t, err == nil, "read identity stat: %v", err)
		if retained != nil {
			Check(t, os.SameFile(identity, retained) && identity.ModTime() == retained.ModTime(), "rerun replaced or rewrote retained identity")
		}
		retained = identity
		files := map[string]string{".env": "SODA_IMAGE=ghcr.io/levitateos/sodaos@" + dockerInstallerManifest + "\nSODA_PROJECT_SUBNET=10.89.0.0/24\n", "compose.yaml": ReadFile(t, "system/docker/compose.yaml")}
		for name, expected := range files {
			path := filepath.Join(f.install, name)
			data, err := os.ReadFile(path)
			Require(t, err == nil && string(data) == expected, "retained %s differs: %v", name, err)
			info, err := os.Stat(path)
			Check(t, err == nil && info.Mode().Perm() == 0o600, "%s is not private", name)
		}
	}
	identity, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
	Require(t, err == nil, "read identity: %v", err)
	WriteFile(t, f.log, nil, 0o600)
	f.metadata["ReleaseTag"] = "v0.3.0-docker-fixture"
	f.writeMetadata(t)
	f.writeRelease(t, false, true)
	result := f.run(t, f.script(t))
	Require(t, result.Code != 0 && strings.Contains(result.Stderr, "Cross-version upgrades"), "changed release was not refused: %+v", result)
	after, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
	Check(t, err == nil && string(after) == string(identity), "changed release altered retained identity")
	log, err := os.ReadFile(f.log)
	Check(t, err == nil && !strings.Contains(string(log), " pull") && !strings.Contains(string(log), " up"), "changed release mutated Docker: %s", log)
}

func TestDockerInstallerRejectsReservedNetworks(t *testing.T) {
	for _, subnet := range []string{"10.88.1.0/24", "10.90.0.0/24"} {
		t.Run(subnet, func(t *testing.T) {
			f := newDockerInstallerFixture(t)
			result := f.run(t, f.script(t), "--project-subnet", subnet)
			f.requireNoInstall(t, result)
			Check(t, strings.Contains(result.Stderr, "reserved network"), "wrong reserved-network refusal: %s", result.Stderr)
		})
	}
}

func TestDockerInstallerRefusesChangedRuntimeBeforeMutation(t *testing.T) {
	for _, fault := range []string{"image", "volume-owner", "partial-volumes", "lost-volumes"} {
		t.Run(fault, func(t *testing.T) {
			f := newDockerInstallerFixture(t)
			script := f.script(t)
			result := f.run(t, script)
			Require(t, result.Code == 0, "fresh installation: %s", result.Stderr)
			identity, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
			Require(t, err == nil, "read identity: %v", err)
			settings := map[string][3]string{
				"image":           {"FIXTURE_CONTAINER_IMAGE", "foreign", "Retained"},
				"volume-owner":    {"FIXTURE_VOLUME_OWNER", "foreign", "Retained"},
				"partial-volumes": {"FIXTURE_MISSING_VOLUME", "state", "missing"},
				"lost-volumes":    {"FIXTURE_MISSING_VOLUME", "all", "missing"},
			}[fault]
			f.env = SetEnv(f.env, settings[0], settings[1])
			WriteFile(t, f.log, nil, 0o600)
			result = f.run(t, script)
			Require(t, result.Code != 0 && strings.Contains(result.Stderr, settings[2]), "changed runtime accepted: %+v", result)
			log, err := os.ReadFile(f.log)
			Check(t, err == nil && !strings.Contains(string(log), " pull") && !strings.Contains(string(log), " up"), "changed runtime mutated Docker: %s", log)
			after, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
			Check(t, err == nil && string(after) == string(identity), "runtime refusal changed retained identity")
		})
	}
}

func TestDockerInstallerReadinessFailureIsRetryable(t *testing.T) {
	f := newDockerInstallerFixture(t)
	script := f.script(t)
	f.env = SetEnv(f.env, "FIXTURE_NOT_READY", "true")
	result := f.run(t, script)
	Require(t, result.Code != 0, "unhealthy runtime admitted")
	Check(t, !strings.Contains(result.Stdout, "Runtime services are ready"), "reported readiness after failed startup")
	identity, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
	Require(t, err == nil, "startup failure did not retain installation identity: %v", err)
	f.env = SetEnv(f.env, "FIXTURE_NOT_READY", "false")
	result = f.run(t, script)
	Require(t, result.Code == 0, "retry failed: %s", result.Stderr)
	after, err := os.ReadFile(filepath.Join(f.install, "identity.json"))
	Check(t, err == nil && string(after) == string(identity), "retry changed retained identity")
}

func TestDockerInstallerTruncatedStdinCannotStart(t *testing.T) {
	f := newDockerInstallerFixture(t)
	script := f.script(t)
	data, err := os.ReadFile(script)
	Require(t, err == nil, "read rendered installer: %v", err)
	WriteFile(t, script, data[:len(data)/2], 0o600)
	f.requireNoInstall(t, f.run(t, script))
	_, err = os.Stat(filepath.Join(f.dir, "downloads.log"))
	Check(t, os.IsNotExist(err), "truncated stdin started downloads")
}

func TestDockerInstallerPrereleaseNeedsExplicitQualification(t *testing.T) {
	f := newDockerInstallerFixture(t)
	f.writeRelease(t, true, false)
	script := f.script(t)
	result := f.run(t, script)
	f.requireNoInstall(t, result)
	Check(t, strings.Contains(result.Stderr, "not qualified"), "prerelease refused for wrong reason: %s", result.Stderr)
	result = f.run(t, script, "--qualification")
	Require(t, result.Code == 0, "explicit prerelease qualification failed: %s", result.Stderr)
	Check(t, strings.Contains(result.Stdout, "not yet been recommended for production"), "prerelease omitted qualification warning")
}

func TestDockerInstallerFailedFileCommitCanRetry(t *testing.T) {
	f := newDockerInstallerFixture(t)
	script := f.script(t)
	f.env = SetEnv(f.env, "FIXTURE_INSTALL_FAILURE", "true")
	result := f.run(t, script)
	Require(t, result.Code != 0, "partial file copy accepted")
	_, err := os.Lstat(f.install)
	Check(t, os.IsNotExist(err), "failed file copy left a partial installation: %v", err)
	_, err = os.Stat(filepath.Join(f.dir, "runtime-active"))
	Check(t, os.IsNotExist(err), "failed file commit started runtime")
	f.env = SetEnv(f.env, "FIXTURE_INSTALL_FAILURE", "false")
	result = f.run(t, script)
	Require(t, result.Code == 0, "partial copy poisoned retry: %s", result.Stderr)
}

func TestDockerInstallerPreservesUnrecognizedDirectory(t *testing.T) {
	f := newDockerInstallerFixture(t)
	Require(t, os.Mkdir(f.install, 0o700) == nil, "create preexisting directory")
	path := filepath.Join(f.install, "operator-file")
	WriteFile(t, path, []byte("retain this data"), 0o600)
	result := f.run(t, f.script(t))
	Require(t, result.Code != 0 && strings.Contains(result.Stderr, "Owned regular file required"), "partial directory admitted: %+v", result)
	entries, err := os.ReadDir(f.install)
	Check(t, err == nil && len(entries) == 1 && entries[0].Name() == "operator-file", "unrecognized directory changed")
	data, err := os.ReadFile(path)
	Check(t, err == nil && string(data) == "retain this data", "operator data changed")
	log, err := os.ReadFile(f.log)
	Check(t, err == nil && !strings.Contains(string(log), " pull") && !strings.Contains(string(log), " up"), "directory refusal mutated Docker: %s", log)
}

func TestDockerInstallerActivatedStateRequiresServicesAndVerifiedHTTPS(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	Require(t, err == nil, "generate fixture CA: %v", err)
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Soda installer fixture CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, ca, ca, public, private)
	Require(t, err == nil, "sign fixture CA: %v", err)
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	for _, fault := range []string{"healthy", "soda-dashboard.service", "soda-proxy.service", "https"} {
		t.Run(fault, func(t *testing.T) {
			f := newDockerInstallerFixture(t)
			script := f.script(t)
			result := f.run(t, script)
			Require(t, result.Code == 0, "fresh installation: %s", result.Stderr)
			WriteFile(t, filepath.Join(f.dir, "activated"), nil, 0o600)
			WriteFile(t, filepath.Join(f.dir, "root.crt"), certificate, 0o600)
			if fault == "https" {
				f.env = SetEnv(f.env, "FIXTURE_HTTPS_FAILURE", "true")
			} else if fault != "healthy" {
				f.env = SetEnv(f.env, "FIXTURE_FAILED_BROWSER_UNIT", fault)
			}
			result = f.run(t, script)
			Check(t, (result.Code == 0) == (fault == "healthy"), "activated readiness result for %s: %+v", fault, result)
			_, err := os.Stat(filepath.Join(f.install, "soda-root.crt"))
			if fault == "healthy" {
				Check(t, err == nil && strings.Contains(result.Stdout, "Then open https://192.168.50.10"), "healthy browser path did not finish: %+v", result)
			} else {
				Check(t, os.IsNotExist(err) && !strings.Contains(result.Stdout, "Then open") && !strings.Contains(result.Stdout, "Lifecycle commands"), "failed browser readiness reported completion: %+v", result)
			}
			if fault == "healthy" || fault == "https" {
				log, err := os.ReadFile(filepath.Join(f.dir, "curl.log"))
				Require(t, err == nil, "read HTTPS invocation: %v", err)
				Check(t, strings.Contains(string(log), "--cacert ") && strings.Contains(string(log), "https://192.168.50.10/api/healthz") && !strings.Contains(string(log), "--insecure") && !strings.Contains(string(log), " -k"), "browser probe did not verify exported CA: %s", log)
			}
		})
	}
}

func TestDockerInstallerCachedSudoWorksWithoutTerminal(t *testing.T) {
	f := newDockerInstallerFixture(t)
	f.noTTY = true
	f.env = SetEnv(f.env, "FIXTURE_NEEDS_SUDO", "true")
	result := f.run(t, f.script(t))
	Require(t, result.Code == 0, "cached sudo required terminal: %s", result.Stderr)
	log, err := os.ReadFile(filepath.Join(f.dir, "sudo.log"))
	Require(t, err == nil, "read sudo invocations: %v", err)
	Check(t, strings.HasPrefix(string(log), "-n -v\n") && !strings.Contains(string(log), "\n-v\n"), "sudo did not use cached admission: %s", log)
	Check(t, strings.Contains(string(log), "--preserve-env=SODA_IMAGE,SODA_PROJECT_SUBNET docker"), "sudo dropped Compose image/subnet environment")
}
