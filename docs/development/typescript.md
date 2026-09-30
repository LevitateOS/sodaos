# Bun and TypeScript development

Run dependency commands from the repository root using the Bun version pinned in
`package.json`. The root package owns shared script/test tooling and the single
`bun.lock`; `tools/lit-check/` is the remaining workspace. Declare dependencies
where they are used, not through another package's transitive installation.
Cockpit's custom workspace, React/PatternFly/Zustand and Vite build are retired;
stock Cockpit branding uses canonical assets without a frontend application.

```sh
bun install --frozen-lockfile
bun run check:source # Go + TypeScript/Lit + local browser + Python
bun run typecheck
bun run test
bun run build
bun run build:preview # local Forgejo branding mount
bun run dev:spaces # local interactive Spaces mock backend and frontend
bun run screenshot --help
```

`build` compiles and minifies Forgejo branding modules. The separate Soda
extension build packages its page and panel entries, styles and locked terminal
assets; see [Lit](lit.md#runtime-and-builds). Neither command installs an appliance.
Browser suites require the compatible Playwright Chromium runtime. Local fixture
checks do not establish native installed behavior.

For Spaces visual development without a VM, use the
[local fixture server](../guides/local-testing.md).

## Local source checks

`bun run check:source` shares the existing Go module verification/tests, complete
TypeScript/Lit checks, `bun run test` and Python `tests/build` suite. It works in a
dirty source checkout without native images or a sealed deployment stage. Native
`check-native.sh` calls this same command **between** its exact-revision/artifact
checks, then runs packaging checks against the selected stage. Its matching-native
Linux, pinned-tool and clean-checkout requirements remain unchanged.

Prerequisites: an installed Go toolchain satisfying `go.mod`, pinned Bun and the
resolved frozen-lock workspace dependencies, Python 3, Bash, OpenSSL and Playwright's
compatible Chromium plus its sandbox/runtime libraries. Source checks use
`GOTOOLCHAIN=local`, `GOWORK=off`, read-only module metadata and `CGO_ENABLED=0`; they
may download modules into Go caches and fetch missing locked terminal distributions,
but do not install workspace dependencies, download Go toolchains, load images or
install an appliance. They write emitted assets,
retained synthetic HTML fixtures and ordinary test-local temporary files. Existing
optional Python checks still report skips: xorriso enables the synthetic ISO test;
`SODA_CADDY_BINARY` selects the separate real loopback Caddy integration. Do not set
installed/provider/native-origin flags for ordinary source checks: those remain
independently gated and are not enabled by the aggregate.

Focused source checks include `bun run test:frontend` for component behavior,
`bun run test:forgejo` for Forgejo branding and presentation, and
`bun run test:lit` for emitted Lit, workspace and terminal behavior. The current extension
package graph is checked by `scripts/build-soda-extension.test.ts`; the persistent
panel fixture checks terminal continuity while native pages navigate. Choose the
smallest affected check. Native installed journeys and their evidence rules belong
in [Testing](testing.md).

The `:prepared` scripts are the same suite bodies used by these wrappers and the
aggregate; direct use requires a preceding `bun run build:forgejo`. Test-specific
module builds remain where they exercise compiler/payload contracts; only repeated
full asset preparation was removed. Local success is not installed/provider or
native-architecture acceptance.

Soda's frontend/build/test tooling requires Bun, without a separate Node runtime.
`bunfig.toml` also selects Bun for dependency executables with Node shebangs.
Dependency lifecycle scripts are not enabled: the selected platforms use locked
prebuilt packages. Parcel watcher's optional Node/node-gyp source-build hook is
not needed for those packages.
Project toolchains and provider runners may still own Node for user workloads.
Prefer Bun file, process, hashing and server APIs in authored tooling. Keep
`node:` imports where they provide a concrete contract: path manipulation,
assertions, exclusive/private filesystem checks, raw HTTP request paths, terminal
input and explicit closure of borrowed Chromium file descriptors. Those supported APIs execute in
Bun and remain strictly typed; they do not introduce a Node process requirement.

## Formatting, linting, and complexity

Pinned `oxfmt` (0.68.0) and `oxlint` (1.81.0) in the root lock are the TypeScript
Oxc analogue of gofumpt and of staticcheck plus gocyclo. They share one parser
family. They do not replace `bun run typecheck` (Oxc has no typechecker we will
use in place of TypeScript 7). Do not add Prettier, ESLint, or Biome on top.
Whole-tree scripts live beside the Go gates; the pre-commit hook is
staged-only so an existing backlog cannot block unrelated commits. Go quality-gate
ownership and the Go threshold remain in [the Go ownership guide](go.md).

| Command | Scope |
| --- | --- |
| `bash scripts/check-oxfmt.sh` (`bun run check:oxfmt`) | Zero-tolerance format on the given `.ts`/`.tsx` paths, or every tracked TypeScript file. Fix with `bunx oxfmt --write <files>`. |
| `bash scripts/check-oxlint.sh` (`bun run check:oxlint`) | Correctness lint (oxlint `correctness`, `--format=agent`, without duplicating `tsc`). Complexity is excluded here. |
| `bash scripts/check-ts-complexity.sh` (`bun run check:ts-complexity`) | Browser payload only (`frontend/`, `assets/branding/`). Cyclomatic complexity strictly below 10, matching shipping Go (`gocyclo -over 9` / oxlint `complexity` max 9). `scripts/` and `tools/` are out of scope. |

`.githooks/pre-commit` runs those three on staged `.ts`/`.tsx` files after the Go
checks. Formatting applies to staged tests as well; complexity does not (and does
not apply to `scripts/` or `tools/`).

## Compiler boundaries

All configurations extend `tsconfig.base.json`: strict checking, checked indexed
access, exact optional properties, consistent filename casing and no emitted output.
`skipLibCheck` skips third-party declaration checking, not our source.

Lit tagged templates require additional binding analysis: ordinary strict TypeScript
does not infer an HTML property's or event's type from its position in a string.
The [frontend improvement guide](typescript.md)
specifies a required analyzer alongside the existing compiler checks, with its own
compatible development-only compiler resolved through the single root lock. Research
demonstrated this path in isolation; the required workflow now runs actual-source
analysis and checker fixtures via [the owned tool](../../tools/lit-check/README.md).
Its analysis-only bundle resolves bare compiler imports explicitly; clean frozen-lock
installation is tested without changing the product compiler. An editor plugin alone
is not a command-line gate. Typed view-helper APIs remain ordinary checked
TypeScript and complement, rather than replace, internal template analysis.

| Configuration | Owner and runtime |
| --- | --- |
| `tsconfig.json` | Root scripts; Bun, with DOM types for Playwright page callbacks |
| `tsconfig.tests.json` | Root tests; Bun plus browser/JSDOM fixtures |
| `tsconfig.browser.json` | Forgejo branding and Sodaspaces browser scripts; DOM types, no automatic Bun/Node globals |
| `tools/lit-check/tsconfig.json` | Analyzer runner checked by product TS7 against its classic compiler API; no browser runtime |

All authored scripts, browser modules and tests are TypeScript. No compiler
configuration admits unchecked JavaScript with `allowJs`/`checkJs`. Browser code
uses DOM APIs; Bun runs only its build tooling. Upstream terminal distributions
remain locked JavaScript assets rather than being rewritten as Soda source.

## Porting source

- Write new authored code in `.ts`/`.tsx`. Convert existing JS/MJS in coherent groups
  with its imports, callers, focused tests and documentation. A rename alone is not
  completion. Keep existing installed targets and opt-in gates intact.
- Prefer inference locally and explicit types at exported/API boundaries. Keep
  types beside the owning code. Share actual contracts; do not add a general type
  registry or duplicate Go models without a concrete consumer.
- Treat parsed JSON, messages, environment and external responses as untrusted.
  Narrow `unknown` with runtime checks before use; TypeScript does not validate data.
- No blanket `any`, `@ts-nocheck`, `@ts-ignore`, unchecked JSON casts or non-null
  assertions to evade errors. Narrow missing values. A necessary external typing
  exception must be local, explained and covered by a behavior check.
- Use `import type` for type-only imports and standard ESM. Browser sources cannot
  use Bun/Node APIs. Prefer explicit relative imports; do not introduce path aliases
  that require a second runtime resolver.
- Keep `strict`, `noUncheckedIndexedAccess` and `exactOptionalPropertyTypes` enabled.
  Optional means absent; include `undefined` explicitly only when the actual contract
  allows it. Do not fill unknown values with invented defaults to silence errors.

`scripts/build-forgejo.ts` uses Bun to emit minified JavaScript into ignored
`.artifacts/forgejo-js/` by default. Native builds pass their own output directory.
The payload map identifies those files with `@build/forgejo-js/`; staging copies
them to Forgejo's public branding URLs. The extension builder in
`scripts/build-soda-extension.ts` bundles the Spaces, Runners, Tailnet and Workspace
entries and packages its own CSS and locked xterm assets. Do not add those modules
to Forgejo's public payload. The shared [Lit runtime](lit.md) is retained for
Forgejo-side modules; extension components use their package graph. Generated
JavaScript must not be tracked beside TypeScript.

Forgejo branding modules use the presentation epoch from `custom/header.tmpl`
(`soda-presentation-revision`) for relative external imports, including Lit.
`lit-build.test.ts` checks that emitted Forgejo imports resolve within the public
payload. The native extension host owns package loading and its asset lifecycle;
Forgejo's branding cache epoch does not version extension package modules.

Run `bun run build:forgejo` before serving a local preview, and resolve payload
entries beginning `@build/forgejo-js/` from that output directory. Never serve a
renamed `.ts` file directly to a browser. `bun run build:preview` projects the
Forgejo branding payload into a local preview mount. The Spaces preview serves
assets from the separate extension build; see [screenshot capture](../design/screenshot-capture.md).

A completed conversion passes `bun run typecheck` and the relevant behavior tests,
including authorization/failure paths. Browser ports also need emitted-asset checks;
local source tests are not native installed proof. Record actual checks and remaining
migration scope in the handoff.
