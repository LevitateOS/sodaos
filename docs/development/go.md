# Go ownership

Navigate as **package + concern file**. A package owns one job you can say in
one sentence. `internal/` is a shallow domain hierarchy: top-level names are
major Soda concepts (`project`, `host`, `web`, `release`, `tailnet`,
`runners`, `store`, `forgejo`, `installer`); subpackages are genuine
subordinate boundaries (`host/project`, `web/api`, `release/deliver`).
Appliance entrypoints live in `cmd/`; support tools live in `tools/`. This
guide owns Go package placement and house style. Product/security boundaries
remain in [architecture](../architecture/overview.md).

## Muscle memory

| If you are adding… | It goes in… |
| --- | --- |
| Project identity, lifecycle/key/OS types, creation profile, validation | `project` — the one canonical definition; never duplicate these DTOs |
| OAuth / login / session / provider / me keys | `web/auth` |
| Product HTTP/WS (environments, spaces, terminal, lifecycle, runners, tailnet settings, pages) | `web/api` |
| Dashboard mux root, namespace gate, `web.New` wiring only | `web` (`Server` wires `Auth` + `API`; no handlers, no aliases) |
| Unix client + thin daemon mux/admission | `host` (`client.go`, `daemon.go`; decode straight into `project` types — no translators) |
| Privileged project env (create/inspect/lifecycle/keys/profiles/os) | `host/project` (package `project`; executes on domain `project` types) |
| Privileged terminal attach | `host/terminal` (package `terminal`) |
| Tailnet companion container runtime | `host/tailnet` (package `tailnet`) |
| Install phase on Linux | `installer/<phase>_linux.go` |
| Build-time installed path consts | `platform` (`legacy.go` / `vendor.go` build-tag pair) |
| Runner composition + native operator identity | `runners` (`operator.go`) |
| Release build primitives / image assembly / qualification / delivery | `release/build`, `release/image`, `release/qualify`, `release/deliver` |
| Domain policy / Forgejo client / SQLite | `tailnet` / `forgejo` / `store` |

Hard size rule: prefer production files under 400 LOC; do not grow a production
`.go` file past 500 LOC—split by noun or phase instead.

`cmd/soda-dashboard` enters through `web` (plus `config`/`store`/`avatar`
for process startup only). `cmd/soda-host` enters through `host`
(plus `tailnet`). `web.Server` constructs `auth`/`api`; `host.Daemon`
wires the three executors. These are the only facades; do not add
forwarding packages or compatibility shims for moved code. The host
Client may re-export `host/terminal` wire types so `web` never imports
the privileged terminal executor directly.

## Placement decision tree

1. SQL / schema / row mapping → `store`
2. Project identity / lifecycle / key / OS types or validation → `project`
   (one canonical definition referenced by HTTP, host, executors, store,
   qualify — never a parallel DTO)
3. Build / image / qualify / deliver → the matching `release/` subpackage —
   never `web`/`host`
4. Outside VM/evidence checks → `acceptance` (release support harness)
5. Tiny reusable primitive → its own small package (`strictjson`, `filelock`, …)
6. External product HTTP client → `forgejo`
7. Domain policy / wire types (no session, no root socket) → domain package
8. Privileged project / terminal / companion → `host/project` /
   `host/terminal` / `host/tailnet` (wire through thin `host.Daemon`)
9. OAuth / session → `web/auth`; product HTTP → `web/api` (wire through
   thin `web.Server`)
10. Still ambiguous → rules in the domain package; HTTP admits and calls;
    privileged packages execute and confirm. Do not invent a fourth package.

**Naming.** Package declaration matches the directory leaf
(`host/project` → `package project`). When an executor imports its domain
twin (`project`, `tailnet`), alias the domain import inside the executor
and alias the executor import at the facade (`projectexec`,
`tailnetexec`) — do not invent concatenated package names. Filename =
concrete product concern (`lifecycle.go`, `enrollment.go`,
`control.go`). Ban new `helpers.go`, `utils.go`, or vague
`management.go`. Matching concern names across layers (`terminal.go`,
`runners.go`, `lifecycle.go`) are intentional. Never resurrect a retired
top-level package path (`internal/projectos`, `internal/linuxhost`,
`internal/installlayout`, `internal/webapp`, `internal/webauth`,
`internal/nativebuild`, `internal/nativequalification`,
`internal/nativefinalization`, `internal/releasedelivery`,
`internal/appliancerelease`, `internal/hostproject`,
`internal/hostterminal`, `internal/hosttailnet`) —
`internal/archcheck` fails the build if they return.

## Ownership map

