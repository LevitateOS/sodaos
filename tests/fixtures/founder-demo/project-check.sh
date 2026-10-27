#!/bin/sh
set -eu
umask 077
for tool in can can-dev can-platform can-soda node git; do
  command -v "$tool" >/dev/null
done
can help check >/dev/null
can help compile >/dev/null
can-soda inventory > "$TMPDIR/northstar-provider-inventory.json"
node -e 'const fs=require("node:fs"); const p=process.env.HOME+"/northstar-write-check"; fs.writeFileSync(p,"ready",{flag:"wx",mode:0o600}); fs.unlinkSync(p)'
if [ -z "${SODA_VERIFICATION_RUN_ID:-}" ]; then
  if [ -f Northstar.can ]; then
    exec node "$SODA_SNAPSHOT/northstar-check.mjs" "$SODA_SNAPSHOT/northstar-check.json" --source-only
  fi
  echo 'Northstar preparation: installed tools and private writable storage ready; application outcomes not checked.'
  exit 0
fi
exec node "$SODA_SNAPSHOT/northstar-check.mjs" "$SODA_SNAPSHOT/northstar-check.json"
