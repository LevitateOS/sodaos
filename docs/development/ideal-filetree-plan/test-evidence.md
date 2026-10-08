# Tests, fixtures and evidence

Tests often reach real production code through controlled peers, but their
authority stops at that boundary. This review found one build-helper result bug,
an unpinned external-CLI source-test selection, and tests that encode the wrong
saved-key confirmation operation. Historical retirement guards and emitter
goldens also preserve implementation details that can be removed selectively.
Keep useful controlled failure cases and actual producer fixtures; do not replace
them with another test framework or infer installed correctness from counts.

Source: `a0bdba8434aeada2a966855695c8807734d9474d`, tree
`a6f6b26742391de46fa8278f9b41dd4e5394b581`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing
dirty guidance files and 39 dependency inputs retain their recorded bytes.
This refines [workflow requirements](workflow-requirements.md),
[observations](observation-reliability.md), [representations](data-representations.md)
and [dependency/architecture cost](dependency-and-architecture-cost.md).
The [tasks](implementation-tasks.md) and [lanes](implementation-lanes.md) remain
the execution plan. No application/test/fixture/dependency edits, executed tests,
builds, database, provider, VM, network or installed operations occurred here.

Discovery accounts for **956 test/fixture/support paths**: 315 command/host/native,
365 Go/browser, 225 release/acceptance and 51 coordinator/harness paths. It includes
named test/fixture subtrees, `_test.go`/`.test.ts` and inline Rust test modules,
including extensionless OCI blobs and Butane snapshots. These are search selectors,
not 956 test functions, body reviews or executed tests. Three Luna medium primaries,
independent cross-challenges and coordinator tracing inspected the defining cases
below; the local selector ledger records discovery separately from body evidence.
The complete responsibility inventory retains other fixtures/configuration duties;
this chapter does not claim a fresh body review of every test assertion.

## Evidence classes and actual entrypoints

| Evidence class | What must be known; what it supports |
| --- | --- |
| Source inspection | Exact assertion, production callee, input/fixture origin and substituted boundary; establishes what the test could exercise, not a performed result |
| Executed focused test | Exact selected case/tool/source/profile, actual assertions reached, refusals and failures; a fake peer remains fake even if the production callee runs |
| Development binary/build | Exact compiler/target/source and built binary used; compilation and local command execution do not establish the appliance, real provider or composed product journey |
| Candidate/native stage check | Selected matching substrate and candidate/source/archive binding; establishes those checks, not installation, routed clients or provider behavior |
| Installed evidence | Actual delivered versions, actor/authority, substrate, workflow, outcomes and cleanup observed; host file/service presence, asset serving and complete factory execution are different scopes |

[check-source.sh](../../../scripts/check-source.sh) verifies pinned Go, runs
`go test ./...`, then source gates and Bun typecheck/browser suites. Some Go tests
compile/run real Rust debug binaries through their helpers. It **does not run the
Rust unit/integration suite**; helper-driven Cargo builds are not Cargo tests.
The [Bun selectors](../../../package.json) distinguish prepared frontend/Forgejo
suites, local browser gates and explicitly selected installed assets. Build wrappers
prepare browser assets and terminal inputs; a prepared-suite invocation alone does
not prove fresh shipping assets.

[check-native.sh](../../../scripts/check-native.sh) requires exact clean source,
matching Linux architecture/tools and selected candidate inputs, runs the Rust
candidate validator and source checks, and rechecks source identity. It can compile
the validator; it does not produce the candidate, install or publish. The installed
[host probe](../../../tests/installed/host.sh),
[operator probe](../../../tests/installed/operator.sh) and
[Project foundation probe](../../../tests/installed/project-foundation.sh) describe
different service/file/label, operator and ordinary-member compiler duties.
Their guards and source definitions are not evidence that they ran. The subsequent
[build/installation join audit](build-and-operational-joins.md) traces those probes
against staged paths and packaged bytes: it records the obsolete libexec prefix
and distinguishes stored-image content checks from running-container identity.
The earlier test audit remains scoped to the assertions above.

Rust [broker tests](../../../cmd/soda-identity/tests/broker.rs) and
[enrollment tests](../../../cmd/soda-identity/tests/enrollment.rs) return normally
when disposable PostgreSQL inputs are absent: Cargo can count these cases as
successful without database assertions. Other PG cases are ignored by default;
Go ephemeral fixtures use test skips. Read exact invocation, output and fixture
admission before reusing a PG receipt. Preserve earlier configured PG proof at its
actual source/scenario; do not change every harness merely to increase a count.

## Production reach and substituted boundaries

