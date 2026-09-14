"""Local timing and process fixtures; never build images or operate a VM."""
import contextlib
import importlib.util
import io
import os
from pathlib import Path
import selectors
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
import build_progress as timing


def clean_env():
    return {key: value for key, value in os.environ.items() if not key.startswith('SODA_BUILD_')}


class Timing(unittest.TestCase):
    def test_monotonic_section_total_and_hours(self):
        output = io.StringIO()
        with patch.dict(os.environ, {'SODA_BUILD_START_NS': '0'}, clear=True), patch.object(timing, 'clock', return_value=3_665_000_000_000), contextlib.redirect_stderr(output):
            timing.emit('DONE', 'Compile fixture', 3_600_000_000_000)
        self.assertIn('section 00:01:05 | total 01:01:05', output.getvalue())

    def test_failed_section_retains_completed_timings_and_exit_code(self):
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, clean_env(), clear=True), contextlib.redirect_stderr(io.StringIO()) as output:
            log = Path(directory) / 'timing.log'
            timing.create_log(log)
            progress = timing.Progress()
            with progress.section('First'):
                pass
            with self.assertRaises(subprocess.CalledProcessError):
                with progress.section('Second'):
                    raise subprocess.CalledProcessError(7, ['private-argument-must-not-be-logged'])
            timing.finish('Fixture', 7)
            text = log.read_text()
            self.assertIn('DONE     First', text)
            self.assertIn('FAILED   Second', text)
            self.assertIn('exit 7', text)
            self.assertNotIn('private-argument', text)
            self.assertIn('SECTION SUMMARY', output.getvalue())
            self.assertEqual(log.stat().st_mode & 0o777, 0o600)

    def test_existing_log_is_never_overwritten(self):
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, clean_env(), clear=True):
            log = Path(directory) / 'timing.log'
            log.write_text('retained')
            with self.assertRaises(FileExistsError):
                timing.create_log(log)
            self.assertEqual(log.read_text(), 'retained')

    def test_child_suppresses_only_final_summary(self):
        with patch.dict(os.environ, {'SODA_BUILD_START_NS': '1', 'SODA_BUILD_CHILD': '1'}, clear=True), contextlib.redirect_stderr(io.StringIO()) as output:
            before = timing.origin()
            with timing.Progress().section('Child step'):
                pass
            timing.finish('Child', 0)
            self.assertEqual(before, timing.origin())
        self.assertIn('DONE     Child step', output.getvalue())
        self.assertNotIn('SUCCESS', output.getvalue())

    def test_shell_checkpoints_preserve_capture_and_failure(self):
        script = '''set -euo pipefail
source "$1/scripts/build-progress.sh"
progress_next 'Captured observation'
result=$(printf 'native-image-id')
[[ "$result" == native-image-id ]]
progress_next 'Failing tool'
bash -c 'exit 7'
printf 'must not run'
'''
        result = subprocess.run(['bash', '-c', script, 'fixture', str(ROOT)], env=clean_env(), capture_output=True, text=True)
        self.assertEqual(result.returncode, 7)
        self.assertEqual(result.stdout, '')
        self.assertIn('DONE     Captured observation', result.stderr)
        self.assertIn('FAILED   Failing tool', result.stderr)
        self.assertIn('exit 7', result.stderr)
        self.assertNotIn('SUCCESS', result.stderr)

    def test_supervisor_terminates_active_child_and_descendant(self):
        for signum in (signal.SIGINT, signal.SIGTERM):
            with tempfile.TemporaryDirectory() as directory:
                marker = Path(directory) / 'descendant-stopped'
                child = Path(directory) / 'child.py'
                child.write_text('''import os, signal, subprocess, sys
from pathlib import Path
if len(sys.argv) == 3:
    def stop(sig, frame):
        Path(sys.argv[1]).write_text('stopped')
        sys.exit(0)
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    print('READY', flush=True)
    signal.pause()
else:
    descendant = subprocess.Popen([sys.executable, __file__, sys.argv[1], 'descendant'])
    def stop(sig, frame):
        descendant.wait(timeout=3)
        sys.exit(128 + sig)
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    signal.pause()
''')
                shell = 'set -euo pipefail; source "$1/scripts/build-progress.sh"; progress_next "Active fixture"; python3 "$2" "$3"'
                process = subprocess.Popen([sys.executable, str(ROOT / 'scripts/build_progress.py'), 'supervise', 'bash', '-c', shell, 'fixture', str(ROOT), str(child), str(marker)], env=clean_env(), stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                try:
                    with selectors.DefaultSelector() as selector:
                        selector.register(process.stdout, selectors.EVENT_READ)
                        self.assertTrue(selector.select(timeout=5), 'fixture did not start')
                        self.assertEqual(process.stdout.readline().strip(), 'READY')
                    process.send_signal(signum)
                    _, error = process.communicate(timeout=8)
                    self.assertEqual(process.returncode, 128 + signum, error)
                    self.assertEqual(marker.read_text(), 'stopped')
                    self.assertIn('CANCELLED', error)
                    self.assertIn('Active fixture', error)
                    self.assertNotIn('SUCCESS', error)
                finally:
                    if process.poll() is None:
                        process.terminate()
                        process.communicate(timeout=8)


class ProductionOwnership(unittest.TestCase):
    def test_both_layouts_delegate_instead_of_copying_component_recipes(self):
        shell = (ROOT / 'scripts/build-native.sh').read_text()
        host = (ROOT / 'internal/hostimage/build_payload.go').read_text()
        legacy = (ROOT / 'tools/soda-host-image/legacy.go').read_text()
        for duplicate in ('scripts/build-forgejo.ts', 'scripts/fetch-tea.py', 'appliance/tailnet.Containerfile'):
            self.assertNotIn(duplicate, shell)
            self.assertNotIn(duplicate, host)
            self.assertNotIn(duplicate, legacy)
        self.assertIn('--legacy-native', shell)
        self.assertEqual(host.count('producer.Assets(context, forgejoContext)'), 1)
        self.assertNotIn('nativeRoot', host)
        self.assertEqual(host.count('producer.Images('), 1)
        self.assertEqual(legacy.count('p.Assets("", "")'), 1)
        self.assertEqual(legacy.count('p.Images('), 1)
        # The native owner no longer executes Python clock/reporting helpers.
        progress = (ROOT / 'internal/nativebuild/progress.go').read_text()
        self.assertNotIn('scripts/build_progress.py', progress)
        controller = (ROOT / 'internal/hostimage/build.go').read_text()
        self.assertIn('acceptance.StartCommand(ctx, cmd)', controller)
        self.assertNotIn('BuildExecution', controller)


class CompleteBuild(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('timed_installer_fixture', ROOT / 'scripts/build-installer.py')
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)

    def run_fixture(self, directory, failure=None, full=True, iso_failure=None):
        root = Path(directory)
        args = ['build-installer.py', '--build-native', '--arch', 'x86_64', '--butane', '/fixture/butane', '--coreos-installer', '/fixture/installer', '--keyring', '/fixture/keyring', '--signer', 'A' * 40, '--out', str(root / 'iso')]
        if not full:
            args.remove('--build-native')
            args += ['--bundle-source', str(root / 'existing-payload')]
        order = []
        def native(command, **kwargs):
            order.append('native')
            self.assertEqual(command, ['bash', str(root / 'scripts/build-native.sh'), 'x86_64'])
            self.assertEqual(kwargs['env']['SODA_BUILD_CHILD'], '1')
            self.assertIn('SODA_BUILD_START_NS', kwargs['env'])
            if failure:
                raise subprocess.CalledProcessError(failure, command)
            verifier = root / '.artifacts/native/x86_64/tools/soda-artifacts'
            verifier.parent.mkdir(parents=True)
            verifier.write_bytes(b'\x7fELFfresh-producer-fixture')
            verifier.chmod(0o755)
        def iso(options, native_verifier=None):
            order.append('iso')
            self.assertEqual(native_verifier, b'\x7fELFfresh-producer-fixture' if full else None)
            self.assertEqual(options.bundle_source, str(root / '.artifacts/native/x86_64' if full else root / 'existing-payload'))
            self.module.progress.next('ISO / Fixture verification')
            if iso_failure:
                raise iso_failure
            self.module.progress.end()
        with patch.dict(os.environ, clean_env(), clear=True), patch.object(self.module, 'ROOT', root), patch.object(self.module, 'preflight'), patch.object(self.module, 'output', return_value='a' * 40), patch.object(self.module.subprocess, 'run', side_effect=native), patch.object(self.module, 'build', side_effect=iso), patch.object(self.module.signal, 'signal'), patch.object(sys, 'argv', args), contextlib.redirect_stderr(io.StringIO()) as output:
            code = self.module.main()
        return code, order, output.getvalue(), (root / 'iso.timing.log').read_text()

    def test_complete_entrypoint_hands_native_payload_to_iso(self):
        with tempfile.TemporaryDirectory() as directory:
            code, order, output, log = self.run_fixture(directory)
        self.assertEqual(code, 0)
        self.assertEqual(order, ['native', 'iso'])
        self.assertIn('DONE     Native phase', log)
        self.assertIn('DONE     ISO phase', log)
        self.assertIn('Application OCI archives', output)
        self.assertEqual(output.count('SUCCESS'), 1)

    def test_produced_verifier_is_snapshot_not_untrusted_bundle_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            verifier = root / '.artifacts/native/x86_64/tools/soda-artifacts'
            verifier.parent.mkdir(parents=True)
            verifier.write_bytes(b'\x7fELFbefore')
            verifier.chmod(0o755)
            with patch.object(self.module, 'ROOT', root):
                snapshot = self.module.produced_verifier('x86_64')
                verifier.write_bytes(b'\x7fELFafter')
                self.assertEqual(snapshot, b'\x7fELFbefore')
                verifier.chmod(0o777)
                with self.assertRaises(ValueError):
                    self.module.produced_verifier('x86_64')
                verifier.chmod(0o755)
                retained = verifier.with_name('retained')
                verifier.rename(retained)
                verifier.symlink_to(retained)
                with self.assertRaises(ValueError):
                    self.module.produced_verifier('x86_64')

    def test_failed_native_build_never_starts_iso(self):
        with tempfile.TemporaryDirectory() as directory:
            code, order, output, log = self.run_fixture(directory, failure=23)
        self.assertEqual(code, 23)
        self.assertEqual(order, ['native'])
        self.assertIn('FAILED   Native phase', log)
        self.assertNotIn('START    ISO phase', log)
        self.assertNotIn('SUCCESS', output)

    def test_standalone_iso_reports_only_its_own_phase(self):
        with tempfile.TemporaryDirectory() as directory:
            code, order, output, log = self.run_fixture(directory, full=False)
        self.assertEqual(code, 0)
        self.assertEqual(order, ['iso'])
        self.assertNotIn('Native phase', log)
        self.assertIn('existing native payload', log)
        self.assertIn('SUCCESS  Soda ISO', log)

    def test_iso_interruption_reports_section_phase_and_total(self):
        with tempfile.TemporaryDirectory() as directory:
            code, order, output, log = self.run_fixture(directory, iso_failure=KeyboardInterrupt())
        self.assertEqual(code, 130)
        self.assertIn('CANCELLED ISO / Fixture verification', log)
        self.assertIn('CANCELLED ISO phase', log)
        self.assertIn('exit 130', log)
        self.assertNotIn('SUCCESS', output)

    def test_occupied_native_output_is_rejected_before_building(self):
        from types import SimpleNamespace
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            native = root / '.artifacts/native/x86_64'
            native.mkdir(parents=True)
            (native / 'retained').write_text('keep')
            tool = root / 'tool'
            tool.write_text('fixture')
            tool.chmod(0o700)
            keyring = root / 'keyring'
            keyring.write_text('fixture')
            args = SimpleNamespace(arch='x86_64', butane=str(tool), coreos_installer=str(tool), xorriso=str(tool), keyring=str(keyring), signer='A' * 40, network_keyfile=None, out=str(root / '.artifacts/iso'), build_native=True, bundle_source=None)
            with patch.object(self.module, 'ROOT', root), patch.object(self.module.platform, 'system', return_value='Linux'), patch.object(self.module.platform, 'machine', return_value='x86_64'), patch.object(self.module, 'output', side_effect=['', self.module.INSTALLER_VERSION]), patch.object(self.module.subprocess, 'run') as run:
                with self.assertRaisesRegex(FileExistsError, 'native output already exists'):
                    self.module.preflight(args)
                run.assert_not_called()
            self.assertEqual((native / 'retained').read_text(), 'keep')
            self.assertFalse((root / '.artifacts/iso.timing.log').exists())


if __name__ == '__main__':
    unittest.main()
