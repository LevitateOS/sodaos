package acceptance

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestLiteralRemoteArguments(t *testing.T) {
	values := []string{"", "a b", "'quoted'", "a; b", "$(false)", "two\nlines"}
	args := append([]string{"printf", "%s\\0"}, values...)
	out, err := exec.Command("/bin/sh", "-c", Quote(args)).Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != strings.Join(values, "\x00")+"\x00" {
		t.Fatalf("arguments changed: %q", out)
	}
}
func TestPinnedSSHOptionsAndNativeRequestBinding(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "key")
	hosts := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(key, []byte("synthetic identity, not used for authentication"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hosts, []byte("synthetic pin, not used for authentication"), 0600); err != nil {
		t.Fatal(err)
	}
	r := Remote{User: "root", Host: "127.0.0.1", Port: 22222, Key: key, KnownHosts: hosts}
	c, err := r.Command([]string{"printf", "%s", "a b"}, strings.NewReader("input"))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(c.Args, " ")
	for _, required := range []string{"StrictHostKeyChecking=yes", "IdentitiesOnly=yes", "GlobalKnownHostsFile=/dev/null", "timeout", "--kill-after=10s"} {
		if !strings.Contains(joined, required) {
			t.Fatal(joined)
		}
	}
	if strings.Contains(joined, "accept-new") || strings.Contains(joined, "ProxyJump") {
		t.Fatal(joined)
	}
	request := filepath.Join(dir, "request.json")
	raw, _ := json.Marshal(RemoteRequest{Revision: strings.Repeat("a", 40), Architecture: "x86_64", Target: "builder", Work: "/private/fresh", Phase: "prepare"})
	if err = os.WriteFile(request, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = r.NativePhase(request, strings.Repeat("b", 40), "x86_64", "builder"); err == nil {
		t.Fatal("accepted wrong revision")
	}
	if _, err = r.NativePhase(request, strings.Repeat("a", 40), "x86_64", "other"); err == nil {
		t.Fatal("accepted wrong target")
	}
	if err = os.Chmod(key, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Args(); err == nil {
		t.Fatal("accepted exposed private identity")
	}
}
func TestFixtureTrustIsKnownBeforeBoot(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(private, "synthetic fixture")
	if err != nil {
		t.Fatal(err)
	}
	encoded := pem.EncodeToMemory(block)
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	hosts := filepath.Join(dir, "known_hosts")
	if err = os.WriteFile(hosts, []byte(knownhosts.Line([]string{"[127.0.0.1]:22222"}, signer.PublicKey())+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "instance.ign")
	contents := map[string]any{"source": "data:;base64," + base64.StdEncoding.EncodeToString(encoded)}
	config := map[string]any{"ignition": map[string]any{"version": "3.5.0"}, "storage": map[string]any{"files": []any{
		map[string]any{"path": "/etc/hostname", "contents": map[string]any{"source": "data:,soda-native-fixture%0A"}},
		map[string]any{"path": "/etc/ssh/ssh_host_ed25519_key", "contents": contents},
	}}}
	body, _ := json.Marshal(config)
	if err = os.WriteFile(input, body, 0600); err != nil {
		t.Fatal(err)
	}
	r := Remote{User: "root", Host: "127.0.0.1", Port: 22222, KnownHosts: hosts}
	if err = VerifyFixtureTrust(input, "soda-native-fixture", r); err != nil {
		t.Fatal(err)
	}
	if err = VerifyFixtureTrust(input, "soda-native-other", r); err == nil {
		t.Fatal("accepted wrong hostname")
	}
	r.Port = 22223
	if err = VerifyFixtureTrust(input, "soda-native-fixture", r); err == nil {
		t.Fatal("accepted unpinned endpoint")
	}
	secrets, err := ProvisioningSecrets(input)
	if err != nil || len(secrets) == 0 {
		t.Fatalf("private bootstrap redaction: %v", err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err = writer.Write(encoded); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	contents["source"] = "data:;base64," + base64.StdEncoding.EncodeToString(compressed.Bytes())
	contents["compression"] = "gzip"
	body, _ = json.Marshal(config)
	if err = os.WriteFile(input, body, 0600); err != nil {
		t.Fatal(err)
	}
	r.Port = 22222
	if err = VerifyFixtureTrust(input, "soda-native-fixture", r); err != nil {
		t.Fatal(err)
	}
	secrets, err = ProvisioningSecrets(input)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, secret := range secrets {
		found = found || bytes.Equal(secret, encoded)
	}
	if !found {
		t.Fatal("compressed private key was not decoded for redaction")
	}
	r.Port = 22223
	if VerifyFixtureTrust(input, "soda-native-fixture", r) == nil {
		t.Fatal("compressed key accepted unpinned endpoint")
	}
	if _, err = inlineData("data:,not-gzip", "gzip"); err == nil {
		t.Fatal("accepted corrupt gzip")
	}
	if _, err = inlineData("data:,plain", "unknown"); err == nil {
		t.Fatal("accepted unknown compression")
	}
	compressed.Reset()
	writer = gzip.NewWriter(&compressed)
	if _, err = writer.Write(bytes.Repeat([]byte("x"), 1024*1024+1)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = inlineData("data:;base64,"+base64.StdEncoding.EncodeToString(compressed.Bytes()), "gzip"); err == nil {
		t.Fatal("accepted decompression overflow")
	}
}
func TestVMArgumentsRetainDiskAndNativeIsolation(t *testing.T) {
	for _, arch := range []string{"x86_64", "aarch64"} {
		c := VMConfig{Name: "soda-native-fixture", Architecture: arch, Work: "/private/owned", Ignition: "/private/input.ign", Firmware: "/firmware/code", SSH: Remote{Port: 22222}}
		args := strings.Join(c.args(), " ")
		for _, s := range []string{"accel=kvm", "-cpu host", "hostfwd=tcp:127.0.0.1:22222-:22", "/private/owned/disk.qcow2", "/private/owned/vars.fd", "opt/com.coreos/config"} {
			if !strings.Contains(args, s) {
				t.Fatal(args)
			}
		}
		if strings.Contains(args, "-daemonize") || strings.Contains(args, "tap,") {
			t.Fatal(args)
		}
	}
}
func TestProcessCancellationBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := StartProcess(ctx, Command{Name: "/bin/sh", Args: []string{"-c", "exit 0"}}, nil, nil); err == nil {
		t.Fatal("started after cancellation")
	}
}
func TestOwnedProcessWaitAndCleanup(t *testing.T) {
	if err := ownedGroupsSupported(); err != nil {
		t.Skip(err)
	}
	p, err := StartProcess(context.Background(), Command{Name: "/bin/sh", Args: []string{"-c", "exit 0"}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = p.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err = p.Stop(); err != nil {
		t.Fatal(err)
	}
	if err = p.Stop(); err != nil {
		t.Fatal(err)
	}
}