| Current test family / defining examples | Production path, fixture origin and proof boundary |
| --- | --- |
| Identity broker/PG/provider tests; [Go/Rust compatibility](../../../internal/identity/client/broker_compat_test.go) | Current Controller/Store/PG and, in the compatibility case, an actual Rust broker process are exercised when configured. Seeded encrypted credentials, synthetic host callbacks and shell JSON-RPC/provider scripts replace external enrollment/execution. This can prove the exercised client/broker or transaction behavior; it is not real provider/auth-service proof |
| [Unix HTTP](../../../lib/unix-http/src/lib.rs), host identity transport, QMP/Phase/pump/redactor tests | Actual production modules run against local sockets, controlled clocks, readers/writers and failure hooks. Chunking, framing/caps, one deadline, buffered events, unterminated metadata, writer/pump failures and redaction regressions are useful local assertions. They do not establish an installed peer, QEMU guest or provider journey |
| Host Factory/terminal/Muse/Tailnet oracles | Current admission/argv/dispatch and socket adapters use mock native executors/hooks; terminal child tests actually launch and reap local children. Neither local direct-child proof nor a mocked supervisor proves Muse PID custody, container termination or listener admission under load |
| Installer/setup/trust and format tests | Pinned Caddy 2.10.2 root PEM verifies original DER/TBS; selected trust tests use actual OpenSSH-produced material when their producer runs. Other generated key/certificate/ELF cases are synthetic refusal/profile fixtures. Preserve raw fingerprints, malformed/weak/unknown-key/signature rejection and strict admission; a checked-in fixture is not live distribution/installation |
| Go API/Store/coordinator tests | Actual routes, domain logic and ephemeral PostgreSQL execute when admitted; session-extension callbacks, Host/Broker/Forgejo observations and mutation responses are often doubles. This proves selected SQL/policy ordering under supplied results, not the external producer's atomicity or complete factory execution |
| [Browser key controls](../../../tests/frontend/project-access-controls.test.ts), workspace/terminal/Forgejo components | Authored components or emitted assets run in fixture pages with local/intercepted APIs, socket simulators and gallery templates. Request/state/render assertions are useful. They do not prove real extension sessions, native authorities, deployed server-rendered pages or installed effects |
| Release build/deliver/image oracles and worker tests | Production parser, orchestration, record and command-runner functions are called. Fake cargo/podman/go/bun hooks and synthetic ELF/OCI outputs substitute native tools; captured Go archives are historical parser inputs, sometimes replayed as fake Podman output. Selected raw-byte/hash and admission assertions remain valid at their input profile; command plans are not production builds |
| [OCI gzip regressions](../../../lib/soda-release-build/src/oci/tests.rs), delivery document tests and asset/CLI tests | The EOF/trailer/member test calls the production archive scanner/decoder, not a copied decoder. Current document tests compare repeated writes and read back emitted content. Checked-in JSON/SVG/Butane inputs support parser/rendering/refusal/no-overwrite/mode checks; template byte snapshots are review aids, not external producer authority |
| [Go build harness](../../../tests/build/helpers.go) and native helper tests | Current Rust Project account/role/provisioning/domain binaries are built and invoked; native Git/command/system effects are often recording stand-ins. Other build tests read source tokens or execute selected shell fragments. Distinguish binary behavior, script dispatch, static wiring and actual installed commands |
| Installed browser/workload drivers and local emulator reuse | The [workspace scenario](../../../tests/installed/sodaspaces-workspace-journey.ts) can drive real UI writes with independent native observations when selected; local tests may supply MatrixNative doubles. Fake SSH/SCP/SFTP and workload commands prove orchestration/response interpretation only. The Lit installed case proves selected assets served in Chromium, not enrollment, terminal custody or factory/provider completion |

No new competing product owner follows from a test double. Retain Go domain/server/
Store/coordinator and Rust privileged/system/runtime boundaries. A test-only clone
of a parser or state machine cannot qualify its production counterpart; prefer a
direct current callee. A stub is appropriate for controllable refusal/race policy,
but cannot stand in for an unresolved upstream capability or native outcome.

## Concrete repairs and selective test retirement

### TEST-CARGO-RESULT-1 — empty stderr erases a failed build

`tests/build/helpers.go::CargoBinary` uses the captured stderr tail as both the
failure record and success discriminator. A non-nil `cmd.Run()` error with empty
stderr leaves an empty failure string, caches it as success and returns
`target/debug/<bin>`. A previous binary at that path can then run despite the
failed current build; without one, the failure appears later at process start.
This is independently confirmed source control flow, not a reproduced execution.

Retain the process error separately from diagnostics; only nil process error
counts as build success. Preserve the existing successful cache and package/bin
selection. No general build service or new test adapter is needed.

### P04-F1 — the tests confirm a different key mutation

