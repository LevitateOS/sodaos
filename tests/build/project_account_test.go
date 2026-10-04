// Port of test_project_account.py: account script/file effects.
//
// project-account is still Python, so a driver per case reproduces the setUp
// doubles plus the test body and reports observations as JSON; every
// assertion below lives in Go.
//
// NOT ported: test_real_key_writer_lock_blocks_new_account_before_effects.
// It drives the real project_keys.py key writer, which no longer exists
// (retired with internal/host/project/project_keys.py; the successor is
// Runtime.AccessKeys in internal/host/project/access_keys.go, covered by
// internal/host/lifecycle_access_keys_test.go). The remaining lock-admission
// coverage (test_lock_contention) is ported below.
package build

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// accountDriver is the setUp doubles plus one case body from
// test_project_account.py, printing a JSON observation object.
const accountDriver = `
import base64, fcntl, importlib.machinery, importlib.util, json, os, sys, types
from pathlib import Path
from unittest.mock import patch

repo, root, case = sys.argv[1], Path(sys.argv[2]), sys.argv[3]
source = Path(repo) / 'project-os/rootfs/usr/libexec/soda/project-account'
loader = importlib.machinery.SourceFileLoader('project_account', str(source))
spec = importlib.util.spec_from_loader(loader.name, loader)
account = importlib.util.module_from_spec(spec)
loader.exec_module(account)

markers, keysdir = root / 'accounts', root / 'keys'
markers.mkdir(mode=0o700)
keysdir.mkdir(mode=0o755)
commands, users = [], {}
patch.object(account, 'ACCOUNTS', markers).start()
patch.object(account, 'KEYS', keysdir).start()
patch.object(account.os, 'geteuid', return_value=0).start()
native_fstat, native_lstat = os.fstat, Path.lstat

def root_owner(info):
    fields = {name: getattr(info, name) for name in dir(info) if name.startswith('st_')}
    fields.update(st_uid=0, st_gid=0)
    return types.SimpleNamespace(**fields)

patch.object(account.os, 'fstat', side_effect=lambda fd: root_owner(native_fstat(fd))).start()
patch.object(Path, 'lstat', lambda path: root_owner(native_lstat(path))).start()

def lookup(name):
    if name not in users:
        raise KeyError(name)
    return users[name]

def run_command(args, **options):
    assert options == {'check': True}, options
    commands.append(args)
    if args[0] == 'useradd':
        home = root / args[-1]
        home.mkdir(mode=0o700)
        users[args[-1]] = types.SimpleNamespace(pw_dir=str(home))
    elif args[0] != 'usermod':
        raise AssertionError('unexpected native command: %r' % (args,))

patch.object(account.pwd, 'getpwnam', side_effect=lookup).start()
patch.object(account.subprocess, 'run', side_effect=run_command).start()

def request(keys=None, admin=False):
    return {'login': 'alice', 'identity': 1, 'admin': admin, 'keys': [] if keys is None else keys}

def attempt(fn, *args, **kwargs):
    try:
        return {'mro': None, 'value': fn(*args, **kwargs)}
    except Exception as failure:
        return {'mro': [c.__name__ for c in type(failure).__mro__]}

def b64(path):
    return base64.b64encode(path.read_bytes()).decode()

def mode(path):
    return path.stat().st_mode & 0o777

obs = {}
if case == 'only_provisions':
    obs['result'] = account.provision(request())
    obs['keys'] = b64(keysdir / 'alice')
    obs['marker'] = b64(markers / 'alice')
    obs['marker_mode'] = mode(markers / 'alice')
    obs['keys_mode'] = mode(keysdir / 'alice')
    obs['shared'] = os.readlink(root / 'alice/shared')
    obs['cmd0'] = commands[0]
    obs['count'] = len(commands)
elif case == 'selected_keys':
    values = request(['ssh-ed25519 YWJj\n'], True)
    account.provision(values)
    obs['keys'] = b64(keysdir / 'alice')
    obs['last'] = commands[-1]
    (root / 'alice/work').write_text('later work')
    before = (keysdir / 'alice').stat().st_ino
    account.provision(values)
    obs['inode_before'] = before
    obs['inode_after'] = (keysdir / 'alice').stat().st_ino
    obs['work'] = (root / 'alice/work').read_text()
    obs['useradds'] = sum(command[0] == 'useradd' for command in commands)
elif case == 'retry_never_erases':
    account.provision(request(['ssh-ed25519 YWJj']))
    obs['retry'] = attempt(account.provision, request())
    obs['keys'] = b64(keysdir / 'alice')
    obs['count'] = len(commands)
elif case == 'join_not_apply':
    account.provision(request())
    obs['keyed'] = attempt(account.provision, request(['ssh-ed25519 YWJj']))
    obs['keys'] = b64(keysdir / 'alice')
    (markers / 'alice').write_text('2')
    obs['drifted'] = attempt(account.provision, request())
    obs['marker'] = b64(markers / 'alice')
elif case == 'occupied_inputs':
    (keysdir / 'alice').symlink_to(root / 'absent')
    obs['occupied'] = attempt(account.provision, request())
    obs['count_before'] = len(commands)
    (keysdir / 'alice').unlink()
    users['alice'] = types.SimpleNamespace(pw_dir=str(root / 'alice'))
    obs['unassociated'] = attempt(account.provision, request())
    obs['count_after'] = len(commands)
elif case == 'key_symlink':
    account.provision(request())
    other = root / 'other'
    other.write_bytes(b'preserve')
    (keysdir / 'alice').unlink()
    (keysdir / 'alice').symlink_to(other)
    obs['attempt'] = attempt(account.provision, request())
    obs['other'] = b64(other)
elif case == 'validation':
    mros = []
    for values in [None, 'key', ['ssh-ed25519 YWJj'] * 33, ['PRIVATE KEY'], ['ssh-ed25519 YWJj\nssh-ed25519 ZGVm']]:
        keyed = request()
        keyed['keys'] = values
        mros.append(attempt(account.provision, keyed))
    rooted = request()
    rooted['login'] = 'root'
    mros.append(attempt(account.provision, rooted))
    obs['attempts'] = mros
    obs['commands'] = commands
elif case == 'lock_contention':
    fd = os.open(keysdir, os.O_RDONLY | os.O_DIRECTORY)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        with patch.object(account.pwd, 'getpwnam', side_effect=AssertionError('account observed before admission')):
            obs['attempt'] = attempt(account.provision, request())
        obs['commands'] = list(commands)
        obs['keys_entries'] = sorted(p.name for p in keysdir.iterdir())
        obs['markers_entries'] = sorted(p.name for p in markers.iterdir())
    finally:
        os.close(fd)
    obs['result'] = account.provision(request())
elif case == 'lock_covers':
    native_sync = os.fsync
    events, failures, contents = [], [], []
    def assert_locked():
        fd = os.open(keysdir, os.O_RDONLY | os.O_DIRECTORY)
        try:
            try:
                fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                return
            failures.append('unlocked')
        finally:
            os.close(fd)
    def run(args, **options):
        assert_locked()
        return run_command(args, **options)
    def sync(fd):
        assert_locked()
        info = os.fstat(fd)
        if account.stat.S_ISREG(info.st_mode):
            expected = ((markers / 'alice', b'1'), (keysdir / 'alice', b'ssh-ed25519 YWJj\n'))
            path, content = next(
                (p, value) for p, value in expected if p.exists() and p.stat().st_ino == info.st_ino)
            contents.append(content == path.read_bytes())
            events.append('marker' if path.parent == markers else 'keyfile')
        else:
            events.append('markers' if info.st_ino == markers.stat().st_ino else 'keys')
        native_sync(fd)
    with patch.object(account.os, 'fsync', side_effect=sync), patch.object(account.subprocess, 'run', side_effect=run):
        account.provision(request(['ssh-ed25519 YWJj'], True))
    obs['events'] = events
    obs['failures'] = failures
    obs['contents'] = contents
    obs['last'] = commands[-1][0]
elif case == 'failed_provisioning':
    with patch.object(account.os, 'fsync', side_effect=OSError('synthetic sync failure')):
        obs['attempt'] = attempt(account.provision, request())
    obs['marker'] = b64(markers / 'alice')
    obs['home_is_dir'] = (root / 'alice').is_dir()
    fd = os.open(keysdir, os.O_RDONLY | os.O_DIRECTORY)
    try:
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            obs['relocked'] = True
        except BlockingIOError:
            obs['relocked'] = False
    finally:
        os.close(fd)
else:
    raise SystemExit('unknown case: ' + case)
print(json.dumps(obs))
`

