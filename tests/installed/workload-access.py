#!/usr/bin/env python3
"""U08 real client/member HTTP and committed PostgreSQL data checks.

Requires psycopg 3 and the retained project-network diagnostic workloads. This
proves that mode only, not the currently blocked default nested bridge. Creates
new run-owned SQL/probe/credential files; retains dirty source and database data.
"""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import urllib.request
import uuid
import psycopg


def main():
    assert os.environ.get('SODA_NATIVE_VALIDATE') == 'soda-test'
    assert len(sys.argv) == 2
    root = Path(sys.argv[1]); st = root.lstat()
    assert root.is_absolute() and not root.is_symlink() and st.st_uid == os.getuid() and not st.st_mode & 0o077
    run = uuid.uuid4().hex
    output = root / ('u08-workload-access-' + run)
    output.mkdir(mode=0o700)
    config = root / 'developer-client.conf'

    def remote(who, command, data=None):
        result = subprocess.run(['ssh', '-F', str(config), 'u08-' + who + '-8417@10.89.0.2', command],
                                input=data, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
        if result.returncode: raise RuntimeError('Native member operation failed')
        return result.stdout

    password = remote('alice', 'head -c 128 "$HOME/.config/soda-u08-workload/database-password"').decode()
    assert re.fullmatch(r'[A-Za-z0-9_-]{43}', password)
    passfile = output / 'pgpass'
    fd = os.open(passfile, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    record = ('10.89.0.2:5432:soda_example:developer:' + password + '\n').encode()
    with os.fdopen(fd, 'wb') as stream: stream.write(record)
    del password
    for who in ('alice', 'bob'):
        remote(who, f'umask 077; mkdir -p "$HOME/.config"; mkdir -m700 "$HOME/.config/u08-db-client-{run}"; cat > "$HOME/.config/u08-db-client-{run}/pgpass"', record)
    del record
    parameters = dict(host='10.89.0.2', port=5432, dbname='soda_example', user='developer', passfile=str(passfile), connect_timeout=10)
    with psycopg.connect(**parameters) as database:
        database.execute('CREATE TABLE IF NOT EXISTS soda_u08_probe (run_id text PRIMARY KEY, value text NOT NULL)')
        database.execute('INSERT INTO soda_u08_probe VALUES (%s, %s)', (run, 'client-committed'))
    # A separate connection observes a committed value, not transaction-local data.
    with psycopg.connect(**parameters) as database:
        assert database.execute('SELECT value FROM soda_u08_probe WHERE run_id=%s', (run,)).fetchone() == ('client-committed',)
    for who in ('alice', 'bob'):
        sql = f"SELECT value FROM soda_u08_probe WHERE run_id='{run}';\n"
        prefix = f'PGPASSFILE="$HOME/.config/u08-db-client-{run}/pgpass" psql -X -w -h 10.89.0.2 -p 5432 -U developer -d soda_example -At -v ON_ERROR_STOP=1'
        assert remote(who, prefix, sql.encode()).strip() == b'client-committed'
    prefix = f'PGPASSFILE="$HOME/.config/u08-db-client-{run}/pgpass" psql -X -w -h 10.89.0.2 -p 5432 -U developer -d soda_example -At -v ON_ERROR_STOP=1'
    remote('bob', prefix, f"UPDATE soda_u08_probe SET value='bob-committed' WHERE run_id='{run}';\n".encode())
    with psycopg.connect(**parameters) as database:
        assert database.execute('SELECT value FROM soda_u08_probe WHERE run_id=%s', (run,)).fetchone() == ('bob-committed',)
    # Preserve the original public source, then retain an ordinary dirty edit.
    previous = remote('alice', 'head -c 1048576 "$HOME/u08-personal-checkout/workload/public/index.html"')
    (output / 'previous-index.html').write_bytes(previous)
    content = ('<!doctype html><title>U08</title><h1>Live bind mount ' + run + '</h1>\n').encode()
    remote('alice', 'cat > "$HOME/u08-personal-checkout/workload/public/index.html"', content)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with opener.open('http://10.89.0.2:8000/', timeout=10) as response:
        assert response.status == 200 and response.read(1048576) == content
    for who in ('alice', 'bob'):
        assert remote(who, 'curl --noproxy "*" --fail --silent --show-error --max-time 10 http://10.89.0.2:8000/') == content
    (output / 'results.json').write_text(json.dumps({'run_id': run, 'network_mode': 'project namespace (nested host mode)',
        'http_bind_edit_client_and_members': True, 'postgres_client_and_members': True, 'committed_value': 'bob-committed',
        'restart_persistence_verified': False, 'default_nested_bridge_verified': False}, indent=2))
    print('Client and Alice/Bob verified live HTTP bind-mount edit and committed PostgreSQL read/write/readback.')
    print('Verified only nested project-network mode. Default bridge and restart/reboot persistence remain unverified.')


if __name__ == '__main__':
    try:
        main()
    except Exception as failure:
        # Do not print DB connection strings, passwords, native bodies or logs.
        print('Workload access incomplete; retained state; failure type: ' + type(failure).__name__, file=sys.stderr)
        sys.exit(1)
