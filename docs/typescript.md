# Bun and TypeScript development

Run dependency commands from the repository root using the Bun version pinned in
`package.json`. The root package owns shared script/test tooling and the single
`bun.lock`; `cockpit/` is its UI workspace. Declare dependencies where they are
used. Root tools must not reach through Cockpit or another package's transitive
installation to load a dependency. Cockpit keeps React, PatternFly and Vite+.

```sh
bun install --frozen-lockfile
bun run typecheck
bun run test
bun run build
bun run screenshot --help
```

`build` builds the two Cockpit pages. It does not install an appliance or build a
standalone dashboard. `test` runs the existing Node frontend/Forgejo tests and
Cockpit's Vite+ tests. Browser/installed checks keep their explicit opt-in flags
and target/action permissions; installing dependencies does not run those journeys.
Node remains pinned for these existing tools. New TypeScript utilities run with
Bun; conversion of existing script/test entrypoints includes their runtime checks.
Do not substitute bare `bun test` for the selected test runners during migration.

## Compiler boundaries

All configurations extend `tsconfig.base.json`: strict checking, checked indexed
access, exact optional properties, consistent filename casing and no emitted output.
`skipLibCheck` skips third-party declaration checking, not our source.

| Configuration | Owner and runtime |
| --- | --- |
| `tsconfig.json` | Root scripts; Bun/Node, with DOM types for Playwright page callbacks |
| `tsconfig.tests.json` | Root tests; Bun/Node plus browser/JSDOM fixtures |
| `tsconfig.browser.json` | Forgejo branding and Sodaspaces browser scripts; DOM types, no automatic Bun/Node globals |
| `cockpit/tsconfig.json` | Existing Cockpit TS/TSX, test and Vite+ build code |

The three root configurations temporarily include legacy JS/MJS with `allowJs`
and `checkJs: false`. This makes the existing files visible to projects/editors
without claiming they have been ported or type checked. New `.ts` files are checked
strictly. Remove legacy inclusion when each area is fully converted; do not disable
checking for converted code. Cockpit is already TypeScript and has no such allowance.

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

The browser-source port must add a build step that emits JavaScript into ignored
`.artifacts/`, update the production payload map, staging and local preview callers,
and verify the emitted assets. That step is not implemented by this scaffolding:
current Forgejo JavaScript is still served from its existing source paths. Preserve
public URLs/module boundaries and upstream xterm assets. Do not track generated
JavaScript beside TypeScript or rewrite third-party JavaScript as our source.

A completed conversion passes `bun run typecheck` and the relevant behavior tests,
including authorization/failure paths. Browser ports also need emitted-asset checks;
local source tests are not native installed proof. Record actual checks and remaining
migration scope in the handoff.