// runAccountCase runs one driver case in a fresh temp root and returns its
// decoded JSON observations.
func runAccountCase(t *testing.T, caseName string) map[string]any {
	t.Helper()
	result := Run(t, RunOpt{}, Python3(t), "-c", accountDriver, RepoRoot, TempDir(t), caseName)
	Require(t, result.Code == 0, "account driver %s failed: %s", caseName, result.Stderr)
	var obs map[string]any
	Require(t, json.Unmarshal([]byte(result.Stdout), &obs) == nil, "parse %s output %q", caseName, result.Stdout)
	return obs
}

// accountAttemptMRO extracts the exception MRO from an attempt observation
// (nil when the driver call succeeded).
func accountAttemptMRO(t *testing.T, value any) []string {
	t.Helper()
	attempt, ok := value.(map[string]any)
	Require(t, ok, "attempt is %T", value)
	raw, ok := attempt["mro"].([]any)
	if !ok {
		return nil
	}
	mro := make([]string, 0, len(raw))
	for _, entry := range raw {
		text, ok := entry.(string)
		Require(t, ok, "mro entry is %T", entry)
		mro = append(mro, text)
	}
	return mro
}

func accountText(t *testing.T, value any) string {
	t.Helper()
	text, ok := value.(string)
	Require(t, ok, "bytes are %T", value)
	raw, err := base64.StdEncoding.DecodeString(text)
	Require(t, err == nil, "decode bytes: %v", err)
	return string(raw)
}

