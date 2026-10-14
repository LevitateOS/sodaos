#!/usr/bin/env bash
# A protected-main push can reuse the exact commit's completed qualification tag.
# PRs, tag pushes and explicit reruns always execute the source suites.
set -euo pipefail
reused=false
if [[ ${GITHUB_EVENT_NAME:-} == push && ${GITHUB_REF:-} == refs/heads/main && ${GITHUB_RUN_ATTEMPT:-} == 1 && ${GITHUB_SHA:-} =~ ^[0-9a-f]{40}$ ]]; then
  tag="ci/soda-main-${GITHUB_SHA:0:12}"
  result=$(mktemp "${TMPDIR:-$HOME}/soda-ci-reuse.XXXXXXXX")
  trap 'rm -f "$result"' EXIT
  if gh api "repos/$GITHUB_REPOSITORY/actions/workflows/ci.yml/runs?head_sha=$GITHUB_SHA&branch=$tag&event=push&per_page=1" > "$result"; then
    if jq -e --arg sha "$GITHUB_SHA" --arg tag "$tag" '
      .workflow_runs | length == 1 and
      (.[0] | (.id | type == "number" and . > 0) and .head_sha == $sha and .head_branch == $tag and
        .event == "push" and .path == ".github/workflows/ci.yml" and
        .status == "completed" and .conclusion == "success")
    ' "$result" >/dev/null; then
      reused=true
      printf 'Reusing source CI run %s for commit %s\n' "$(jq -r '.workflow_runs[0].id' "$result")" "$GITHUB_SHA"
    fi
  fi
fi
printf 'reused=%s\n' "$reused" >> "$GITHUB_OUTPUT"
