"""A missing welcome hook must not pass the quiet noninteractive check."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class OperatorProbe(unittest.TestCase):
    def test_missing_failed_noisy_and_quiet_hooks(self):
        source = (ROOT / 'tests/installed/operator.sh').read_text().splitlines()
        probe = next(line for line in source if line.startswith('quiet=$('))
        assertion = next(line for line in source if line.startswith('[[ "$quiet"'))
        for content, success in [(None, False), ('false\n', False), ('echo noise\n', False), (':\n', True)]:
            with self.subTest(content=content), tempfile.TemporaryDirectory() as directory:
                hook = Path(directory) / 'hook'
                if content is not None:
                    hook.write_text(content)
                command = probe.replace('/etc/profile.d/soda-console-welcome.sh', str(hook)) + '\n' + assertion
                result = subprocess.run(['bash', '-ec', command], env=os.environ,
                                        capture_output=True, timeout=5)
                self.assertEqual(result.returncode == 0, success)


if __name__ == '__main__':
    unittest.main()