func accountStrings(t *testing.T, value any) []string {
	t.Helper()
	raw, ok := value.([]any)
	Require(t, ok, "list is %T", value)
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		text, ok := entry.(string)
		Require(t, ok, "list entry is %T", entry)
		out = append(out, text)
	}
	return out
}

func TestAccountOnlyProvisionsLockedHomeMarkerSharedAndEmptyKeyfile(t *testing.T) {
	obs := runAccountCase(t, "only_provisions")
	result, ok := obs["result"].(map[string]any)
	Require(t, ok, "result is %T", obs["result"])
	Check(t, result["login"] == "alice" && result["identity"] == float64(1), "result = %v", result)
	Check(t, accountText(t, obs["keys"]) == "", "keys = %q", accountText(t, obs["keys"]))
	Check(t, accountText(t, obs["marker"]) == "1", "marker = %q", accountText(t, obs["marker"]))
	Check(t, obs["marker_mode"] == float64(0o600), "marker mode = %v", obs["marker_mode"])
	Check(t, obs["keys_mode"] == float64(0o644), "keys mode = %v", obs["keys_mode"])
	Check(t, obs["shared"] == "/srv/project/shared", "shared = %v", obs["shared"])
	cmd := accountStrings(t, obs["cmd0"])
	found := false
	for i, word := range cmd {
		if word == "--password" && i+1 < len(cmd) {
			Check(t, cmd[i+1] == "!", "password = %q", cmd[i+1])
		}
		found = found || word == "--create-home"
	}
	Check(t, found, "--create-home missing in %v", cmd)
	Require(t, len(cmd) >= 2, "cmd0 = %v", cmd)
	Check(t, cmd[len(cmd)-2] == "soda-project" && cmd[len(cmd)-1] == "alice", "cmd0 tail = %v", cmd)
	Check(t, obs["count"] == float64(1), "commands = %v", obs["count"])
}

func TestAccountSelectedKeysAndCreationOwnerPrivilegeRemainReal(t *testing.T) {
	obs := runAccountCase(t, "selected_keys")
	Check(t, accountText(t, obs["keys"]) == "ssh-ed25519 YWJj\n", "keys wrong")
	last := accountStrings(t, obs["last"])
	Require(t, len(last) == 5, "last = %v", last)
	Check(t, last[0] == "usermod" && last[1] == "--append" && last[2] == "--groups" &&
		last[3] == "wheel" && last[4] == "alice", "last = %v", last)
	Check(t, obs["inode_before"] == obs["inode_after"], "keyfile replaced")
	Check(t, obs["work"] == "later work", "work = %v", obs["work"])
	Check(t, obs["useradds"] == float64(1), "useradds = %v", obs["useradds"])
}

func TestAccountOnlyRetryNeverErasesExistingKeys(t *testing.T) {
	obs := runAccountCase(t, "retry_never_erases")
	Check(t, raisedAs(accountAttemptMRO(t, obs["retry"]), "ValueError"), "retry accepted")
	Check(t, accountText(t, obs["keys"]) == "ssh-ed25519 YWJj\n", "keys wrong")
	Check(t, obs["count"] == float64(1), "commands = %v", obs["count"])
}

