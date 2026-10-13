# Shared supporting slices

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## First-principles refresh (2026-10-13)

Baseline: `fde19669440455a77e4c81c4f08ccc60a72276eb`, plus the named H06
gate/test patches below. Current helper owners and actual consumers were reconciled
with retained reviews and the Go/Rust/TypeScript/PostgreSQL decisions
(GUIDANCE-01/03/05). Historical scope is preserved. **Met** is the bounded
source model, **partial** retains a concrete workflow/evidence gap, and native
outcomes marked **unknown** were not executed. Helpers must not acquire domain
authority merely because several callers share a mechanism.

| Slice / purpose | First principles | Current assessment and evidence | Desired gap | Dependency-ordered implementation / check |
| --- | --- | --- | --- | --- |
| H01 Private IPC and service lifetime: provide admitted bounded request/stream transport. | Kernel/socket identity and transport limits precede domain admission; stopping a listener is distinct from retiring admitted children/streams. | **Partial.** `lib/host/src/daemon/server.rs` uses Hyper/Tokio, bounded connection/backend permits and joined terminal tasks; `internal/host/client.go` consumes the private host transport. Real namespace/lifetime behavior is **unknown**. | Qualify current peer and shutdown boundaries with actual processes; retain native libraries rather than another transport bridge. | Before dependent host journeys, reuse daemon/client refusal tests, then prove one actual private peer/request/stream shutdown boundary. Domain authorization stays with its callers. |
| H02 Storage mechanics: provide PostgreSQL connections, transactions and schema admission. | A transaction implements its domain's invariant; concurrency proof must match PostgreSQL; schema/key admission cannot create a second canonical ledger. | **Partial.** `internal/store/store.go`, `schema.go`, `readiness_work.go` and `readiness_creations.go` retain domain-local SQL and bounded readiness transactions; `cmd/soda-identity/src/pg.rs` retains distinct broker connection mechanics. No new live PG check ran here. | Preserve current database choice and validate changed invariant/lock order with the existing disposable fixture. | After O01, exercise only changed schema/key/transaction contracts with explicit fixture DSN. Historical SQLite single-connection proof is inapplicable; no migration/alternate store is proposed. |
| H03 Encoding and parsing: enforce caller-owned bounded wire representations. | Syntax helpers grant no authority; duplicate/unknown fields, size, UTF-8 and completion belong to each protocol's admitted representation; shared syntax does not imply shared semantics. | **Met** for inspected Go boundary: `internal/strictjson/decode.go` bounds input and rejects invalid UTF-8/duplicate/unknown/trailing content. Current Rust callers use `lib/host/src/json/` and `cmd/soda-identity/src/strict.rs`; retired syntax engine remains retired. | Keep one mature syntax implementation per language and caller-owned DTO/field contracts; compare real consumers before consolidation. | Before a changed wire caller, run its existing malformed/producer-vector tests. No blanket cross-domain DTO, alias package or generic parser framework. |
| H04 Configuration and filesystem primitives: admit protected inputs and mechanical file/lock effects. | Caller owns policy, target, lock order and lifecycle; helper must preserve private paths/descriptors and report uncertainty without expanding cleanup scope. | **Partial.** `internal/config/config.go` admits configuration; `internal/filelock/filelock.go` waits cancellably with native flock; `lib/host/src/tcontrol_policy.rs` and installer key helpers retain caller-specific protected file ownership. Native permission/concurrency behavior is **unknown**. | Resolve demonstrated caller-specific failures before sharing helpers; do not normalize away differing secret/ownership contracts. | Before dependent O/N/I effects, run existing restricted-input/lock/cancellation tests for changed owner, then selected native filesystem boundary checks. No new cleanup coordinator. |
| H05 Branding, avatars and attribution: retain canonical deterministic visuals and provenance. | Rendering is deterministic and bounded; artwork/licensing remain canonical; native authentication/product authority is independent of appearance. | **Met** for inspected renderer model: `internal/avatar/avatar.go` validates 32-hex seed/size, shares immutable style and creates one renderer per call; `assets/branding/` retains canonical native-host consumers. Installed visual integration is **unknown**. | Preserve deliberate Forgejo design breadth and stock Cockpit branding; verify actual asset/staging consumers when they change. | After changed asset consumers/D03, run established avatar snapshot and native branding source checks; inspect one installed browser appearance only for the affected integration. No new shell or visual redesign. |
| H06 Developer tooling and verification infrastructure: report reliable scoped source/test results. | A checker failure cannot become compliance; actual file arguments and tool status must survive the gate; source PASS is not native qualification. | **Partial, two concrete corrections implemented.** `scripts/check-oxfmt.sh` and `check-ts-complexity.sh` formerly swallowed tool failures and split paths. Both now preserve status and explicit/default filename boundaries. The complexity gate retains canonical max 9 and lets `check-oxlint.sh` own unrelated correctness rules. Existing `tests/build/source_checks_test.go` and `complexity_scope_test.go` exercise copied actual gates. | Keep domain test/receipt owners and evidence scope explicit; other tool gates remain outside this correction's proof. | Prove the actual pinned rule profile before changing flags, reproduce false success, fix current entrypoints and run focused drivers. Five oxfmt and five complexity boundary cases pass; real Oxlint confirms threshold/parser/input behavior below. No new config/profile owner or checker framework. |

