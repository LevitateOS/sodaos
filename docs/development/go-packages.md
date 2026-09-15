# Soda Go package convention

One codebase, one organizational philosophy. A developer who understands one
`internal/` package should be able to navigate another without relearning it.
The question is never "where did someone happen to put this" but "according
to the convention, this should be here".

This document describes the convention as implemented. It extends the
[Go ownership guide](go.md) (which owns package placement, SQL locality and
house style); where the two overlap, `go.md` wins.

## Ownership principle

A package owns one job statable in one sentence. That sentence lives in the
`// Package` clause of the package's primary file (`client.go`, `daemon.go`,
`server.go`, …) and states what the package owns **and** what it does not
own. A standalone `doc.go` is only warranted when the contract needs more
than a few lines; do not create `doc.go` just to hold a sentence that
already has a home.

## Canonical file roles

Not every package needs every file. Never create empty ceremonial files.
When a category exists, it uses this filename:

| Exists when… | Filename |
| --- | --- |
| Package-owned domain types shared across capabilities | `types.go` |
| Package configuration struct, defaults, validation | `config.go` |
| Package error vocabulary with real `errors.Is` callers | `errors.go` |
| Stateful primary object and its construction | `service.go`, or a better domain noun (`daemon.go`, `client.go`, `server.go`, `api.go`, `control.go`, `store.go`) |

Role consistency, not textual uniformity: do not rename a precise name
(`model.go`, `payload.go`, `control_types.go`) to `types.go` merely for
symmetry. A type used by exactly one capability stays in that capability's
file. Constructors are only `New` (in-memory), `Open` (live resource),
`Load` (decode/validate path or bytes).

## Capability files

Behavior is organized by capability, named in domain language:
`repositories.go`, `access_keys.go`, `tailnet.go`, `enrollment.go`,
`lifecycle.go`, `profiles.go`. One file = one concept a human can name.
Banned: `utils.go`, `helpers.go`, `common.go`, `misc.go`, `manager.go`,
`logic.go`, `impl.go`, and synonyms. Resource lifecycles use one verb set
(`create`/`get`/`list`/`update`/`delete`, or a single `lifecycle.go` when
the operations are inseparable); do not invent per-package synonyms for the
same action, but do not erase real domain distinctions either
(`provisioning` Ignition secrets vs `create` project environments are
different jobs in different packages).

File size is a smell, not a rule: prefer under 400 lines, split past 500
only when the file holds independently nameable responsibilities. A large
file that is one coherent concept (one privileged boundary, one build
pipeline, one policy store) stays large and says so in review.

## Platform files

OS boundaries use Go file selection, never scattered `runtime.GOOS` checks:

`process.go` + `process_linux.go` + `process_other.go`

The shared contract lives in the neutral file. Build-tag variants that
select packaging contracts (`platform`'s `legacy.go`/`vendor.go`)
follow the same rule.

## Tests mirror production

`repositories.go` ↔ `repositories_test.go`, `oauth.go` ↔ `oauth_test.go`.
Several tests may exercise one capability; keep them together until the
test file itself is hard to navigate. Shared package-local test
infrastructure lives in `test_helpers_test.go`, never in a vague
`behavior_test.go` / `everything_test.go` / `misc_test.go`. Exceptions with
honest names are allowed: `boundaries_test.go` for architecture-boundary
tests, and one package-contract test (e.g. `installer_test.go`) when it
genuinely covers the package's combined surface. Facade integration tests
(e.g. `web/*_test.go` exercising the server) stay with the facade and
reference the owning packages (`api.*`, `auth.*`) directly — forwarding
aliases between internal packages are banned and `internal/archcheck`
fails if `web/aliases.go` returns. The host Client may re-export
`host/terminal` wire types so transport never imports that executor.

## Interfaces and errors

Interfaces belong to the consumer that needs the abstraction, stay small,
and disappear when they have one implementation and no testing or
architectural boundary requires them. No dependency-injection machinery.

Errors wrap causes with `%w`. Package sentinels exist only with a real
in-tree `errors.Is` caller. Libraries never log; long-lived processes use
`slog`; one-shot CLIs print once to stderr and exit.

## Package boundaries

Top-level `internal/` names are major Soda concepts (`project`, `host`,
`web`, `release`, `tailnet`, `runners`, `store`, `forgejo`, `installer`,
plus small primitives). Subpackages express genuine subordinate
boundaries only: privilege execution under `host/` (`host/project`,
`host/terminal`, `host/tailnet`), HTTP transport under `web/`
(`web/api`, `web/auth`), release construction under `release/`
(`release/build`, `release/image`, `release/qualify`,
`release/deliver`). No `internal/models`, `internal/services`,
`internal/utils` or other horizontal dumping grounds; no micro-packages;
no splitting `tailnet` / `runners` / `store`; no resurrecting retired
top-level paths (`internal/projectos`, `internal/linuxhost`,
`internal/installlayout`, `internal/webapp`, `internal/webauth`,
`internal/nativebuild`, `internal/nativequalification`,
`internal/nativefinalization`, `internal/releasedelivery`,
`internal/appliancerelease`, `internal/hostproject`,
`internal/hostterminal`, `internal/hosttailnet`). Package declarations
match directory leaves; colliding domain twins are resolved with import
aliases (`domain`, `projectexec`, `tailnetexec`), not concatenated
package names.

Dependencies run one way, from orchestration toward capabilities:

```text
binaries (cmd/, tools/)
  ↓
transport (web, host daemon mux)
  ↓
domain orchestration and policy (project, tailnet, runners, store, web/api)
  ↓
privileged execution and release construction (host/*, installer, release/*)
  ↓
external adapters and primitives (forgejo, filelock, strictjson, platform)
```

Domain types are defined once in the owning package and referenced
directly — never duplicated as parallel DTOs, never translated field by
field, never re-exported through aliases. Privileged wire types owned by
an executor may appear on the `host` Client surface so transport never
imports that executor. Each side of a privilege boundary validates its
own inputs against the domain validators (`project.Valid*`); duplicated
validation logic anywhere else is a smell.
Moving files inside a package is cheap; moving symbols between packages
changes architecture — do it only when ownership is clearly wrong, keep
the new ownership explainable in one sentence, and avoid cycles.
`internal/archcheck` mechanically enforces the retired names and the
banned edges above.

## When NOT to create a file

- A `types.go` for types owned by one capability.
- An `errors.go` without a real error vocabulary.
- A `config.go` for a struct small enough to live with its owner.
- A `doc.go` duplicating a `// Package` clause that already states the
  ownership contract.
- A new file to avoid touching an existing file; a split with no nameable
  concept on each side.
