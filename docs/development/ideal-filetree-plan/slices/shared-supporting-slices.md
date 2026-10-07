# Shared supporting slices

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## H01 Private IPC and service lifetime

Provide admitted private request/stream transport and explicit service startup/shutdown.

- **Entrypoints:** systemd socket activation; soda-host main; Go host/broker clients; private HTTP/mux dispatch.
- **Owned data:** Listener descriptors, admission/stream slots, in-flight requests and transient process/service handles; No repository policy or competing run ledger.
- **Authority:** Kernel peer/socket ownership and fixed operation admission; domain authority remains in consuming slices.
- **Dependencies:** [G01](forgejo-integration.md#g01-browser-authority-and-contributions); [G02](forgejo-integration.md#g02-background-service-admission); Native systemd/Unix sockets; Domain operation slices.
- **Source files:** [rust/soda-host/src/main.rs:224](../../../../rust/soda-host/src/main.rs#L224); [rust/soda-host/src/gmux_admission.rs](../../../../rust/soda-host/src/gmux_admission.rs); [rust/soda-host/src/gmux_server.rs](../../../../rust/soda-host/src/gmux_server.rs); [internal/host/client.go:56](../../../../internal/host/client.go#L56); [internal/identity/client/client.go](../../../../internal/identity/client/client.go); [appliance/services/soda-host.socket](../../../../appliance/services/soda-host.socket).
- **Tests:** [rust/soda-host/tests/gmux_smoke.rs:1048](../../../../rust/soda-host/tests/gmux_smoke.rs#L1048) — Scripted-backend route/parser/stream-slot tests and local Unix-server harness; [rust/soda-host/src/main.rs:389](../../../../rust/soda-host/src/main.rs#L389) — Inline flag/root/listener and exit-code contract tests; [internal/identity/client/client_test.go](../../../../internal/identity/client/client_test.go) — Broker client request/response tests.
- **Unclear boundaries:** Transport admission is not domain authorization. Mux smoke uses scripted backends and has stale introductory wiring prose; it does not alone prove the production backend, shutdown or installed daemon.
- **Evidence status:** Current source mapped; correctness and installed behavior unreviewed.
- **Validity review:** [H01 record](../reviews/H01.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## H02 Storage mechanics

Support current PostgreSQL connections, transactions, schema admission and mechanical integrity checks.

- **Entrypoints:** store.Open()/OpenEncrypted(); schema initialization; Broker Store.open_encrypted() and PostgreSQL connection implementation.
- **Owned data:** Connection/transaction handles, schema_version and encryption-key check metadata; Domain table rows remain assigned to their domain slices.
- **Authority:** Operator-configured restricted DSN/key inputs and database role permissions; callers own product decisions.
- **Dependencies:** [O01](operator-administration.md#o01-first-boot-database-provisioning); [O03](operator-administration.md#o03-existing-install-credential-maintenance); [I02](identity-brokering.md#i02-encrypted-credential-custody); Native PostgreSQL; Domain persistence slices.
- **Source files:** [internal/store/store.go:137](../../../../internal/store/store.go#L137); [internal/store/schema.go:12](../../../../internal/store/schema.go#L12); [internal/store/grants.go](../../../../internal/store/grants.go); [rust/soda-identity/src/pg.rs](../../../../rust/soda-identity/src/pg.rs); [rust/soda-identity/src/store.rs:61](../../../../rust/soda-identity/src/store.rs#L61).
- **Tests:** [internal/store/schema_test.go:11](../../../../internal/store/schema_test.go#L11) — Fixture-dependent current-schema creation, integrity and old/unversioned-store refusal; [internal/store/grants_test.go:11](../../../../internal/store/grants_test.go#L11) — Encryption-key continuity across store reopen; [rust/soda-identity/src/pg.rs:481](../../../../rust/soda-identity/src/pg.rs#L481) — Inline DSN/command-tag vectors.
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
- **Source files:** [internal/config/config.go:163](../../../../internal/config/config.go#L163); [internal/filelock/filelock.go:15](../../../../internal/filelock/filelock.go#L15); [rust/soda-host/src/iconfig.rs](../../../../rust/soda-host/src/iconfig.rs); [rust/soda-host/src/tailnet_files.rs](../../../../rust/soda-host/src/tailnet_files.rs); [rust/soda-project-terminal/src/fs.rs](../../../../rust/soda-project-terminal/src/fs.rs).
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