H06 implementation evidence: exact Go 1.26.7 executed the focused
`TestOxfmtGatePreservesFormatterFailureAndPaths` and
`TestTSComplexityGatePreservesAnalyzerFailureAndPaths` tests in `./tests/build`
with `-count=1` (ten passing subcases).
The original gate returned zero/PASS with the fake formatter's `formatter
unavailable` stderr and exit 2 on an ordinary filename; the fixed gate passes
clean-result, differences, operational-failure, explicit spaced-path and default
NUL-path cases. The copied original TypeScript complexity gate returned
zero/PASS on a real malformed ordinary frontend input under installed Oxlint
1.81.0; the changed gate reports the actual parse failure and exits nonzero.
Real-tool profile checks show that `-A all -D complexity` would reset the rule's
configured options and incorrectly admit complexity 10. The selected
`-A correctness` preserves max 9: a combined complexity-9/10 fixture reports
only the latter; a concrete unrelated `no-unreachable` error remains the general
lint gate's responsibility; missing inputs and parse failures still fail.
`bash -n` passes for both changed scripts. All scratch/cache/temp
paths were under `/home`; no formatter install, whole-tree source scan,
native build or operational command ran. Independent A challenge inspected the
actual script/test diff and accepted retained diagnostics/status, explicit/default
filename handling and failure of direct discovery under `set -e`; it found no
blocking objection within this formatter boundary. A's final TypeScript challenge
also accepted preserved max-9 options, direct status/discovery failure handling,
payload scope and retained diagnostics against the exact copied script/config
and selected real-tool fixtures; no blocking objection remained. Fake-tool results
prove gate behavior; the real
Oxlint fixture checks support only this selected profile, not native qualification.
A also checked H03's bounded strictjson contract and H05's validated deterministic
renderer boundary. Those accepted source claims do not qualify installed consumers.

## H01 Private IPC and service lifetime

Provide admitted private request/stream transport and explicit service startup/shutdown.

- **Entrypoints:** systemd socket activation; soda-host main; Go host/broker clients; private HTTP/mux dispatch.
- **Owned data:** Listener descriptors, admission/stream slots, in-flight requests and transient process/service handles; No repository policy or competing run ledger.
- **Authority:** Kernel peer/socket ownership and fixed operation admission; domain authority remains in consuming slices.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [G02](forgejo-integration.md#g02-background-service-admission); Native systemd/Unix sockets; Domain operation slices.
- **Source files:** `rust/soda-host/src/main.rs:224 (historical source locator)`; `rust/soda-host/src/gmux_admission.rs (historical source locator)`; `rust/soda-host/src/gmux_server.rs (historical source locator)`; [internal/host/client.go:56](../../../../internal/host/client.go#L56); [internal/identity/client/client.go](../../../../internal/identity/client/client.go); `appliance/services/soda-host.socket (historical source locator)`.
- **Tests:** `rust/soda-host/tests/gmux_smoke.rs:1048 (historical source locator)` — Scripted-backend route/parser/stream-slot tests and local Unix-server harness; `rust/soda-host/src/main.rs:389 (historical source locator)` — Inline flag/root/listener and exit-code contract tests; [internal/identity/client/client_test.go](../../../../internal/identity/client/client_test.go) — Broker client request/response tests.
- **Unclear boundaries:** Transport admission is not domain authorization. Mux smoke uses scripted backends and has stale introductory wiring prose; it does not alone prove the production backend, shutdown or installed daemon.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.
- **Validity review:** [H01 record](../reviews/H01.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## H02 Storage mechanics

Support current PostgreSQL connections, transactions, schema admission and mechanical integrity checks.

- **Entrypoints:** store.Open()/OpenEncrypted(); schema initialization; Broker Store.open_encrypted() and PostgreSQL connection implementation.
- **Owned data:** Connection/transaction handles, schema_version and encryption-key check metadata; Domain table rows remain assigned to their domain slices.
- **Authority:** Operator-configured restricted DSN/key inputs and database role permissions; callers own product decisions.
- **Dependencies:** [O01](operator-administration.md#o01-first-boot-database-provisioning); [O03](operator-administration.md#o03-existing-install-credential-maintenance); [I02](identity-brokering.md#i02-encrypted-credential-custody); Native PostgreSQL; Domain persistence slices.
- **Source files:** [internal/store/store.go:137](../../../../internal/store/store.go#L137); [internal/store/schema.go:12](../../../../internal/store/schema.go#L12); [internal/store/grants.go](../../../../internal/store/grants.go); `rust/soda-identity/src/pg.rs (historical source locator)`; `rust/soda-identity/src/store.rs:61 (historical source locator)`.
- **Tests:** [internal/store/schema_test.go:11](../../../../internal/store/schema_test.go#L11) — Fixture-dependent current-schema creation, integrity and old/unversioned-store refusal; [internal/store/grants_test.go:11](../../../../internal/store/grants_test.go#L11) — Encryption-key continuity across store reopen; `rust/soda-identity/src/pg.rs:481 (historical source locator)` — Inline DSN/command-tag vectors.
- **Unclear boundaries:** The Go store and broker share schema/code concepts but must not gain duplicate domain authority. Confirm deployed database separation and which store is canonical for each identity record during I02 review; schema tests may skip without fixture inputs.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.
- **Validity review:** [H02 record](../reviews/H02.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## H03 Encoding and parsing

Implement bounded, explicit wire decoding/encoding used by the established callers.

- **Entrypoints:** strictjson.Decode(); caller-owned Serde request/DTO admission and response projections. The shared Rust syntax engine is retired by L04.
- **Owned data:** Transient decoded values, declared field specs and shared contract vectors; no independent durable product state.
- **Authority:** Established caller/protocol contracts define accepted representations; parser helpers grant no operation authority.
- **Dependencies:** [H01](#h01-private-ipc-and-service-lifetime); [G03](forgejo-integration.md#g03-authoritative-native-reads); Domain wire contracts.
- **Source files:** [internal/strictjson/decode.go:15](../../../../internal/strictjson/decode.go#L15); [lib/host/src/json/mod.rs](../../../../lib/host/src/json/mod.rs); [cmd/soda-identity/src/strict.rs](../../../../cmd/soda-identity/src/strict.rs); [cmd/soda-identity/src/wire.rs](../../../../cmd/soda-identity/src/wire.rs).
- **Tests:** [internal/strictjson/decode_test.go:15](../../../../internal/strictjson/decode_test.go#L15) — Duplicate/unknown fields, object shape, UTF-8 and size-limit tests; [scripts/wire_contracts_test.go:50](../../../../scripts/wire_contracts_test.go#L50) — Recorded wire vectors and limits; actual Serde caller admission and producer-byte checks recorded in [L04](../library-adoption.md#l04-json-and-base64-profiles). Retired engine-only vectors remain historical evidence.
- **Unclear boundaries:** Similar parser/DTO code is not proof of equivalent semantics or a justified shared abstraction. Caller-specific fields/limits stay with their slices; fixture agreement can preserve an obsolete assumption and needs contract review.
- **Evidence status:** L03/L04 source boundaries verified at their recorded scopes; historical structural findings and installed behavior retain separate status.
- **Validity review:** [H03 record](../reviews/H03.md) — actual scope, model, findings, challenge and target-allocation status.

## H04 Configuration and filesystem primitives

- **Validity review:** [H04 record](../reviews/H04.md) — actual reviewed scope, findings and pending completion dimensions.

Read protected configuration/inputs and perform caller-owned mechanical file/lock operations.

- **Entrypoints:** config.Load()/GrantKey()/Secret(); host config admission; filelock.Acquire(); native restricted file and directory helpers.
- **Owned data:** Admitted configuration values, descriptors, lock state and caller-selected protected files; No new policy, cleanup coordinator or recovery state.
- **Authority:** Host/operator-protected paths and exact caller effects; owning domain controls lock ordering and mutation authority.
- **Dependencies:** [H01](#h01-private-ipc-and-service-lifetime); [H02](#h02-storage-mechanics); [O02](operator-administration.md#o02-operator-identity-bootstrap); [O03](operator-administration.md#o03-existing-install-credential-maintenance); Native filesystem/advisory locks.
- **Source files:** [internal/config/config.go:163](../../../../internal/config/config.go#L163); [internal/filelock/filelock.go:15](../../../../internal/filelock/filelock.go#L15); `rust/soda-host/src/iconfig.rs (historical source locator)`; `rust/soda-host/src/tailnet_files.rs (historical source locator)`; `rust/soda-project-terminal/src/fs.rs (historical source locator)`.
- **Tests:** [internal/config/grant_key_test.go:11](../../../../internal/config/grant_key_test.go#L11) — Private/exact grant-key input checks; [internal/config/load_test.go:11](../../../../internal/config/load_test.go#L11) — Configuration size/trailing-input and private-origin cases; [internal/filelock/filelock_test.go:22](../../../../internal/filelock/filelock_test.go#L22) — Cancelled waits, reader/writer exclusion and descriptor errors.
- **Unclear boundaries:** Generic filesystem helpers must not take over caller-specific credential/identity/lifecycle semantics. Consolidation of Rust helper copies remains open until real callers and error/ownership behavior are compared.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

## H05 Branding, avatars and attribution

- **Validity review:** [H05 record](../reviews/H05.md) — actual reviewed scope, findings and pending completion dimensions.

Maintain established shared visual assets, deterministic avatar rendering and attribution.

- **Entrypoints:** avatar.Render(); public /-/soda/avatars/v1/ route; Branding render/build consumers; soda-avatars utility.
- **Owned data:** Canonical assets, style-v1 definition and versioned rendering snapshots; licenses/provenance; No alternate browser login/session or standalone Soda shell.
- **Authority:** Established presentation/asset contracts; native hosts retain their authentication and product authority.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [N01](networking.md#n01-private-origins-tls-and-activation); [D02](release-and-installation.md#d02-pinned-input-acquisition); [D03](release-and-installation.md#d03-candidate-production); Native Forgejo/Cockpit presentation.
- **Source files:** [internal/avatar/avatar.go](../../../../internal/avatar/avatar.go); [internal/web/avatars.go](../../../../internal/web/avatars.go); [assets/branding/theme/README.md](../../../../assets/branding/theme/README.md); [assets/branding/cockpit/README.md](../../../../assets/branding/cockpit/README.md); [docs/design/branding.md](../../../design/branding.md).
- **Tests:** [internal/avatar/avatar_test.go:36](../../../../internal/avatar/avatar_test.go#L36) — Deterministic versioned snapshots, bounds and concurrent rendering; [tests/forgejo/cockpit-branding.test.ts:8](../../../../tests/forgejo/cockpit-branding.test.ts#L8) — Canonical assets, local font/background/native-theme source assertions; [tests/forgejo/branding.test.ts](../../../../tests/forgejo/branding.test.ts) — Forgejo branding source assertions.
- **Unclear boundaries:** Domain UI behavior belongs to its capability slices. Shared artwork/rendering does not justify replacing native pages or authentication; visual/source checks do not establish installed browser correctness.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.

## H06 Developer tooling and verification infrastructure

- **Validity review:** [H06 record](../reviews/H06.md) — actual reviewed scope, findings and pending completion dimensions.

Support source/architecture checks, disposable fixtures, previews and reusable verification drivers.

- **Entrypoints:** bun check:source/typecheck/test; scripts/check-source.sh; Architecture checker, Lit analyzer adapter, preview/screenshot and image comparison utilities.
- **Owned data:** Source inventories, fixture vectors and disposable analyzer/preview/evidence artifacts; Manifests/locks describe existing tooling inputs; domain test expectations remain with their slices.
- **Authority:** Developer/operator task authority and existing tooling contracts; no qualification authority from a source-check PASS.
- **Dependencies:** All domain test slices; [D06](release-and-installation.md#d06-installed-qualification); Pinned Go/Bun/TypeScript/Cargo inputs.
- **Source files:** [scripts/check-source.sh:1](../../../../scripts/check-source.sh#L1); [scripts/check-lit.ts](../../../../scripts/check-lit.ts); [tools/lit-check/check.ts](../../../../tools/lit-check/check.ts); [internal/archcheck/arch_test.go](../../../../internal/archcheck/arch_test.go); [scripts/screenshot.ts](../../../../scripts/screenshot.ts); [tools/png-equal/main.go](../../../../tools/png-equal/main.go); [package.json](../../../../package.json); [Living ideal filetree plan](../ideal-filetree-plan.md) — Its 122 maintained Markdown sections own planning/coverage upkeep; [internal/factory/control/st15_demo_native_test.go:579](../../../../internal/factory/control/st15_demo_native_test.go#L579) — Native fixture broker process/file-log setup and cleanup.
- **Tests:** [tests/build/source_checks_test.go:142](../../../../tests/build/source_checks_test.go#L142) — Scripted command ordering, environment and stop-on-failure behavior; [tools/png-equal/main_test.go:14](../../../../tools/png-equal/main_test.go#L14) — Decoded-pixel equality and bounded image dimensions; [tools/lit-check/check.ts:67](../../../../tools/lit-check/check.ts#L67) — --fixtures mode asserts positive/negative analyzer diagnostics and missing-input failures; not run.
- **Unclear boundaries:** Tool tests prove their own asserted scope. Domain tests/drivers stay linked to domain slices; fixture source, source checks, browser execution and installed qualification are distinct. Analyzer docs are a coverage pointer, not a test result.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.
