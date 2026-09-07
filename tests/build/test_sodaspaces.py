"""Production staging/preflight in temporary filesystems; never host installation."""
import ast
import json
import os
from pathlib import Path
import runpy
import shutil
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
FILES = ('templates/custom/header.tmpl', 'templates/custom/footer.tmpl',
         'public/assets/sodaspaces.css', 'public/assets/sodaspaces.js')
PREFIX = 'rootfs/var/lib/soda/forgejo/gitea/'


class SodaspacesPackaging(unittest.TestCase):
    def test_browser_probe_refuses_bad_inputs_without_secret_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / 'input.json'
            source.write_text('invalid JSON SYNTHETIC_AUTH_SECRET_MARKER')
            source.chmod(0o600)
            for permission in ([], ['--allow-auth-transitions']):
                result = subprocess.run(['node', str(ROOT / 'tests/installed/sodaspaces.mjs'),
                                         str(source), str(root / 'not-created'), *permission],
                                        capture_output=True, text=True, timeout=15)
                self.assertEqual(result.returncode, 1)
                self.assertIn('private input validation', result.stdout)
                self.assertNotIn('SYNTHETIC_AUTH_SECRET_MARKER', result.stdout + result.stderr)
                self.assertFalse((root / 'not-created').exists())

    def test_browser_probe_does_not_finalize_an_occupied_run(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            root.chmod(0o700)
            run = root / 'sodaspaces-run'
            run.mkdir(mode=0o700)
            (run / 'retained-marker').write_text('retained')
            private = root / 'synthetic-input'
            private.write_text('synthetic fixture, not a real credential or CA')
            private.chmod(0o600)
            config = {'origin': 'https://example.invalid', 'target': 'fixture',
                      'revision': '1' * 40, 'repository_path': '/alice/repo',
                      'repository_id': '42', 'oauth_client_id': 'synthetic-client',
                      'ca_file': str(private), 'users': [
                          {'id': '1', 'login': 'alice', 'password_file': str(private)},
                          {'id': '2', 'login': 'bob', 'password_file': str(private)}]}
            request = root / 'request.json'
            request.write_text(json.dumps(config))
            request.chmod(0o600)
            # Source/transport doubles only. Never query a provider or start a browser.
            bin_dir = root / 'bin'
            bin_dir.mkdir()
            git = bin_dir / 'git'
            git.write_text('#!/bin/sh\ncase "$1" in rev-parse) echo ' + '1' * 40 + ';; status) :;; *) exit 1;; esac\n')
            git.chmod(0o755)
            guard = root / 'no-network.cjs'
            guard.write_text("require('node:https').request = () => { require('node:fs').writeFileSync(" + json.dumps(str(root / 'unexpected-network')) + ", 'refused'); throw Error('test transport refused'); };\n")
            result = subprocess.run(['node', '--require', str(guard), str(ROOT / 'tests/installed/sodaspaces.mjs'),
                                     str(request), str(root), '--allow-auth-transitions'],
                                    env={**os.environ, 'PATH': str(bin_dir) + os.pathsep + os.environ['PATH'],
                                         'SODA_NATIVE_VALIDATE': 'fixture'},
                                    capture_output=True, text=True, timeout=15)
            self.assertEqual(result.returncode, 1)
            self.assertEqual(sorted(p.name for p in run.iterdir()), ['retained-marker'])
            self.assertEqual((run / 'retained-marker').read_text(), 'retained')
            self.assertFalse((root / 'unexpected-network').exists())

    def test_actual_stage_recipe_with_synthetic_build_inputs(self):
        # No generated artifact is placed in the production .artifacts/native tree.
        with tempfile.TemporaryDirectory() as tmp:
            checkout = Path(tmp)
            (checkout / 'scripts').mkdir()
            shutil.copyfile(ROOT / 'scripts/stage.py', checkout / 'scripts/stage.py')
            (checkout / 'assets').symlink_to(ROOT / 'assets', target_is_directory=True)
            (checkout / 'appliance').symlink_to(ROOT / 'appliance', target_is_directory=True)
            (checkout / 'cmd/soda-dashboard').mkdir(parents=True)
            build = checkout / '.artifacts/native/x86_64'
            (build / 'bin').mkdir(parents=True)
            (build / 'bin/soda-dashboard').write_text('synthetic; never executed')
            (build / 'github-actions-runner').mkdir()
            for page in ('tailscale', 'runners'):
                (checkout / 'cockpit/dist' / f'soda-{page}').mkdir(parents=True)
            previous = os.umask(0o077)
            try:
                with patch('sys.argv', ['stage.py', '--arch', 'x86_64']), patch('platform.system', return_value='Linux'), patch('platform.machine', return_value='x86_64'):
                    runpy.run_path(str(checkout / 'scripts/stage.py'), run_name='__main__')
            finally:
                os.umask(previous)
            stage = build / 'rootfs'
            for name in FILES:
                p = stage / PREFIX.removeprefix('rootfs/') / name
                self.assertEqual(p.read_bytes(), (ROOT / 'appliance/forgejo' / name).read_bytes())
                self.assertEqual(stat.S_IMODE(p.stat().st_mode), 0o644)
                for parent in p.parents:
                    if parent == stage:
                        break
                    self.assertEqual(stat.S_IMODE(parent.stat().st_mode), 0o755)

    def test_original_source_notices_in_metadata(self):
        module = runpy.run_path(str(ROOT / 'scripts/native-build-info.py'))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ('appliance', 'project-os', 'cockpit', 'docs', 'scripts',
                         'go.mod', 'go.sum', 'LICENSE', 'NOTICE'):
                (root / name).symlink_to(ROOT / name)
            stage = root / '.artifacts/native/x86_64'
            stage.mkdir(parents=True)
            for name in ('base', 'project-os', 'dashboard', 'forgejo', 'caddy'):
                (stage / (name + '.iid')).write_text('sha256:' + '1' * 64)
            def synthetic_output(args):
                return '[]' if '{{json .RepoDigests}}' in args else 'synthetic metadata; no commands run'
            with patch.dict(module['collect'].__globals__, output=synthetic_output), patch('platform.system', return_value='Linux'), patch('platform.machine', return_value='x86_64'):
                module['collect'](root, 'x86_64', '1' * 40)
            for source, name in (('LICENSE', 'soda-LICENSE'), ('NOTICE', 'soda-NOTICE')):
                self.assertEqual((stage / 'notices' / name).read_bytes(), (ROOT / source).read_bytes())

    def test_destination_refusal_before_writes(self):
        installer = (ROOT / 'scripts/install-native.sh').read_text()
        start = installer.index('python3 - "$bundle/build-info.json"', installer.index('# Inspect existing destination ancestors'))
        end = installer.index('\nPY\n', start)
        program = installer[start:end].split("<<'PY'\n", 1)[1]
        tree = ast.parse(program)
        # Execute the actual readonly function; its production caller is fixed '/'.
        self.assertEqual(ast.unparse(tree.body[-1]), "check_destinations(Path('/'), json.loads(Path(sys.argv[1]).read_text())['Files'])")
        namespace = {}
        exec(compile(ast.Module(body=tree.body[:-1], type_ignores=[]), '<installer-preflight>', 'exec'), namespace)
        check = namespace['check_destinations']
        self.assertLess(end, installer.index('install -d -m 0700 /etc/soda'))
        self.assertLess(end, installer.index('configure_network apply'))
        real_stat = Path.stat
        def root_owned(path, *args, **kwargs):
            values = list(real_stat(path, *args, **kwargs))
            values[4] = 0  # Simulate host root ownership, never chown the test host.
            return os.stat_result(values)
        with tempfile.TemporaryDirectory() as tmp, patch.object(Path, 'stat', root_owned):
            root = Path(tmp)
            payload = [PREFIX + name for name in FILES]
            check(root, payload)
            self.assertEqual(list(root.iterdir()), [])
            for name in FILES:
                target = root / PREFIX.removeprefix('rootfs/') / name
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text('operator bytes')
                with self.assertRaisesRegex(SystemExit, 'occupied Sodaspaces'):
                    check(root, payload)
                self.assertEqual(target.read_text(), 'operator bytes')
                target.unlink()  # Only exact temporary fixture files.
            target = root / PREFIX.removeprefix('rootfs/') / FILES[0]
            target.symlink_to(root / 'missing')
            with self.assertRaisesRegex(SystemExit, 'symlink'):
                check(root, payload)
            target.unlink()
            os.mkfifo(target)
            with self.assertRaisesRegex(SystemExit, 'occupied Sodaspaces'):
                check(root, payload)
            target.unlink()
            target.parent.chmod(0o777)
            with self.assertRaisesRegex(SystemExit, 'unsafe installation'):
                check(root, payload)
            target.parent.chmod(0o755)
            # Keep the one stock CoreOS link; refuse other links in ancestry.
            (root / 'usr').mkdir()
            (root / 'var/usrlocal').mkdir()
            (root / 'usr/local').symlink_to(root / 'var/usrlocal', target_is_directory=True)
            check(root, ['rootfs/usr/local/bin/tool'])
            (root / 'var/usrlocal/bin').symlink_to(root / 'elsewhere')
            with self.assertRaisesRegex(SystemExit, 'symlink'):
                check(root, ['rootfs/usr/local/bin/tool'])
