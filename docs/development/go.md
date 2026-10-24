# Go ownership

Navigate as **package + concern file**. A package owns one job you can say in
one sentence. `internal/` is a shallow domain hierarchy: top-level names are
major Soda concepts (`project`, `host`, `web`, `tailnet`,
`store`, `forgejo`, `factory`, `identity`); subpackages are genuine
subordinate boundaries (`web/api`, `forgejo/publish`, `identity/client`,
`factory/control`). Release construction, installation and privileged
execution live in Rust (`lib/`, `cmd/`). Appliance entrypoints live in
`cmd/`; support tools live in `tools/`. This
guide owns Go package placement and house style. Product/security boundaries
remain in [architecture](../architecture/overview.md).

## Muscle memory

| If you are adding… | It goes in… |
| --- | --- |
| Project identity, lifecycle/key/OS types, creation profile, validation | `project` — the one canonical definition; never duplicate these DTOs |
| Admitted application artifact/image/data references, administrator grant and native observations | `application` — pure wire/domain records; Canlang owns semantics and bindings |
| Supervised factory runs: status/stop/reconcile coordinator plus pause/resume/retry/takeover lifecycle controls and readiness intake/reconciliation/assessment | `factory/control`; `cmd/soda-factory` is a thin client of its private operator endpoint |
| Run identity, outcome, display bindings and operator/control command records | `factory` — pure domain types and validation; no runtime or SQL |
| Provider connection, delegation and execution lease types | `identity` — canonical domain records; no runtime or SQL |
| Serialized provider custody and enrollment | `cmd/soda-identity` wires private service/runtime; `identity` keeps domain records |
| Muse native device enrollment and immutable CLI credentials | `cmd/soda-identity` providers (muse) — installed upstream CLI; no Meta refresh service |
| Codex app-server protocol | `cmd/soda-identity` providers (codex) — verified CLI protocol and private credential files |
| Trusted broker Unix transport | `identity/client` — service callers only; never mounted in workspaces |
| Native extension identity / profile / me keys | `web/auth` |
| Product HTTP/WS (environments, spaces, factory views, terminal, lifecycle, tailnet settings, pages) | `web/api` |
| Dashboard mux root, namespace gate, `web.New` wiring only | `web` (`Server` wires `Auth` + `API`; no handlers, no aliases) |
| Unix client for the privileged host daemon | `host` (`client.go`, `terminal_client.go`, `factory_client.go`; decode straight into `project` types — no translators) |
| Privileged project env (create/inspect/lifecycle/keys/profiles/os) | `lib/host` (Rust; executes on admitted requests) — Go `host` only calls it |
| Privileged terminal attach | `lib/host` terminal lifecycle (Rust); Go `host/terminal_client.go` calls it |
| Narrow factory Git publication validation | `forgejo/publish` (unprivileged; validates verified candidate bundles, executes persisted conditional branch/PR operations) |
| Supervised factory run orchestration | `factory/control` run supervisor (durable receipts; typed `host` clients + broker handshake) |
| Tailnet companion container runtime | `lib/host` tailnet lifecycle (Rust); Go `host/tailnet.go` calls it |
| Install phase on Linux | `cmd/soda-install` (Rust) |
| Release build primitives / image assembly / delivery | `lib/soda-release-build`, `lib/soda-release-image`, `lib/soda-release-deliver` (Rust) |
| Domain policy / Forgejo client / PostgreSQL store | `tailnet` / `forgejo` / `store` |

Keep files cohesive. Split by responsibility when it improves the change; line
counts alone do not justify refactoring or block working code.

`cmd/soda-dashboard` enters through `web` (plus `config`/`store`/`avatar`
for process startup only). The `soda-host` daemon is the Rust
`lib/host` binary; `host` keeps the Go Unix client surface.
`web.Server` constructs `auth`/`api` and the
`factory/control` coordinator. `cmd/soda-factory` is a thin Unix client of the
coordinator's private operator endpoint and keeps no database. Do not add
forwarding packages or compatibility shims for moved code. The host
client types live in `host` itself so `web` never imports
the privileged executor directly.

