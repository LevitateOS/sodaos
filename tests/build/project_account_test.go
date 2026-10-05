// Account binary/file effects, driven against the compiled Rust
// project-account with redirected roots and PATH doubles.
//
// The retired interpreter-driven twin asserted per-case exception types
// through an in-process driver; the binary collapses every failure to one
// stderr line and exit 1, so these tests assert that exact contract plus
// the filesystem and argv effects. Durability ordering under the lock is
// covered by the crate's fsync_order_holds_lock_and_orders_durably unit
// test, which observes each sync from inside the provision.
package build

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// accountFailure is the only failure the binary ever reports.
const accountFailure = "account provisioning unconfirmed; inspect native account and managed files\n"

func accountBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-project-account", "project-account")
}

// accountEnv is one hermetic binary run: temp root, managed dirs, and
// useradd/usermod doubles that log argv and simulate user creation.
type accountEnv struct {
	root string
	env  []string
}

const accountUseraddDouble = `#!/bin/sh
printf '%s\n' "useradd $*" >>"$SODA_PROJECT_ACCOUNT_TEST_ROOT/commands.log"
login=
for last in "$@"; do login=$last; done
home=$SODA_PROJECT_ACCOUNT_TEST_ROOT/home/$login
mkdir -p "$home"
printf '%s:%s\n' "$login" "$home" >>"$SODA_PROJECT_ACCOUNT_TEST_ROOT/passwd"
`

const accountUsermodDouble = `#!/bin/sh
printf '%s\n' "usermod $*" >>"$SODA_PROJECT_ACCOUNT_TEST_ROOT/commands.log"
`

func newAccountEnv(t *testing.T) *accountEnv {
	t.Helper()
	root := TempDir(t)
	for _, dir := range []string{"accounts", "keys", "bin", "home"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.Chmod(filepath.Join(root, "accounts"), 0o700); err != nil {
		t.Fatalf("chmod accounts: %v", err)
	}
	WriteFile(t, filepath.Join(root, "bin", "useradd"), []byte(accountUseraddDouble), 0o755)
	WriteFile(t, filepath.Join(root, "bin", "usermod"), []byte(accountUsermodDouble), 0o755)
	env := SetEnv(os.Environ(), "SODA_PROJECT_ACCOUNT_TEST_ROOT", root)
	env = SetEnv(env, "PATH", filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
	return &accountEnv{root: root, env: env}
}

// run feeds stdin to the binary and captures the exact byte contract.
func (e *accountEnv) run(t *testing.T, stdin []byte, extra ...string) ProcResult {
	t.Helper()
	cmd := exec.Command(accountBinary(t))
	cmd.Env = e.env
	for i := 0; i+1 < len(extra); i += 2 {
		cmd.Env = SetEnv(cmd.Env, extra[i], extra[i+1])
	}
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, ok := err.(*exec.ExitError)
		Require(t, ok, "run project-account: %v", err)
		code = exit.ExitCode()
	}
	return ProcResult{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

func (e *accountEnv) commands(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.root, "commands.log"))
	if os.IsNotExist(err) {
		return nil
	}
	Require(t, err == nil, "read commands log: %v", err)
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func (e *accountEnv) read(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.root, rel))
	Require(t, err == nil, "read %s: %v", rel, err)
	return string(data)
}

func (e *accountEnv) mode(t *testing.T, rel string) os.FileMode {
	t.Helper()
	info, err := os.Stat(filepath.Join(e.root, rel))
	Require(t, err == nil, "stat %s: %v", rel, err)
	return info.Mode().Perm()
}

func accountDoc(login string, identity int64, admin bool, keys []string) []byte {
	body, err := json.Marshal(map[string]any{
		"login": login, "identity": identity, "admin": admin, "keys": keys,
	})
	if err != nil {
		panic(err)
	}
	return body
}

func requireAccountFailure(t *testing.T, result ProcResult, what string) {
	t.Helper()
	Check(t, result.Code == 1, "%s: exit = %d", what, result.Code)
	Check(t, result.Stdout == "", "%s: stdout = %q", what, result.Stdout)
	Check(t, result.Stderr == accountFailure, "%s: stderr = %q", what, result.Stderr)
}

