# Library integrations through current callers

This is the current caller map for the library-adoption audit. It records the
selected dependency, the upstream capability actually used, the application
surface that remains, its production consumers and its test support. It makes
moved machinery visible alongside deleted engines. The existing [task list](../implementation-tasks.md),
[lane schedule](../implementation-lanes.md) and [adoption chapter](../library-adoption.md)
remain the execution plan.

The [concurrency and termination audit](../concurrency-and-termination.md) at
`94c40095` traces retained runtimes, tasks, transports, locks and children through
their actual callers. It preserves completed transport/capture repairs and assigns
separate dashboard, shared-admission, Muse and worker-cleanup corrections.

The [authority and state audit](../authority-and-state.md) at `88a40b5d` traces
current grants, atomic transitions, credential/signing custody and uncertain
mutation replay, preserving completed trust and terminal-fencing repairs.

The [resource-bounds audit](../resource-bounds.md) at `88a40b5d` distinguishes
pre-allocation admission from downstream refusal, retained/expanded copies and
aggregate producer results. Its four correction packets refine existing owners.

The [file/descriptor/process custody audit](../file-and-process-custody.md) at
`1d8c4e11` checks opened-file admission, confinement and cleanup result ownership.
It assigns four bounded corrections and a dormant-method retention decision.

The [dependency and architectural cost audit](../dependency-and-architecture-cost.md)
at `1cb4bbd8` checks selected feature/version/toolchain and shipping costs. It
assigns five narrow dependency, feature and bootstrap-interface cuts, while
preserving real adapter/state owners and conditional artifact-notice questions.

The [test/evidence audit](../test-evidence.md) at `a0bdba84` traces assertions
into production callees and distinguishes actual producer fixtures, controlled
peers, captured emitter goldens and source/build/installed evidence. It assigns
a build-result repair and narrow test-selector/retirement work while refining
the existing saved-key confirmation and profile-dependent regression scopes.

The [build/installation/operational join audit](../build-and-operational-joins.md)
at `bc28a07c` traces command discovery, staged service/configuration paths, browser
assets and candidate/media/installed identities. It assigns a stale host-probe
path correction and narrows running-service identity claims to actual observations.

The [total maintenance assessment](../integration-maintenance-result.md) at
`797e6ec8` compares removed duties with all remaining conversions, runtimes,
synchronization, errors, dependencies and operational work. It gives each of the
73 entries a recommendation and assesses local joins and held optional candidates.

The [independent consequential challenge](../independent-conclusions.md) at
`24fc3ea7` examines those recommendations and 75 canonical prior claims against
contracts and callers. It corrects five misplaced references and separates a
new CA algorithm-profile question from the current feature-only cut. Agreement
retains explicit source, behavior and producer-evidence limits.

The subsequent [selected-version adapter challenge](adapter-challenges.md) at
`2dc3bce9` evaluates these surfaces against upstream APIs and actual callers.
It selects typed PostgreSQL, PEM, descriptor-flag and SDK snapshot cuts; JSON
profile changes and upstream SDK transport capabilities have explicit
prerequisites. Synchronous/string interface alternatives and parked browser
ownership are assessed separately. This census identifies a retained duty,
not a requirement to retain the present wrapper.

The subsequent [representation audit](../data-representations.md) at `463ce772`
checks duplicate/alias/null/number and output rules against actual producers.
It adds concrete evidence/formatter/provisioning cuts and narrows shared-model,
ordered-tree and native-schema changes to explicit profile/ownership decisions.

Source: `f8b22b0bcf274f42bd6395a2ddf89872135ecb2d`, tree
`607899f44a2c3b1368aa09e0f2fa61c4fe703927`, inspected 2026-10-07. The application
source is unchanged from the recorded [baseline](../review-baseline.md). The
12 already dirty guidance documents and 39 dependency inputs retain their recorded
bytes. These four new H06 documents and their navigation/inventory updates are
an explicit documentation delta; they do not advance older behavioral or source
validity evidence.

## Map coverage

| Map | Current scope |
| --- | --- |
| [Cryptography, encoding and wire profiles](crypto-profiles.md) | 20 direct external Rust crates; calendar facade; selected standard IP and monotonic-time adapters; reverse callers of retained JSON, Base64 and SSH profiles |
| [Native transport, storage and release engines](native-engines.md) | 18 direct external Rust crates; Unix HTTP facade and four consumers; PostgreSQL row/parameter/transaction surface; WebSocket upgrade and pump; file/descriptor, CLI, archive and rendering consumers |
| [Go services, browser libraries and developer consumers](go-browser.md) | All current direct Go module/API integrations, standard-library replacement profiles, SDK and terminal reverse callers, browser runtime and declared developer dependencies, Go tool directives and dependency-only support |

