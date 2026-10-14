#!/bin/bash
# Source packaging checks: no image build, container startup or registry access.
set -euo pipefail
cd "$(dirname "$0")/.."
if (( $# != 0 )); then
  echo 'usage: check-docker-packaging.sh' >&2
  exit 2
fi

# Fedora packages float by design; their actual RPM inventory is bound by the
# artifact verifier. All other Hadolint warnings and errors remain failures.
hadolint --ignore DL3041 system/docker/Dockerfile

mkdir -p .artifacts/docker-packaging/client
compose=$(mktemp .artifacts/docker-packaging/compose.XXXXXX.json)
trap 'rm -f -- "$compose"' EXIT
image="ghcr.io/levitateos/sodaos@sha256:$(printf 'a%.0s' {1..64})"
SODA_IMAGE="$image" SODA_PROJECT_SUBNET=10.89.0.0/24 \
  docker --config "$PWD/.artifacts/docker-packaging/client" compose \
  --env-file /dev/null -f system/docker/compose.yaml config --format json > "$compose"

# Admit the parsed deployment contract, including persistent Project storage.
# Extra binds (especially the host root or Docker socket) fail this selection.
jq -e --arg image "$image" '
  (.services | keys) == ["soda"] and
  (.volumes | keys) == ["config", "home", "state"] and
  (.services.soda |
    .image == $image and .platform == "linux/amd64" and
    .privileged == true and .userns_mode == "host" and
    .network_mode == "host" and .cgroup == "private" and
    .command == null and .entrypoint == null and
    .restart == "unless-stopped" and .stop_grace_period == "5m0s" and
    (.tmpfs | sort) == ["/run", "/tmp"] and
    .environment.SODA_PROJECT_SUBNET == "10.89.0.0/24" and
    (.volumes | length) == 3 and
    (.volumes | all(.type == "volume" and .read_only != true)) and
    (.volumes | map({source, target}) | sort_by(.target)) == [
      {source: "config", target: "/etc"},
      {source: "home", target: "/home"},
      {source: "state", target: "/var"}
    ]
  )
' "$compose" > /dev/null || {
  echo 'Docker Compose deployment contract differs; see docs/guides/docker.md' >&2
  exit 1
}
echo 'Dockerfile and Compose packaging checks passed.'