## Placement decision tree

1. SQL / schema / row mapping → `store`
2. Project identity / lifecycle / key / OS types or validation → `project`
   (one canonical definition referenced by HTTP, host, executors and store —
   never a parallel DTO)
3. Build / image / deliver → the matching `lib/soda-release-*` Rust
   crate — never `web`/`host`
4. Outside VM/evidence checks → `acceptance` (release support harness)
5. Tiny reusable primitive → its own small package (`strictjson`, `filelock`, …)
6. External product HTTP client → `forgejo`
7. Domain policy / wire types (no session, no root socket) → domain package
8. Privileged project / terminal / companion → `lib/host` (Rust);
   Go callers wire through the thin `host` Unix clients
9. Native browser authority → `web/auth`; product HTTP → `web/api` (wire through
   thin `web.Server`)
10. Still ambiguous → rules in the domain package; HTTP admits and calls;
    privileged packages execute and confirm. Do not invent a fourth package.

**Naming.** Package declaration matches the directory leaf
(`web/api` → `package api`). Filename =
concrete product concern (`lifecycle.go`, `enrollment.go`,
`control.go`). Package names stay leaf-simple; do not invent
concatenated package names. Ban new `helpers.go`, `utils.go`, or vague
`management.go`. Matching concern names across layers (`terminal.go`,
`lifecycle.go`) are intentional. Never resurrect a retired
top-level package path (`internal/projectos`, `internal/linuxhost`,
`internal/installlayout`, `internal/webapp`, `internal/webauth`,
`internal/nativebuild`, `internal/nativequalification`,
`internal/nativefinalization`, `internal/releasedelivery`,
`internal/appliancerelease`, `internal/hostproject`,
`internal/hostterminal`, `internal/hosttailnet`,
`internal/release`, `internal/installer`, `internal/runners`).

## Ownership map

| Package | Owns | Does not own | Look here first |
| --- | --- | --- | --- |
| `acceptance` | Outside support VM/evidence checks | Product HTTP, PostgreSQL store, install UX | `tools/acceptance` |
| `application` | Exact admitted application references, administrator grant, replacement and observation validation | Canlang descriptor/provider semantics, runtime, HTTP, SQL | `reference.go`, `administration.go` |
| `avatar` | Robot SVG render | Identity lookup | `avatar.go` |
| `config` | Dashboard/operator JSON load | Secrets at rest, migrations | `config.go` |
| `filelock` | Advisory file locks | Business policy | `filelock.go` |
| `factory` | Supervised run identity/outcome and operator command records (references canonical `identity` leases and `project` IDs) | Forgejo collaboration, runtime execution, PostgreSQL store | `types.go`, `run.go` |
| `identity` | Connection, delegation, execution lease records and validation | HTTP, native execution, SQL | `types.go` |
| `cmd/soda-identity` | Provider-specific account custody, enrollment, revocation and reconciliation | Browser authority, factory publication | concern files |
| `cmd/soda-identity` providers | Muse native enrollment and immutable credential validation | Soda grants, custom upstream refresh | `providers/muse` |
| `cmd/soda-identity` providers | Codex app-server protocol and isolated CLI credential cache | Soda delegation policy | `providers/codex` |
| `identity/client` | Private broker Unix client | Browser authority, credential persistence | `client.go` |
| `forgejo` | Forgejo HTTP API client | Forgejo DB, upstream rules | `client.go` |
| `host` | Unix clients for the privileged host daemon | Project/terminal/companion execution guts | `client.go`, `terminal_client.go`, `factory_client.go` |
| `lib/host` (Rust) | Privileged project/terminal/companion execution | HTTP admission, Tailnet policy | `src/` concern modules |
| `cmd/soda-install` (Rust) | Console-to-CoreOS install adapter | Host daemon, PostgreSQL store | concern files |
| `project` | Canonical project domain: profile, lifecycle/key/OS types, validation | I/O, HTTP, privileged execution | `types.go`, `profile.go`, `project.go` |
| `lib/soda-release-*` (Rust) | Release build primitives, image assembly, delivery | Qualify/sign/install UX | per-crate concern files |
| `store` | PostgreSQL schema + row ops, including the factory run/command ledger | HTTP, host execute | `store.go`, `schema.go`, `factory.go` |
| `strictjson` | Bounded single-object JSON decode | Domain validation | `decode.go` |
| `tailnet` | Tailnet policy/identity/`Control` | Companion launch | `control.go`, `policy.go` |
| `web` | HTTP root mux + wiring only | Business handlers | `server.go` |
| `web/api` | Product API + settings + terminal WS | OAuth state machine | concern files |
| `web/auth` | Native extension identity, profiles and account keys | Environment/terminal APIs | `service.go`, `provider.go` |