| Package | Owns | Does not own | Look here first |
| --- | --- | --- | --- |
| `acceptance` | Outside support VM/evidence checks | Product HTTP, SQLite, install UX | `tools/soda-acceptance` |
| `archcheck` | Package-topology boundary tests | Product behavior | `arch_test.go` |
| `avatar` | Robot SVG render | Identity lookup | `avatar.go` |
| `config` | Dashboard/operator JSON load | Secrets at rest, migrations | `config.go` |
| `filelock` | Advisory file locks | Business policy | `filelock.go` |
| `forgejo` | Forgejo HTTP API client | Forgejo DB, upstream rules | `client.go` |
| `host` | Unix client + thin Daemon mux/admission | Project/terminal/companion guts | `client.go`, `daemon.go`, `project.go` |
| `host/project` | Privileged project env execution | HTTP admission, Tailnet policy, terminal attach | `create.go`, `lifecycle.go` |
| `host/terminal` | Privileged terminal attach | Project create | `service.go`, `types.go` |
| `host/tailnet` | Companion container runtime | Tailnet policy (`tailnet`) | `companion.go`, `runtime.go` |
| `installer` | Console-to-CoreOS install adapter | Host daemon, SQLite | phase `*_linux.go` files |
| `platform` | Build-time installed path consts | Scratch `/run` paths | `legacy.go` / `vendor.go` |
| `project` | Canonical project domain: profile, lifecycle/key/OS types, validation | I/O, HTTP, privileged execution | `types.go`, `profile.go`, `project.go` |
| `release` | Release-construction overview only | Any build/qualify/publish logic | `doc.go` |
| `release/build` | Shared build primitives | Qualify/sign/install UX | `production.go`, `oci.go` |
| `release/deliver` | Payload model, signing, publication | Building images | `payload.go`, `publish.go`, `finalize.go` |
| `release/image` | Host image assemble/prepare | Qualification, publish | `build.go`, `prepare.go` |
| `release/qualify` | Artifact admission + guest state | Image build, update client | `inputs.go`, `state.go` |
| `runners` | Local CI runner composition + operator identity | Forgejo Actions UI | `model.go`, `native.go`, `operator.go` |
| `store` | SQLite schema + row ops | HTTP, host execute | `store.go`, `migrations.go` |
| `strictjson` | Bounded single-object JSON decode | Domain validation | `decode.go` |
| `tailnet` | Tailnet policy/identity/`Control` | Companion launch | `control.go`, `policy.go` |
| `testoci` | Inert OCI test fixtures | Production images | `fixture.go` |
| `web` | HTTP root mux + wiring only | Business handlers | `server.go` |
| `web/api` | Product API + settings + terminal WS | OAuth state machine | concern files |
| `web/auth` | OAuth/session/provider/login | Environment/terminal APIs | `service.go`, `provider.go` |

## Cross-cut owners

| Concern | Meaning / policy | Privileged execute | HTTP |
| --- | --- | --- | --- |
| Project | `project` (types + validation) | `host/project` | `web/api` (+ thin `host` daemon routes) |
| Tailnet | `tailnet` | `host/tailnet` | `web/api` (+ thin `host` daemon routes) |
| Terminal | access/lifetime in `web/api` + store | `host/terminal` | `web/api` |
| Runners | `runners` incl. `operator.go` | runner cmds via `host` daemon routes | `web/api` |

## Release debug map

| Symptom | Open first | Then |
| --- | --- | --- |
| Image/candidate will not build | `release/build`, `tools/soda-build` | `release/image` |
| Media/assemble wrong | `release/image` | `installer` phases |
| Guest/fixture will not qualify | `release/qualify` | `acceptance` |
| Sign/publish | `release/deliver` | `tools/soda-release` |

Do not casually start in `release/qualify` for a build failure.

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
| Tests | Assert exported contracts; reopen SQL/schema guts only inside `store` tests |

## Hard invariants (persistence)

1. **SQL locality.** Product SQLite lives only in `store`. Callers use exported
   methods and `store.ErrNotFound`.
2. **One schema owner.** Migrations and `SchemaVersion()` change only in `store`.
3. **One implementation per concern.** No dual paths or parallel query styles.
4. **Layer duties stay layered.** HTTP admits; privileged packages execute;
   domain packages own meaning; `store` owns rows.
5. **No SQL escape hatch.** Do not export `*sql.DB` or generic `Exec`/`Query`.

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
`tailnet`/`runners`/`store`, dual shims for old god method sets, and rewriting
`scripts/` into `internal/`.

## Enforcement

`scripts/check-sql-locality.sh` fails when sources outside `internal/store`
import `database/sql`, call `sql.Open`, or embed SQL verb literals. It runs from
`bun run check:source`.

`internal/archcheck` fails when a retired package name returns, when
`web/aliases.go` is recreated, or when production code crosses a banned
ownership edge (transport reaching executors or release, release reaching
transport or the daemon, executors reaching up or out, leaves reaching up,
the dashboard binary reaching the daemon directly). Run it with
`go test ./internal/archcheck/`; keep its rules in sync with the tables
above whenever ownership genuinely moves.