The Rust union is **38 direct external dependency names**, with no overlap or
missing manifest name between its two maps. Versions come from current manifests
and locks, not the original replacement proposals. Direct requirements, enabled
features, coexisting transitive versions and test-only use are distinguished in
the entries. All selected Rust crate sources were available in the local Cargo
registry and inspected at their selected versions. That source identity supports
the stated capability; it does not prove every upstream implementation path.

Go has 47 explicit module requirements and an existing 90-module selection whose
dependency inputs remain unchanged. `pgx/v5` has direct application imports even
though `go.mod` labels its requirement indirect. Other dependency-only modules
are named with their versions; they are not counted as separately adopted Soda
engines. The SDK's nominal `v0.0.0` is replaced by the sibling checkout at
`c92db11c14b773c9cc20ccfa4b853b4c017e8717`, tree
`1791fdbb1adc97e86b14eda36475d89147531439`. Its authority and repository DTOs are
local SDK contracts, not Forgejo framework model types.

Go's module and shipping build select 1.26.7. Rust's `stable` channel floats:
the earlier observed 1.99.0 toolchain does not make standard-library behavior an
immutable Cargo dependency. Bun is pinned to 1.4.2. The browser map distinguishes
root TypeScript 7.0.2 from the Lit-check workspace's 5.9.3 and distinguishes
type-only xterm declarations from the locked JavaScript/CSS shipped at runtime.

## Application libraries and composition

These are source-owned libraries at the same revision, not extra registry
engines. A path dependency, a public reexport or a renamed helper is still
application code. The manifests establish package joins; the entries below
identify the actual consumers and retained duties. Detailed per-library profiles
and helpers are in the three maps.

| Source-owned library | Current surface and retained responsibility | Consumers and composition |
| --- | --- | --- |
| [soda-unix-http](../../../../lib/unix-http/src/lib.rs) | `request`, `Limits`, `Response`, `Error`; Hyper HTTP/1 connection/body handling through Tokio; Soda request admission, response budgets and error mapping remain | Exactly four production users: [identity runtime](../../../../cmd/soda-identity/src/runtime.rs), [host identity client](../../../../lib/host/src/iclient.rs), [host Tailnet native client](../../../../lib/host/src/tcontrol_native.rs), [factory CLI](../../../../cmd/soda-factory/src/main.rs). The native map covers their request/response conversions and socket/deadline custody |
| [soda-wire-time](../../../../lib/wire-time/src/lib.rs) | `parse`, `parse_nanos`, `format`, `compact_utc` delegate calendar/RFC3339 work to `time`; caller zero-time, precision and error policy remain | Identity wire DTOs, PG maintenance timestamps, Project terminal, host factory deadlines/identity/Tailnet/terminal, candidate naming and acceptance timestamps. The crypto map enumerates all ten production source consumers and test support |
| [soda-build-tools](../../../../lib/release-inputs/src/lib.rs) | Shared reader predicates and DTOs, `Elf64Le`/`elf64_le_header`, optional `trust_key::parse_p256_public_key`; typed P-256 parsing returns the original admitted DER; domain admission and raw fingerprint meaning remain | Host Muse validation; release build/deliver/image; acceptance and release-assets packages. Trust callers and their tests are in the crypto map; ELF readers, release-input DTO mappings and package edges remain application responsibilities |
| [soda-release-build](../../../../lib/soda-release-build/src/lib.rs) | Build-input, CoreOS, OCI inspection and production duties; shared library error and producer-profile conversions remain | Host [iconfig](../../../../lib/host/src/daemon/config.rs) calls `files::require_native`; release-tools [artifacts](../../../../lib/soda-release-tools/src/artifacts.rs) and [pipeline](../../../../lib/soda-release-tools/src/pipeline.rs) consume CoreOS/image/build DTOs and errors. Worker runtime has test-only CoreOS fixture imports |
| [soda-release-deliver](../../../../lib/soda-release-deliver/src/lib.rs) | Payload/document/admission and OCI layer mechanics; `Image`/`Payload`/model reexports; release JSON and error policy remain | Host iconfig loads payloads; Historical release-build consumers: `lib/soda-release-build/src/oci/content.rs` and `lib/soda-release-build/src/oci.rs` (source locators at this inventory’s original revision); release-tools [check CLI](../../../../lib/soda-release-tools/src/check_cli.rs) and pipeline consume admission, content, native, payload and document APIs |
| [soda-release-image](../../../../lib/soda-release-image/src/lib.rs) | Image production, staging, packaging and event duties; `ProductionInputs`, `Runner`, `Cancel`, `foreign::Production`, request/model/error types remain | Release-tools pipeline maps build inputs, cancellation, progress, secrets, image results and errors into image production; image oracle fixtures implement test production/media surfaces |

