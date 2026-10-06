package acceptance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