## Cross-cut owners

### Application administration placement

`application` mirrors the mechanical native request/observation contract in
`lib/host/src/application/wire.rs`. `store` owns current application references,
immutable revision history and observation CAS. A replacement preserves the
administrator, application name/identity, owner team, private data, origin and
port. Unknown native outcomes remain a nullable observation; desired image
references never imply readiness. Applications reserve distinct hostnames,
private data identities and loopback ports in the shared store.

An operator admits the initial administrator grant. Existing records are read
and changed only for that stable Forgejo identity. App-local users and bootstrap
credentials remain in Canlang; they do not grant Soda administration. The Rust
host executes against admitted image labels and preprovisioned private data.
`host.Client` sends the canonical native request and validates confirmed
observations. `web/api` admits live native-extension actors: operator registration,
install and image replacement, or the stored administrator's inspect/start/stop.
It records unknown state before native effects, serializes local actions and
uses store CAS to publish matching outcomes. Spaces shows subordinate app
controls; navigation never installs or starts an app. Local HTTP and browser
tests do not constitute installed-appliance qualification. Domain records have
no transport, runtime, provider configuration or database dependencies.

### Factory interface placement

The [target factory interfaces](../architecture/factory-interfaces.md) reuse these
owners. The tables above describe existing placement; apply the following narrow
dependency changes with the implementation,
not as an invitation to reorganize `internal/`:

- `factory` owns policy, acceptance, assessment, command, attempt, assignment,
  run and usage records. Reference canonical `identity` records; use Project IDs
  rather than inventing a second Project DTO. `project` owns preparation and
  checkout/lifecycle records. `store` is still the only SQL owner.
- `factory/control` is the run supervisor: it composes `store`, narrow
  typed `host` client/`identity/client` interfaces and canonical Project
  records; privileged execution lives in the Rust host, not in Go.
- `web` wires the coordinator and its explicit lifecycle. `web/api` admits calls
  into `factory/control`; it still cannot call native executor packages directly.
  `cmd/soda-dashboard` continues to enter through `web`. `cmd/soda-factory`
  controls address the coordinator's private operator endpoint, not a second
  local controller/database.
- `host` supplies typed Unix-client methods for the Rust host daemon. Fixed
  Project preparation/launch orchestration executes in `lib/host`, using
  managed terminal execution and the private identity client for the
  broker handshake. Native receipts are protected mechanical files; neither
  executor imports `store` or decides factory policy. Canonical
  factory/Project/identity request records cross the facade directly,
  without forwarding DTO packages.
- `identity` owns acquisition identity, lease and process-binding types;
  `cmd/soda-identity` enforces execution uniqueness and credential custody over
  its private store. Broker runtime adapters use the fixed host process
  contract instead of disposable-container or hard-coded worker assumptions.

