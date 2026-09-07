"""Exercise the actual read-only RPM preflights without running an installer."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class HostPackages(unittest.TestCase):
    def test_capability_providers_and_missing_dependencies(self):
        for source in ['scripts/install-native.sh', 'tests/installed/host.sh']:
            lines = [line for line in (ROOT / source).read_text().splitlines() if line.startswith('rpm -q ')]
            self.assertEqual(len(lines), 2)
            for missing in ['', 'nodejs', 'zlib', 'cockpit-ws']:
                with self.subTest(source=source, missing=missing), tempfile.TemporaryDirectory() as name:
                    fake = Path(name) / 'rpm'
                    fake.write_text('''#!/bin/sh
[ "$1" = -q ] || exit 99
shift
provides=no
if [ "$1" = --whatprovides ]; then provides=yes; shift; fi
for package do
    [ "$package" != "$MISSING" ] || exit 1
    case "$package" in
        nodejs|zlib) [ "$provides" = yes ] || exit 1 ;;
    esac
done
''')
                    fake.chmod(0o755)
                    result = subprocess.run(['/bin/sh', '-ec', '\n'.join(lines)],
                                            env={**os.environ, 'PATH':name, 'MISSING':missing},
                                            capture_output=True, timeout=5)
                    self.assertEqual(result.returncode, 1 if missing else 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
