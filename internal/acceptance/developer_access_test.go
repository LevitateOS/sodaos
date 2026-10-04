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

func writeAccessRequest(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	request := map[string]any{
		"target":         "test-target-01",
		"revision":       strings.Repeat("a", 40),
		"project_id":     "p" + strings.Repeat("b", 24),
		"subnet":         "10.89.0.0/24",
		"browser_result": filepath.Join(root, "browser.json"),
		"host_key_file":  filepath.Join(root, "hostkey"),
		"users": []any{
			map[string]any{"id": "1001", "login": "alice", "key_file": filepath.Join(root, "alice-key"), "administrator": true},
			map[string]any{"id": "1002", "login": "bob", "key_file": filepath.Join(root, "bob-key"), "administrator": false},
		},
	}
	if mutate != nil {
		mutate(request)
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeAccessSupport(t *testing.T, root string) {
	t.Helper()
	project := "p" + strings.Repeat("b", 24)
	revision := strings.Repeat("a", 40)
	user := func(id, login string) map[string]any {
		return map[string]any{"id": id, "login": login, "connection": map[string]any{
			"environment": map[string]any{"id": project, "running": true, "ip": "10.89.0.11"},
			"host_key":    testHostKey(),
			"fingerprint": testFingerprint(),
		}}
	}
	browser := map[string]any{
		"target": "test-target-01", "revision": revision, "outcome": "passed-scoped-journey",
		"access": map[string]any{
			"native_join_confirmed": true, "reservation_id": project,
			"users": []any{user("1001", "alice"), user("1002", "bob")},
		},
	}
	data, err := json.Marshal(browser)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "browser.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hostkey"), []byte(testHostKey()+" test-comment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "alice-key"), []byte("alice-private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bob-key"), []byte("bob-private"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAccessRequest(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "test-target-01")
	root := accessFixture(t)
	writeAccessSupport(t, root)
	writeAccessRequest(t, root, nil)
	request, err := loadAccessRequest(root)
	if err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if request.publicKey != testHostKey() || len(request.users) != 2 || request.sshConfig != "/dev/null" {
		t.Errorf("request = %+v", request)
	}
	cases := map[string]func(map[string]any){
		"bad target":      func(m map[string]any) { m["target"] = "bad target!" },
		"bad revision":    func(m map[string]any) { m["revision"] = "short" },
		"bad project":     func(m map[string]any) { m["project_id"] = "x" },
		"public subnet":   func(m map[string]any) { m["subnet"] = "11.0.0.0/8" },
		"extra key":       func(m map[string]any) { m["extra"] = 1 },
		"missing users":   func(m map[string]any) { delete(m, "users") },
		"one user":        func(m map[string]any) { m["users"] = m["users"].([]any)[:1] },
		"swapped admin":   func(m map[string]any) { m["users"].([]any)[0].(map[string]any)["administrator"] = false },
		"duplicate ids":   func(m map[string]any) { m["users"].([]any)[1].(map[string]any)["id"] = "1001" },
		"missing browser": func(m map[string]any) { m["browser_result"] = "/nonexistent/browser.json" },
	}
	for name, mutate := range cases {
		root := accessFixture(t)
		writeAccessSupport(t, root)
		writeAccessRequest(t, root, mutate)
		if _, err := loadAccessRequest(root); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	t.Setenv("SODA_NATIVE_VALIDATE", "other-target")
	valid := accessFixture(t)
	writeAccessSupport(t, valid)
	writeAccessRequest(t, valid, nil)
	if _, err := loadAccessRequest(valid); err == nil {
		t.Error("environment mismatch accepted")
	}
}

// stubAccessTransport installs ssh-keygen, ssh, scp and sftp emulators that
// prove the full success path without a network.
func stubAccessTransport(t *testing.T) (state, log string) {
	t.Helper()
	dir := t.TempDir()
	state = filepath.Join(dir, "remote-state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	log = filepath.Join(dir, "argv.log")
	t.Setenv("STUBSTATE", state)
	t.Setenv("STUBLOG", log)
	t.Setenv("STUBFINGERPRINT", testFingerprint())
	keygen := "#!/bin/sh\nprintf '256 %s %s (ED25519)\\n' \"$STUBFINGERPRINT\" \"$2\"\n"
	ssh := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" >> \"$STUBLOG\"\n" +
		"login=\"\"; key=\"\"; prev=\"\"; last=\"\"\n" +
		"for arg in \"$@\"; do\n" +
		"  case \"$arg\" in *@10.*) login=\"${arg%%@*}\";; esac\n" +
		"  if [ \"$prev\" = -i ]; then key=\"$arg\"; fi\n" +
		"  prev=\"$arg\"; last=\"$arg\"\n" +
		"done\n" +
		"case \" $@ \" in *' -tt '*) cat >/dev/null; printf '\\r\\nSODA-PTY:%s\\r\\nmore\\r\\n' \"$login\"; exit 0;; esac\n" +
		"if [ \"$last\" = true ]; then case \"$key\" in *alice*) echo 'debug: Permission denied (publickey,gssapi-keyex).' >&2; exit 255;; esac; exit 8; fi\n" +
		"case \"$key\" in *\"$login\"*) ;; *) exit 8;; esac\n" +
		"case \"$last\" in\n" +
		"  *'id -un'*) printf '%s\\n1000\\n/home/%s\\n' \"$login\" \"$login\";;\n" +
		"  *'uid_map'*) printf '0 100000 65536\\n';;\n" +
		"  *'sudo -n'*) if [ \"$login\" = alice ]; then printf '0\\n'; else echo \"Sorry, user $login is not allowed to execute.\" >&2; exit 1; fi;;\n" +
		"  *'mkdir '*) exit 0;;\n" +
		"  *) exit 9;;\n" +
		"esac\n"
	scp := "#!/bin/sh\n" +
		"src=\"\"; dst=\"\"; for arg in \"$@\"; do src=\"$dst\"; dst=\"$arg\"; done\n" +
		"remote=\"$dst\"; case \"$src\" in *:* ) remote=\"$src\";; esac; login=\"${remote%%@*}\"\n" +
		"case \"$src\" in *:* ) cp \"$STUBSTATE/scp-$login\" \"$dst\";; * ) cp \"$src\" \"$STUBSTATE/scp-$login\";; esac\n"
	sftp := "#!/bin/sh\n" +
		"target=\"\"; for arg in \"$@\"; do target=\"$arg\"; done; login=\"${target%%@*}\"\n" +
		"while IFS= read -r line; do\n" +
		"  set -- $line; op=\"$1\"; a=\"$(printf '%s' \"$2\" | tr -d '\"')\"; b=\"$(printf '%s' \"$3\" | tr -d '\"')\"\n" +
		"  case \"$op\" in put) cp \"$a\" \"$STUBSTATE/sftp-$login\";; get) cp \"$STUBSTATE/sftp-$login\" \"$b\";; *) exit 9;; esac\n" +
		"done\n"
	for name, body := range map[string]string{"ssh-keygen": keygen, "ssh": ssh, "scp": scp, "sftp": sftp} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return state, log
}

func TestRunDeveloperAccessEndToEnd(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "test-target-01")
	preserveUmask(t)
	stubAccessTransport(t)
	root := accessFixture(t)
	writeAccessSupport(t, root)
	writeAccessRequest(t, root, nil)
	var stdout bytes.Buffer
	if err := RunDeveloperAccess([]string{root}, &stdout); err != nil {
		t.Fatalf("end-to-end access failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Declared memberships passed native SSH, PTY, SCP/SFTP, owner sudo and cross-user key denial.") {
		t.Errorf("success output = %q", stdout.String())
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var output string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "sodaspaces-access-") {
			output = filepath.Join(root, entry.Name())
		}
	}
	if output == "" {
		t.Fatalf("no output directory in %v", entries)
	}
	results, err := os.ReadFile(filepath.Join(output, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded accessResults
	if json.Unmarshal(results, &decoded) != nil {
		t.Fatalf("results invalid: %s", results)
	}
	if decoded.Outcome != "passed-scoped-access" || decoded.CrossUserKey != "public-key authentication denied" {
		t.Errorf("outcome = %+v", decoded)
	}
	if len(decoded.Users) != 2 || decoded.Users[0].Login != "alice" || decoded.Users[1].IP != "10.89.0.11" {
		t.Errorf("users = %+v", decoded.Users)
	}
	if decoded.Transport != "direct project IP" || decoded.ClientArch != machineArch() || decoded.ScriptSHA256 == "" {
		t.Errorf("results = %+v", decoded)
	}
	if len(decoded.ScriptSHA256) != 64 {
		t.Errorf("script digest length = %d", len(decoded.ScriptSHA256))
	}
	for _, c := range decoded.ScriptSHA256 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("script digest not lowercase hex: %q", decoded.ScriptSHA256)
		}
	}
	// Key order must match the retired Python insertion order.
	ordered := []string{`"revision"`, `"target"`, `"project"`, `"client"`, `"client_arch"`, `"transport"`, `"script_sha256"`, `"users"`, `"cross_user_key"`, `"outcome"`}
	previous := -1
	for _, key := range ordered {
		next := strings.Index(string(results), key)
		if next <= previous {
			t.Errorf("key order breaks at %s:\n%s", key, results)
		}
		previous = next
	}
	if !strings.HasSuffix(string(results), "}\n") {
		t.Error("results lack trailing newline")
	}
	payload, err := os.ReadFile(filepath.Join(output, "payload"))
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != 8192*(len("sodaspaces-access-")+32+1) {
		t.Errorf("payload size = %d", len(payload))
	}
	known, err := os.ReadFile(filepath.Join(output, "alice-known-hosts"))
	if err != nil {
		t.Fatal(err)
	}
	if string(known) != "10.89.0.11 "+testHostKey()+"\n" {
		t.Errorf("known-hosts = %q", known)
	}
}
