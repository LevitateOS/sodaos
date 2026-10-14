stage=$1
digest=$2
arch=$3
case "$arch" in x86_64|aarch64) ;; *) exit 1;; esac
test "$(uname -m)" = "$arch"
test -d "$stage" && test ! -L "$stage"
test -f "$stage/archive.tar.gz" && test ! -L "$stage/archive.tar.gz"
printf '%s  %s\n' "$digest" "$stage/archive.tar.gz" | sha256sum -c - >/dev/null
# Stop previews and compiler/platform processes explicitly before maintenance.
for process in /proc/[0-9]*/exe; do
    executable=$(readlink "$process" 2>/dev/null || true)
    case "$executable" in /opt/canlang/*) echo 'Canlang tools are running; stop them before maintenance' >&2; exit 1;; esac
done
if test -e /opt/canlang; then
    test -d /opt/canlang && test ! -L /opt/canlang
    test -f /opt/canlang/tools.json
fi
tar -xzf "$stage/archive.tar.gz" --no-same-owner -C "$stage"
test -x "$stage/canlang/bin/can"
test -x "$stage/canlang/libexec/node"
test -f "$stage/canlang/tools.json"
for name in can can-dev can-platform can-soda; do
    if test -e "/usr/local/bin/$name" || test -L "/usr/local/bin/$name"; then
        test -L "/usr/local/bin/$name"
        test "$(readlink "/usr/local/bin/$name")" = "/opt/canlang/bin/$name"
    fi
done
previous="/opt/.canlang-before-$digest"
test ! -e "$previous" && test ! -L "$previous"
if test -d /opt/canlang; then mv /opt/canlang "$previous"; fi
if ! mv "$stage/canlang" /opt/canlang; then
    if test -d "$previous"; then mv "$previous" /opt/canlang; fi
    exit 1
fi
for name in can can-dev can-platform can-soda; do ln -sfn "/opt/canlang/bin/$name" "/usr/local/bin/$name"; done
# This task-owned staging directory contains only the verified archive.
rm "$stage/archive.tar.gz"
rmdir "$stage"
# Previous tool bytes are retained for explicit operator cleanup. Preview and
# example state lives elsewhere and is never part of tool installation.
