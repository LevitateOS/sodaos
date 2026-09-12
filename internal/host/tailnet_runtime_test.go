package host

import (
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/tailnet"
)

func TestTailnetProcessIdentityRejectsHostNamespacesAndAmbiguousMappings(t *testing.T) {
	for _, mode := range []string{"valid", "host user", "host net", "extra map", "root map", "wrong pid", "dead", "missing start", "bad boot", "missing namespace"} {
		t.Run(mode, func(t *testing.T) {
			read := func(path string) ([]byte, error) {
				switch path {
				case "/proc/123/stat":
					fields := strings.Fields("S 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 456 1")
					if mode == "dead" {
						fields[0] = "Z"
					}
					if mode == "missing start" {
						fields[19] = "0"
					}
					pid := "123"
					if mode == "wrong pid" {
						pid = "124"
					}
					return []byte(pid + " (comm with ) spaces) " + strings.Join(fields, " ")), nil
				case "/proc/123/uid_map", "/proc/123/gid_map":
					data := "0 524288 262144\n"
					if mode == "extra map" {
						data += "300000 900000 1\n"
					}
					if mode == "root map" {
						data = "0 0 262144\n"
					}
					return []byte(data), nil
				case "/proc/sys/kernel/random/boot_id":
					if mode == "bad boot" {
						return []byte("bad"), nil
					}
					return []byte("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n"), nil
				}
				return nil, errors.New("unexpected proc input")
			}
			link := func(path string) (string, error) {
				kind := "user"
				if strings.HasSuffix(path, "/net") {
					kind = "net"
				}
				inode := "11"
				if strings.Contains(path, "/123/") {
					inode = "22"
				}
				if mode == "host "+kind {
					inode = "11"
				}
				if mode == "missing namespace" {
					return "", errors.New("gone")
				}
				return kind + ":[" + inode + "]", nil
			}
			out, e := processRunIdentity(123, read, link)
			if mode == "valid" {
				if e != nil || out.UID != 524288 || out.GID != 524288 || out.Start != "456" || out.UserNS != "user:[22]" || out.NetNS != "net:[22]" {
					t.Fatal(out, e)
				}
			} else if e == nil {
				t.Fatal("unsafe process identity accepted")
			}
		})
	}
}
func TestTailnetCompanionRecipeKeepsSecretsAndUnrelatedNamespacesOutside(t *testing.T) {
	run := projectRun{Target: tailnet.RunTarget{Project: "p" + strings.Repeat("a", 24), Container: strings.Repeat("b", 64), Run: strings.Repeat("c", 64)}, UID: 524288, GID: 524288}
	image := "sha256:" + strings.Repeat("d", 64)
	args, e := companionCreateArgs(run, image)
	if e != nil {
		t.Fatal(e)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--userns=container:" + run.Target.Container, "--network=container:" + run.Target.Container, "--pid=private", "--ipc=private", "--uts=private", "--cgroupns=private", "--cap-drop=ALL", "--cap-add=NET_ADMIN", "--device=/dev/net/tun", "--no-hosts", "--log-driver=none", "--pull=never", image, "--entrypoint=/usr/local/bin/tailscaled", "--state=/var/lib/tailscale/tailscaled.state", "--no-logs-no-support"} {
		if !strings.Contains(joined, want) {
			t.Fatal("missing fixed recipe argument", want)
		}
	}
	for _, forbidden := range []string{"--privileged", "--rm", "--replace", "--env", "SYS_MODULE", ":U", "--network=host", "--pid=host", "--userns=host", "/var/lib/soda-tailnet", "/run/podman", "/proc/", "tskey", "client_secret"} {
		if strings.Contains(joined, forbidden) {
			t.Fatal("unsafe companion argument", forbidden)
		}
	}
	count := 0
	for i, a := range args {
		if a == "--volume" {
			count++
			if i+1 >= len(args) || !strings.HasPrefix(args[i+1], "/run/soda-tailnet/"+run.Target.Project+"/"+run.Target.Run+"/") {
				t.Fatal("non-run mount")
			}
		}
	}
	if count != 3 {
		t.Fatal("unexpected mount count")
	}
	for _, bad := range []string{"latest", "docker.io/tailscale/tailscale:latest", "sha256:bad", "--privileged"} {
		if _, e := companionCreateArgs(run, bad); e == nil {
			t.Fatal("mutable/caller image accepted")
		}
	}
	run.Target.Container = "../other"
	if _, e := companionCreateArgs(run, image); e == nil {
		t.Fatal("invalid namespace target accepted")
	}
}
