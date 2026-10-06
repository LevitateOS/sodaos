#!/bin/bash
# No-Python gate: the language port ends with zero Python in tracked source.
# Fails on (1) tracked *.py files, (2) tracked files with a Python shebang,
# (3) tracked Go/Rust/shell/TS/package.json code that executes the Python
# interpreter. Deliberately out of scope: prose mentions (*.md), image
# package installs (*Containerfile*, *Dockerfile* - third-party tooling such
# as podman-compose keeps its own interpreter), JSON data files other than
# package.json scripts, and guest-toolchain fixtures that merely name
# "python3" as data without an argv shape.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0

found=$(git ls-files '*.py' || true)
if [ -n "$found" ]; then
  printf 'tracked Python files remain:\n%s\n' "$found"
  fail=1
fi

shebangs=$(git ls-files | while IFS= read -r f; do
  [ -f "$f" ] || continue
  line=$(head -c 128 -- "$f" 2>/dev/null | tr -d '\000' | head -n 1 || true)
  case "$line" in '#!'*python*) printf '%s\n' "$f" ;; esac
done)
if [ -n "$shebangs" ]; then
  printf 'Python shebang programs remain:\n%s\n' "$shebangs"
  fail=1
fi

# The gate's own enforcement text names the banned shapes, so it must not
# scan itself; every other tracked file in the classes below is checked.
self='scripts/check-no-python.sh'
gofiles=$(git ls-files '*.go' | grep -v "^$self\$" || true)
if [ -n "$gofiles" ]; then
  # shellcheck disable=SC2086
  hits=$(grep -nE 'exec\.Command(Context)?\s*\(\s*"(/usr/bin/)?python|LookPath\s*\(\s*"(/usr/bin/)?python|syscall\.Exec\([^)]*"(/usr/bin/)?python|"(/usr/bin/)?python[0-9.]*"\s*,' $gofiles || true)
  if [ -n "$hits" ]; then
    printf 'Go code executes the Python interpreter:\n%s\n' "$hits"
    fail=1
  fi
fi

rsfiles=$(git ls-files '*.rs' | grep -v "^$self\$" || true)
if [ -n "$rsfiles" ]; then
  # shellcheck disable=SC2086
  hits=$(grep -nE 'Command::new\s*\(\s*"(/usr/bin/)?python|"(/usr/bin/)?python[0-9.]*"\s*,' $rsfiles || true)
  if [ -n "$hits" ]; then
    printf 'Rust code executes the Python interpreter:\n%s\n' "$hits"
    fail=1
  fi
fi

# Extensionless programs (appliance/bin, project helpers, hooks) can hide
# invocations behind any name, so they join *.sh. Containerfiles stay out:
# image package installs are third-party tooling, not SodaOS Python.
shfiles=$(git ls-files '*.sh' 'appliance/bin/*' 'system/project/rootfs/usr/libexec/soda/*' '.githooks/*' | grep -v "^$self\$" || true)
tsfiles=$(git ls-files '*.ts' '*.js' '*.mjs' 'package.json' | grep -v "^$self\$" || true)
token='(^|[][:space:];'"'"'"`=([|&{,])(sudo[[:space:]]+)?(/usr/bin/)?python[0-9.]*([][:space:];,'"'"'"`)}{&]|$)'
scripthits=""
for class in "$shfiles" "$tsfiles"; do
  [ -n "$class" ] || continue
  # shellcheck disable=SC2086
  hits=$(printf '%s\n' "$class" | xargs grep -nE "$token" -- || true)
  [ -n "$hits" ] && scripthits+="$hits"$'\n'
done
if [ -n "$scripthits" ]; then
  printf 'shell/TS/package code invokes the Python interpreter:\n%s' "$scripthits"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  printf 'no-python gate failed.\n'
  exit 1
fi
printf 'no-python gate passed: zero tracked Python.\n'
