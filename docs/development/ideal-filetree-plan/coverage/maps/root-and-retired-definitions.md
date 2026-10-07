# Root selectors and retired artifacts

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-13ee4b2252c9"></a>

## [Cargo.lock](../../../../../Cargo.lock)

Current artifact role, concrete consumer/provenance and established contract owner inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2885; whole file | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Tool-maintained dependency identities; Cargo/Bun/Go consumers and preserved locked dependency resolution — docs/development/ideal-filetree-plan/coverage/inventory/root-files.md; Current source/README provenance and explicit native/browser/doc build consumer inspection |

<a id="coverage-b33563055168"></a>

## [README.md](../../../../../README.md)

Current artifact role, concrete consumer/provenance and established contract owner inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–5, 77–85; Brand identity; License and attribution | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Brand identity; License and attribution — Current source clauses/section; canonical workflow requirements at519b76bd |
| 6–7, 32–33; Manual admission/fresh disposable workspace model; Disposable factory workspace topology clauses | [F07](../../slices/factory-coordination.md#f07-assignment-and-dispatch) | obsolete | Manual admission/fresh disposable workspace model; Disposable factory workspace topology clauses — Current source clauses/section; canonical workflow requirements at519b76bd |
| 7–10, 19; Bounded execution/outcome/publication description; Explicit current operator-command interface | [F08](../../slices/factory-coordination.md#f08-run-lifecycle-and-intervention) | retained | Bounded execution/outcome/publication description; Explicit current operator-command interface — Current source clauses/section; canonical workflow requirements at519b76bd |
| 8–9; Fixed one-repair/manual-merge clauses | [F10](../../slices/factory-coordination.md#f10-independent-review-and-correction) | obsolete | Fixed one-repair/manual-merge clauses — Current source clauses/section; canonical workflow requirements at519b76bd |
| 11–15, 47; Persistent human Project access; Explicit human Join clause | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Persistent human Project access; Explicit human Join clause — Current source clauses/section; canonical workflow requirements at519b76bd |
| 16–19; Native Forgejo frontend and protected Go API | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | retained | Native Forgejo frontend and protected Go API — Current source clauses/section; canonical workflow requirements at519b76bd |
| 17; Removed OAuth clause | [G01](../../slices/forgejo-integration.md#g01-browser-authority-and-contributions) | obsolete | Removed OAuth clause — Current source clauses/section; canonical workflow requirements at519b76bd |
| 16–17; Spaces entry/drawer description requiring current page/panel reconciliation | [S02](../../slices/spaces-and-terminals.md#s02-workspace-lifetime-and-navigation) | unresolved | Spaces entry/drawer description requiring current page/panel reconciliation — Current source clauses/section; canonical workflow requirements at519b76bd |
| 18; Deferred local Runners clause | [F11](../../slices/factory-coordination.md#f11-candidate-verification-assessment) | obsolete | Deferred local Runners clause — Current source clauses/section; canonical workflow requirements at519b76bd |
| 20–29, 55–76; Documentation entrypoints; Source/guide navigation | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Documentation entrypoints; Source/guide navigation — Current source clauses/section; canonical workflow requirements at519b76bd |
| 30–43; Host/Forgejo/Caddy topology | [H01](../../slices/shared-supporting-slices.md#h01-private-ipc-and-service-lifetime) | retained | Host/Forgejo/Caddy topology — Current source clauses/section; canonical workflow requirements at519b76bd |
| 36; SQLite/Soda OAuth topology clause | [H02](../../slices/shared-supporting-slices.md#h02-storage-mechanics) | obsolete | SQLite/Soda OAuth topology clause — Current source clauses/section; canonical workflow requirements at519b76bd |
| 44–50; Retained Project root, association and native Git boundary | [P01](../../slices/projects.md#p01-repository-association-and-creation) | retained | Retained Project root, association and native Git boundary — Current source clauses/section; canonical workflow requirements at519b76bd |
| 51–54; LAN access/route obligations | [N02](../../slices/networking.md#n02-project-lan-access) | retained | LAN access/route obligations — Current source clauses/section; canonical workflow requirements at519b76bd |
| 61; line61 rust/soda-factory source-path token | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | obsolete | Stale source path within otherwise retained navigation — Current source clauses/section; canonical workflow requirements at519b76bd |

<a id="coverage-b5e19b9328f3"></a>

## [internal/filelock/filelock.go](../../../../../internal/filelock/filelock.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–36; file scaffold; Acquire | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; Current declaration duty: Acquire — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7a00c66ef34a"></a>

## [internal/filelock/filelock_test.go](../../../../../internal/filelock/filelock_test.go)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–78; file scaffold; openTestLock; TestCancelledWaitDoesNotAcquireOrRetainTheLock; TestSharedHoldersExcludeWriterUntilBothRelease; TestCancellationAndDescriptorErrors | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Current scaffold duty: file scaffold; 5 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-7ae45ad102ea"></a>

## [package.json](../../../../../package.json)

Current artifact role, concrete consumer/provenance and established contract owner inspected

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–10, 12–25, 27, 29–47; lines 1-10: Bun workspace/package identity and build command registry; lines 12-25: Source/type/lint/browser-fixture check selectors; lines 27-27: Explicit screenshot tool dispatch; lines 29-30: Local preview/dev entry points; lines 31-47: Locked dependency/profile inventory and registry closure | [H06](../../slices/shared-supporting-slices.md#h06-developer-tooling-and-verification-infrastructure) | retained | Bun workspace/package identity and build command registry; 5 named units assigned here; remaining selectors preserve each duty — Current file contents; canonical workflow-requirements at519b76bd and native build/consumer selectors |
| 11, 28; lines 11-11: Browser asset build dispatch; lines 28-28: scripts.build:forgejo Bun build-forgejo asset-bundle consumer | [D03](../../slices/release-and-installation.md#d03-candidate-production) | retained | Browser asset build dispatch; scripts.build:forgejo Bun build-forgejo asset-bundle consumer — Current file contents; canonical workflow-requirements at519b76bd and native build/consumer selectors |
| 26; lines 26-26: Opt-in installed asset evidence selector | [D06](../../slices/release-and-installation.md#d06-installed-qualification) | retained | Opt-in installed asset evidence selector — Current file contents; canonical workflow-requirements at519b76bd and native build/consumer selectors |
| 28; lines 28-28: scripts.build:forgejo cargo soda-fetch-terminal distribution producer | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) | retained | scripts.build:forgejo cargo soda-fetch-terminal distribution producer — Current file contents; canonical workflow-requirements at519b76bd and native build/consumer selectors |

## Removed predecessor navigation

These anchors identify earlier mapped source. Their paths no longer define this current map;
the current inventory and normalized predecessor ledger preserve replacement/retirement evidence.
No historical source body is counted as current coverage.

<a id="coverage-e25a660ce02b"></a>

Former source `.agents/plans/2026-09-16-remove-rpm-pinning.md`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2a6c4de4fd4d"></a>

Former source `.containerignore`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-87320095d7c4"></a>

Former source `.githooks/pre-commit`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-bc37d034bad5"></a>

Former source `.gitignore`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-ba6bdb5315b1"></a>

Former source `.oxfmtrc.json`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2113aef2ff3d"></a>

Former source `.oxlintrc.json`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-a54ff182c7e8"></a>

Former source `AGENTS.md`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2e9d962a0832"></a>

Former source `Cargo.toml`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-c693279643b8"></a>

Former source `LICENSE`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-dfb14fbb9e7d"></a>

Former source `NOTICE`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-bfd0ef82a011"></a>

Former source `bun.lock`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-d20a1d3c8f2b"></a>

Former source `bunfig.toml`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-7a882221a14c"></a>

Former source `factory-os/Containerfile`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-33ef32bf6c23"></a>

Former source `go.mod`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-3295df723452"></a>

Former source `go.sum`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-9c0d294c05fc"></a>

Former source `installer`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-2b1bde2cf3a8"></a>

Former source `rust-toolchain.toml`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-619e7e2459fb"></a>

Former source `soda-candidate`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-0b280a445be1"></a>

Former source `tsconfig.base.json`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-7053a2fc9b8a"></a>

Former source `tsconfig.browser.json`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-b55cdbef4907"></a>

Former source `tsconfig.json`; consult its pinned earlier Git source and the current coverage disposition.

<a id="coverage-e05a8a3e3685"></a>

Former source `tsconfig.tests.json`; consult its pinned earlier Git source and the current coverage disposition.
