"""Production staging/preflight in temporary filesystems; never host installation."""
import ast
import hashlib
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
         'public/assets/sodaspaces.css', 'public/assets/sodaspaces.js',
         'public/assets/sodaspaces-terminal.js', 'public/assets/sodaspaces-terminal.css',
         'public/assets/sodaspaces-drawer.js', 'public/assets/sodaspaces-drawer.css')
VENDOR_FILES = tuple('public/assets/soda-terminal/' + f['file'] for item in json.loads((ROOT / 'appliance/terminal-assets.lock.json').read_text()) for f in item['files'])
PREFIX = 'rootfs/var/lib/soda/forgejo/gitea/'


class SodaspacesPackaging(unittest.TestCase):
    def test_vm_web_tunnel_uses_only_the_native_browser_origin(self):
        with tempfile.TemporaryDirectory() as tmp:
            ssh = Path(tmp) / 'ssh'
            ssh.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            ssh.chmod(0o755)
            result = subprocess.run(['bash', str(ROOT / 'scripts/test-vm.sh'), 'web-tunnel'],
                                    env={**os.environ, 'PATH': tmp + os.pathsep + os.environ['PATH']},
                                    capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0)
            self.assertIn('Forgejo + Sodaspaces https://localhost:24444', result.stdout)
            self.assertIn('127.0.0.1:24444:127.0.0.1:24444', result.stdout)
            self.assertNotIn('24443', result.stdout)

    def test_browser_probe_refuses_bad_inputs_without_secret_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / 'input.json'
            source.write_text('invalid JSON SYNTHETIC_AUTH_SECRET_MARKER')
            source.chmod(0o600)
            for permission in ([], ['--allow-auth-transitions'], ['--allow-auth-transitions', '--allow-environment-access']):
                result = subprocess.run(['node', str(ROOT / 'tests/installed/sodaspaces.mjs'),
                                         str(source), str(root / 'not-created'), *permission],
                                        capture_output=True, text=True, timeout=15)
                self.assertEqual(result.returncode, 1)
                self.assertIn('private input validation', result.stdout)
                self.assertNotIn('SYNTHETIC_AUTH_SECRET_MARKER', result.stdout + result.stderr)
                self.assertFalse((root / 'not-created').exists())

    def test_access_probe_rejects_private_bad_request_before_native_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            root.chmod(0o700)
            request = root / 'target.json'
            request.write_text('{"SYNTHETIC_PRIVATE_MARKER":true}')
            request.chmod(0o600)
            result = subprocess.run(['python3', str(ROOT / 'tests/installed/developer-access.py'), str(root)],
                                    capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 1)
            self.assertIn('Developer access incomplete', result.stderr)
            self.assertNotIn('SYNTHETIC_PRIVATE_MARKER', result.stdout + result.stderr)
            self.assertEqual(list(root.iterdir()), [request])

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
            shutil.copytree(ROOT / 'assets', checkout / 'assets')
            for asset in (checkout / 'assets').rglob('*'):
                asset.chmod(0o700 if asset.is_dir() else 0o600)
            shutil.copytree(ROOT / 'appliance', checkout / 'appliance')
            (checkout / 'internal/nativebuild').mkdir(parents=True)
            shutil.copyfile(ROOT / 'internal/nativebuild/forgejo-payload.json', checkout / 'internal/nativebuild/forgejo-payload.json')
            for name in ('LICENSE', 'NOTICE'):
                shutil.copyfile(ROOT / name, checkout / name)
            (checkout / 'cmd/soda-dashboard').mkdir(parents=True)
            build = checkout / '.artifacts/native/x86_64'
            (build / 'bin').mkdir(parents=True)
            (build / 'bin/soda-dashboard').write_text('synthetic; never executed')
            (build / 'github-actions-runner').mkdir()
            (build / 'forgejo-locales').mkdir()
            (build / 'forgejo-locales/locale_en-US.ini').write_text('synthetic full-catalog output; not native proof')
            # Synthetic bytes and lock only inside this temporary checkout.
            (build / 'terminal-assets').mkdir()
            lock_path = checkout / 'appliance/terminal-assets.lock.json'
            lock = json.loads(lock_path.read_text())
            for item in lock:
                for asset in item['files']:
                    data = ('synthetic asset ' + asset['file']).encode()
                    (build / 'terminal-assets' / asset['file']).write_bytes(data)
                    asset['sha256'] = hashlib.sha256(data).hexdigest()
            lock_path.write_text(json.dumps(lock))
            for page in ('tailscale', 'runners'):
                (checkout / 'cockpit/dist' / f'soda-{page}').mkdir(parents=True)
            previous = os.umask(0o077)
            try:
                with patch('sys.argv', ['stage.py', '--arch', 'x86_64']), patch('platform.system', return_value='Linux'), patch('platform.machine', return_value='x86_64'):
                    runpy.run_path(str(checkout / 'scripts/stage.py'), run_name='__main__')
            finally:
                os.umask(previous)
            stage = build / 'rootfs'
            for asset in (stage / PREFIX.removeprefix('rootfs/') / 'public/assets').rglob('*'):
                self.assertEqual(stat.S_IMODE(asset.stat().st_mode), 0o755 if asset.is_dir() else 0o644)
            for name in FILES:
                p = stage / PREFIX.removeprefix('rootfs/') / name
                self.assertEqual(p.read_bytes(), (checkout / 'appliance/forgejo' / name).read_bytes())
                self.assertEqual(stat.S_IMODE(p.stat().st_mode), 0o644)
                for parent in p.parents:
                    if parent == stage:
                        break
                    self.assertEqual(stat.S_IMODE(parent.stat().st_mode), 0o755)

            for name, origin in json.loads((checkout / 'internal/nativebuild/forgejo-payload.json').read_text()).items():
                original = build / origin.removeprefix('@build/') if origin.startswith('@build/') else checkout / origin
                target = stage / PREFIX.removeprefix('rootfs/') / name
                self.assertEqual(target.read_bytes(), original.read_bytes())
                self.assertEqual(stat.S_IMODE(target.stat().st_mode), 0o644)

            for name in VENDOR_FILES:
                p = stage / PREFIX.removeprefix('rootfs/') / name
                self.assertEqual(p.read_bytes(), ('synthetic asset ' + Path(name).name).encode())
                self.assertEqual(stat.S_IMODE(p.stat().st_mode), 0o644)
                self.assertEqual(stat.S_IMODE(p.parent.stat().st_mode), 0o755)

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
            payload = [PREFIX + name for name in FILES + VENDOR_FILES]
            check(root, payload)
            self.assertEqual(list(root.iterdir()), [])
            for name in FILES + VENDOR_FILES:
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
