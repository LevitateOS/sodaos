package host

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type captureCreate struct{ args []string }

func (c *captureCreate) Run(_ context.Context, _ []byte, executable string, args ...string) ([]byte, error) {
	if executable != "/usr/bin/podman" {
		return nil, errors.New("unexpected executable")
	}
	if reflect.DeepEqual(args, []string{"network", "exists", "soda-projects"}) {
		return nil, nil
	}
	c.args = append([]string(nil), args...)
	return nil, errors.New("stop before native creation")
}

func TestCreateUsesFixedNamespacedNetworkCapability(t *testing.T) {
	commands := &captureCreate{}
	d := Daemon{Exec: commands, Config: Config{Network: "soda-projects", Image: "localhost/soda-project-os:dev"}}
	id := "p123456789012345678901234"
	if _, err := d.create(context.Background(), Create{ID: id, Owner: 2}); err == nil {
		t.Fatal("failed native creation reported success")
	}
	want := []string{"create", "--name", "soda-" + id, "--label", "org.soda.project=" + id,
		"--label", "org.soda.owner=2", "--network", "soda-projects", "--userns=auto:size=262144",
		"--systemd=always", "--cgroupns=private", "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN",
		"--device=/dev/fuse", "--security-opt=label=disable", "localhost/soda-project-os:dev"}
	if !reflect.DeepEqual(commands.args, want) {
		t.Fatalf("unexpected native creation contract: %q", commands.args)
	}
}

type noExec struct{ called bool }

func (n *noExec) Run(context.Context, []byte, string, ...string) ([]byte, error) {
	n.called = true
	return nil, errors.New("unexpected command")
}
func TestInvalidProjectNeverExecutes(t *testing.T) {
	n := &noExec{}
	d := Daemon{Exec: n}
	if _, _, err := d.inspect(context.Background(), "../../other"); err == nil || n.called {
		t.Fatal("untrusted project reached executor")
	}
}
func TestInvalidAccountNeverExecutes(t *testing.T) {
	n := &noExec{}
	d := Daemon{Exec: n}
	if err := d.account(context.Background(), Account{Login: "root;id", Identity: 1, Keys: []string{"x"}}); err == nil || n.called {
		t.Fatal("untrusted account reached executor")
	}
}