func TestAccountOnlyProvisionsLockedHomeMarkerSharedAndEmptyKeyfile(t *testing.T) {
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, false, []string{}))
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	Check(t, result.Stdout == "{\"login\": \"alice\", \"identity\": 1}\n", "stdout = %q", result.Stdout)
	Check(t, result.Stderr == "", "stderr = %q", result.Stderr)
	Check(t, env.read(t, "keys/alice") == "", "keys = %q", env.read(t, "keys/alice"))
	Check(t, env.read(t, "accounts/alice") == "1", "marker = %q", env.read(t, "accounts/alice"))
	Check(t, env.mode(t, "accounts/alice") == 0o600, "marker mode = %o", env.mode(t, "accounts/alice"))
	Check(t, env.mode(t, "keys/alice") == 0o644, "keys mode = %o", env.mode(t, "keys/alice"))
	link, err := os.Readlink(filepath.Join(env.root, "home/alice/shared"))
	Require(t, err == nil, "readlink shared: %v", err)
	Check(t, link == "/srv/project/shared", "shared = %q", link)
	commands := env.commands(t)
	Require(t, len(commands) == 1, "commands = %v", commands)
	Check(t, commands[0] == "useradd --create-home --shell /bin/bash --password ! --groups soda-project alice",
		"useradd = %q", commands[0])
}

func TestAccountSelectedKeysAndCreationOwnerPrivilegeRemainReal(t *testing.T) {
	env := newAccountEnv(t)
	body := accountDoc("alice", 1, true, []string{"ssh-ed25519 YWJj\n"})
	result := env.run(t, body)
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	Check(t, env.read(t, "keys/alice") == "ssh-ed25519 YWJj\n", "keys wrong")
	commands := env.commands(t)
	Require(t, len(commands) == 2, "commands = %v", commands)
	Check(t, commands[1] == "usermod --append --groups wheel alice", "last = %q", commands[1])
	WriteFile(t, filepath.Join(env.root, "home/alice/work"), []byte("later work"), 0o644)
	before, err := os.Stat(filepath.Join(env.root, "keys/alice"))
	Require(t, err == nil, "stat keys: %v", err)
	result = env.run(t, body)
	Require(t, result.Code == 0, "rerun exit = %d: %q", result.Code, result.Stderr)
	after, err := os.Stat(filepath.Join(env.root, "keys/alice"))
	Require(t, err == nil, "stat keys: %v", err)
	Check(t, os.SameFile(before, after), "keyfile replaced on rerun")
	Check(t, env.read(t, "home/alice/work") == "later work", "work lost")
	useradds := 0
	for _, command := range env.commands(t) {
		if strings.HasPrefix(command, "useradd ") {
			useradds++
		}
	}
	Check(t, useradds == 1, "useradds = %d", useradds)
}

func TestAccountOnlyRetryNeverErasesExistingKeys(t *testing.T) {
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, false, []string{"ssh-ed25519 YWJj"}))
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "retry")
	Check(t, env.read(t, "keys/alice") == "ssh-ed25519 YWJj\n", "keys wrong")
	Check(t, len(env.commands(t)) == 1, "commands = %v", env.commands(t))
}

func TestAccountJoinIsNotKeyApplyAndPreservesDrift(t *testing.T) {
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, false, []string{}))
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{"ssh-ed25519 YWJj"})), "keyed join")
	Check(t, env.read(t, "keys/alice") == "", "keys = %q", env.read(t, "keys/alice"))
	WriteFile(t, filepath.Join(env.root, "accounts/alice"), []byte("2"), 0o600)
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "drifted join")
	Check(t, env.read(t, "accounts/alice") == "2", "marker = %q", env.read(t, "accounts/alice"))
}

