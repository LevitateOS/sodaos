# This socket belongs to THIS project's runtime, never the host engine.
# Project-owner/wheel membership is required to administer shared workloads.
export CONTAINER_HOST=unix:///run/soda-podman/podman.sock
export PODMAN_COMPOSE_PROVIDER=/usr/local/bin/podman-compose
