# Resolve the pinned Ruff binary. Sourced by the Python quality-gate scripts.
# Prefer PATH / python3 -m ruff at the pinned version; otherwise uvx with that pin.
RUFF_PIN=0.16.6
export NO_COLOR=1

_ruff_version() {
  "$@" version 2>/dev/null | awk '{print $2}'
}

ruff_cmd=()
if command -v ruff >/dev/null 2>&1; then
  got=$(_ruff_version ruff)
  if [ "$got" != "$RUFF_PIN" ]; then
    printf 'ruff %s required (found %s); install with: python3 -m pip install -r requirements-ruff.txt\n' "$RUFF_PIN" "${got:-unknown}"
    exit 1
  fi
  ruff_cmd=(ruff)
elif python3 -m ruff version >/dev/null 2>&1; then
  got=$(_ruff_version python3 -m ruff)
  if [ "$got" != "$RUFF_PIN" ]; then
    printf 'ruff %s required (found %s); install with: python3 -m pip install -r requirements-ruff.txt\n' "$RUFF_PIN" "${got:-unknown}"
    exit 1
  fi
  ruff_cmd=(python3 -m ruff)
elif command -v uvx >/dev/null 2>&1; then
  ruff_cmd=(uvx "ruff@${RUFF_PIN}")
else
  printf 'ruff %s is not installed; run: python3 -m pip install -r requirements-ruff.txt\n' "$RUFF_PIN"
  exit 1
fi
