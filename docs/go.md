# Go ownership

Navigate as **package + concern file**. A package owns one job you can say in
one sentence. Types are wiring facades or domain owners—not dumping grounds for
unrelated product domains. Flat `internal/*` stays. Appliance entrypoints live in
`cmd/`; support tools live in `tools/`. This guide owns Go package placement and
house style. Product/security boundaries remain in [architecture](architecture.md).

## Muscle memory

| If you are adding… | It goes in… |
| --- | --- |
| OAuth / login / session / provider / me keys | `webauth` |
| Product HTTP/WS (environments, spaces, terminal, lifecycle, runners, tailnet settings, pages) | `webapp` |
| Mux root, namespace gate, `web.New` wiring only | `web` |
| Wire DTOs + Unix client + thin daemon mux/admission | `host` |
| Privileged project env (create/inspect/lifecycle/keys/profiles/runners/os) | `hostproject` |
| Privileged terminal attach | `hostterminal` |
| Tailnet companion container runtime | `hosttailnet` |
| Install phase on Linux | `installer/<phase>_linux.go` |
| Domain policy / Forgejo client / SQLite | `tailnet` / `forgejo` / `store` |

Hard size rule: prefer production files under 400 LOC; do not grow a production
`.go` file past 500 LOC—split by noun or phase instead.

`cmd/soda-dashboard` imports only `web`. `cmd/soda-host` imports only `host`.
Facades construct `webauth`/`webapp` and `hostproject`/`hostterminal`/`hosttailnet`.

## Placement decision tree

1. SQL / schema / row mapping → `store`
2. Build / image / qualify / deliver / accept → matching release package
   (`nativebuild`, `hostimage`, `nativequalification`, `releasedelivery`,
   `acceptance`, …) — never `web`/`host`
3. Tiny reusable primitive → its own small package (`strictjson`, `filelock`, …)
4. External product HTTP client → `forgejo`
5. Domain policy / wire types (no session, no root socket) → domain package
6. Privileged project / terminal / companion → `hostproject` / `hostterminal` /
   `hosttailnet` (wire through thin `host.Daemon`)
7. OAuth / session → `webauth`; product HTTP → `webapp` (wire through thin `web`)
8. Still ambiguous → rules in the domain package; HTTP admits and calls;
   privileged packages execute and confirm. Do not invent a fourth package.

**Naming.** Package = durable owner. Filename = concrete product concern
(`lifecycle.go`, `enrollment.go`, `control.go`). Ban new `helpers.go`,
`utils.go`, or vague `management.go`. Matching concern names across layers
(`terminal.go`, `runners.go`, `lifecycle.go`) are intentional.

## Ownership map

| Package | Owns | Does not own | Look here first |
| --- | --- | --- | --- |
| `acceptance` | Outside support VM/evidence checks | Product HTTP, SQLite, install UX | `tools/soda-acceptance` |
| `appliancerelease` | Immutable payload metadata | Signing, upgrade client | `payload.go` |
| `avatar` | Robot SVG render | Identity lookup | `avatar.go` |
| `config` | Dashboard/operator JSON load | Secrets at rest, migrations | `config.go` |
| `filelock` | Advisory file locks | Business policy | `filelock.go` |
| `forgejo` | Forgejo HTTP API client | Forgejo DB, upstream rules | `client.go` |
| `host` | Client, DTOs, thin Daemon mux/admission | Project/terminal/companion guts | `client.go`, `daemon.go` |
| `hostproject` | Privileged project env ops | Terminal PTY, companion | `lifecycle.go`, create/inspect |
| `hostterminal` | Privileged terminal attach | Project create | `terminal.go` |
| `hosttailnet` | Companion container runtime | Tailnet policy (`tailnet`) | `companion.go` |
| `hostimage` | Host image assemble/prepare | Qualification, publish | `build.go`, `prepare.go` |
| `installer` | Console-to-CoreOS install adapter | Host daemon, SQLite | phase `*_linux.go` files |
| `installlayout` | Build-time installed path consts | Scratch `/run` paths | `legacy.go` / `vendor.go` |
| `linuxhost` | Operator Linux identity boundary | Runner lifecycle | `operator.go` |
| `nativebuild` | Shared build primitives | Qualify/sign/install UX | `production.go`, `oci.go` |
| `nativefinalization` | Protected final signing connection | Build execution | `finalize.go` |
| `nativequalification` | Artifact admission + guest state | Image build, update client | `inputs.go`, `state.go` |
| `projectos` | Creation profile identity | Image registry/runtime | `profile.go` |
| `releasedelivery` | Sigstore/release binding | Building images | `model.go`, `publish.go` |
| `runners` | Local CI runner composition | Forgejo Actions UI | `model.go`, `native.go` |
| `store` | SQLite schema + row ops | HTTP, host execute | `store.go`, `migrations.go` |
| `strictjson` | Bounded single-object JSON decode | Domain validation | `decode.go` |
| `tailnet` | Tailnet policy/identity/`Control` | Companion launch | `control.go`, `policy.go` |
| `testoci` | Inert OCI test fixtures | Production images | `fixture.go` |
| `web` | HTTP root mux + wiring only | Business handlers | `server.go` |
| `webauth` | OAuth/session/provider/login | Environment/terminal APIs | `auth.go`, `provider.go` |
| `webapp` | Product API + settings + terminal WS | OAuth state machine | concern files |

## Cross-cut owners

| Concern | Meaning / policy | Privileged execute | HTTP |
| --- | --- | --- | --- |
| Tailnet | `tailnet` | `hosttailnet` | `webapp` (+ thin `host` daemon routes) |
| Terminal | access/lifetime in `webapp` + store | `hostterminal` | `webapp` |
| Runners | `runners` | `hostproject` / runner cmds | `webapp` |
| Lifecycle | project unit semantics in `hostproject` | `hostproject` | `webapp` |

## Release debug map

| Symptom | Open first | Then |
| --- | --- | --- |
| Image/candidate will not build | `nativebuild`, `tools/soda-build` | `hostimage` |
| Media/assemble wrong | `hostimage` | `installer` phases |
| Guest/fixture will not qualify | `nativequalification` | `acceptance` |
| Sign/publish | `releasedelivery`, `nativefinalization` | `tools/soda-release` |

Do not casually start in `nativequalification` for a build failure.

## `scripts/` Go tests

[`scripts/*.go`](../scripts/) are Forgejo presentation/branding/template checks
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

Hexagonal/ports-everywhere, ORM/sqlc mandate, `internal/appliance` vs
`internal/release` directory churn, interface DI graphs, repository wrappers
around `*Store`, micro-packages (`pages`, `forgejo_keys`, `hostd`), splitting
`tailnet`/`runners`/`store`, dual shims for old god method sets, and rewriting
`scripts/` into `internal/`.

## Enforcement

`scripts/check-sql-locality.sh` fails when sources outside `internal/store`
import `database/sql`, call `sql.Open`, or embed SQL verb literals. It runs from
`bun run check:source`.
