# Invoked only by the root maintenance owner in an exact admitted container.
set -euo pipefail
program=/usr/libexec/soda/project-terminal
stage=/usr/libexec/soda/.project-terminal.stage
action=$1
expected=$2
safe() {
 test ! -L "$1"
 test "$(stat -c %u "$1")" = 0
 test $(( 8#$(stat -c %a "$1") & 8#22 )) = 0
}
for parent in /usr /usr/libexec /usr/libexec/soda; do safe "$parent"; test -d "$parent"; done
safe "$program"
test -f "$program" && test -x "$program"
actual=$(sha256sum "$program"); actual=${actual%% *}
if test "$action" = observe; then printf '%s\n' "$actual"; exit; fi
# A running but disconnected session, reservation, subscription or factory state is
# never taken as idle. Only the proven empty-managed-state transition is admitted.
idle() {
 if test -e /run/soda-terminals; then
  safe /run/soda-terminals
  test -d /run/soda-terminals
  test -z "$(find /run/soda-terminals -mindepth 1 -maxdepth 1 -print -quit)"
 else
  test "${SODA_STOPPED:-0}" = 1
 fi
 if test -e /var/lib/soda/factory; then
  safe /var/lib/soda/factory
  test -d /var/lib/soda/factory
  test -z "$(find /var/lib/soda/factory -mindepth 1 -maxdepth 1 -print -quit)"
 fi
 if test "${SODA_STOPPED:-0}" != 1; then
  test -z "$(systemctl list-units --all --no-legend --plain 'soda-terminal-*' 'soda-factory-*')"
 fi
}
# The guest broker and terminal controls lock this same stable directory. The
# host admission lock also covers factory and identity effects throughout.
if test -d /run/soda-terminals; then
 exec 9</run/soda-terminals
 flock -n -x 9
fi
idle
if test "$action" = idle; then printf '%s\n' "$actual"; exit; fi
if test "$action" = retire; then
 test "$actual" = "$expected"
 old=$3
 new=$4
 [[ "$old" =~ ^[a-f0-9]{64}$ && "$new" =~ ^[a-f0-9]{64}$ ]]
 old_copy=/usr/libexec/soda/.project-terminal.previous-$old
 new_copy=/usr/libexec/soda/.project-terminal.previous-$new
 # Admit the complete declared set before the first unlink. A missing copy is
 # already retired, including after an interrupted previous retirement.
 for hash in "$old" "$new"; do
  copy=/usr/libexec/soda/.project-terminal.previous-$hash
  if test -e "$copy" || test -L "$copy"; then
   safe "$copy"; test -f "$copy"; test -x "$copy"
   test "$(stat -c %h "$copy")" = 1
   test "$(stat -c %s "$copy")" -le 33554432
   saved=$(sha256sum "$copy"); test "${saved%% *}" = "$hash"
  fi
 done
 for copy in "$old_copy" "$new_copy"; do
  if test -e "$copy"; then rm -- "$copy"; fi
  test ! -e "$copy" && test ! -L "$copy"
 done
 sync -f /usr/libexec/soda
 printf '%s\n' "$actual"
 exit
fi
test "$action" = install
test "$actual" = "$3"
previous=/usr/libexec/soda/.project-terminal.previous-$actual
. /etc/os-release
test "$ID:$VERSION_ID" = rocky:10.2
/usr/bin/tmux -V >/dev/null
for term in xterm-256color screen-256color; do /usr/bin/infocmp "$term" >/dev/null; done
if test -e "$stage"; then safe "$stage"; test -d "$stage"; else mkdir -m 0700 "$stage"; fi
test ! -L "$stage/project-terminal"
if test -e "$stage/project-terminal"; then safe "$stage/project-terminal"; test -f "$stage/project-terminal"; rm -- "$stage/project-terminal"; fi
tar --extract --file=- --directory="$stage" --no-same-owner --no-same-permissions project-terminal
safe "$stage/project-terminal"
test -f "$stage/project-terminal"
chmod --reference="$program" "$stage/project-terminal"
chown --reference="$program" "$stage/project-terminal"
new=$(sha256sum "$stage/project-terminal"); test "${new%% *}" = "$expected"
# Release the guest lock only for read-only helper probes; the host exclusion
# remains held. Recheck empty state after reacquiring before any replacement.
flock -u 9
safe /var/lib/soda/accounts
count=0
for marker in /var/lib/soda/accounts/*; do
 test -e "$marker" || continue
 safe "$marker"; test -f "$marker"; test "$(stat -c %s "$marker")" -le 32
 login=${marker##*/}; identity=$(cat "$marker")
 reply=$("$stage/project-terminal" list '' "$login" "$identity" 0 0 10 "$expected" '' '')
 test "$(printf '%s' "$reply" | /usr/bin/jq -c .)" = '{"type":"metadata","terminals":[]}'
 count=$((count+1)); test "$count" -le 256
done
# No accounts means no existing managed state; native loader execution still
# must succeed far enough to return the fixed invalid-arguments response.
if test "$count" = 0; then
 reply=$("$stage/project-terminal" 2>/dev/null) && exit 1 || test "$reply" = '{"type":"closed","reason":"launch_failed"}'
fi
flock -n -x 9
idle
if test -e "$previous"; then
 safe "$previous"; test -f "$previous"
 saved=$(sha256sum "$previous"); test "${saved%% *}" = "$actual"
else
 test ! -L "$stage/previous"
 if test -e "$stage/previous"; then safe "$stage/previous"; test -f "$stage/previous"; fi
 cp --preserve=mode,ownership -- "$program" "$stage/previous"
 sync -f "$stage/previous"
 mv -T -- "$stage/previous" "$previous"
fi
sync -f "$stage/project-terminal"
mv -T -- "$stage/project-terminal" "$program"
sync -f /usr/libexec/soda
observed=$(sha256sum "$program"); test "${observed%% *}" = "$expected"
rmdir -- "$stage"
printf '%s\n' "$expected"
