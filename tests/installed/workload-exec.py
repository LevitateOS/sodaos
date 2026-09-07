#!/usr/bin/env python3
"""U08: ordinary exec into the selected different-UID PostgreSQL workload.

Run after workloads.sh in the approved fresh fixture. No lifecycle, credentials,
process attachment/debugging, data writes or workload replacement.
"""
import ipaddress
import json
import os
from pathlib import Path
import subprocess
import sys


def main():
    assert os.environ.get('SODA_NATIVE_VALIDATE') == 'soda-test'
    assert len(sys.argv) == 2
    fixture = Path(sys.argv[1])
    st = fixture.lstat()
    assert fixture.is_absolute() and not fixture.is_symlink() and fixture.is_dir()
    assert st.st_uid == os.getuid() and not st.st_mode & 0o077
    target = json.loads((fixture / 'target.json').read_text())
    ip = target['ip']
    assert ipaddress.ip_address(ip) in ipaddress.ip_network('10.89.0.0/24')
    ssh = ['ssh', '-F', str(fixture / 'developer-client.conf'), '-o', 'BatchMode=yes', '-o', 'ForwardAgent=no', '-o', 'ClearAllForwardings=yes']
    owner = 'u08-alice-8417@' + ip
    command = '''set -eu
podman exec workload_database_1 grep -Eq '^Uid:[[:space:]]+999[[:space:]]+999[[:space:]]+999[[:space:]]+999$' /proc/1/status
test "$(podman exec workload_database_1 id -u)" = 0
test "$(podman exec --user postgres workload_database_1 id -u)" = 999
podman exec --user postgres workload_database_1 psql -X -U developer -d soda_example -At -c 'select current_database()'
podman exec -t --user postgres workload_database_1 true
'''
    result = subprocess.run(ssh + [owner, 'sh -se'], input=command, text=True, capture_output=True, timeout=60)
    assert result.returncode == 0, 'Different-UID exec failed; keep native state for diagnosis'
    assert result.stdout.strip() == 'soda_example'
    denied = subprocess.run(ssh + ['u08-bob-8417@' + ip, 'podman exec workload_database_1 true'], text=True, capture_output=True, timeout=30)
    assert denied.returncode != 0 and 'permission denied' in denied.stderr.lower(), 'Expected socket permission denial, not transport failure'
    print('Root/default-user, explicit PostgreSQL-user and PTY exec passed against UID-999 PID 1; ordinary member denied engine access.')


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        print('Workload exec check incomplete (' + type(error).__name__ + ').', file=sys.stderr)
        sys.exit(1)
