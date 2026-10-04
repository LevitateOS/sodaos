// Port of test_project_factory_roles.py: factory helper state machine.
//
// project-factory-roles is still Python, so a driver per case reproduces the
// setUp doubles plus the test body and reports observations as JSON; every
// assertion below lives in Go.
package build

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// rolesDriver is the setUp doubles plus one case body from
// test_project_factory_roles.py, printing a JSON observation object.
const rolesDriver = `
import base64, hashlib, importlib.machinery, importlib.util, json, os, sys, types
from pathlib import Path
from unittest.mock import patch

repo, root, case = sys.argv[1], Path(sys.argv[2]), sys.argv[3]
source = Path(repo) / 'project-os/rootfs/usr/libexec/soda/project-factory-roles'
loader = importlib.machinery.SourceFileLoader('project_factory_roles', str(source))
spec = importlib.util.spec_from_loader(loader.name, loader)
roles = importlib.util.module_from_spec(spec)
loader.exec_module(roles)

PID = 'f0123456789abcdef01234567'
COMMIT = 'c' * 40

def canonical_digest(files):
    digest = hashlib.sha256()
    for name in sorted(files):
        digest.update(name.encode() + b'\x00' + files[name])
    return digest.hexdigest()

def approve_request(**overrides):
    raw = {'setup.sh': b'true\n', 'check.sh': b'true\n'}
    files = {name: base64.b64encode(contents).decode() for name, contents in raw.items()}
    request = {'op': 'approve', 'id': PID, 'role': 'soda-coder',
               'setup_digest': canonical_digest(raw), 'source_commit': COMMIT,
               'files': files, 'bundle': base64.b64encode(b'bundle').decode(), 'credential': ''}
    request.update(overrides)
    return request

factory = root / 'factory'
users, commands = {}, []
patch.object(roles, 'FACTORY', factory).start()
patch.object(roles, 'PREPARATIONS', factory / 'preparations').start()
patch.object(roles, 'CREDENTIALS', factory / 'credentials').start()
patch.object(roles, 'HOLD', factory / 'maintenance-hold').start()
patch.object(roles, 'LOCK', factory / 'lock').start()
patch.object(roles.os, 'geteuid', return_value=0).start()
patch.object(roles.os, 'chown', return_value=None).start()
native_fstat, native_lstat = os.fstat, Path.lstat

def root_owner(info):
    fields = {name: getattr(info, name) for name in dir(info) if name.startswith('st_')}
    fields.update(st_uid=0, st_gid=0)
    return types.SimpleNamespace(**fields)

def home_owner(path, info):
    text = str(path)
    for user in users.values():
        if text == user.pw_dir or text.startswith(user.pw_dir + '/'):
            info.st_uid, info.st_gid = user.pw_uid, user.pw_gid
    return info

patch.object(roles.os, 'fstat', side_effect=lambda fd: root_owner(native_fstat(fd))).start()
patch.object(Path, 'lstat', lambda path: home_owner(path, root_owner(native_lstat(path)))).start()

def lookup(name):
    if name not in users:
        raise KeyError(name)
    return users[name]

def groups():
    return [types.SimpleNamespace(gr_name=name, gr_mem=[name]) for name in users]

def primary(gid):
    for name, user in users.items():
        if user.pw_gid == gid:
            return types.SimpleNamespace(gr_name=name)
    raise KeyError(gid)

def run_command(args, **options):
    commands.append(args)
    if args[0] == 'useradd':
        home = root / args[-1]
        home.mkdir(mode=0o700, exist_ok=True)
        uid = 2000 + len(users)
        users[args[-1]] = types.SimpleNamespace(
            pw_name=args[-1], pw_dir=str(home), pw_uid=uid, pw_gid=uid, pw_shell=roles.NOLOGIN)
        return types.SimpleNamespace(stdout=b'')
    if args[0] == roles.GIT:
        return types.SimpleNamespace(stdout=b'')
    raise AssertionError('unexpected native command: %r' % (args,))

patch.object(roles.pwd, 'getpwnam', side_effect=lookup).start()
patch.object(roles.subprocess, 'run', side_effect=run_command).start()
patch.object(roles.grp, 'getgrall', side_effect=groups).start()
patch.object(roles.grp, 'getgrgid', side_effect=primary).start()

def approve(**overrides):
    return roles.do_approve(approve_request(**overrides))

def record(missing='', refusal=''):
    verified = {'uid': str(os.getuid()), 'login': 'soda-coder', 'groups': 'soda-coder'}
    if refusal:
        verified['refusal'] = refusal
    return roles.do_record({'op': 'record', 'id': PID, 'tools': [], 'missing': missing, 'verified': verified})

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
if case == 'ensure':
    obs['result'] = roles.do_ensure({'op': 'ensure'})
    obs['names'] = [command[0] for command in commands]
    obs['cmd0'] = commands[0]
    obs['count_before'] = len(commands)
    roles.do_ensure({'op': 'ensure'})
    obs['count_after'] = len(commands)
elif case == 'ensure_refuses':
    home = root / 'soda-coder'
    home.mkdir(mode=0o700)
    users['soda-coder'] = types.SimpleNamespace(
        pw_name='soda-coder', pw_dir=str(home), pw_uid=0, pw_gid=0, pw_shell='/bin/bash')
    obs['attempt'] = attempt(roles.do_ensure, {'op': 'ensure'})
elif case == 'approve_snapshot':
    result = approve()
    snapshot = factory / 'preparations' / PID / 'snapshot'
    obs['approved'] = result['approved']
    obs['setup'] = b64(snapshot / 'setup.sh')
    obs['setup_mode'] = mode(snapshot / 'setup.sh')
    obs['git_clone'] = any(command[:2] == [roles.GIT, 'clone'] for command in commands)
    obs['verify_tmp'] = (snapshot / 'verify-tmp').exists()
    obs['repeated'] = bool(approve()['repeated'])
    obs['bad_digest'] = attempt(approve, setup_digest='e' * 64)
elif case == 'approve_rejects':
    bad = [dict(role='root'), dict(id='../escape'), dict(setup_digest='zz'),
           dict(source_commit='short'), dict(credential='../x'),
           dict(files={'setup.sh': base64.b64encode(b'x').decode()}),
           dict(bundle='!!!'), dict(extra=1)]
    obs['attempts'] = [attempt(approve, **values) for values in bad]
    obs['commands'] = commands
elif case == 'record_phases':
    approve()
    obs['waiting'] = bool(record(missing='python3')['waiting'])
    state = roles.do_inspect({'op': 'inspect', 'id': PID})
    obs['state'] = [state['phase'], state['missing'], state['ready']]
    obs['changed_missing'] = attempt(record, missing='other-tool')
elif case == 'launcher_refusal':
    approve()
    record(refusal='role holds unexpected groups')
    obs['phase'] = roles.do_inspect({'op': 'inspect', 'id': PID})['phase']
    obs['start'] = attempt(roles.do_start, {'op': 'start', 'id': PID})
elif case == 'start_spawns':
    approve()
    record()
    started_path = factory / 'preparations' / PID / 'started.json'
    def fake_fork():
        started_path.write_text(json.dumps({'pid': 999, 'pgid': 999}))
        return 999
    with patch.object(roles.os, 'fork', side_effect=fake_fork):
        with patch.object(roles.os, 'waitpid', return_value=(999, 0)):
            with patch.object(roles, 'group_alive', return_value=True):
                obs['result'] = roles.do_start({'op': 'start', 'id': PID})
    with patch.object(roles, 'group_alive', return_value=True):
        obs['phase'] = roles.do_inspect({'op': 'inspect', 'id': PID})['phase']
elif case == 'dead_supervisor':
    approve()
    record()
    (factory / 'preparations' / PID / 'started.json').write_text(json.dumps({'pid': 999, 'pgid': 999}))
    with patch.object(roles, 'group_alive', return_value=False):
        obs['phase'] = roles.do_inspect({'op': 'inspect', 'id': PID})['phase']
        obs['start'] = attempt(roles.do_start, {'op': 'start', 'id': PID})
elif case == 'stop_bars':
    stopped = roles.do_stop({'op': 'stop', 'id': PID})
    obs['unknown'] = [stopped['known'], stopped['retirement']]
    obs['approve_blocked'] = attempt(approve)
    other = approve_request(id='f123456789abcdef012345678')
    roles.do_approve(other)
    roles.do_record({'op': 'record', 'id': other['id'], 'tools': [], 'missing': '',
                     'verified': {'uid': '1', 'login': 'soda-coder', 'groups': 'soda-coder'}})
    with patch.object(roles, 'group_alive', return_value=False):
        stopped = roles.do_stop({'op': 'stop', 'id': other['id']})
    obs['known'] = [stopped['known'], stopped['retirement']]
    state = roles.do_inspect({'op': 'inspect', 'id': other['id']})
    obs['state'] = [state['phase'], state['stopped']]
elif case == 'hold_release':
    approve()
    obs['held'] = bool(roles.do_hold({'op': 'hold', 'revision': 1})['hold']['active'])
    obs['approve_blocked'] = attempt(approve, id='f123456789abcdef012345678')
    obs['release_wrong_rev'] = attempt(roles.do_release, {'op': 'release', 'revision': 2})
    directory = factory / 'preparations' / PID
    (directory / 'started.json').write_text(json.dumps({'pid': 999, 'pgid': 999}))
    with patch.object(roles, 'group_alive', return_value=True):
        obs['release_running'] = attempt(roles.do_release, {'op': 'release', 'revision': 1})
    roles.do_stop({'op': 'stop', 'id': PID})
    obs['released'] = bool(roles.do_release({'op': 'release', 'revision': 1})['hold']['active'])
elif case == 'proc_group':
    proot = root / 'proc'
    (proot / '46').mkdir(parents=True)
    (proot / '46' / 'stat').write_text('46 (python3) S 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
    (proot / '47').mkdir()
    (proot / '47' / 'stat').write_text('47 (my) proc) S 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
    account = types.SimpleNamespace(pw_uid=2000, pw_gid=2000)
    obs['alive46'] = bool(roles.group_alive(46, proot))
    obs['alive45'] = bool(roles.group_alive(45, proot))
    obs['owned'] = bool(roles.leader_owned_by({'pid': 46, 'pgid': 46}, account, proot))
    obs['wrong_pgid'] = bool(roles.leader_owned_by({'pid': 46, 'pgid': 45}, account, proot))
    obs['missing_pid'] = bool(roles.leader_owned_by({'pid': 999, 'pgid': 46}, account, proot))
    (proot / '46' / 'stat').write_text('46 (python3) Z 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
    (proot / '47' / 'stat').write_text('47 (sleep) Z 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
    obs['zombie_alive'] = bool(roles.group_alive(46, proot))
    obs['zombie_owned'] = bool(roles.leader_owned_by({'pid': 46, 'pgid': 46}, account, proot))
elif case == 'run_as_role':
    patch.object(roles.os, 'geteuid', return_value=os.getuid()).start()
    account = types.SimpleNamespace(pw_uid=os.getuid(), pw_gid=os.getgid())
    work = root / 'work'
    work.mkdir()
    log = str(root / 'test.log')
    obs['code'] = roles.run_as_role(
        ['/bin/sh', '-c', 'echo out; exit 3'], {'PATH': '/usr/bin:/bin'}, str(work), log, account)
    obs['log'] = b64(Path(log))
    with patch.object(roles, 'LOG_CAP', 4):
        roles.run_as_role(['/bin/sh', '-c', 'echo overflow'],
                          {'PATH': '/usr/bin:/bin'}, str(work), log, account)
    obs['truncated'] = b64(Path(log))
else:
    raise SystemExit('unknown case: ' + case)
print(json.dumps(obs))
`

