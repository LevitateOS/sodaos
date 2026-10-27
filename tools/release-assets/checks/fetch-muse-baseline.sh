#!/bin/bash
# Maintained integration check: the DEFAULT soda-fetch-muse entrypoint must
# refuse stale pins and unknown upstreams, and reuse current cached bytes.
# Uses a local fixture channel (SODA_MUSE_CHANNEL_URL) plus the real binary;
# no credentials, model calls, or upstream downloads. Fails loudly on any
# unexpected exit code or mutated fixture.
set -u
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
BIN="$ROOT/target/debug/soda-fetch-muse"
WORK="$(mktemp -d /home/vince/soda-fetch-check.XXXXXX)"
PORT=18931
fail=0

note() { echo "fetch-check: $1"; }
die() { echo "fetch-check FAIL: $1"; fail=1; }

"$ROOT/target/debug/soda-fetch-muse" --help >/dev/null 2>&1 || {
  cargo build --offline -p soda-release-assets --bin soda-fetch-muse >/dev/null 2>&1 ||
    { die "cannot build soda-fetch-muse"; }
}

# Fixture world NEW (9.9.99) with a tiny payload whose digest we control.
printf 'staged-bytes-current' > "$WORK/payload-new"
NEW_SUM="$(sha256sum "$WORK/payload-new" | cut -d' ' -f1)"
NEW_SIZE="$(stat -c%s "$WORK/payload-new")"
cat > "$WORK/channel.json" <<EOF
{"channel":"muse-stable","version":"9.9.99-NEW.1","manifest_url":"http://127.0.0.1:$PORT/manifest.json","urgency":"none","notification_text":"","state":"public","min_version":null}
EOF
cat > "$WORK/manifest.json" <<EOF
{"version":"9.9.99-NEW.1","checksum_algorithm":"sha256","artifacts":{"x86_linux":{"url":"http://127.0.0.1:$PORT/download?file=muse-x86-linux","checksum":"$NEW_SUM","size":$NEW_SIZE}}}
EOF
# Stale pin (9.9.98) with its own cached bytes.
printf 'staged-bytes-old' > "$WORK/payload-old"
OLD_SUM="$(sha256sum "$WORK/payload-old" | cut -d' ' -f1)"
cat > "$WORK/stale-pin.json" <<EOF
{"version":"9.9.98-OLD.1","artifacts":{"x86_64":{"file":"muse-x86-linux","sha256":"$OLD_SUM","size":17}}}
EOF
cp "$WORK/payload-old" "$WORK/out-stale"
BEFORE="$(sha256sum "$WORK/out-stale" | cut -d' ' -f1)"
# Current pin + matching cached bytes (reuse case, no download expected).
cat > "$WORK/current-pin.json" <<EOF
{"version":"9.9.99-NEW.1","artifacts":{"x86_64":{"file":"muse-x86-linux","sha256":"$NEW_SUM","size":$NEW_SIZE}}}
EOF
cp "$WORK/payload-new" "$WORK/out-current"
MTIME_BEFORE="$(stat -c%Y "$WORK/out-current")"
# Unknown upstream: well-formed HTTP, malformed channel document.
printf 'not-json{' > "$WORK/bad.json"

python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$WORK" >/dev/null 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null' EXIT
sleep 1

export SODA_MUSE_CHANNEL_URL="http://127.0.0.1:$PORT/channel.json"
OUT=$("$BIN" --arch x86_64 --manifest "$WORK/stale-pin.json" --out "$WORK/out-stale" 2>&1); RC=$?
[ "$RC" = 2 ] || die "stale pin exit=$RC (want 2): $OUT"
[ "$(sha256sum "$WORK/out-stale" | cut -d' ' -f1)" = "$BEFORE" ] || die "stale run mutated dest"
echo "$OUT" | grep -q "9.9.98-OLD.1" || die "stale message names no drift: $OUT"
note "stale cached pin refused (exit 2), dest untouched"

OUT=$("$BIN" --arch x86_64 --manifest "$WORK/current-pin.json" --out "$WORK/out-current" 2>&1); RC=$?
[ "$RC" = 0 ] || die "current pin exit=$RC (want 0): $OUT"
[ "$(stat -c%Y "$WORK/out-current")" = "$MTIME_BEFORE" ] || die "current run re-downloaded"
echo "$OUT" | grep -q "admitted current" || die "no frozen tuple printed: $OUT"
note "current pin reused without download (exit 0)"

export SODA_MUSE_CHANNEL_URL="http://127.0.0.1:$PORT/bad.json"
OUT=$("$BIN" --arch x86_64 --manifest "$WORK/current-pin.json" --out "$WORK/out-current" 2>&1); RC=$?
[ "$RC" = 2 ] || die "unknown upstream exit=$RC (want 2): $OUT"
note "unknown upstream refused (exit 2)"

export SODA_MUSE_CHANNEL_URL="http://127.0.0.1:9/none"
OUT=$("$BIN" --arch x86_64 --manifest "$WORK/current-pin.json" --out "$WORK/out-current" 2>&1); RC=$?
[ "$RC" = 1 ] || die "dead upstream exit=$RC (want 1): $OUT"
note "dead upstream is operational failure (exit 1), claims nothing"

kill $SRV 2>/dev/null; trap - EXIT
rm -rf "$WORK"
[ "$fail" = 0 ] && note "PASS" || die "FAILED"
exit "$fail"
