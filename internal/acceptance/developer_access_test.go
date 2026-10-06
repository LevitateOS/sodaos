package acceptance

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func testFingerprint() string { return "SHA256:" + strings.Repeat("A", 43) }

func testHostKey() string { return "ssh-ed25519 " + strings.Repeat("B", 68) }

func TestParseAccessSubnet(t *testing.T) {
	prefix, broadcast, err := parseAccessSubnet("10.89.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if broadcast.String() != "10.89.0.255" {
		t.Errorf("broadcast = %s", broadcast)
	}
	if !prefix.Contains(netip.MustParseAddr("10.89.0.11")) {
		t.Error("member IP not contained")
	}
	for _, bad := range []string{
		"10.89.0.5/24", // host bits set: strict networks only
		"11.0.0.0/8",   // public
		"10.0.0.0/7",   // not entirely private
		"fd00::/64",    // IPv6 rejected
		"not-a-subnet", // garbage
		"10.89.0.0/33", // bad bits
		"",             // empty
	} {
		if _, _, err := parseAccessSubnet(bad); err == nil {
			t.Errorf("subnet accepted: %q", bad)
		}
	}
}

func writeTestKey(t *testing.T, dir, name string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("private-key-bytes-never-printed"), mode); err != nil {
		t.Fatal(err)
	}
	// Force exact bits: WriteFile honors the process umask, so without this
	// the "group-readable" case collapses to 0600 under umask 077.
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateAccessUser(t *testing.T) {
	dir := t.TempDir()
	key := writeTestKey(t, dir, "alice-key", 0o600)
	good := `{"id": "1001", "login": "alice", "key_file": "` + key + `", "administrator": true}`
	user, err := validateAccessUser(json.RawMessage(good))
	if err != nil || user.id != "1001" || !user.administrator {
		t.Errorf("valid user rejected: %+v %v", user, err)
	}
	badIDs := []string{"0", "01", "abc", "", "100000000000000000000"}
	for _, id := range badIDs {
		body := `{"id": "` + id + `", "login": "alice", "key_file": "` + key + `", "administrator": true}`
		if _, err := validateAccessUser(json.RawMessage(body)); err == nil {
			t.Errorf("id accepted: %q", id)
		}
	}
	for _, login := range []string{"root", "Alice", "a", "toolonglogin12345678901234567890123", "has space", ""} {
		if login == "a" {
			continue // single lowercase letter is valid; covered below
		}
		body := `{"id": "1001", "login": "` + login + `", "key_file": "` + key + `", "administrator": true}`
		if _, err := validateAccessUser(json.RawMessage(body)); err == nil {
			t.Errorf("login accepted: %q", login)
		}
	}
	if _, err := validateAccessUser(json.RawMessage(`{"id": "1001", "login": "a", "key_file": "` + key + `", "administrator": false}`)); err != nil {
		t.Errorf("minimal login rejected: %v", err)
	}
	for _, admin := range []string{"1", `"true"`, "null", `[]`} {
		body := `{"id": "1001", "login": "alice", "key_file": "` + key + `", "administrator": ` + admin + `}`
		if _, err := validateAccessUser(json.RawMessage(body)); err == nil {
			t.Errorf("administrator accepted: %s", admin)
		}
	}
	extra := `{"id": "1001", "login": "alice", "key_file": "` + key + `", "administrator": true, "other": 1}`
	if _, err := validateAccessUser(json.RawMessage(extra)); err == nil {
		t.Error("extra user key accepted")
	}
	missing := `{"id": "1001", "login": "alice", "administrator": true}`
	if _, err := validateAccessUser(json.RawMessage(missing)); err == nil {
		t.Error("missing key_file accepted")
	}
	open := writeTestKey(t, dir, "open-key", 0o644)
	openBody := `{"id": "1001", "login": "alice", "key_file": "` + open + `", "administrator": true}`
	if _, err := validateAccessUser(json.RawMessage(openBody)); err == nil {
		t.Error("group-readable key accepted")
	}
	absent := `{"id": "1001", "login": "alice", "key_file": "/nonexistent/key", "administrator": true}`
	if _, err := validateAccessUser(json.RawMessage(absent)); err == nil {
		t.Error("missing key file accepted")
	}
}

func TestQuoteAccessPath(t *testing.T) {
	quoted, err := quoteAccessPath("/plain/path/file")
	if err != nil || quoted != `"/plain/path/file"` {
		t.Errorf("plain quote = %q %v", quoted, err)
	}
	quoted, err = quoteAccessPath(`/we\ird/pa"th`)
	if err != nil || quoted != `"/we\\ird/pa\"th"` {
		t.Errorf("tricky quote = %q %v", quoted, err)
	}
	for _, bad := range []string{"has\nnewline", "has\rcr"} {
		if _, err := quoteAccessPath(bad); err == nil {
			t.Errorf("multiline path accepted: %q", bad)
		}
	}
}

func TestParseAccessUID(t *testing.T) {
	for input, want := range map[string]int{"1000": 1000, " 42 ": 42, "+5": 5, "-3": -3, "0": 0, "1_0": 10, "007": 7} {
		got, err := parseAccessUID(input)
		if err != nil || got != want {
			t.Errorf("uid %q = %d %v", input, got, err)
		}
	}
	for _, bad := range []string{"", "abc", "1__", "_1", "1_", "1.5", "--3", "++3"} {
		if _, err := parseAccessUID(bad); err == nil {
			t.Errorf("uid accepted: %q", bad)
		}
	}
}

func TestAccessSSHOptions(t *testing.T) {
	request := &accessRequest{sshConfig: "/dev/null"}
	user := accessRequestUser{keyFile: "/keys/alice"}
	got := accessSSHOptions(request, user, "/out/alice-known-hosts")
	want := []string{
		"-F", "/dev/null",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		"-o", "IdentityAgent=none",
		"-o", "PreferredAuthentications=publickey",
		"-o", "ForwardAgent=no",
		"-o", "ClearAllForwardings=yes",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=/out/alice-known-hosts",
		"-i", "/keys/alice",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("options differ:\n%q", got)
	}
}

func accessFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

// preserveUmask restores the process umask after probes that restrict it.
func preserveUmask(t *testing.T) {
	t.Helper()
	current := syscall.Umask(0o077)
	syscall.Umask(current)
	t.Cleanup(func() { syscall.Umask(current) })
}

func TestRunDeveloperAccessValidation(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	preserveUmask(t)
	root := accessFixture(t)
	if err := os.WriteFile(filepath.Join(root, "target.json"), []byte(`{"SYNTHETIC_PRIVATE_MARKER":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RunDeveloperAccess([]string{root}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("bad request accepted")
	}
	if !strings.HasPrefix(err.Error(), "Developer access incomplete; retained probe state; failure type: ") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_MARKER") {
		t.Errorf("failure leaks request: %q", err.Error())
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "target.json" {
		t.Errorf("probe created output before validation: %v", entries)
	}
	if err := RunDeveloperAccess([]string{root, "extra"}, &bytes.Buffer{}); err == nil {
		t.Error("extra argv accepted")
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "wrong-target")
	if err := RunDeveloperAccess([]string{accessFixture(t)}, &bytes.Buffer{}); err == nil {
		t.Error("wrong env accepted")
	}
}