Keep construction in facades and behavior in the existing concern packages.
No new coordinator daemon, parallel runtime abstraction or generic persistence
API is required by this placement.

### Current cross-cut map

| Concern | Meaning / policy | Privileged execute | HTTP |
| --- | --- | --- | --- |
| Project | `project` (types + validation) | `lib/host` (Rust; Go `host` calls it) | `web/api` |
| Tailnet | `tailnet` | `lib/host` (Rust; Go `host` calls it) | `web/api` |
| Terminal | access/lifetime in `web/api` + store | `lib/host` (Rust; Go `host` calls it) | `web/api` |

## Release debug map

| Symptom | Open first | Then |
| --- | --- | --- |
| Image/candidate will not build | `lib/soda-release-build`, `lib/soda-release-tools` (`soda-build`) | `lib/soda-release-image` |
| Media/assemble wrong | `lib/soda-release-image` | `cmd/soda-install` |
| Sign/publish | `lib/soda-release-deliver` | explicit grant (no operator CLI) |


## `scripts/` Go tests

[`scripts/*.go`](../../scripts/) are Forgejo presentation/branding/template checks
(`package scripts`). They are not appliance runtime and not a third product
library root. Own them with Forgejo UI docs; do not treat them as `internal/`.

## House style (new code; migrate on touch)

| Rule | Choice |
| --- | --- |
| Package doc | One sentence: the job and what it is *not* |
| Constructors | Only `New` (in-memory), `Open` (live resource), `Load` (decode/validate path/bytes) |
| Errors | Wrap causes with `%w`; package sentinel only when an in-tree `errors.Is` caller already exists |
| Logging | Libraries never log; long-lived processes use `slog`; one-shot CLIs print once to stderr and exit |
| CLI | `flag.NewFlagSet` → `run() error` → main prints/exits |
| Absence from store | Use `errors.Is(err, store.ErrNotFound)`. Do not import `database/sql` outside `store`. |
| File size | Prefer &lt;400 LOC; split before a production file exceeds 500 LOC |
| Tests | Assert exported contracts; reopen SQL/schema guts only inside `store` tests, except the named existing integration fixtures below |

## Hard invariants (persistence)

1. **SQL locality.** Product PostgreSQL lives only in `store`. Callers use exported
   methods and `store.ErrNotFound`.
2. **One schema owner.** Migrations and `SchemaVersion()` change only in `store`.
3. **One implementation per concern.** No dual paths or parallel query styles.
4. **Layer duties stay layered.** HTTP admits; privileged packages execute;
   domain packages own meaning; `store` owns rows.
5. **No SQL escape hatch.** Do not export `*sql.DB` or generic `Exec`/`Query`.
6. **Native parameters.** Product PostgreSQL statements use `$1…$n` in argument
   order, directly through the existing database or transaction. Keep values
   parameterized and admitted dynamic identifiers local to `store`. The external
   Forgejo SQLite seed fixture retains SQLite placeholders.

## Definition of done (any Go change)

- Touched package has an accurate `// Package …` line when it exports more than one symbol.
- New constructors use only `New` / `Open` / `Load`.
- Handlers and privileged ops land in the muscle-memory table packages—not on thin facades.
- No production file grown past 500 LOC without a split.
- Errors wrap causes; no new sentinel unless a real `errors.Is` caller exists in-tree.
- No library logging; CLI/daemon logging matches the table above.
- You can explain the package's one job in ≤10 seconds in review.

## Explicit non-goals

Hexagonal/ports-everywhere, ORM/sqlc mandate, re-litigating the `release/`,
`host/`, `web/` hierarchies, interface DI graphs, repository wrappers around
`*Store`, micro-packages (`pages`, `forgejo_keys`, `hostd`), splitting
`tailnet`/`store`, dual shims for old god method sets, and rewriting
`scripts/` into `internal/`.

## Enforcement