The release-tools pipeline is a significant conversion join: `BuildCoreOSImage`,
`BuildImage`, `DeliverImage`, `DeliverPayload`, permits/trust, secret files and
image production inputs are separate application models. Their existence is
recorded here without treating the separation as either a required contract or
an approved simplification.

## Local-library source joins

This source-use census closes the six declared path-dependency names against
the current tracked Rust callers. Source selectors include imports/reexports as
well as calls; the profile maps and composition table above give their retained
model and lifetime duties. Test-only ranges stay in the fixture column.

| Local package | Manifest consumers | Current production source references | Test/fixture source references |
| --- | --- | --- | --- |
| `soda-unix-http` | [cmd/soda-factory/Cargo.toml](../../../../cmd/soda-factory/Cargo.toml); [cmd/soda-identity/Cargo.toml](../../../../cmd/soda-identity/Cargo.toml); [lib/host/Cargo.toml](../../../../lib/host/Cargo.toml) | [cmd/soda-factory/src/main.rs](../../../../cmd/soda-factory/src/main.rs) `L207, L214, L220`; [cmd/soda-identity/src/runtime.rs](../../../../cmd/soda-identity/src/runtime.rs) `L29, L36, L42, L43, L44`; [lib/host/src/iclient.rs](../../../../lib/host/src/iclient.rs) `L492, L499, L505`; [lib/host/src/tcontrol_native.rs](../../../../lib/host/src/tcontrol_native.rs) `L63, L70` |  |
| `soda-wire-time` | [cmd/soda-identity/Cargo.toml](../../../../cmd/soda-identity/Cargo.toml); [cmd/soda-pg-maintenance/Cargo.toml](../../../../cmd/soda-pg-maintenance/Cargo.toml); [cmd/soda-project-terminal/Cargo.toml](../../../../cmd/soda-project-terminal/Cargo.toml); [lib/host/Cargo.toml](../../../../lib/host/Cargo.toml); [lib/soda-release-tools/Cargo.toml](../../../../lib/soda-release-tools/Cargo.toml); [tools/acceptance/Cargo.toml](../../../../tools/acceptance/Cargo.toml) | [cmd/soda-identity/src/wire_time.rs](../../../../cmd/soda-identity/src/wire_time.rs) `L47, L56, L63, L67`; [cmd/soda-pg-maintenance/src/lib.rs](../../../../cmd/soda-pg-maintenance/src/lib.rs) `L41`; [cmd/soda-project-terminal/src/timex.rs](../../../../cmd/soda-project-terminal/src/timex.rs) `L15`; [lib/host/src/factory/deadline.rs](../../../../lib/host/src/factory/deadline.rs) `L13, L30`; [lib/host/src/iclient.rs](../../../../lib/host/src/iclient.rs) `L102`; [lib/host/src/tailnet/domain/time.rs](../../../../lib/host/src/tailnet/domain/time.rs) `L5`; [lib/host/src/tcontrol_provider.rs](../../../../lib/host/src/tcontrol_provider.rs) `L339`; [lib/host/src/terminal/time.rs](../../../../lib/host/src/terminal/time.rs) `L3`; [lib/soda-release-tools/src/candidate_prompts/defaults.rs](../../../../lib/soda-release-tools/src/candidate_prompts/defaults.rs) `L56`; [tools/acceptance/src/timestamps.rs](../../../../tools/acceptance/src/timestamps.rs) `L8, L14` | [tools/acceptance/src/timestamps/tests.rs](../../../../tools/acceptance/src/timestamps/tests.rs) `L7, L9` |
| `soda-build-tools` | [lib/host/Cargo.toml](../../../../lib/host/Cargo.toml); [lib/soda-release-build/Cargo.toml](../../../../lib/soda-release-build/Cargo.toml); [lib/soda-release-deliver/Cargo.toml](../../../../lib/soda-release-deliver/Cargo.toml); [lib/soda-release-image/Cargo.toml](../../../../lib/soda-release-image/Cargo.toml); [tools/acceptance/Cargo.toml](../../../../tools/acceptance/Cargo.toml); [tools/release-assets/Cargo.toml](../../../../tools/release-assets/Cargo.toml) | [lib/host/src/muse/validate.rs](../../../../lib/host/src/muse/validate.rs) `L460, L463`; [lib/soda-release-build/src/coreos.rs](../../../../lib/soda-release-build/src/coreos.rs) `L103, L331`; [lib/soda-release-build/src/coreos_stream.rs](../../../../lib/soda-release-build/src/coreos_stream.rs) `L262, L268, L274, L326`; [lib/soda-release-build/src/elf.rs](../../../../lib/soda-release-build/src/elf.rs) `L5`; [lib/soda-release-build/src/files.rs](../../../../lib/soda-release-build/src/files.rs) `L133, L138`; [lib/soda-release-build/src/forgejo.rs](../../../../lib/soda-release-build/src/forgejo.rs) `L13`; [lib/soda-release-build/src/http.rs](../../../../lib/soda-release-build/src/http.rs) `L157`; [lib/soda-release-build/src/lib.rs](../../../../lib/soda-release-build/src/lib.rs) `L108, L109`; [lib/soda-release-build/src/live_inputs.rs](../../../../lib/soda-release-build/src/live_inputs.rs) `L226, L231, L244, L250, L256`; [lib/soda-release-build/src/production_images.rs](../../../../lib/soda-release-build/src/production_images.rs) `L9`; [lib/soda-release-build/src/production_inputs.rs](../../../../lib/soda-release-build/src/production_inputs.rs) `L10`; [lib/soda-release-deliver/src/buildx/mod.rs](../../../../lib/soda-release-deliver/src/buildx/mod.rs) `L11`; [lib/soda-release-deliver/src/lib.rs](../../../../lib/soda-release-deliver/src/lib.rs) `L75`; [lib/soda-release-deliver/src/model/trust.rs](../../../../lib/soda-release-deliver/src/model/trust.rs) `L51`; [lib/soda-release-image/src/model.rs](../../../../lib/soda-release-image/src/model.rs) `L44, L48, L52`; [lib/soda-release-image/src/model/trust.rs](../../../../lib/soda-release-image/src/model/trust.rs) `L119`; [tools/acceptance/src/coreos.rs](../../../../tools/acceptance/src/coreos.rs) `L14, L15, L16`; [tools/acceptance/src/native_phase.rs](../../../../tools/acceptance/src/native_phase.rs) `L240`; [tools/acceptance/src/report/mod.rs](../../../../tools/acceptance/src/report/mod.rs) `L20, L43, L48`; [tools/release-assets/src/fetch/muse.rs](../../../../tools/release-assets/src/fetch/muse.rs) `L11, L91`; [tools/release-assets/src/fetch/tea.rs](../../../../tools/release-assets/src/fetch/tea.rs) `L147` | [lib/soda-release-build/tests/oracle/inputs.rs](../../../../lib/soda-release-build/tests/oracle/inputs.rs) `L5`; [lib/soda-release-deliver/src/model/tests.rs](../../../../lib/soda-release-deliver/src/model/tests.rs) `L119`; [lib/soda-release-image/src/model/tests.rs](../../../../lib/soda-release-image/src/model/tests.rs) `L89`; [lib/soda-release-image/src/sys.rs](../../../../lib/soda-release-image/src/sys.rs) `L304` |
| `soda-release-deliver` | [lib/host/Cargo.toml](../../../../lib/host/Cargo.toml); [lib/soda-release-build/Cargo.toml](../../../../lib/soda-release-build/Cargo.toml); [lib/soda-release-tools/Cargo.toml](../../../../lib/soda-release-tools/Cargo.toml) | `lib/host/src/iconfig/mod.rs` (`../../../../lib/host/src/iconfig/mod.rs`; historical source selector) `L137`; `lib/soda-release-build/src/oci.rs` (`../../../../lib/soda-release-build/src/oci.rs`; historical source selector) `L239`; `lib/soda-release-build/src/oci/content.rs` (`../../../../lib/soda-release-build/src/oci/content.rs`; historical source selector) `L8`; [lib/soda-release-tools/src/check_cli.rs](../../../../lib/soda-release-tools/src/check_cli.rs) `L108`; [lib/soda-release-tools/src/pipeline.rs](../../../../lib/soda-release-tools/src/pipeline.rs) `L37, L38, L39, L40, L41` | `lib/soda-release-build/src/oci/tests.rs` (`../../../../lib/soda-release-build/src/oci/tests.rs`; historical source selector) `L6`; [lib/soda-release-deliver/tests/oracle/artifacts.rs](../../../../lib/soda-release-deliver/tests/oracle/artifacts.rs) `L3, L4, L5, L6, L7, L8, L9, L147, L154, L161, L168, L175`; [lib/soda-release-deliver/tests/oracle/fetch_state.rs](../../../../lib/soda-release-deliver/tests/oracle/fetch_state.rs) `L1, L2, L3, L4, L22, L38, L60`; [lib/soda-release-deliver/tests/oracle/main.rs](../../../../lib/soda-release-deliver/tests/oracle/main.rs) `L10, L11, L15, L16, L17, L114, L344` |
| `soda-release-build` | [lib/host/Cargo.toml](../../../../lib/host/Cargo.toml); [lib/soda-release-tools/Cargo.toml](../../../../lib/soda-release-tools/Cargo.toml) | `lib/host/src/iconfig/mod.rs` (`../../../../lib/host/src/iconfig/mod.rs`; historical source selector) `L141`; [lib/soda-release-tools/src/artifacts.rs](../../../../lib/soda-release-tools/src/artifacts.rs) `L331, L340, L346`; [lib/soda-release-tools/src/pipeline.rs](../../../../lib/soda-release-tools/src/pipeline.rs) `L27, L28, L32, L33, L36`; [lib/soda-release-tools/src/worker/runtime.rs](../../../../lib/soda-release-tools/src/worker/runtime.rs) `L419` | [lib/soda-release-build/tests/oracle.rs](../../../../lib/soda-release-build/tests/oracle.rs) `L12, L13`; [lib/soda-release-build/tests/oracle/inputs.rs](../../../../lib/soda-release-build/tests/oracle/inputs.rs) `L6, L7, L8, L11, L12, L13, L14, L15, L16, L103`; [lib/soda-release-build/tests/oracle/oci.rs](../../../../lib/soda-release-build/tests/oracle/oci.rs) `L5, L6, L99`; [lib/soda-release-build/tests/oracle/production.rs](../../../../lib/soda-release-build/tests/oracle/production.rs) `L5, L6, L7, L134, L141` |
| `soda-release-image` | [lib/soda-release-tools/Cargo.toml](../../../../lib/soda-release-tools/Cargo.toml) | [lib/soda-release-tools/src/pipeline.rs](../../../../lib/soda-release-tools/src/pipeline.rs) `L42, L43, L44, L45, L46, L47, L586` | [lib/soda-release-image/tests/oracle.rs](../../../../lib/soda-release-image/tests/oracle.rs) `L6, L18`; [lib/soda-release-image/tests/oracle/media.rs](../../../../lib/soda-release-image/tests/oracle/media.rs) `L2, L159, L167, L175, L183, L191, L199, L207, L215`; [lib/soda-release-image/tests/oracle/staging.rs](../../../../lib/soda-release-image/tests/oracle/staging.rs) `L55, L85, L93, L96, L99, L102, L105, L113, L116, L119, L127, L136, L143, L147, L153, L156, L168, L178, L184, L185` |

