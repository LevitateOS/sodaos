# Provider freshness shared contract (UP-20261026-v1 T00, coordinator-frozen)

Frozen 2026-10-26. Lane A implements bodies; lanes B and C consume only the
exported names below. Coordinator owns `Cargo.toml` files, workspace
membership and `Cargo.lock`; lanes request dependency changes instead of
editing them. No unit tests (`#[test]`/`#[cfg(test)]` banned); verify through
integration drivers against real systems.

## Crate `soda-provider-tools` (`lib/provider-tools/`)

- `Provider`: `Codex | Muse`; `as_str`, `parse`, `metadata_url(channel)`,
  `executable_relpath` (`bin/codex`, `bin/muse`).
- `ReleaseIdentity`: `{provider, version, digest (hex sha256), size, artifact_url}`.
- `Freshness`: `Current | Stale{latest} | Unknown{reason} | UpdateFailed{reason}`.
- `Selection`: `{provider, release, checked_at_secs, admitted_at_secs, freshness}`;
  `bundle_dir(root)`, `executable(root)`.
- `Status`: `{root, now_secs, entries}`.
- `Error`: `Denied | Failed | Uncertain | Blocked`; `blocks_new_work()` is true
  only for `Blocked`.
- `config`: `store_root()` (`SODA_PROVIDERS_ROOT` else
  `/var/lib/soda/providers`), `channel()` (`SODA_PROVIDERS_CHANNEL` else
  `stable`), `check_ttl()` (default 300s).
- `discover::resolve_upstream(provider)`: live official metadata only. Codex:
  `https://api.github.com/repos/openai/codex/releases/latest`, native asset
  `codex-package-x86_64-unknown-linux-musl.tar.zst` (+ SHA256SUMS where
  published). Muse: `https://api.meta.ai/muse-code/channels/muse-stable` ->
  validated manifest -> artifact URL/checksum/size.
- `discover::probe_executable(provider, exe)`: no-model, no-auth behavior
  probes (`--help`, provider protocol handshake, Muse model-profile resolve).
- `refresh::refresh(provider)`: check, admit verified bundle when absent,
  atomically replace `current-<provider>.json`. Unchanged bytes: no
  re-download. Every mutation (unchanged advance, same-digest repair,
  new-digest publish, stale-block) is compare-and-swapped against the exact
  observed base: a concurrent newer winner is retained, never overwritten.
  Damaged destinations quarantine-swap under the store lock. Failure:
  previous selection untouched, timestamps not refreshed;
  newer-known-but-unadmitted returns `Blocked`.
- `refresh::fetch_verified(release, staging)`: bounded download, sha256 + size
  check, archive safety (no symlink/absolute escape, native arch enforced).
  `.tar.zst` via system `zstd` (`/usr/bin/zstd` present); no new crates.
- `resolve::resolve_new(provider)`: refresh when due, else current selection
  executable; `Blocked` when stale/failed-newer-known. For NEW work only.
- `resolve::resolve_existing(provider, digest)`: exact retained executable;
  fails closed. For existing runs and recovery.
- `resolve::acquire/release`: `<root>/refs/<token>.json` lifetime guards.
- `store::read_selection/write_selection (atomic temp+rename+fsync)/evaluate/cleanup`.
  `cleanup` retains current, previous and referenced bundles; returns bytes
  reclaimed.

## Command `soda-provider-update` (`cmd/soda-provider-update/`)

`check | refresh | status [--provider codex|muse] [--json]`.
Exits: 0 success/current, 1 operational error, 2 known-stale/failed blocks
new provider work.

## Consumers

- Lane B (`cmd/soda-identity/src/providers/`, host daemon/dbackend/terminal
  resolution): resolve paths at session creation via `resolve_new`; bind
  in-flight sessions/enrollments/leases and recovery to `resolve_existing` +
  `acquire`; never cache one hash forever.
- Lane C (Project launcher/maintenance access, systemd service + timer at
  5-minute cadence, packaging, bootstrap/baseline inputs): read-only
  staging path for Projects (no write access to the shared store),
  launch-time due-check, installed scheduling fixtures under isolated roots.
- T06 Factory dispatch adoption is coordinator-owned after writer
  reconciliation; lanes B/C must not edit shared Go Factory dispatch files.
  Persisted cold-recovery Factory integration is required by this contract,
  not yet integrated: the accepted execution reference lifetime covers live
  host executions only, and cold recovery still resolves through
  `resolve_existing` fail-closed until T06 lands.
