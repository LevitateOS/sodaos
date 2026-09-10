"""Actual account script/file effects with mocked root, pwd and account commands.

Only fresh test-owned directories are used; never creates Linux host accounts.
"""
import importlib.machinery
import importlib.util
import os
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2] / 'project-os/rootfs/usr/libexec/soda/project-account'
loader = importlib.machinery.SourceFileLoader('project_account', str(SOURCE))
spec = importlib.util.spec_from_loader(loader.name, loader)
account = importlib.util.module_from_spec(spec)
loader.exec_module(account)


class ProjectAccount(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.markers, self.keys = self.root / 'accounts', self.root / 'keys'
        self.markers.mkdir(mode=0o700)
        self.keys.mkdir(mode=0o755)
        self.commands, self.users = [], {}
        self.addCleanup(patch.stopall)
        patch.object(account, 'ACCOUNTS', self.markers).start()
        patch.object(account, 'KEYS', self.keys).start()
        patch.object(account.os, 'geteuid', return_value=0).start()
        patch.object(account.pwd, 'getpwnam', side_effect=self.lookup).start()
        patch.object(account.subprocess, 'run', side_effect=self.run_command).start()
        native_fstat, native_lstat = os.fstat, Path.lstat
        def root_owner(info):
            values = list(info)
            values[4] = values[5] = 0
            return os.stat_result(values)
        patch.object(account.os, 'fstat', side_effect=lambda fd: root_owner(native_fstat(fd))).start()
        patch.object(Path, 'lstat', lambda path: root_owner(native_lstat(path))).start()

    def lookup(self, name):
        if name not in self.users:
            raise KeyError(name)
        return self.users[name]

    def run_command(self, args, **options):
        self.commands.append(args)
        self.assertEqual(options, {'check': True})
        if args[0] == 'useradd':
            home = self.root / args[-1]
            home.mkdir(mode=0o700)
            self.users[args[-1]] = types.SimpleNamespace(pw_dir=str(home))
        elif args[0] != 'usermod':
            self.fail('unexpected native command')

    def request(self, keys=None, admin=False):
        return {'login': 'alice', 'identity': 1, 'admin': admin, 'keys': [] if keys is None else keys}

    def test_account_only_provisions_locked_home_marker_shared_and_empty_keyfile(self):
        self.assertEqual(account.provision(self.request()), {'login': 'alice', 'identity': 1})
        self.assertEqual((self.keys / 'alice').read_bytes(), b'')
        self.assertEqual((self.markers / 'alice').read_bytes(), b'1')
        self.assertEqual((self.markers / 'alice').stat().st_mode & 0o777, 0o600)
        self.assertEqual((self.keys / 'alice').stat().st_mode & 0o777, 0o644)
        self.assertEqual(os.readlink(self.root / 'alice/shared'), '/srv/project/shared')
        self.assertIn('--create-home', self.commands[0])
        self.assertEqual(self.commands[0][self.commands[0].index('--password') + 1], '!')
        self.assertEqual(self.commands[0][-2:], ['soda-project', 'alice'])
        self.assertEqual(len(self.commands), 1)

    def test_selected_keys_and_creation_owner_privilege_remain_real(self):
        request = self.request(['ssh-ed25519 YWJj\n'], True)
        account.provision(request)
        self.assertEqual((self.keys / 'alice').read_bytes(), b'ssh-ed25519 YWJj\n')
        self.assertEqual(self.commands[-1], ['usermod', '--append', '--groups', 'wheel', 'alice'])
        (self.root / 'alice/work').write_text('later work')
        before = (self.keys / 'alice').stat().st_ino
        account.provision(request)
        self.assertEqual((self.keys / 'alice').stat().st_ino, before)
        self.assertEqual((self.root / 'alice/work').read_text(), 'later work')
        self.assertEqual(sum(command[0] == 'useradd' for command in self.commands), 1)

    def test_account_only_retry_never_erases_existing_keys(self):
        account.provision(self.request(['ssh-ed25519 YWJj']))
        with self.assertRaises(ValueError):
            account.provision(self.request())
        self.assertEqual((self.keys / 'alice').read_bytes(), b'ssh-ed25519 YWJj\n')
        self.assertEqual(len(self.commands), 1)

    def test_join_is_not_key_apply_and_preserves_drift(self):
        account.provision(self.request())
        with self.assertRaises(ValueError):
            account.provision(self.request(['ssh-ed25519 YWJj']))
        self.assertEqual((self.keys / 'alice').read_bytes(), b'')
        (self.markers / 'alice').write_text('2')
        with self.assertRaises(ValueError):
            account.provision(self.request())
        self.assertEqual((self.markers / 'alice').read_bytes(), b'2')

    def test_occupied_inputs_and_unassociated_users_refuse_before_account_commands(self):
        (self.keys / 'alice').symlink_to(self.root / 'absent')
        with self.assertRaises(ValueError):
            account.provision(self.request())
        self.assertEqual(self.commands, [])
        (self.keys / 'alice').unlink()
        self.users['alice'] = types.SimpleNamespace(pw_dir=str(self.root / 'alice'))
        with self.assertRaises(FileNotFoundError):
            account.provision(self.request())
        self.assertEqual(self.commands, [])

    def test_key_symlink_is_not_followed(self):
        account.provision(self.request())
        other = self.root / 'other'
        other.write_bytes(b'preserve')
        (self.keys / 'alice').unlink()
        (self.keys / 'alice').symlink_to(other)
        with self.assertRaises(OSError):
            account.provision(self.request())
        self.assertEqual(other.read_bytes(), b'preserve')

    def test_validation_before_native_effects(self):
        for values in [None, 'key', ['ssh-ed25519 YWJj'] * 33, ['PRIVATE KEY'], ['ssh-ed25519 YWJj\nssh-ed25519 ZGVm']]:
            request = self.request()
            request['keys'] = values
            with self.assertRaises(ValueError):
                account.provision(request)
        request = self.request()
        request['login'] = 'root'
        with self.assertRaises(ValueError):
            account.provision(request)
        self.assertEqual(self.commands, [])

    def test_native_password_ssh_policy_is_not_relaxed(self):
        rootfs = SOURCE.parents[3]
        config = '\n'.join(path.read_text() for path in (rootfs / 'etc/ssh').rglob('*.conf'))
        self.assertIn('PasswordAuthentication no', config)
        self.assertIn('PermitRootLogin no', config)
        self.assertIn('KbdInteractiveAuthentication no', config)


if __name__ == '__main__':
    unittest.main()
