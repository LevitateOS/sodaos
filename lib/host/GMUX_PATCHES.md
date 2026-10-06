# GMUX integrator patches (PR26 daemon-mux skeleton)

The skeleton is four self-contained modules plus a smoke harness. Nothing
is wired into the crate yet: this file is the exact cutover checklist.

Files (all new, no existing file touched):

- `src/gmux_backend.rs` — `ExecBackend` trait + `StubBackend` (501).
- `src/gmux_admission.rs` — request validators, mutation gate, terminal
  stream cap, `SO_PEERCRED` / `SO_PEERPIDFD` attestation helpers.
- `src/gmux_routes.rs` — all 33 routes, subsystem error mapping, response
  envelopes, websocket 101 + RFC 6455 accept key.
- `src/gmux_server.rs` — Unix-socket HTTP server + `systemd_listener()`.
- `tests/gmux_smoke.rs` — 32 tests, includes the modules via `#[path]`.

Sources ported from: `internal/host/daemon.go` (mux, gate, dispatch),
`identity.go`, `tailnet.go`, `terminal.go`, `terminal/service.go`
(websocket admission, limits), `terminal/muse_socket_linux.go` (peer
credentials), `cmd/soda-host/main.go` (socket activation); server loop
pattern from `rust/soda-identity/src/http.rs`. There is no
`rust/soda-host/src/daemon.rs`; the Go daemon is the contract.

## 1. Module wiring (`src/lib.rs`)

Append, keeping the existing alphabetical order (`gmux_*` sorts after
`domain` and before `json`):

```rust
pub mod gmux_admission;
pub mod gmux_backend;
pub mod gmux_routes;
pub mod gmux_server;
```

All cross-module paths use `crate::gmux_*`, so the modules compile
unchanged both under `#[path]` (this smoke test) and under `lib.rs`.
After wiring, delete the four `#[path]` includes in
`tests/gmux_smoke.rs` and `use soda_host::gmux_*` instead (or leave the
test as-is; both keep passing).

## 2. Stub -> real adapter

Implement `ExecBackend` once over the texec + pfactory executors:

```rust
pub struct DaemonBackend { /* project/terminal/tailnet/identity handles */ }
impl gmux_backend::ExecBackend for DaemonBackend { /* ... */ }
```

Per-method contract notes (deviations break wire parity):

- Bodies arrive as raw JSON bytes and MUST be strict-decoded by the
  backend (unknown fields rejected), replacing `strictjson.Decode`.
  The mux enforces only transport admission + body limits.
- `/account` MUST return exactly `{"ok":true}` on success (Go's
  `dispatchMutation` synthesizes this; there is no backend value).
- `factory_harness(body, image)`: stamp `image` onto the harness pin
  before validation, like Go (`pin.Image = d.Config.Image`, then
  `pin.Validate()`); validation failure maps to `Unavailable`.
- `identity_action("launch")` is never called (launch has its own
  method). Inside `identity_action`, decode the delivery envelope and
  reproduce Go's `identityOperation` routing: `Factory` leases go to
  the factory-run callbacks (validate/stop/finish; anything else is
  `Denied`), `muse-project`-scoped Muse leases to the muse runtime,
  everything else to the terminal identity path. Unknown actions MUST
  return `Denied`, never execute.
- `tailnet("project"/"policy")`: reproduce Go's container-identity
  fencing — resolve the project container BEFORE the observation and
  re-resolve AFTER; mismatch or error maps to `Internal` (mux: 502).
  `policy` MUST reject any action other than `inspect` with `Invalid`.
- `terminal_accept` = Go's `attachLauncher` assertion + stream
  registration that must succeed BEFORE the 101 is emitted.
  `pump_terminal` owns the upgraded stream to session end: first text
  frame (<= `TERMINAL_REQUEST_LIMIT` bytes, strict `TerminalRequest`
  JSON), expiry deadline, launch, bidirectional pump with
  `TERMINAL_FRAME_LIMIT` reads. The mux holds the `TerminalSlot` for
  the pump lifetime; the backend MUST NOT re-count streams.
- `BackendError::Unimplemented` MUST NEVER be returned by the real
  backend. Every subsystem maps it to 501, which exists only to make
  skeleton gaps visible.
- `DaemonConfig` booleans fold Go's nil-runtime checks: set
  `tailnet_management=false` when tailnet is disabled
  (`d.Tailnet == nil` -> 503), `terminal_available=false` with no
  terminal service (503), `identity_available=false` likewise (400).

## 3. Route mounts

None required: `gmux_routes::dispatch` already serves the complete
table (`ROUTE_TABLE`, 33 entries: 5 identity, 6 tailnet, 1 terminal,
8 project, 5 prepare, 8 factory). Do not add mounts; add a test if
you add a route.

## 4. Binary main

`soda-host` currently has no Rust binary. Create it (suggested
`rust/soda-host/src/main.rs` with `[[bin]]`, or a `bins/` entry per
repo layout) along these lines:

```rust
fn main() -> anyhow::Result<()> /* or plain Result<(), Box<dyn Error>> */ {
    // 1. Parse -config / -tailnet-action / -project exactly like
    //    cmd/soda-host/main.go (same flags, same exit statuses:
    //    0 ok, 78 tailnet-preparation, 1 other).
    // 2. Load + validate host.json (port LoadConfig/validateRuntimeConfig;
    //    unknown fields rejected).
    // 3. let listener = gmux_server::systemd_listener()?; // fd 3, root-only
    // 4. let server = Server::new(Arc::new(DaemonBackend::new(cfg)?), config);
    // 5. On SIGTERM/SIGINT: server.shutdown(); drain server.inflight();
    //    close terminal streams (backend-owned); 5s shutdown horizon.
    // 6. server.serve(&listener);
}
```

Per-request deadlines are a BACKEND responsibility: Go applies a
3-minute native timeout, 20s tailnet timeout and 12h terminal
lifetime. The mux applies socket read timeouts (5s head, 30s body)
only. Wire the Go horizons into the adapter (backend contexts), not
into `gmux_server`.

The muse-launch unixpacket listener (`OpenMuseListener`/`ServeMuse`,
0666 + empty-dir gate) is a SEPARATE listener the integrator wires
with `gmux_admission::{muse_peer, close_pidfd}`; it is not part of
the HTTP mux and must not share its socket.

## 5. Go files to delete at cutover

Delete only when the Rust binary serves the systemd socket AND the
Go tests for the deleted surface are retired in the same commit:

Server half (replaced by this mux):

- `internal/host/daemon.go` (mux, gate, dispatch, config load)
- `internal/host/identity.go` — server half only (`identityLaunch`,
  `identityOperation`, `factoryIdentityOperation`,
  `identityHandler`); the `Client.Identity*` methods below move with §6
- `internal/host/tailnet.go` — server half only (through
  `tailnetHandler`); `tailnetCall` + `Client.Tailnet*` move with §6
- `internal/host/terminal.go` — `terminalHandler`/`CloseTerminals`
  (keep the type re-exports + `ValidTerminalName` until clients port)
- `internal/host/muse.go` (`museRuntime`, `OpenMuseListener`,
  `ServeMuse`) once the Rust muse-launch listener lands
- `cmd/soda-host/main.go`, `cmd/soda-host/launch.go` (replaced by §4)

NOT this cutover (owned by sibling lanes / later PRs):

- `internal/host/client.go`, `terminal_client.go`, `factory_client.go`
  and the `Client.*` methods in `identity.go`/`tailnet.go`/`project.go`
  — the Go Unix CLIENT surface; other Go callers still use it.
- `internal/host/project/*`, `terminal/*`, `tailnet/*`, `publish/*` —
  executor implementations; texec/pfactory lanes delete their own.
- `internal/host/access_keys.go`, `lifecycle.go`, `os.go`,
  `prepare.go`, `profiles.go`, `project.go` — thin facades over the
  executors; delete with the executor cutovers.

## 6. Cargo dependencies

NONE missing. The skeleton uses only `std` + `libc` (already in
`soda-host/Cargo.toml`). SHA-1 and base64 for the websocket accept
key are inline in `gmux_routes.rs` (~90 lines, RFC 6455 test vector
pinned); do NOT add `sha1`/`base64` crates for this — the crate is
deliberately dependency-free and the handshake has no other crypto
needs. If a future backend needs JSON, reuse the crate's own `json`
module, not serde.

## 7. Deliberate deviations from Go (reviewer checklist)

- `Unimplemented` -> 501 in every subsystem (Go has no 501 path).
  Real backends never return it, so it vanishes at cutover.
- Chunked request bodies and duplicate Content-Length are rejected
  with 400 (Go's net/http would accept chunked; no in-repo client
  sends it; matches `soda-identity/src/http.rs`).
- Unknown `/identity/*` actions are denied in the mux with 409. Go
  forwards them to `Terminal.Identity`, which denies with the same
  409 — same wire behavior, one less backend call.
- Pre-upgrade terminal backend failures map to 503 (Go only has
  post-upgrade launch failures, which close the connection; 503 is
  the closest pre-upgrade signal in `terminalHandler`).
- Over-limit heads are drained (bounded 1 MiB) before the 431 so the
  client observes the status instead of a reset. Go closes without
  draining; the status code is the same.
- The mutation gate is a blocking mutex, not Go's
  context-cancellable channel acquisition. Cancelled waiters in Go
  leave the queue; here the per-request backend deadline (§4) bounds
  the wait instead. `try_acquire` exists for non-blocking callers.
- `peer_cred` is attested on demand, not per connection: Go gates
  nothing on UID/GID for the main socket (filesystem authorization),
  and neither does the mux. The helper exists for the muse-launch
  listener (§4) and future backend policy.