## What remains after adoption

| Area | Current evidence exposed by the caller maps | Question for the next simplification review |
| --- | --- | --- |
| JSON | Serde supplies syntax and codec machinery. Soda still defines strict/tolerant admission, integer/null/duplicate-field handling, raw-object/array visitors, ordered value trees, Go/Python string/number formatting and producer-specific output paths | Which policies are required by current workflows or actual producers, and which can callers consume directly from upstream? Changing signed bytes requires its own contract proof |
| Identity PostgreSQL | Tokio-postgres supplies connection/query/protocol machinery. `Dsn`, connection-driver ownership, `Row` getters, crate-visible `Param` encoding and `Store`/`Tx` remain | Which conversions serve domain admission or lifetime guarantees, and which duplicate library types or methods? Transaction exclusion and cancellation obligations survive a smaller surface |
| HTTP and terminals | Hyper/Tungstenite and Go HTTP/coder libraries supply protocol engines. Unix request limits, authority, upgrade read-ahead, terminal frames, output queues and pump ownership remain with application callers | Trace each retained layer to a real caller/authority/lifetime duty before consolidation; preserve the complete upgrade-to-pump join |
| Cryptography | Typed libraries supply curve validation, signatures, hashes, key/certificate formats and entropy | Preserve current key purpose, raw fingerprint bytes and signing authority while assessing profile conversion duplication |
| Release formats and process I/O | Library codecs and filesystem/syscall bindings replaced primitives. Archive metadata, selection, budgets, completion, descriptors, cancellation and evidence finalization remain | Separate producer requirements and resource custody from inherited format/process emulation; count transferred helpers and fixture copies |
| Go and browser composition | SDK-to-Soda models, Store injection, host terminal wrappers, Lit elements and narrowed xterm renderer types are explicit | Assess application/domain mapping separately from generic API emulation; a dependency declared for development can still feed a shipped asset |
| SQLite | One developer probe reads copied legacy dashboard database bytes; a compiled helper writes staged Forgejo test edges | Current reachability is distinct from product persistence. The retained observation assumption requires its existing workflow review; library presence does not establish it as valid |

