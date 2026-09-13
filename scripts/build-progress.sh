# Sourced only by build-native.sh. Commands remain in the original shell so
# adding checkpoints cannot disable errexit inside a command wrapper.
progress_helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/build_progress.py"
export SODA_BUILD_START_NS=${SODA_BUILD_START_NS:-$(python3 "$progress_helper" clock)}
progress_label=''
progress_started=0
progress_done() {
  if [[ -n "$progress_label" ]]; then
    python3 "$progress_helper" emit DONE "$progress_label" "$progress_started"
    progress_label=''
  fi
}
progress_next() {
  progress_done
  progress_label="$1"
  progress_started=$(python3 "$progress_helper" clock)
  python3 "$progress_helper" emit START "$progress_label"
}
progress_exit() {
  local result=$?
  trap - EXIT
  if [[ -n "$progress_label" ]]; then
    local outcome=FAILED
    if [[ $result == 0 ]]; then outcome=DONE; fi
    if [[ $result == 130 || $result == 143 ]]; then outcome=CANCELLED; fi
    python3 "$progress_helper" emit "$outcome" "$progress_label" "$progress_started" || true
  fi
  python3 "$progress_helper" finish 'Native payload' "$result" || true
  exit "$result"
}
trap progress_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
