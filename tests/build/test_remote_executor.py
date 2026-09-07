"""Synthetic dispatcher tests: no real Git, build, SSH, VM or provider command."""
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import types
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[2] / 'internal/acceptance/remote_executor.py'
spec = importlib.util.spec_from_file_location('native_remote_executor', SOURCE)
executor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(executor)
REVISION = 'a' * 40


class RemoteExecutorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.work = Path(self.temp.name) / 'run'
        self.request = dict(Revision=REVISION, Architecture='x86_64', Target='synthetic-builder', Work=str(self.work), Phase='prepare')
        self.calls = []

    def run_command(self, command, **kwargs):
        self.calls.append(command)
        if 'clone' in command:
            Path(command[-1]).mkdir()
        return subprocess.CompletedProcess(command, 0)

    def output(self, command, **kwargs):
        return (REVISION + '\n').encode() if command[1] == 'rev-parse' else b''

    def invoke(self, phase, **changes):
        request = dict(self.request, Phase=phase, **changes)
        with patch.object(executor.sys, 'stdin', types.SimpleNamespace(buffer=io.BytesIO(json.dumps(request).encode()))), \
             patch.object(executor.sys, 'stdout', io.StringIO()), \
             patch.object(executor.platform, 'system', return_value='Linux'), \
             patch.object(executor.platform, 'machine', return_value='x86_64'), \
             patch.object(executor.platform, 'node', return_value='synthetic-builder'), \
             patch.object(executor.subprocess, 'run', side_effect=self.run_command), \
             patch.object(executor.subprocess, 'check_output', side_effect=self.output):
            executor.main()

    def test_explicit_phase_order_without_automatic_work(self):
        self.invoke('prepare')
        self.assertEqual(len(self.calls), 2)
        self.assertFalse((self.work / 'build.started').exists())
        self.invoke('build')
        self.assertEqual(self.calls[-1], ['bash', 'scripts/build-native.sh', 'x86_64'])
        self.assertFalse((self.work / 'check.started').exists())
        self.invoke('check')
        self.assertEqual(self.calls[-1], ['bash', 'scripts/check-native.sh', 'x86_64'])
        self.invoke('bundle')
        self.assertIn('--revision', self.calls[-1])
        self.assertTrue((self.work / 'bundle.completed').is_file())
        self.assertFalse(any('install-native.sh' in part for call in self.calls for part in call))

    def test_unknown_phase_target_arch_and_revision_fail_before_creation(self):
        for phase, changes in [('install', {}), ('prepare', {'Target': 'other'}), ('prepare', {'Architecture': 'aarch64'}), ('prepare', {'Revision': 'main'})]:
            with self.subTest(phase=phase, changes=changes):
                with self.assertRaises(ValueError):
                    self.invoke(phase, **changes)
                self.assertFalse(self.work.exists())
                self.assertEqual(self.calls, [])

    def test_occupied_prepare_and_phase_replay_do_not_run_commands(self):
        self.invoke('prepare')
        before = len(self.calls)
        with self.assertRaises(FileExistsError):
            self.invoke('prepare')
        self.assertEqual(len(self.calls), before)
        self.invoke('build')
        before = len(self.calls)
        with self.assertRaises(FileExistsError):
            self.invoke('build')
        self.assertEqual(len(self.calls), before)

    def test_missing_prerequisite_stale_binding_and_dirty_source(self):
        self.invoke('prepare')
        before = len(self.calls)
        with self.assertRaises(ValueError):
            self.invoke('check')
        with self.assertRaises(ValueError):
            self.invoke('build', Revision='b' * 40)
        self.output = lambda command, **kwargs: (REVISION + '\n').encode() if command[1] == 'rev-parse' else b' M source.go\n'
        with self.assertRaises(ValueError):
            self.invoke('build')
        self.assertEqual(len(self.calls), before)
        self.assertFalse((self.work / 'build.started').exists())

    def test_failed_build_retains_started_without_completion(self):
        self.invoke('prepare')
        def fail(command, **kwargs):
            raise subprocess.CalledProcessError(23, command)
        self.run_command = fail
        with self.assertRaises(subprocess.CalledProcessError):
            self.invoke('build')
        self.assertTrue((self.work / 'build.started').is_file())
        self.assertFalse((self.work / 'build.completed').exists())


if __name__ == '__main__':
    unittest.main()
