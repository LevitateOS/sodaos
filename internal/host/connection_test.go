package host

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"
)

type connectionExec struct {
	t       *testing.T
	key     []byte
	running bool
	calls   int
}

func (e *connectionExec) Run(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
	e.calls++
	if command != "/usr/bin/podman" {
		e.t.Fatal("unexpected executable")
	}
	id := "p0123456789abcdef01234567"
	if reflect.DeepEqual(args, []string{"inspect", "soda-" + id}) {
		return json.Marshal([]any{map[string]any{"Config": map[string]any{"Labels": map[string]string{"org.soda.project": id, "org.soda.owner": "1"}}, "State": map[string]bool{"Running": e.running}, "NetworkSettings": map[string]any{"Networks": map[string]any{"soda": map[string]string{"IPAddress": "10.89.0.2"}}}}})
	}
	if !reflect.DeepEqual(args, []string{"exec", "soda-" + id, "/usr/bin/head", "-c", "16385", "/etc/ssh/ssh_host_ed25519_key.pub"}) {
		e.t.Fatal("public key read escaped fixed boundary", args)
	}
	return e.key, nil
}
func TestConnectionReadsOnlyFixedPublicKey(t *testing.T) {
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	exec := &connectionExec{t: t, key: ssh.MarshalAuthorizedKey(key), running: true}
	daemon := Daemon{Config: Config{Network: "soda", Subnet: "10.89.0.0/24"}, Exec: exec}
	result, err := daemon.connection(context.Background(), "p0123456789abcdef01234567")
	if err != nil || result.Fingerprint != ssh.FingerprintSHA256(key) || exec.calls != 2 {
		t.Fatal("public host key not returned", err)
	}
	exec.calls = 0
	exec.running = false
	result, err = daemon.connection(context.Background(), "p0123456789abcdef01234567")
	if err != nil || result.HostKey != "" || exec.calls != 1 {
		t.Fatal("stopped container entered", err)
	}
	exec.calls = 0
	if _, err = daemon.connection(context.Background(), "../../other"); err == nil || exec.calls != 0 {
		t.Fatal("invalid target executed")
	}
}
func TestConnectionRejectsMalformedKey(t *testing.T) {
	exec := &connectionExec{t: t, key: []byte("not a public key"), running: true}
	daemon := Daemon{Config: Config{Network: "soda", Subnet: "10.89.0.0/24"}, Exec: exec}
	if _, err := daemon.connection(context.Background(), "p0123456789abcdef01234567"); err == nil {
		t.Fatal("invalid host key accepted")
	}
}
