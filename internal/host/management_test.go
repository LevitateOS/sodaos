package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type managementExec func(context.Context, []byte, string, ...string) ([]byte, error)

func (f managementExec) Run(ctx context.Context, in []byte, cmd string, args ...string) ([]byte, error) {
	return f(ctx, in, cmd, args...)
}

func TestLifecycleUsesExistingUnitAndRetainsIdentity(t *testing.T) {
	for _, action := range []string{"inspect", "start", "stop"} {
		t.Run(action, func(t *testing.T) {
			running, enabled := false, false
			if action == "stop" {
				running, enabled = true, true
			}
			mutations := 0
			id := "p0123456789abcdef01234567"
			d := Daemon{Config: Config{Network: "soda-projects", Subnet: "10.89.0.0/24"}}
			d.Exec = managementExec(func(_ context.Context, _ []byte, cmd string, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				if cmd == "/usr/bin/systemctl" {
					if args[0] == "show" {
						state := "disabled"
						if enabled {
							state = "enabled"
						}
						return []byte("LoadState=loaded\nFragmentPath=/etc/systemd/system/soda-project@.service\nDropInPaths=\nUnitFileState=" + state + "\n"), nil
					}
					verb := "enable"
					if action == "stop" {
						verb = "disable"
					}
					if joined != verb+" --now soda-project@"+id+".service" {
						t.Fatal("unbounded lifecycle command", joined)
					}
					mutations++
					running, enabled = action == "start", action == "start"
					return nil, nil
				}
				if cmd != "/usr/bin/podman" {
					t.Fatal("unexpected command")
				}
				if strings.Contains(joined, "--format") {
					return []byte(fmt.Sprintf(`{"id":%q,"running":%t,"project":%q,"owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64), running, id)), nil
				}
				if joined != "inspect soda-"+id {
					t.Fatal("lifecycle bypassed native unit", joined)
				}
				return []byte(fmt.Sprintf(`[{"Config":{"Labels":{"org.soda.project":%q,"org.soda.owner":"1"}},"State":{"Running":%t},"NetworkSettings":{"Networks":{}}}]`, id, running)), nil
			})
			state, err := d.lifecycle(t.Context(), Lifecycle{Project: id, Action: action})
			if err != nil {
				t.Fatal(err)
			}
			if state.Environment.ID != id || state.BootEnabled != enabled || state.Environment.Running != running {
				t.Fatal("wrong observed state")
			}
			want := 1
			if action == "inspect" {
				want = 0
			}
			if mutations != want {
				t.Fatal("unexpected mutation count")
			}
		})
	}
}
func TestLifecycleRefusesUnexpectedUnitBeforeMutation(t *testing.T) {
	for _, unit := range []string{"LoadState=not-found\n", "LoadState=loaded\nFragmentPath=/etc/systemd/system/other.service\nDropInPaths=\nUnitFileState=enabled\n", "LoadState=loaded\nFragmentPath=/etc/systemd/system/soda-project@.service\nDropInPaths=/etc/systemd/system/override.conf\nUnitFileState=enabled\n"} {
		d := Daemon{Exec: managementExec(func(_ context.Context, _ []byte, cmd string, args ...string) ([]byte, error) {
			if cmd == "/usr/bin/systemctl" {
				if args[0] != "show" {
					t.Fatal("unexpected unit was mutated")
				}
				return []byte(unit), nil
			}
			return []byte(fmt.Sprintf(`{"id":%q,"running":true,"project":"p0123456789abcdef01234567","owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64))), nil
		})}
		if _, err := d.lifecycle(t.Context(), Lifecycle{Project: "p0123456789abcdef01234567", Action: "stop"}); err == nil {
			t.Fatal("unexpected unit accepted")
		}
	}
}

func TestManagementRejectsCallerSelectedTargetsBeforeExec(t *testing.T) {
	calls := 0
	d := Daemon{Exec: managementExec(func(context.Context, []byte, string, ...string) ([]byte, error) {
		calls++
		return nil, errors.New("unexpected")
	})}
	for _, in := range []Lifecycle{{Project: "/host", Action: "start"}, {Project: "p0123456789abcdef01234567", Action: "destroy"}} {
		if _, err := d.lifecycle(t.Context(), in); err == nil {
			t.Fatal("invalid lifecycle accepted")
		}
	}
	for _, in := range []AccessKeys{{Project: "p0123456789abcdef01234567", Login: "root", Identity: 1}, {Project: "p0123456789abcdef01234567", Login: "alice", Identity: 1, Apply: true, Revision: "bad"}} {
		if _, err := d.accessKeys(t.Context(), in); err == nil {
			t.Fatal("invalid key operation accepted")
		}
	}
	if calls != 0 {
		t.Fatal("invalid target reached host")
	}
}
func TestEmbeddedKeyProgramLoadsAndRefusesLocalUnprivilegedAccount(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("this check must never use a host root account")
	}
	path, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("Python unavailable")
	}
	d := Daemon{Exec: managementExec(func(_ context.Context, in []byte, cmd string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[1] == "inspect" {
			return []byte(fmt.Sprintf(`{"id":%q,"running":true,"project":"p0123456789abcdef01234567","owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64))), nil
		}
		// Exercise the actual assembled fixed Python source, substituting only the
		// external Podman boundary. No root account/filesystem or native project used.
		p := exec.Command(path, "-I", "-c", args[7])
		p.Stdin = bytes.NewReader(in)
		out, e := p.CombinedOutput()
		if e == nil || string(out) != "native key operation not confirmed\n" {
			t.Fatal("embedded program failed to load/refuse safely")
		}
		return nil, e
	})}
	if _, err := d.accessKeys(t.Context(), AccessKeys{Project: "p0123456789abcdef01234567", Login: "alice", Identity: 1}); err == nil {
		t.Fatal("unprivileged native update accepted")
	}
}

func TestKeyPreviewUsesExistingMarkerValidatorAndVerifiedContainer(t *testing.T) {
	id := "p0123456789abcdef01234567"
	calls := 0
	d := Daemon{Exec: managementExec(func(_ context.Context, in []byte, cmd string, args ...string) ([]byte, error) {
		calls++
		if len(args) > 1 && args[1] == "inspect" {
			return []byte(fmt.Sprintf(`{"id":%q,"running":true,"project":%q,"owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64), id)), nil
		}
		if cmd != "/usr/bin/podman" || strings.Join(args[:7], " ") != "--remote=false exec --interactive "+strings.Repeat("a", 64)+" /usr/bin/python3 -I -c" {
			t.Fatal("unexpected native key command")
		}
		if !strings.Contains(args[7], "types.ModuleType('project_terminal')") || !strings.Contains(args[7], "from project_terminal import account_for") {
			t.Fatal("identity validator was not reused")
		}
		var body map[string]any
		if json.Unmarshal(in, &body) != nil || body["login"] != "alice" || body["identity"] != float64(1) || body["apply"] != false {
			t.Fatal("unexpected native key input")
		}
		return []byte(`{"revision":"` + strings.Repeat("a", 64) + `","keys":[]}`), nil
	})}
	if _, err := d.accessKeys(t.Context(), AccessKeys{Project: id, Login: "alice", Identity: 1}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("unexpected native operations")
	}
}
