#!/bin/bash
# SQL locality: product SQLite stays in internal/store.
# See docs/development/go.md. No appliance installation or provider setup.
set -euo pipefail
cd "$(dirname "$0")/.."

violations=""
while IFS= read -r -d '' file; do
  case "$file" in
    ./internal/store/*) continue ;;
  esac
  if grep -nE '"database/sql"|sql\.Open[[:space:]]*\(|`(SELECT|INSERT|UPDATE|DELETE|CREATE TABLE|ALTER TABLE|PRAGMA)[[:space:]]|"[[:space:]]*(SELECT|INSERT|UPDATE|DELETE|CREATE TABLE|ALTER TABLE|PRAGMA)[[:space:]]' "$file" >/tmp/soda-sql-locality.hits 2>/dev/null; then
    while IFS= read -r hit; do
      violations+="${file}:${hit}"$'\n'
    done </tmp/soda-sql-locality.hits
  fi
done < <(find ./internal ./cmd ./tools ./appliance -name '*.go' -print0)

rm -f /tmp/soda-sql-locality.hits

if [ -n "$violations" ]; then
  printf '%s' "$violations"
  printf 'SQL locality failed: database/sql, sql.Open, and SQL verb literals are allowed only in internal/store (docs/development/go.md).\n'
  exit 1
fi
printf 'SQL locality passed: no database/sql usage outside internal/store.\n'
