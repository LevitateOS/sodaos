#!/bin/sh
# Run in the project's workload-example checkout as its owner.
# First create native secret soda-example-db from a restricted local password
# file: podman secret create soda-example-db /absolute/private/password-file
# The password is never passed in argv or printed by Compose.
# Use `check` to finish readiness/SQL observations without another up/build.
set -eu
: "${SODA_NATIVE_VALIDATE:?Explicit native validation authorization required}"
[ "$SODA_NATIVE_VALIDATE" = soda-test ]
[ "$#" -le 1 ] || { echo 'usage: workloads.sh [start|check]' >&2; exit 2; }
case "${1-start}" in
    start)
        podman secret inspect soda-example-db >/dev/null
        podman compose up -d --build
        ;;
    check) ;; # Explicit read-only resume; never replay an uncertain up/build.
    *) echo 'usage: workloads.sh [start|check]' >&2; exit 2 ;;
esac
podman compose ps
# A detached native start is not application readiness. Retry only these reads,
# with a finite limit; do not rebuild, restart or replace anything on failure.
attempt=0
until curl --fail --silent --show-error --max-time 2 http://127.0.0.1:8000/; do
    attempt=$((attempt + 1))
    [ "$attempt" -lt 30 ] || { echo 'HTTP readiness failed' >&2; exit 1; }
    sleep 1
done
attempt=0
until podman compose exec -T database pg_isready -U developer -d soda_example; do
    attempt=$((attempt + 1))
    [ "$attempt" -lt 30 ] || { echo 'PostgreSQL readiness failed' >&2; exit 1; }
    sleep 1
done
podman compose exec -T database psql -X -v ON_ERROR_STOP=1 -U developer -d soda_example -c 'select current_database();'
printf 'Also connect to ports 8000 and 5432 from Bob and a real developer client.\n'
# No automatic down -v or other data deletion.
