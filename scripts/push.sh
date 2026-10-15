#!/usr/bin/env bash
# Qualify and fast-forward the maintained branch within one bounded operation.
set -euo pipefail
budget=${SODA_PUSH_TIMEOUT_SECONDS:-300}
[[ $budget =~ ^[0-9]+$ ]] && (( budget >= 2 && budget <= 300 )) || {
  echo 'Push budget must be between 2 and 300 seconds.' >&2; exit 2;
}
if [[ ${SODA_PUSH_BOUNDED:-} != 1 ]]; then
  if timeout --signal=TERM --kill-after=1s "$((budget - 1))s" env SODA_PUSH_BOUNDED=1 bash "$0" "$@"; then
    exit 0
  else
    result=$?
    if (( result == 124 || result == 137 )); then
      echo 'Push budget expired. Inspect origin before retrying if publication had started.' >&2
    fi
    exit "$result"
  fi
fi
[[ $# == 0 ]]
cd "$(git rev-parse --show-toplevel)"
[[ -z $(git status --porcelain --untracked-files=normal) ]] || {
  echo 'Commit source changes before qualification.' >&2; exit 1;
}
candidate=$(git rev-parse HEAD)
branch=main
repository=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
tag="ci/soda-main-$candidate"
mkdir -p .artifacts/push-gate
receipt=$(mktemp .artifacts/push-gate/push.XXXXXXXX)
trap 'rm -f "$receipt"' EXIT
if git rev-parse --verify "refs/tags/$tag" >/dev/null 2>&1; then
  [[ $(git rev-parse "$tag^{commit}") == "$candidate" ]]
else
  git tag "$tag" "$candidate"
fi
git push origin "refs/tags/$tag"
printf 'Qualifying %s; total push budget %ss.\n' "$candidate" "$budget"
while (( SECONDS < budget - 15 )); do
  gh api "repos/$repository/actions/workflows/ci.yml/runs?head_sha=$candidate&branch=$tag&event=push&per_page=1" > "$receipt"
  if jq -e --arg sha "$candidate" --arg tag "$tag" '
    .workflow_runs | length == 1 and
    (.[0] | .head_sha == $sha and .head_branch == $tag and .event == "push" and .path == ".github/workflows/ci.yml")
  ' "$receipt" >/dev/null; then
    run=$(jq -r '.workflow_runs[0].id' "$receipt")
    url=$(jq -r '.workflow_runs[0].html_url' "$receipt")
    if [[ $(jq -r '.workflow_runs[0].status' "$receipt") == completed ]]; then
      [[ $(jq -r '.workflow_runs[0].conclusion' "$receipt") == success ]] || {
        printf 'Qualification failed: %s\n' "$url" >&2; exit 1;
      }
      gh api "repos/$repository/commits/$candidate/check-runs?filter=all&per_page=100" > "$receipt"
      jq -e --arg sha "$candidate" --arg run "/actions/runs/$run/" '
        any(.check_runs[]; .name == "CI" and .head_sha == $sha and .app.id == 15368 and
          .status == "completed" and .conclusion == "success" and (.details_url | contains($run)))
      ' "$receipt" >/dev/null
      remote=$(git ls-remote origin "refs/heads/$branch" | cut -f1)
      [[ $remote =~ ^[0-9a-f]{40}$ ]]
      git merge-base --is-ancestor "$remote" "$candidate"
      printf 'Publishing qualified commit after %ss: %s\n' "$SECONDS" "$url"
      git push origin "$candidate:refs/heads/$branch"
      [[ $(git ls-remote origin "refs/heads/$branch" | cut -f1) == "$candidate" ]]
      printf 'Published %s in %ss.\n' "$candidate" "$SECONDS"
      exit 0
    fi
    # Stop promptly when a required child has already failed, rather than
    # spending the remaining publication budget waiting for sibling jobs.
    if gh api "repos/$repository/actions/runs/$run/jobs?per_page=100" --jq \
      'any(.jobs[]; .status == "completed" and (.conclusion == "failure" or .conclusion == "cancelled" or .conclusion == "timed_out"))' | grep -qx true; then
      printf 'Qualification failed: %s\n' "$url" >&2
      exit 1
    fi
  fi
  sleep 3
done
echo 'Qualification did not finish within the push budget; maintained branch was not advanced.' >&2
exit 1