func TestAccountOccupiedInputsAndUnassociatedUsersRefuseBeforeCommands(t *testing.T) {
	env := newAccountEnv(t)
	Require(t, os.Symlink(filepath.Join(env.root, "absent"), filepath.Join(env.root, "keys/alice")) == nil, "dangle")
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "occupied input")
	Check(t, len(env.commands(t)) == 0, "commands = %v", env.commands(t))
	Require(t, os.Remove(filepath.Join(env.root, "keys/alice")) == nil, "unlink")
	WriteFile(t, filepath.Join(env.root, "passwd"),
		[]byte("alice:"+filepath.Join(env.root, "home/alice")+"\n"), 0o644)
	Require(t, os.MkdirAll(filepath.Join(env.root, "home/alice"), 0o755) == nil, "home")
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "unassociated user")
	Check(t, len(env.commands(t)) == 0, "commands = %v", env.commands(t))
}

func TestAccountKeySymlinkIsNotFollowed(t *testing.T) {
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, false, []string{}))
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	WriteFile(t, filepath.Join(env.root, "other"), []byte("preserve"), 0o644)
	Require(t, os.Remove(filepath.Join(env.root, "keys/alice")) == nil, "unlink")
	Require(t, os.Symlink(filepath.Join(env.root, "other"), filepath.Join(env.root, "keys/alice")) == nil, "link")
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "symlinked keyfile")
	Check(t, env.read(t, "other") == "preserve", "other overwritten")
}

func TestAccountValidationBeforeNativeEffects(t *testing.T) {
	env := newAccountEnv(t)
	many := make([]string, 33)
	for i := range many {
		many[i] = "ssh-ed25519 YWJj"
	}
	bodies := [][]byte{
		accountDoc("alice", 1, false, nil),
		[]byte(`{"login":"alice","identity":1,"admin":false,"keys":"key"}`),
		accountDoc("alice", 1, false, many),
		accountDoc("alice", 1, false, []string{"PRIVATE KEY"}),
		accountDoc("alice", 1, false, []string{"ssh-ed25519 YWJj\nssh-ed25519 ZGVm"}),
		accountDoc("root", 1, false, []string{}),
	}
	for i, body := range bodies {
		requireAccountFailure(t, env.run(t, body), string(rune('a'+i)))
	}
	Check(t, len(env.commands(t)) == 0, "effects before validation: %v", env.commands(t))
}

func TestAccountLockContentionRefusesBeforeObservationOrCommands(t *testing.T) {
	env := newAccountEnv(t)
	held, err := os.Open(filepath.Join(env.root, "keys"))
	Require(t, err == nil, "open keys: %v", err)
	defer held.Close()
	Require(t, unix.Flock(int(held.Fd()), unix.LOCK_EX|unix.LOCK_NB) == nil, "hold lock")
	requireAccountFailure(t, env.run(t, accountDoc("alice", 1, false, []string{})), "contention")
	Check(t, len(env.commands(t)) == 0, "commands = %v", env.commands(t))
	for _, dir := range []string{"keys", "accounts"} {
		entries, err := os.ReadDir(filepath.Join(env.root, dir))
		Require(t, err == nil, "readdir %s: %v", dir, err)
		Check(t, len(entries) == 0, "%s entries = %v", dir, entries)
	}
	Require(t, unix.Flock(int(held.Fd()), unix.LOCK_UN) == nil, "release lock")
	result := env.run(t, accountDoc("alice", 1, false, []string{}))
	Require(t, result.Code == 0, "relock exit = %d: %q", result.Code, result.Stderr)
	Check(t, result.Stdout == "{\"login\": \"alice\", \"identity\": 1}\n", "relock stdout = %q", result.Stdout)
}

func TestAccountLockCoversFileDurabilityAndAccountCommands(t *testing.T) {
	// Binary-observable remainder of the durability case: one provision
	// writes the marker then the keyfile with exact bytes and modes and
	// runs useradd before the usermod grant. The per-sync lock-holding
	// and fsync order are asserted inside the crate, where each sync is
	// observable.
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, true, []string{"ssh-ed25519 YWJj"}))
	Require(t, result.Code == 0, "exit = %d: %q", result.Code, result.Stderr)
	Check(t, env.read(t, "accounts/alice") == "1", "marker wrong")
	Check(t, env.mode(t, "accounts/alice") == 0o600, "marker mode wrong")
	Check(t, env.read(t, "keys/alice") == "ssh-ed25519 YWJj\n", "keys wrong")
	Check(t, env.mode(t, "keys/alice") == 0o644, "keys mode wrong")
	commands := env.commands(t)
	Require(t, len(commands) == 2, "commands = %v", commands)
	Check(t, strings.HasPrefix(commands[0], "useradd "), "first = %q", commands[0])
	Check(t, strings.HasPrefix(commands[1], "usermod "), "last = %q", commands[1])
}

