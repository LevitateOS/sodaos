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

Store custody follows the consumer's operating-system authority. The native
`soda-identity` unit sets `SODA_PROVIDERS_ROOT=/home/soda-identity/providers`
inside its existing private, durable home. Its `soda-identity` user owns admission,
selection, locking and active references there under the unit's `UMask=0077`.
The root host and root updater retain `/var/lib/soda/providers`; the broker gains
no access to that root-owned store. Both use the same resolver and verified
official-upstream admission. Broker enrollment and lease bindings carry no store
path into the host's independent, exact harness binding.

For initial operator preparation, run the existing `soda-provider-update refresh
--provider codex --json` as `soda-identity`, with the same store environment and
private umask. An empty store must complete actual admission; copying a selection
record or credentials is not a substitute. New-session due checks continue to
refresh that owner's store, and stale, unknown or unverifiable selections retain
their existing refusals.

- Lane B (`cmd/soda-identity/src/providers/`, host daemon/dbackend/terminal
  resolution): resolve paths at session creation via `resolve_new`; bind
  in-flight sessions/enrollments/leases and recovery to `resolve_existing` +
  `acquire`; never cache one hash forever.
- Lane C (Project launcher/maintenance access, systemd service + timer at
  5-minute cadence, packaging, bootstrap/baseline inputs): read-only
  staging path for Projects (no write access to the shared store),
  launch-time due-check, installed scheduling fixtures under isolated roots.
- T06 Factory dispatch adoption LANDED as V6 (commits `3c9c2cfc` Go +
  `d779b446` Rust, 2026-10-26): new packets record the live staged pin as
  their adopted bundle (`Assignment.AdoptedHarness`, `RunAdmission.Tool`);
  recovery binds the adopted bytes with no current-pin oracle; retries
  continue their assignment's bundle; rows predating proof keep the strict
  legacy rule and fail recovery closed. Host reservations snapshot current
  (`resolve_current`) or bind the recorded sidecar (`bind_recorded`) with a
  digest/version coherence check and release the ref after confirmed
  retirement. Stale saved policy versions no longer refuse fresh admission.
  Remaining: cross-process restart end-to-end (admit A, update to B,
  restart, recover A) is T07 combined-integration work; R01/C01 stay
  Codex-owned.
- V7 correction (2026-10-26, UP-20261026-V7-START): a retry with a NEW
  execution ID is a new provider session and adopts the CURRENT pin, never
  the assignment's adopted bundle (`retryHarnessPin` deleted; only the SAME
  already-captured execution/session recovers its recorded adopted bytes).
  Host `Factory::launch` classifies via recorded authority first: existing
  receipts never consult current or re-check the original execution
  deadline; only genuinely new runs require current pin + current deadline.
  `dbackend` reserve preserves the durable binding + lifetime ref under
  unconfirmed unit state (fail-closed default) and releases only after
  confirmed retirement through the single owner. Joined acceptance red to
  green against real `Factory::launch` + private-state files
  (`oracle_v7_recorded_receipt_survives_current_advance`,
  `oracle_v7_recorded_receipt_survives_own_deadline_expiry`); Go retry
  conformance probe green; suites match baseline (4 pre-existing PG-fixture
  failures unchanged); `cargo check -p soda-host` clean.