[TestSavedKeyRemovalIsOwnOnlyAndNeverNativeRevocation](../../../internal/web/lifecycle_access_keys_test.go)
inserts one saved key and permits its deletion with an empty body. The browser
case expects an unconditional DELETE. `TestAccessKeyPreviewApplyAndLastKeyConfirmation`
instead checks `confirm_empty` when applying an empty **Project-installed key set**.
That is not the owner's required explicit confirmation before removing the final
**saved development key**. The tests can encode the missing rule rather than
detect it; title similarity cannot close this existing finding.

Refine existing P04-F1 at the frontend → authenticated DELETE → authoritative
Store mutation. Preserve owner-only deletion and the rule that removing a saved
preference does not revoke installed Project access. Confirmation admission must
use the state at mutation time, including concurrent removals. Keep Project apply
and saved-key removal as separate assertions; no new product decision is needed.

### TEST-SOURCE-CLI-GATE-1 — source selection includes an ambient external CLI

[muse_exec_test.go](../../../tests/build/muse_exec_test.go) belongs to generic
`go test ./...`. It locates an ambient Muse executable or a user-specific fallback
without version/source admission. Availability alone selects the echo and bad-auth
cases; the latter invokes a Meta provider/model with synthetic invalid credentials.
No explicit operational selector isolates this external path. Source does not
establish an offline preflight guarantee for that unpinned binary. No actual
network request, credential consumption or failed execution was observed here.

If retained, select this CLI check explicitly with a known binary/version and
declared effects under existing native-support conventions; remove the personal
fallback. Keep deterministic source fixtures in the ordinary source suite. An
actual CLI check should record its precise development/provider scope and must
not be promoted to broker custody or installed factory proof. Do not add a CLI
bridge, another model/provider adapter or a general test-selection framework.

### TEST-RETIREMENT-GUARDS-1 — historical paths and tokens freeze the cutover

The six test functions in [project_keys_test.go](../../../tests/build/project_keys_test.go),
[terminal_test.go](../../../tests/build/terminal_test.go) and
[project_os_observation_test.go](../../../tests/build/project_os_observation_test.go)
check absent Python/Go predecessors and text in current Rust files. Tokens such as
a function spelling, literal limit expression or comment do not execute key,
terminal or OS behavior, and can obstruct harmless successor refactors.

Their absent-Go/Python duties overlap the current no-Python gate and
[architecture test](../../../internal/archcheck/arch_test.go)::TestDependencyDirection.
That test checks Go boundaries; it does not prove the Rust replacement's behavior.
Select deletion of these redundant history/token guards after checking their
substantive assertions against current Rust behavior tests and shipping joins.
Retain canonical language/import/authority assertions and actual key/terminal/OS
regressions; do not add a subsystem to police each retired filename. Remaining
candidate/script wiring checks are static evidence and require their own current
purpose assessment, not blanket deletion.

| Packet / single accountable owner | Scope and prerequisites | Acceptance for later authorized implementation |
| --- | --- | --- |
| **TEST-CARGO-RESULT-1 / C**, build-test helper | Distinct process result plus existing cache/error reporting; exact failure branch established | Empty-stderr exit/exec failure fails before returning/dispatching a binary path and is never cached as success; process error survives; successful builds remain reusable |
| **P04-F1 / B**, saved-key API/Store/Spaces | Existing owner confirmation requirement; implement the current-state authoritative gate with its already allocated correction | Final-key deletion without confirmation refuses; explicit confirmation succeeds; concurrent changes cannot bypass admission; nonfinal/owner-only behavior and no native revocation remain; Project apply tests stay separate |
| **TEST-SOURCE-CLI-GATE-1 / C**, build/source test selector | Availability/personal fallback and unpinned external auth case; decide retention and exact selected CLI/effects before a native invocation | Ordinary source selection does not invoke an ambient provider CLI; retained explicit check uses the admitted subject and reports its scoped outcome; deterministic fixtures remain useful |
| **TEST-RETIREMENT-GUARDS-1 / C**, build-test history cleanup/L18 | Six named test functions only; compare canonical current ownership checks and current Rust behavior coverage first | Redundant history/token gates disappear; retain current Rust behavior tests and canonical no-Python/Go boundary assertions, without claiming fresh execution or a Rust ownership guard; no copied successor or new harness |

Use Luna low for the settled build-result join, explicit selector and redundant
guard removal after prerequisites; Luna medium for the P04 state/authority change
and independent review of each coherent packet. The coordinator owns explicit
staging, shared manifests/locks and resource-heavy checks. This audit does not
apply these source/test changes.

## Profile-dependent oracle cleanup and evidence labels

