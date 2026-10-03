# candidate-storage.sh — one candidate storage configuration (D1).
#
# Heavy worker state lives on /home, never on the small root filesystem.
# Setup sources this file; the worker configuration records the resolved
# root and the controller enforces that BuildHome/Runtime sit under it.
# Engine state (rootless Podman/Buildah storage) follows the worker HOME,
# Go module/build caches and Playwright browsers live under it too, and
# setup scratch uses the scratch directory instead of /tmp.
#
# Overrides are for development and tests only; qualification uses these
# defaults.
: "${SODA_CANDIDATE_ROOT:=/home/soda-candidate}"
: "${SODA_CANDIDATE_HOME:=$SODA_CANDIDATE_ROOT/home}"
: "${SODA_CANDIDATE_RUN:=$SODA_CANDIDATE_ROOT/run}"
: "${SODA_CANDIDATE_SCRATCH:=$SODA_CANDIDATE_ROOT/scratch}"

# Legacy root-backed locations. Setup migrates from them engine-aware and
# preserves them; nothing deletes legacy state (see D2 retention).
SODA_LEGACY_HOME="/var/lib/soda-candidate-home"
SODA_LEGACY_RUN="/var/lib/soda-candidate-run"
