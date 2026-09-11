# Bun and TypeScript development

Run dependency commands from the repository root using the Bun version pinned in
`package.json`. The root package owns shared script/test tooling and the single
`bun.lock`; `cockpit/` is its UI workspace. Declare dependencies where they are
used. Root tools must not reach through Cockpit or another package's transitive
installation to load a dependency. Cockpit keeps React, PatternFly and Vite+.

```sh
bun install --frozen-lockfile
bun run check:source # Go + TypeScript/Lit + local browser/Cockpit + Python
bun run typecheck
bun run test
bun run build
bun run build:preview # local Forgejo branding mount
bun run screenshot --help
```

`build` compiles and minifies the Forgejo browser assets and the two Cockpit
pages. It does not install an appliance or build a standalone dashboard. `test`
prepares locked terminal assets and emitted Forgejo modules once, then runs the
frontend, native Forgejo page fixtures, drawer layout, Forgejo and Cockpit suites.
The page group requires the authorized local Forgejo fixture at `localhost:3300`
and its saved screenshot-account credential. It reuses `TestNativeConnectionFixture`,
prepares a fresh canonical public payload and isolated Soda OAuth/DB state, then
runs the existing page consumers with that fixture's own native login. Missing native
host/assets fail this group; no handwritten HTML fallback or implicit installation
is used. Synthetic operation APIs remain separate from native/provider proof.
The local Lit runtime and operator settings-link browser checks are enabled in the
Forgejo group. Cockpit builds through Vite+'s programmatic API under Bun; it retains
React and PatternFly. Installed/provider checks keep their explicit opt-in flags
and target/action permissions; installing dependencies does not run those journeys.

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

Focused commands prepare their own assets:

- `bun run test:frontend` — frontend unit/browser tests; conditional page/layout
  journeys are executed by the commands below, not silently counted as covered here.
- `bun run test:pages` — uncached Go producers followed by all three real HTML/CSP
  browser consumers (Spaces, operator runners and repository settings). Missing or
  empty fixture output fails before Chromium. `test:spaces-page` is a compatibility
  alias for this expanded group. Run directories are printed and retained on failure.
- `bun run test:layout` — the integrated drawer layout fixture.
- `bun run test:forgejo` — Forgejo source tests plus local Lit runtime/settings-link.
- `bun run test:lit` — focused emitted Lit runtime, settings-link and workspace tests.

For the selected retained-page transition, the **existing** native fixture also
accepts `SODA_CONNECTION_PREDECESSOR` (an absolute verified b8af68c export),
`SODA_CONNECTION_PREDECESSOR_REVISION` (its full revision) and
`SODA_CONNECTION_PREDECESSOR_MANIFEST_SHA256` (its independently established
`build-info.json` hash). Verify an older export with its own trusted verifier first;
the current inventory rules are not retroactive. Pass these variables to
`bun run test:pages`; all normal consumers still run. The fixture binds three genuine
old module files to that manifest, models the inspected zero-age revalidation policy,
and tests first candidate graph navigation/Back/Forward in a fresh browser context.
Its phase file controls only the local public-file responder. No native HTML/API
replacement, appliance cutover or old backend/page execution is involved. Without
these inputs, the stock six-hour candidate-bytes/legacy-URL cache case remains.
Neither asset-only case proves old-document retirement or terminal preservation.

For genuine predecessor-document execution, additionally provide
`SODA_CONNECTION_PREDECESSOR_BINARY`: a regular bounded executable extracted with
Podman from that verified dashboard OCI. Its bytes must match the same inventory's
`rootfs/usr/local/libexec/soda/soda-dashboard` entry (the image's executable path is
`/usr/local/bin/soda-dashboard`). A mismatch refuses before OAuth-app/process startup.
No rebuild or hand-written old HTML is substituted. After all ordinary consumers and
the native parent finish, a separate browser context uses the old process, complete
old public-file tree and a fresh synthetic v6 database. The private `backend-phase`
file switches to current handlers only after confirmed old-process exit; those
handlers migrate that **same database** to v9 with the same fixture key/client.
The old page remains open for its real Refresh control and subsequent departure.
The browser observes actual pagehide retirement and the actual history outcome:
retired BFCache owner or a network reload/current owner, labelled separately. Headers
are not changed to force BFCache. The current passing run used a network reload.