The presence of a wrapper alone is not a defect. Conversely, calling a library
does not prove that the previous engine vanished. The named remaining symbols,
reverse callers, exposed types, formatter/visitor profiles and test fakes are
the inputs for that assessment; this step creates no new adapter or corrective
implementation packet.

## Proposed libraries that are not current integrations

| Historical proposal | Current disposition and exact retained owner |
| --- | --- |
| Aho-Corasick 1.1.5 / RED01 | Optional L16 adoption remains deferred; it is not a current production dependency. L16.G aggregate secret-input bounds remain a prerequisite. Current evidence redaction and completed L01/L12 repairs retain their scopes |
| rust-ini 0.21.3 / CFG01 | L17 native evidence and candidate assessment are complete; the candidate failed continuation admission and is not admitted. The current [Forgejo marker and path parser](../../../../cmd/soda-forgejo-domain/src/marker.rs) remains. Semantic fit holds this parser decision |
| postgres 0.19.14 / PG01 | Historical proposal superseded by selected tokio-postgres 0.7.18; use its current caller map and cancellation/transaction boundaries |
| ssh-key 0.6.7 / CF-03 | Historical proposal superseded by selected 0.7.0-rc.11; its actual feature/version profile is recorded in the crypto map |

Native path, formatting and process operations do not introduce another registry
library. Their selected standard-library and local policy owners remain in the
[responsibility maps](../coverage/maps/README.md) and the relevant native caller
records (PATH01, FMT01/CLI01, PROC01/PROC02, CF-08, KEEP01). This map does not expand
those replacements into an inventory of every unrelated standard-library call.

