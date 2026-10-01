"""Factory helper state machine with mocked root, accounts and git.

Only fresh test-owned directories are used; no Linux accounts, processes or
native git state are created. The detached supervisor body runs only in the
native M2 proof; here spawn is simulated at the fork boundary.
"""

import base64
import hashlib
import importlib.machinery
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2] / 'project-os/rootfs/usr/libexec/soda/project-factory-roles'
loader = importlib.machinery.SourceFileLoader('project_factory_roles', str(SOURCE))
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
    request = {
        'op': 'approve',
        'id': PID,
        'role': 'soda-coder',
        'setup_digest': canonical_digest(raw),
        'source_commit': COMMIT,
        'files': files,
        'bundle': base64.b64encode(b'bundle').decode(),
        'credential': '',
    }
    request.update(overrides)
    return request


class FactoryRoles(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.factory = self.root / 'factory'
        self.users, self.commands = {}, []
        self.addCleanup(patch.stopall)
        patch.object(roles, 'FACTORY', self.factory).start()
        patch.object(roles, 'PREPARATIONS', self.factory / 'preparations').start()
        patch.object(roles, 'CREDENTIALS', self.factory / 'credentials').start()
        patch.object(roles, 'HOLD', self.factory / 'maintenance-hold').start()
        patch.object(roles, 'LOCK', self.factory / 'lock').start()
        patch.object(roles.os, 'geteuid', return_value=0).start()
        patch.object(roles.os, 'chown', return_value=None).start()
        patch.object(roles.pwd, 'getpwnam', side_effect=self.lookup).start()
        patch.object(roles.subprocess, 'run', side_effect=self.run_command).start()
        patch.object(roles.grp, 'getgrall', side_effect=self.groups).start()
        patch.object(roles.grp, 'getgrgid', side_effect=self.primary).start()
        native_fstat, native_lstat = os.fstat, Path.lstat

        def root_owner(info):
            fields = {name: getattr(info, name) for name in dir(info) if name.startswith('st_')}
            fields.update(st_uid=0, st_gid=0)
            return types.SimpleNamespace(**fields)

        def home_owner(path, info):
            text = str(path)
            for user in self.users.values():
                if text == user.pw_dir or text.startswith(user.pw_dir + '/'):
                    info.st_uid, info.st_gid = user.pw_uid, user.pw_gid
            return info

        patch.object(roles.os, 'fstat', side_effect=lambda fd: root_owner(native_fstat(fd))).start()
        patch.object(Path, 'lstat', lambda path: home_owner(path, root_owner(native_lstat(path)))).start()

    def lookup(self, name):
        if name not in self.users:
            raise KeyError(name)
        return self.users[name]

    def groups(self):
        return [types.SimpleNamespace(gr_name=name, gr_mem=[name]) for name in self.users]

    def primary(self, gid):
        for name, user in self.users.items():
            if user.pw_gid == gid:
                return types.SimpleNamespace(gr_name=name)
        raise KeyError(gid)

    def run_command(self, args, **options):
        self.commands.append(args)
        if args[0] == 'useradd':
            home = self.root / args[-1]
            home.mkdir(mode=0o700, exist_ok=True)
            uid = 2000 + len(self.users)
            self.users[args[-1]] = types.SimpleNamespace(
                pw_name=args[-1], pw_dir=str(home), pw_uid=uid, pw_gid=uid, pw_shell=roles.NOLOGIN
            )
            return types.SimpleNamespace(stdout=b'')
        if args[0] == roles.GIT:
            return types.SimpleNamespace(stdout=b'')
        self.fail('unexpected native command')

    def approve(self, **overrides):
        return roles.do_approve(approve_request(**overrides))

    def record(self, missing='', refusal=''):
        verified = {'uid': str(os.getuid()), 'login': 'soda-coder', 'groups': 'soda-coder'}
        if refusal:
            verified['refusal'] = refusal
        return roles.do_record({'op': 'record', 'id': PID, 'tools': [], 'missing': missing, 'verified': verified})

    def test_ensure_provisions_locked_roles_without_extra_groups(self):
        self.assertEqual(roles.do_ensure({'op': 'ensure'}), {'roles': ['soda-coder', 'soda-reviewer']})
        self.assertEqual([command[0] for command in self.commands], ['useradd', 'useradd'])
        self.assertIn('--shell', self.commands[0])
        self.assertEqual(self.commands[0][self.commands[0].index('--password') + 1], '!')
        roles.do_ensure({'op': 'ensure'})
        self.assertEqual(len(self.commands), 2)

    def test_ensure_refuses_interactive_or_grouped_accounts(self):
        home = self.root / 'soda-coder'
        home.mkdir(mode=0o700)
        self.users['soda-coder'] = types.SimpleNamespace(
            pw_name='soda-coder', pw_dir=str(home), pw_uid=0, pw_gid=0, pw_shell='/bin/bash'
        )
        with self.assertRaises(ValueError):
            roles.do_ensure({'op': 'ensure'})

    def test_approve_writes_protected_snapshot_and_verifies_bundle(self):
        result = self.approve()
        snapshot = self.factory / 'preparations' / PID / 'snapshot'
        self.assertEqual(result['approved'], PID)
        self.assertEqual((snapshot / 'setup.sh').read_bytes(), b'true\n')
        self.assertEqual((snapshot / 'setup.sh').stat().st_mode & 0o777, 0o644)
        self.assertTrue(any(command[:2] == [roles.GIT, 'clone'] for command in self.commands))
        self.assertFalse((snapshot / 'verify-tmp').exists())
        repeated = self.approve()
        self.assertTrue(repeated['repeated'])
        with self.assertRaises(ValueError):
            self.approve(setup_digest='e' * 64)

    def test_approve_rejects_untrusted_inputs_before_effects(self):
        bad = [
            dict(role='root'),
            dict(id='../escape'),
            dict(setup_digest='zz'),
            dict(source_commit='short'),
            dict(credential='../x'),
            dict(files={'setup.sh': base64.b64encode(b'x').decode()}),
            dict(bundle='!!!'),
            dict(extra=1),
        ]
        for values in bad:
            with self.assertRaises(ValueError):
                self.approve(**values)
        self.assertEqual(self.commands, [])

    def test_record_reports_waiting_and_failed_phases(self):
        self.approve()
        self.assertTrue(self.record(missing='python3')['waiting'])
        state = roles.do_inspect({'op': 'inspect', 'id': PID})
        self.assertEqual((state['phase'], state['missing'], state['ready']), ('waiting', 'python3', False))
        with self.assertRaises(ValueError):
            self.record(missing='other-tool')

    def test_launcher_refusal_fails_without_start(self):
        self.approve()
        self.record(refusal='role holds unexpected groups')
        state = roles.do_inspect({'op': 'inspect', 'id': PID})
        self.assertEqual(state['phase'], 'failed')
        with self.assertRaises(ValueError):
            roles.do_start({'op': 'start', 'id': PID})

    def test_start_spawns_supervisor_and_reports_running(self):
        self.approve()
        self.record()
        started_path = self.factory / 'preparations' / PID / 'started.json'

        def fake_fork():
            started_path.write_text(json.dumps({'pid': 999, 'pgid': 999}))
            return 999

        with patch.object(roles.os, 'fork', side_effect=fake_fork):
            with patch.object(roles.os, 'waitpid', return_value=(999, 0)):
                with patch.object(roles, 'group_alive', return_value=True):
                    result = roles.do_start({'op': 'start', 'id': PID})
        self.assertEqual(result['pgid'], 999)
        with patch.object(roles, 'group_alive', return_value=True):
            state = roles.do_inspect({'op': 'inspect', 'id': PID})
        self.assertEqual(state['phase'], 'running')

    def test_dead_supervisor_reports_interrupted(self):
        self.approve()
        self.record()
        directory = self.factory / 'preparations' / PID
        (directory / 'started.json').write_text(json.dumps({'pid': 999, 'pgid': 999}))
        with patch.object(roles, 'group_alive', return_value=False):
            state = roles.do_inspect({'op': 'inspect', 'id': PID})
            self.assertEqual(state['phase'], 'interrupted')
            with self.assertRaises(ValueError):
                roles.do_start({'op': 'start', 'id': PID})

    def test_stop_bars_unknown_identity_and_retires_known(self):
        stopped = roles.do_stop({'op': 'stop', 'id': PID})
        self.assertEqual((stopped['known'], stopped['retirement']), (False, 'confirmed'))
        with self.assertRaises(ValueError):
            self.approve()
        other = approve_request(id='f123456789abcdef012345678')
        roles.do_approve(other)
        roles.do_record(
            {
                'op': 'record',
                'id': other['id'],
                'tools': [],
                'missing': '',
                'verified': {'uid': '1', 'login': 'soda-coder', 'groups': 'soda-coder'},
            }
        )
        with patch.object(roles, 'group_alive', return_value=False):
            stopped = roles.do_stop({'op': 'stop', 'id': other['id']})
        self.assertEqual((stopped['known'], stopped['retirement']), (True, 'confirmed'))
        state = roles.do_inspect({'op': 'inspect', 'id': other['id']})
        self.assertEqual((state['phase'], state['stopped']), ('stopped', True))

    def test_hold_denies_approve_and_release_needs_revision_and_quiescence(self):
        self.approve()
        self.assertTrue(roles.do_hold({'op': 'hold', 'revision': 1})['hold']['active'])
        with self.assertRaises(ValueError):
            roles.do_approve(approve_request(id='f123456789abcdef012345678'))
        with self.assertRaises(ValueError):
            roles.do_release({'op': 'release', 'revision': 2})
        directory = self.factory / 'preparations' / PID
        (directory / 'started.json').write_text(json.dumps({'pid': 999, 'pgid': 999}))
        with patch.object(roles, 'group_alive', return_value=True):
            with self.assertRaises(ValueError):
                roles.do_release({'op': 'release', 'revision': 1})
        roles.do_stop({'op': 'stop', 'id': PID})
        self.assertFalse(roles.do_release({'op': 'release', 'revision': 1})['hold']['active'])

    def test_proc_group_reads_pgrp_not_session(self):
        proot = self.root / 'proc'
        (proot / '46').mkdir(parents=True)
        (proot / '46' / 'stat').write_text('46 (python3) S 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
        (proot / '47').mkdir()
        (proot / '47' / 'stat').write_text('47 (my) proc) S 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
        self.assertTrue(roles.group_alive(46, proot))
        self.assertFalse(roles.group_alive(45, proot))
        account = types.SimpleNamespace(pw_uid=2000, pw_gid=2000)
        self.assertTrue(roles.leader_owned_by({'pid': 46, 'pgid': 46}, account, proot))
        self.assertFalse(roles.leader_owned_by({'pid': 46, 'pgid': 45}, account, proot))
        self.assertFalse(roles.leader_owned_by({'pid': 999, 'pgid': 46}, account, proot))
        (proot / '46' / 'stat').write_text('46 (python3) Z 1 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
        (proot / '47' / 'stat').write_text('47 (sleep) Z 46 46 45 0 -1 0 0 0 0 0 0 0 0 0 0 1 0 1 0 0 0 0')
        self.assertFalse(roles.group_alive(46, proot))
        self.assertFalse(roles.leader_owned_by({'pid': 46, 'pgid': 46}, account, proot))

    def test_run_as_role_captures_bounded_output_and_exit(self):
        # This test forks for real as the invoking user, so restore the real
        # euid: the setgroups drop applies to the privileged supervisor only.
        patch.object(roles.os, 'geteuid', return_value=os.getuid()).start()
        account = types.SimpleNamespace(pw_uid=os.getuid(), pw_gid=os.getgid())
        work = self.root / 'work'
        work.mkdir()
        log = str(self.root / 'test.log')
        code = roles.run_as_role(
            ['/bin/sh', '-c', 'echo out; exit 3'], {'PATH': '/usr/bin:/bin'}, str(work), log, account
        )
        self.assertEqual(code, 3)
        self.assertEqual(Path(log).read_bytes(), b'out\n')
        with patch.object(roles, 'LOG_CAP', 4):
            roles.run_as_role(['/bin/sh', '-c', 'echo overflow'], {'PATH': '/usr/bin:/bin'}, str(work), log, account)
        self.assertTrue(Path(log).read_bytes().endswith(b'[output truncated]\n'))


if __name__ == '__main__':
    unittest.main()