This phase uses the existing fixture account/login owner and leaves its new OAuth
app, separate databases, logs and private browser evidence retained. The old process
has an absent helper socket: no project provisioning or terminal process proof is
implied. The legacy required `admin_token_file` is an unused absolute placeholder;
no admin token is created/read/borrowed. This does not mutate retained appliances,
load their credentials or establish their installed cache/CSP behavior.

For optional populated visual review, set `SODA_PAGE_CAPTURES` to an existing
absolute private directory. The same consumers reuse `scripts/screenshot.ts` on
their authenticated fixture pages; no second login or scenario runner is started.
See [capture scope and guards](screenshot-capture.md#capturing-the-existing-native-page-consumers).
The normal suite still runs with this flag absent; screenshots never replace its
behavior/authorization assertions or installed evidence.

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

## Compiler boundaries

All configurations extend `tsconfig.base.json`: strict checking, checked indexed
access, exact optional properties, consistent filename casing and no emitted output.
`skipLibCheck` skips third-party declaration checking, not our source.

Lit tagged templates require additional binding analysis: ordinary strict TypeScript
does not infer an HTML property's or event's type from its position in a string.
The [frontend improvement guide](frontend-improvement-plan.md#required-analyzer-integration)
specifies a required analyzer alongside the existing compiler checks, with its own
compatible development-only compiler resolved through the single root lock. Research
demonstrated this path in isolation; the required workflow now runs actual-source
analysis and checker fixtures via [the owned tool](../tools/lit-check/README.md).
Its analysis-only bundle resolves bare compiler imports explicitly; clean frozen-lock
installation is tested without changing the product compiler. An editor plugin alone
is not a command-line gate. Typed view-helper APIs remain ordinary checked
TypeScript and complement, rather than replace, internal template analysis.

| Configuration | Owner and runtime |
| --- | --- |
| `tsconfig.json` | Root scripts; Bun, with DOM types for Playwright page callbacks |
| `tsconfig.tests.json` | Root tests; Bun plus browser/JSDOM fixtures |
| `tsconfig.browser.json` | Forgejo branding and Sodaspaces browser scripts; DOM types, no automatic Bun/Node globals |
| `cockpit/tsconfig.json` | Existing Cockpit TS/TSX, test and Vite+ build code |
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
them to the existing public URLs. Modules retain their public import boundaries,
and upstream xterm assets retain their exact locked bytes and licenses. The shared
[Lit runtime](lit.md) is the explicit bundled exception: component `lit` imports
resolve to its staged relative URL, and existing imports remain external. Generated
JavaScript must not be tracked beside TypeScript.

Changed Soda module graphs use the presentation epoch from
`custom/header.tmpl` (`soda-presentation-revision`). The compiler appends that
`?v=` value to every relative external import, including dynamic page imports and
Lit. Bump the epoch and the three Soda entry URLs plus changed workspace/settings
styles together; `lit-build.test.ts` checks the emitted closure and entry registry.
Do not version only the top-level script: Forgejo caches assets privately for six
hours. New documents receive the new graph; already-open documents keep their
loaded code until navigation/reload. This is not live code replacement or an
atomic mixed-version rollout guarantee.

Run `bun run build:forgejo` before serving a local preview, and resolve payload
entries beginning `@build/forgejo-js/` from that output directory. Never serve a
renamed `.ts` file directly to a browser. `bun run build:preview` projects the same payload into the local preview branding
mount; see [screenshot capture](screenshot-capture.md). The isolated drawer layout fixture uses
the same emitted assets and checks their integration with the locked renderer.

A completed conversion passes `bun run typecheck` and the relevant behavior tests,
including authorization/failure paths. Browser ports also need emitted-asset checks;
local source tests are not native installed proof. Record actual checks and remaining
migration scope in the handoff.