## Evidence and completion limits

Luna medium agents inspected caller bodies and profile/facade joins and performed
independent challenges across the Rust and Go/browser packets. Luna low checked
the dependency partition, exact version selections and unchanged baseline inputs.
Review corrected driver-registration users missed by imports, test-only calls in
mixed files, upstream versus Soda types and incorrect SQLite persistence labels.
Only tracked current source contributes to caller coverage; ignored historical
source copies are excluded.

The map records direct API-use source files and reverse consumers of the retained
integration surfaces, including public/module/crate types, reexports, conversions,
fixtures and developer-only use. Imports are seeds, not complete caller evidence.
Named selector/hosting joins and the independently challenged nontrivial profiles
are the evidence at this scope. Generic primitive use does not imply inspection
of every possible interprocedural execution path or transitive library body.

This step completes integration/caller accounting at the pinned source. It does
not close the workflow correctness audit, approve every retained adapter, regenerate
the complete desired tree or advance native qualification. Existing unresolved
requirements, correctness findings, parked seams and optional-library gates keep
their current owners. No application edit, dependency change, test, build or
native/provider operation is part of this step.

The subsequent [observation-reliability audit](../observation-reliability.md)
checks the deadline, capture, redaction and admission mechanisms used by later
evidence. Its findings remain separate from optional replacement decisions and
do not change this caller map's source pin or completed accounting scope.
