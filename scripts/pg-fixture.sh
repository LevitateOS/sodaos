#!/bin/bash
# Ephemeral PostgreSQL 17 for Soda/Forgejo tests (A10 runtime fixture).
# Same pinned image and role==database shape as the appliance cluster, with
# random per-run passwords that never appear in argv, stdout or logs.
#
#   eval "$(scripts/pg-fixture.sh start)"   # prints KEY=VALUE assignments
#   ... run tests against $SODA_PG_HOST:$SODA_PG_PORT ...
#   scripts/pg-fixture.sh stop "$SODA_PG_CONTAINER"
#
# Start output (passwords exposed as file paths only):
#   SODA_PG_CONTAINER  container name for stop
#   SODA_PG_HOST       127.0.0.1
#   SODA_PG_PORT       mapped loopback port
#   SODA_PG_DATABASES  space-separated databases created (default "forgejo soda")
#   SODA_PG_DIR        secret dir holding <role>.passwd (mode 0600); caller removes it
#   SODA_PG_SUPER_PASSWORD_FILE  postgres superuser password file
#
# SODA_PG_FIXTURE_DIR reuses a caller-owned dir instead of mktemp.
# SODA_PG_DATABASES overrides the role/database list (names: [a-z0-9_]+).
# Exit 3 when the container engine or image is unavailable: callers skip.
set -euo pipefail

# Must match Image= in appliance/services/soda-postgres.container.
IMAGE='docker.io/library/postgres:17@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675'

command -v podman >/dev/null || { echo 'podman unavailable' >&2; exit 3; }

case "${1:-}" in
start) ;;
stop)
	[ -n "${2:-}" ] || { echo 'container name required' >&2; exit 2; }
	exec podman rm -f "$2" >/dev/null
	;;
*) echo "usage: $0 {start|stop <container>}" >&2; exit 2;;
esac

# An explicitly empty SODA_PG_DATABASES starts a bare cluster (superuser
# only) so callers can test role provisioning itself.
databases="${SODA_PG_DATABASES-forgejo soda}"
for db in $databases; do
	case "$db" in ''|*[!a-z0-9_]*|'pg_'*) echo "refusing database name: $db" >&2; exit 2;; esac
done

if ! podman image exists "$IMAGE" >/dev/null 2>&1 && ! podman pull "$IMAGE" >/dev/null 2>&1; then
	echo 'postgres fixture image unavailable' >&2
	exit 3
fi

dir="${SODA_PG_FIXTURE_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/soda-pg-fixture.XXXXXX")}"
mkdir -p "$dir"
for role in postgres $databases; do
	head -c 33 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$dir/$role.passwd"
	chmod 600 "$dir/$role.passwd"
done

name="soda-pg-test-$$-$RANDOM"
podman run -d --rm --name "$name" --pull=never \
	-p '127.0.0.1::5432' \
	-v "$dir/postgres.passwd:/run/secrets/soda-pg-super:ro,z" \
	-e POSTGRES_PASSWORD_FILE=/run/secrets/soda-pg-super \
	"$IMAGE" >/dev/null

cleanup() { podman rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

deadline=$((SECONDS + 90))
until podman exec -u postgres "$name" pg_isready -q 2>/dev/null; do
	[ $SECONDS -lt "$deadline" ] || { echo 'postgres fixture never became ready' >&2; exit 1; }
	sleep 1
done

sql=''
for db in $databases; do
	pw="$(cat "$dir/$db.passwd")"
	sql+="CREATE ROLE \"$db\" LOGIN PASSWORD '$pw'; CREATE DATABASE \"$db\" OWNER \"$db\";"
done
podman exec -i -u postgres "$name" psql -v ON_ERROR_STOP=1 -f - <<<"$sql" >/dev/null

trap - EXIT
port="$(podman port "$name" 5432 | head -1)"
port="${port##*:}"
echo "SODA_PG_CONTAINER=$name"
echo "SODA_PG_HOST=127.0.0.1"
echo "SODA_PG_PORT=$port"
echo "SODA_PG_DATABASES=\"$databases\""
echo "SODA_PG_DIR=$dir"
echo "SODA_PG_SUPER_PASSWORD_FILE=$dir/postgres.passwd"