func TestAccountJoinIsNotKeyApplyAndPreservesDrift(t *testing.T) {
	obs := runAccountCase(t, "join_not_apply")
	Check(t, raisedAs(accountAttemptMRO(t, obs["keyed"]), "ValueError"), "keyed join accepted")
	Check(t, accountText(t, obs["keys"]) == "", "keys = %q", accountText(t, obs["keys"]))
	Check(t, raisedAs(accountAttemptMRO(t, obs["drifted"]), "ValueError"), "drifted join accepted")
	Check(t, accountText(t, obs["marker"]) == "2", "marker = %q", accountText(t, obs["marker"]))
}

func TestAccountOccupiedInputsAndUnassociatedUsersRefuseBeforeCommands(t *testing.T) {
	obs := runAccountCase(t, "occupied_inputs")
	Check(t, raisedAs(accountAttemptMRO(t, obs["occupied"]), "ValueError"), "occupied input accepted")
	Check(t, obs["count_before"] == float64(0), "commands = %v", obs["count_before"])
	Check(t, raisedAs(accountAttemptMRO(t, obs["unassociated"]), "FileNotFoundError"), "unassociated accepted")
	Check(t, obs["count_after"] == float64(0), "commands = %v", obs["count_after"])
}

func TestAccountKeySymlinkIsNotFollowed(t *testing.T) {
	obs := runAccountCase(t, "key_symlink")
	Check(t, raisedAs(accountAttemptMRO(t, obs["attempt"]), "OSError"), "symlink followed")
	Check(t, accountText(t, obs["other"]) == "preserve", "other overwritten")
}

func TestAccountValidationBeforeNativeEffects(t *testing.T) {
	obs := runAccountCase(t, "validation")
	attempts, ok := obs["attempts"].([]any)
	Require(t, ok && len(attempts) == 6, "attempts = %v", obs["attempts"])
	for i, attempt := range attempts {
		Check(t, raisedAs(accountAttemptMRO(t, attempt), "ValueError"), "input %d accepted", i)
	}
	commands, ok := obs["commands"].([]any)
	Require(t, ok, "commands is %T", obs["commands"])
	Check(t, len(commands) == 0, "effects before validation: %v", commands)
}

func TestAccountLockContentionRefusesBeforeObservationOrCommands(t *testing.T) {
	obs := runAccountCase(t, "lock_contention")
	Check(t, raisedAs(accountAttemptMRO(t, obs["attempt"]), "BlockingIOError"), "contention not refused")
	commands, ok := obs["commands"].([]any)
	Require(t, ok, "commands is %T", obs["commands"])
	Check(t, len(commands) == 0, "commands = %v", commands)
	Check(t, len(accountStrings(t, obs["keys_entries"])) == 0, "keys entries = %v", obs["keys_entries"])
	Check(t, len(accountStrings(t, obs["markers_entries"])) == 0, "markers entries = %v", obs["markers_entries"])
	result, ok := obs["result"].(map[string]any)
	Require(t, ok, "result is %T", obs["result"])
	Check(t, result["login"] == "alice", "relock provision = %v", result)
}

func TestAccountLockCoversFileDurabilityAndAccountCommands(t *testing.T) {
	obs := runAccountCase(t, "lock_covers")
	Check(t, len(accountStrings(t, obs["failures"])) == 0, "unlocked effects")
	events := accountStrings(t, obs["events"])
	Require(t, len(events) == 4, "events = %v", events)
	Check(t, events[0] == "marker" && events[1] == "markers" && events[2] == "keyfile" && events[3] == "keys",
		"events = %v", events)
	contents, ok := obs["contents"].([]any)
	Require(t, ok && len(contents) == 2, "contents = %v", obs["contents"])
	Check(t, contents[0] == true && contents[1] == true, "contents = %v", contents)
	Check(t, obs["last"] == "usermod", "last = %v", obs["last"])
}

func TestAccountFailedProvisioningReleasesLockWithoutRemovingPartialFiles(t *testing.T) {
	obs := runAccountCase(t, "failed_provisioning")
	Check(t, raisedAs(accountAttemptMRO(t, obs["attempt"]), "OSError"), "sync failure swallowed")
	Check(t, accountText(t, obs["marker"]) == "1", "marker = %q", accountText(t, obs["marker"]))
	Check(t, obs["home_is_dir"] == true, "home missing")
	Check(t, obs["relocked"] == true, "lock stranded")
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