func TestAccountFailedProvisioningReleasesLockWithoutRemovingPartialFiles(t *testing.T) {
	env := newAccountEnv(t)
	result := env.run(t, accountDoc("alice", 1, false, []string{}), "SODA_PROJECT_ACCOUNT_FAIL_SYNC", "1")
	requireAccountFailure(t, result, "sync failure")
	Check(t, env.read(t, "accounts/alice") == "1", "marker = %q", env.read(t, "accounts/alice"))
	info, err := os.Stat(filepath.Join(env.root, "home/alice"))
	Require(t, err == nil, "home missing: %v", err)
	Check(t, info.IsDir(), "home is not a dir")
	relock, err := os.Open(filepath.Join(env.root, "keys"))
	Require(t, err == nil, "open keys: %v", err)
	defer relock.Close()
	Check(t, unix.Flock(int(relock.Fd()), unix.LOCK_EX|unix.LOCK_NB) == nil, "lock stranded")
}

func TestAccountStdinContractBounds(t *testing.T) {
	env := newAccountEnv(t)
	for _, body := range [][]byte{
		{},
		[]byte("not json"),
		[]byte("null"),
		[]byte("[]"),
		[]byte("{}"),
		[]byte("\xff\xfe"),
		[]byte(`{"login": "alice"}`),
		bytes.Repeat([]byte("x"), 65537),
	} {
		requireAccountFailure(t, env.run(t, body), "malformed stdin")
	}
	// A 65536-byte valid document succeeds; the padding spreads over
	// four keys so each stays under the key length limit.
	prefix := `{"login": "alice", "identity": 1, "admin": false, "keys": [`
	suffix := `]}`
	per := (65536 - len(prefix) - len(suffix) - 3*2 - 4*8) / 4
	lens := [4]int{per, per, per, per}
	lens[3] += 65536 - (len(prefix) + len(suffix) + 6 + 4*8 + 4*per)
	var edge bytes.Buffer
	edge.WriteString(prefix)
	for i, n := range lens {
		Require(t, 6+n <= 16384, "key %d too long", i)
		if i > 0 {
			edge.WriteString(", ")
		}
		edge.WriteString(`"ssh-x ` + strings.Repeat("A", n) + `"`)
	}
	edge.WriteString(suffix)
	Require(t, edge.Len() == 65536, "edge len = %d", edge.Len())
	result := env.run(t, edge.Bytes())
	Require(t, result.Code == 0, "edge exit = %d: %q", result.Code, result.Stderr)
	Check(t, result.Stdout == "{\"login\": \"alice\", \"identity\": 1}\n", "edge stdout = %q", result.Stdout)
	Check(t, len(env.commands(t)) == 1, "commands = %v", env.commands(t))
}

func TestAccountNativePasswordSSHPolicyIsNotRelaxed(t *testing.T) {
	var matches []string
	err := filepath.WalkDir(filepath.Join(RepoRoot, "project-os/rootfs/etc/ssh"),
		func(path string, entry os.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
				matches = append(matches, path)
			}
			return err
		})
	Require(t, err == nil, "walk ssh conf: %v", err)
	Require(t, len(matches) > 0, "no ssh conf files")
	var config strings.Builder
	for _, match := range matches {
		data, err := os.ReadFile(match)
		Require(t, err == nil, "read %s: %v", match, err)
		config.Write(data)
		config.WriteString("\n")
	}
	for _, want := range []string{"PasswordAuthentication no", "PermitRootLogin no", "KbdInteractiveAuthentication no"} {
		Check(t, strings.Contains(config.String(), want), "%q missing", want)
	}
}