// runRolesCase runs one driver case in a fresh temp root and returns its
// decoded JSON observations.
func runRolesCase(t *testing.T, caseName string) map[string]any {
	t.Helper()
	result := Run(t, RunOpt{}, Python3(t), "-c", rolesDriver, RepoRoot, TempDir(t), caseName)
	Require(t, result.Code == 0, "roles driver %s failed: %s", caseName, result.Stderr)
	var obs map[string]any
	Require(t, json.Unmarshal([]byte(result.Stdout), &obs) == nil, "parse %s output %q", caseName, result.Stdout)
	return obs
}

// rolesAttemptMRO extracts the exception MRO from an attempt observation
// (nil when the driver call succeeded).
func rolesAttemptMRO(t *testing.T, value any) []string {
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

func rolesText(t *testing.T, value any) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(value.(string))
	Require(t, err == nil, "decode bytes: %v", err)
	return string(raw)
}

func rolesStrings(t *testing.T, value any) []string {
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

func TestRolesEnsureProvisionsLockedRolesWithoutExtraGroups(t *testing.T) {
	obs := runRolesCase(t, "ensure")
	result, ok := obs["result"].(map[string]any)
	Require(t, ok, "result is %T", obs["result"])
	Check(t, len(rolesStrings(t, result["roles"])) == 2, "roles = %v", result["roles"])
	names := rolesStrings(t, obs["names"])
	Require(t, len(names) == 2, "commands = %v", names)
	Check(t, names[0] == "useradd" && names[1] == "useradd", "commands = %v", names)
	cmd := rolesStrings(t, obs["cmd0"])
	found := false
	for i, word := range cmd {
		if word == "--password" && i+1 < len(cmd) {
			Check(t, cmd[i+1] == "!", "password = %q", cmd[i+1])
		}
		found = found || word == "--shell"
	}
	Check(t, found, "--shell missing in %v", cmd)
	Check(t, obs["count_before"] == obs["count_after"], "rerun issued commands")
}

func TestRolesEnsureRefusesInteractiveOrGroupedAccounts(t *testing.T) {
	obs := runRolesCase(t, "ensure_refuses")
	Check(t, raisedAs(rolesAttemptMRO(t, obs["attempt"]), "ValueError"), "mro = %v", obs["attempt"])
}

func TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle(t *testing.T) {
	obs := runRolesCase(t, "approve_snapshot")
	Check(t, obs["approved"] == "f0123456789abcdef01234567", "approved = %v", obs["approved"])
	Check(t, rolesText(t, obs["setup"]) == "true\n", "setup = %q", rolesText(t, obs["setup"]))
	Check(t, obs["setup_mode"] == float64(0o644), "setup mode = %v", obs["setup_mode"])
	Check(t, obs["git_clone"] == true, "no git clone")
	Check(t, obs["verify_tmp"] == false, "verify-tmp remains")
	Check(t, obs["repeated"] == true, "repeat not reported")
	Check(t, raisedAs(rolesAttemptMRO(t, obs["bad_digest"]), "ValueError"), "bad digest accepted")
}

func TestRolesApproveRejectsUntrustedInputsBeforeEffects(t *testing.T) {
	obs := runRolesCase(t, "approve_rejects")
	attempts, ok := obs["attempts"].([]any)
	Require(t, ok && len(attempts) == 8, "attempts = %v", obs["attempts"])
	for i, attempt := range attempts {
		Check(t, raisedAs(rolesAttemptMRO(t, attempt), "ValueError"), "input %d accepted", i)
	}
	commands, ok := obs["commands"].([]any)
	Require(t, ok, "commands is %T", obs["commands"])
	Check(t, len(commands) == 0, "effects before refusal: %v", commands)
}

func TestRolesRecordReportsWaitingAndFailedPhases(t *testing.T) {
	obs := runRolesCase(t, "record_phases")
	Check(t, obs["waiting"] == true, "not waiting")
	state, ok := obs["state"].([]any)
	Require(t, ok && len(state) == 3, "state = %v", obs["state"])
	Check(t, state[0] == "waiting" && state[1] == "python3" && state[2] == false, "state = %v", state)
	Check(t, raisedAs(rolesAttemptMRO(t, obs["changed_missing"]), "ValueError"), "missing change accepted")
}

func TestRolesLauncherRefusalFailsWithoutStart(t *testing.T) {
	obs := runRolesCase(t, "launcher_refusal")
	Check(t, obs["phase"] == "failed", "phase = %v", obs["phase"])
	Check(t, raisedAs(rolesAttemptMRO(t, obs["start"]), "ValueError"), "start after refusal accepted")
}

func TestRolesStartSpawnsSupervisorAndReportsRunning(t *testing.T) {
	obs := runRolesCase(t, "start_spawns")
	result, ok := obs["result"].(map[string]any)
	Require(t, ok, "result is %T", obs["result"])
	Check(t, result["pgid"] == float64(999), "pgid = %v", result["pgid"])
	Check(t, obs["phase"] == "running", "phase = %v", obs["phase"])
}

func TestRolesDeadSupervisorReportsInterrupted(t *testing.T) {
	obs := runRolesCase(t, "dead_supervisor")
	Check(t, obs["phase"] == "interrupted", "phase = %v", obs["phase"])
	Check(t, raisedAs(rolesAttemptMRO(t, obs["start"]), "ValueError"), "start after interrupt accepted")
}

func TestRolesStopBarsUnknownIdentityAndRetiresKnown(t *testing.T) {
	obs := runRolesCase(t, "stop_bars")
	unknown, ok := obs["unknown"].([]any)
	Require(t, ok && len(unknown) == 2, "unknown = %v", obs["unknown"])
	Check(t, unknown[0] == false && unknown[1] == "confirmed", "unknown = %v", unknown)
	Check(t, raisedAs(rolesAttemptMRO(t, obs["approve_blocked"]), "ValueError"), "approve after stop accepted")
	known, ok := obs["known"].([]any)
	Require(t, ok && len(known) == 2, "known = %v", obs["known"])
	Check(t, known[0] == true && known[1] == "confirmed", "known = %v", known)
	state, ok := obs["state"].([]any)
	Require(t, ok && len(state) == 2, "state = %v", obs["state"])
	Check(t, state[0] == "stopped" && state[1] == true, "state = %v", state)
}

func TestRolesHoldDeniesApproveAndReleaseNeedsRevisionAndQuiescence(t *testing.T) {
	obs := runRolesCase(t, "hold_release")
	Check(t, obs["held"] == true, "hold not active")
	Check(t, raisedAs(rolesAttemptMRO(t, obs["approve_blocked"]), "ValueError"), "approve under hold accepted")
	Check(t, raisedAs(rolesAttemptMRO(t, obs["release_wrong_rev"]), "ValueError"), "wrong revision released")
	Check(t, raisedAs(rolesAttemptMRO(t, obs["release_running"]), "ValueError"), "running preparation released")
	Check(t, obs["released"] == false, "hold still active")
}

func TestRolesProcGroupReadsPgrpNotSession(t *testing.T) {
	obs := runRolesCase(t, "proc_group")
	Check(t, obs["alive46"] == true, "group 46 not alive")
	Check(t, obs["alive45"] == false, "session 45 reported alive")
	Check(t, obs["owned"] == true, "leader not owned")
	Check(t, obs["wrong_pgid"] == false, "wrong pgid owned")
	Check(t, obs["missing_pid"] == false, "missing pid owned")
	Check(t, obs["zombie_alive"] == false, "zombie group alive")
	Check(t, obs["zombie_owned"] == false, "zombie leader owned")
}

func TestRolesRunAsRoleCapturesBoundedOutputAndExit(t *testing.T) {
	obs := runRolesCase(t, "run_as_role")
	Check(t, obs["code"] == float64(3), "code = %v", obs["code"])
	Check(t, rolesText(t, obs["log"]) == "out\n", "log = %q", rolesText(t, obs["log"]))
	truncated := rolesText(t, obs["truncated"])
	Check(t, len(truncated) >= len("[output truncated]\n") &&
		truncated[len(truncated)-len("[output truncated]\n"):] == "[output truncated]\n",
		"log not truncated: %q", truncated)
}
