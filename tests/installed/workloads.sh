#!/bin/sh
# Run in the project's workload-example checkout as its owner.
# First create native secret soda-example-db from a restricted local password
# file: podman secret create soda-example-db /absolute/private/password-file
# The password is never passed in argv or printed by Compose.
set -eu
: "${SODA_NATIVE_VALIDATE:?Explicit native validation authorization required}"
podman secret inspect soda-example-db >/dev/null
podman compose up -d --build
podman compose ps
curl --fail http://127.0.0.1:8000/
podman compose exec -T database psql -U developer -d soda_example -c 'select current_database();'
printf 'Also connect to ports 8000 and 5432 from Bob and a real developer client.\n'
# No automatic down -v or other data deletion.