**TEST-REL-ORACLE-1 / C** refines **SIMP-REL-WIRE-1** and the representation review.
The source/test cuts in the [finding allocation](execution-findings.md) now
replace unsupported captured-Go emitter/error equality with current schema,
admission, semantic transition, read-back and deterministic-write checks in the
[delivery](../../../lib/soda-release-deliver/tests/oracle/main.rs), build and
image tests. Original signed inputs/digests and raw fingerprints remain exact;
delivery also retains current candidate binding-stage and nonqualifying-evidence
refusal assertions. Rendering an error through `Display` does not establish a
requirement for the old Go message.

The production oracle's command vector remains a current fake-runner assertion:
it records the production caller's tool/argument text and ordering, including
current Rust build selectors. This comparison does not preserve formatter output
or prove general argv boundaries, native command execution or installation.
Synthetic ELF and replayed OCI inputs are explicit in its source label. A
supported current Go SDK producer still owns its actual contract; historical
internal-emitter fixtures do not establish that producer. No new compatibility
model or test framework is introduced. C11.M-tests is complete at its allocated
source/test maintenance scope; C11.V/R04 qualification remains separate.

**TEST-RETIRED-PYTHON-OUTPUT-1 / B** similarly refines the existing Go probe
representation cut. Acceptance [workload tests](../../../internal/acceptance/workload_exec_test.go)
freeze old Python shell bytes even though the actual consumer is `sh -se`;
developer-access output tests preserve old JSON insertion order. Retire only
unsupported layout/order equality after checking any retained external evidence
consumer, while keeping required probe commands, fields and refusal semantics.
The migration missing-config no-op test is a separate **C** caller/contract
question; absence of a current service path alone cannot delete a supported direct
CLI contract. Do not group that question into a new compatibility obligation.

**TEST-ACCEPTANCE-SSH-EMULATOR-1 / B** is an evidence-label correction:
[TestRunDeveloperAccessEndToEnd](../../../internal/acceptance/developer_access_journey_test.go)
supplies fake SSH/keygen/SCP/SFTP, chosen PTY/UID/sudo/denial replies and local file
copies. Its real orchestrator can validly assert normal CLI success text and
artifact shape. Name/document the test as emulated orchestration and keep any
receipt at that scope; it cannot prove the printed native claims. No production
test mode or receipt framework is required. **TEST-REL-PRODUCTION-BOUNDARY-1 / C**
likewise labels fake-tool production sequencing; the broad Production bridge is
valid stage composition. Its labeling does not depend on or gate the separate
**COST-BOOTSTRAP-SEAM-1** cleanup.

## Regressions belong with their demonstrated defects

Carry existing acceptance into the corresponding fix, using current owners and
the smallest sufficient controlled failure. An absent case alone is not another
production defect or a reason to invent an exhaustive matrix.

| Existing correction / owner | Relevant test follow-through; present boundary |
| --- | --- |
| AUTH-I-ACQ-1/CLOSE-1, ID-CFG-01, RES-I-PROBE-OUTPUT-1 / A | Existing broker/provider/settings fixtures need the actual reserve/link and lookup failure windows, one-open pre-allocation cap, bounded/read-failed/stalled output checks alongside their fixes. Successful broker lifecycle and tiny version replies do not establish these branches |
| CON-M01/M02 / A; CON-G01/G02 and Factory authority findings / B | Existing listener/supervisor/coordinator fixtures should force the recorded custody, drain, shared-admission and stale/uncertain authority cases when corrected. Local child reaping or fake native responses cannot certify real process/container retirement or atomic native mutations |
| B03.C, L16.G, OBS-D01/G01/R01/W01/S01, RES-Go and snapshot custody / established A/B/C owners | Preserve exact cap+one/truncation, aggregate input/derived retention, phase expiry, output failure and cleanup/record semantics from their chapters. Tie each assertion to its producer and bound location; a downstream refusal does not prove pre-allocation admission |
| CUST-C-BUTANE-CLEANUP-1 / C | Existing lifecycle test covers unlink success and asserts “partial output removed.” Add focused failed/unconfirmed unlink and post-create clone-error ownership checks with the fix; no successful-removal claim without its observed result and no general fault framework |
| Completed L01/L02/L03/L05/L06/L08/L09/L12 and D05 EOF | Keep actual deadline/capture/redaction failures, key/signature/entropy admission, raw bytes, cancellation/transaction exclusion, upgrade/pump and same-file/EOF regressions. Removal of obsolete formatter/retirement tests does not reopen or weaken those completed duties |

Prior R03.L and adoption receipts retain their exact source/tool/profile/case
scope, including earlier build failures and later focused successes. Unchanged
application source permits bounded reuse for the same exercised unit and question;
it does not make a receipt a fresh run at this documentation HEAD or prove composed
installed behavior. Source/assertion/caller tracing and independent challenge are
complete for the listed cases; whole-suite behavioral coverage, held producer
profiles and installed journeys remain explicit. Counts, source snapshots and
reviewer agreement do not close those questions.
