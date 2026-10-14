# Invoked in a private mount/PID/user namespace only after exact stopped-root
# inspection. No init or user workloads run. Preserve, never hide retained state.
set -euo pipefail
for path in /run; do
 test ! -L "$path"; test -d "$path"; test "$(stat -c %u "$path")" = 0
 test $((8#$(stat -c %a "$path") & 8#22)) = 0
done
if test -e /run/soda-terminals; then
 test ! -L /run/soda-terminals; test -d /run/soda-terminals
 test -z "$(find /run/soda-terminals -mindepth 1 -maxdepth 1 -print -quit)"
fi
mount -t tmpfs -o mode=0755,nosuid,nodev tmpfs /run
mkdir -m 0755 /run/soda-terminals
SODA_STOPPED=1
