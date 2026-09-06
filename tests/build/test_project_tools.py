"""Authored tests over local fixtures/process doubles, not native build proof."""
import importlib.util
import io
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


class ProjectTools(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / 'scripts'
        scripts.mkdir()
        source = Path(__file__).resolve().parents[2] / 'scripts/build-project-tools.py'
        copied = scripts / source.name
        shutil.copyfile(source, copied)
        spec = importlib.util.spec_from_file_location('project_tools_fixture', copied)
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        lock = self.root / 'project-os/locks'
        lock.mkdir(parents=True)
        (lock / 'tea-source.toml').write_text('version = "0.15.1"\nsource_archive = "tea.tar.gz"\n')
        license_dir = self.root / 'project-os/licenses'
        license_dir.mkdir()
        (license_dir / 'tea-LICENSE').write_text('fixture license\n')
        archives = self.root / '.artifacts/tools'
        archives.mkdir(parents=True)
        with tarfile.open(archives / 'tea.tar.gz', 'w:gz') as tar:
            for name in ['tea/go.mod', 'tea/Makefile']:
                body = b'fixture source, never executed\n'
                info = tarfile.TarInfo(name)
                info.size = len(body)
                tar.addfile(info, io.BytesIO(body))

    def test_native_guard_precedes_mutation(self):
        for system, machine in [('Darwin', 'x86_64'), ('Linux', 'aarch64')]:
            with self.subTest(system=system, machine=machine), patch.object(self.module.platform, 'system', return_value=system), patch.object(self.module.platform, 'machine', return_value=machine), patch.object(self.module.subprocess, 'run') as run:
                with self.assertRaises(SystemExit):
                    self.module.build('x86_64')
                run.assert_not_called()
                self.assertFalse((self.root / '.artifacts/native').exists())

    def test_native_build_inputs_and_project_only_outputs(self):
        calls = []

        def run(args, **kwargs):
            calls.append(args)
            if args[0] == 'make':
                self.assertEqual(kwargs['env']['GOOS'], 'linux')
                self.assertEqual(kwargs['env']['GOARCH'], 'amd64')
                self.assertEqual(kwargs['env']['CGO_ENABLED'], '0')
                self.assertNotIn('GOFLAGS', kwargs['env'])
                self.assertEqual(kwargs['env']['TEA_VERSION'], '0.15.1')
                (Path(kwargs['cwd']) / 'tea').write_text('not a real binary; process double\n')
            return subprocess.CompletedProcess(args, 0, stdout='Version: \x1b[1m0.15.1\x1b[0m\tgolang: 1.26.7\tgo-sdk: v1.2.0\n')

        with patch.dict(self.module.os.environ, {'GOFLAGS': '-mod=readonly'}), patch.object(self.module.platform, 'system', return_value='Linux'), patch.object(self.module.platform, 'machine', return_value='x86_64'), patch.object(self.module.subprocess, 'run', side_effect=run):
            self.module.build('x86_64')
        output = self.root / '.artifacts/native/x86_64/project-tools'
        self.assertEqual((output / 'bin/tea').stat().st_mode & 0o777, 0o755)
        self.assertEqual((output / 'licenses/tea/LICENSE').read_text(), 'fixture license\n')
        self.assertEqual(calls[0], [str(self.root / 'scripts/fetch-tea-source.sh')])
        self.assertEqual(calls[1], ['make', 'BUILDMODE=-buildvcs=false -mod=readonly', 'build'])
        self.assertEqual(calls[2], [str(output / 'bin/tea'), '--version'])
        self.assertFalse((self.root / '.artifacts/native/x86_64/rootfs').exists())

    def test_different_native_version_is_rejected(self):
        def run(args, **kwargs):
            if args[0] == 'make':
                (Path(kwargs['cwd']) / 'tea').write_text('fixture binary\n')
            return subprocess.CompletedProcess(args, 0, stdout='Version: \x1b[1m0.15.10\x1b[0m\tgolang: 1.26.7\n')

        with patch.object(self.module.platform, 'system', return_value='Linux'), patch.object(self.module.platform, 'machine', return_value='x86_64'), patch.object(self.module.subprocess, 'run', side_effect=run):
            with self.assertRaisesRegex(SystemExit, 'version does not match'):
                self.module.build('x86_64')


if __name__ == '__main__':
    unittest.main()