`scripts/check-sql-locality.sh` fails when sources outside `internal/store`
import `database/sql`, call `sql.Open`, or embed SQL verb literals. It runs from
`bun run check:source`.

The gate retains exact-file exceptions for existing integration fixtures that
need state unavailable through the public Store API. These operate on disposable
test databases; production SQL and new test files remain subject to the gate.
When changing these fixtures, use existing Store methods where they express the
same scenario. Do not add a production SQL escape hatch just to seed a test.

| Existing fixture | Required direct database state |
| --- | --- |
| `internal/factory/control/dispatch_expiry_test.go` | Exhaust recorded roots before dispatch closure and explicit retry |
| `internal/factory/control/dispatch_accounting_test.go` | Remove settlement/checkpoint metadata to exercise retained elapsed-time recovery and clamping without waiting for real deadlines |
| `internal/factory/control/dispatch_restore_helpers_test.go` | Change a retained sponsorship receipt to challenge recovery without a public corruption writer |
| `internal/factory/control/dispatch_preparation_test.go` | Consume an attempt, expire admission receipts, and temporarily hide a policy table to exercise operational failure |
| `internal/factory/control/dispatch_source_preparation_test.go` | Expire a ready source receipt during admission and confirm that no admission rows remain |
| `internal/factory/control/traversal_test.go` | Remove admission headers so their independent cap does not obscure dispatch traversal limits |
| `internal/factory/control/readiness_sweep_test.go` | Isolate the cycle bound and inspect durable cursor state across acceptance changes |
| `internal/factory/control/staged_seed_test.go` | Seed an external Forgejo SQLite fixture whose dependency edges have no REST writer |
| `internal/factory/control/serving_priority_starvation_test.go` | Remove emitted admission headers to isolate priority traversal, fresh members and cursor bounds |
| `internal/factory/control/review_carried_history_test.go` | Seed and corrupt attributable assignment/run/view history to challenge carried-finding retirement |
| `internal/web/api/factory_completion_budget_test.go` | Seed completion/reassessment prefixes and observe budget/cursor rollback with scripted native RPC evidence |
| `internal/web/api/factory_journey_readback_test.go` | Seed and corrupt persisted assignment/publication/completion records for registered-HTTP projection, order and refusal |
| `internal/factory/control/mcp_settle_pg_test.go` | Seed a persisted review assignment to isolate terminal receipt settlement from admission and dispatch |
| `internal/factory/control/intent_plan_choice_test.go` | Seed canonical merge evidence and alter generation/task bindings to challenge exact accepted outcome coverage |
| `internal/factory/control/stall_serve_test.go` | Seed and corrupt protected repair evidence, compare immutable spending, and challenge atomic diagnosis/draft/feed recovery with complete native RPC fixtures |
| `internal/web/api/factory_mcp_bound_test.go` | Seed exact active worker/cause/preparation tuples and corrupt attributed read projections through authenticated MCP transport |
| `internal/web/api/factory_feed_route_test.go` | Seed finished assignment history to check feed enrichment without recreating a full execution lifecycle |
| `internal/factory/control/delivery_admission_test.go` | Seed a merged completion event so delivered intent admits a scoped Test Pilot; no public writer records a standalone completion outside the merge flow |
| `internal/factory/control/app_verification_test.go` | Seed a finished assignment with its merged completion to challenge application verification; no public writer records a standalone finished assignment outside dispatch |
| `internal/web/api/factory_mcp_live_test.go` | Seed an assigned assignment, ready preparation, open dispatch and active run to activate a staged worker MCP capability; no public writer records this exact bound set outside the dispatch admission flow under test |

The separate `internal/acceptance/workload_access.go` also
has an exact exemption: its `psql` statements target disposable workload
databases, not the product Store.

## Function complexity

Use [complexity reports](complexity.md) when they help a concrete refactor.
`bash scripts/check-complexity.sh --language go` reports changed production
functions. Scores and file lengths do not block commits or require exceptions.
