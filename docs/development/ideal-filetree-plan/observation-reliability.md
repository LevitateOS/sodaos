# Reliability of audit observations

This bounded source audit reviews the mechanisms that later checks use to observe
completion, failure and retained evidence. It carries forward **B03.C**, **L16.G**
and **H06-F1**, and records four additional source findings. Completed repairs
remain complete at their recorded scope. Optional matcher or library replacements
are separate decisions.

The later [concurrency and termination audit](concurrency-and-termination.md) at
`94c40095` freshly traces these deadline/capture owners and records OBS-W01's
conditional release-worker cleanup risk. OBS-D01/G01/R01 keep their existing
identifiers and scopes; the four findings recorded here are not counted again.

The later [resource-bounds audit](resource-bounds.md) at `88a40b5d` locates
L16.G collection admission and OBS-G01/R01 capture/diagnostic retention precisely,
separately from optional matcher replacement. It records four additional producer
result/capture gaps under their own IDs and preserves these earlier findings.

The later [custody audit](file-and-process-custody.md) at `1d8c4e11` adds
Project snapshot admission and enrollment/Butane cleanup-report corrections,
while preserving completed capture and same-descriptor repairs.

## Subject, authority and evidence

| Input | Identity / scope |
| --- | --- |
| Inspected source | `06f1db9147042508e9315ba98003c693cc66bb56`, tree `520b97284294df5eb3792067a496211c87d83c7d` |
| Application baseline | `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; later baseline, requirement, coverage and caller-map commits are documentation changes |
| Contracts | [Native support](../native-support.md), especially the evidence record, command/VM deadlines and producer versus qualification distinctions; [workflow requirements](workflow-requirements.md); [review guidance](review-assignments.md#guidance-conflicts-and-controlling-decisions) |
| Mutable guidance | The 12 pre-existing dirty plan/review inputs and 39 dependency inputs retain their baseline bytes; working hashes are in the ignored audit receipt |
| Review method | Three Luna medium primaries, cross-challenges and coordinator caller tracing; source and authored tests inspected; no tests, builds, VM, database or provider operations executed |
| Reuse | [L01](library-adoption.md#l01-deadline-and-evidence-repair), [L12](library-adoption.md#l12-file-fd-and-process-ownership), [B03.C](implementation-tasks.md#b03-dispatch-and-intervention), [L16.G](library-adoption.md#l16-evidence-matching), [H06-F1](reviews/H06.md) and their stated earlier receipts |

Matching source, contract and review question permits the bounded reuse below.
A completed historical test run is not a new execution receipt. The 80-slice
validity review remains pinned at `f7e9cf9d`; this chapter does not advance it or
establish installed qualification. H06 owns this audit document; the defining
application/tool responsibilities keep their existing owners.

## Reviewed observation boundaries

| Actual caller / consumer | Mechanisms inspected | Source conclusion |
| --- | --- | --- |
| Rust acceptance driver → local commands, CoreOS fetch and VM/QMP → finalizer | Phase propagation, child/pump ownership, EOF, both writer closes, sticky failures, status joins and publication | Earlier repairs remain present; SSH readiness and artifact-key fidelity have separate findings below |
| Private inputs / Ignition → known-secret collection → capture and structured evidence | Individual input admission, variants, pattern/output/pending limits, streaming URL handling and final leak scan | Downstream limits remain present; aggregate collection still precedes admission, L16.G |
| Optional operator Unix endpoint → strict decoder → factory actions | Authenticated peer/principal, outer body cap, exact-object EOF and request read errors | B03.C remains open; no authority bypass is inferred |
| Go installed probes → SSH, Git, psql and lifecycle snapshot | Context, local process/capture ownership, output checks and caller errors | Capture admission and descendant-held-pipe completion need correction, OBS-G01; developer scope remains distinct |
| Release image Runner → worker result / build log → producer receipt consumer | Concurrent drains, cancellation, capture cap, progress/log completion, receipt validation and cleanup joins | Earlier drain repair remains present; diagnostic RecallLog has a separate resource-profile gap, OBS-R01 |
| Explicit shipping Go inputs → complexity script → pre-commit/check consumer | Selected-file existence, pinned analyzer stdout/stderr/exit semantics and PASS decision | Existing H06-F1 remains open |
| Browser/matrix helpers and screenshot fixture consumer | Attachment/close, per-call SSH limits, bounded retained matrix text, awaited screenshot/metadata errors and actual callers | No additional supported finding; native-browser tests only cover preflight refusal, and screenshots label synthetic scope |

Domain assertions, all other checker implementations, remote/native completion
and unknown-secret discovery are outside this review. The native-browser helper
does not await exit after SIGKILL escalation; its only discovered runtime caller
is a preflight-refusal test. Matrix's outer poll deadline is checked between
individually bounded synchronous calls. Neither establishes a verified hard
cleanup/poll deadline for an installed journey. Its possible systemctl timeout
versus not-found ambiguity lacks a supported complete-output counterexample and
is held, not added to the correction queue.

## Preserved repairs and bounded reuse

| Completed subject | Current source and authored assertion evidence | Reuse limit |
| --- | --- | --- |
| Phase child deadline, D06-F2 / L01 | [Phase](../../../tools/acceptance/src/process/phase.rs) preserves a finite child of an unbounded parent and the earlier of two finite deadlines | Does not prove every caller passes/checks that phase; OBS-D01 is another caller boundary |
| Absolute QMP and VM pump custody, L01 | [QMP](../../../tools/acceptance/src/qmp.rs) checks one finite phase during connect, negotiation, writes and response matching, including buffered input; VM/process code joins owned pumps | No native QEMU execution in this pass |
| CoreOS EOF/finalization, L01 | [CoreOS](../../../tools/acceptance/src/coreos.rs) completes pump ownership and both writer closes before using buffers; metadata parsing accepts no trailing newline; injected pump/close assertions remain | Tests were read, not rerun |
| Command capture failure separation, L01 | [Command execution](../../../tools/acceptance/src/command/execute.rs) joins capture and close errors, discards incomplete buffers and preserves evidence failure separately from native exit | Expected denial cannot make a capture failure a completed observation |
| Streaming confidentiality and budgets, L01/L11 | [Redaction](../../../tools/acceptance/src/evidence/redaction.rs) admits patterns and bounds input, expanded output, pending/URL buffers and tee retention; malformed/slashless and byte-split URL assertions remain | Known-secret and declared URL-component profiles only; no universal unknown-secret claim |
| File and structured-output custody, L12/L01 | Same-descriptor bounded private reads; [evidence store](../../../tools/acceptance/src/evidence/store.rs) bounds serialization, propagates write/sync/leak errors and links the final record exclusively after checks | These downstream bounds do not bound earlier collection or recover already lost map entries |
| Image child drain / cleanup, D03-F1 | [Build runner](../../../lib/soda-release-image/src/build_runner.rs) drains both pipes concurrently, bounds captured stdout, handles cancellation and refuses stalled descendant drain | RecallLog retention has its own gap; a producer receipt is not installed qualification |
| SDK request/manifest admission, L15 | The earlier exact SDK source check at `c92db11c` established cap+one, read-error and exact-object admission for its 64 KiB request / 1 MiB manifest profiles | No SDK upgrade, caller-authority or native proof is inferred; B03.C is a different outer limiter |

## Findings and correction boundaries

### B03.C — operator body limiter creates an artificial EOF

[Operator handling](../../../internal/factory/control/operator.go) wraps the body
in `io.LimitReader(..., 4096)` before the strict decoder. A valid authorized
command followed by whitespace filling the first 4 KiB can be admitted even when
additional bytes or a later reader error follow. The decoder sees the limiter's
EOF. Malformed or incomplete prefixes still fail. The optional endpoint retains
its private socket, peer UID/principal and action checks; no tracked current
service enabling it was established in this bounded trace.

Carry forward the [recorded L15 concern](library-adoption.md#l15-sdk-input-admission)
under **B / B03.C**. Read at most cap+one and refuse overrun/read failure before
the existing schema decoder. Preserve exact-cap success, request ID, principal
and status/stop/reconcile behavior. This narrow repair does not wait for the
separate factory authority/transaction decisions in B03.C.

### L16.G — secret variants accumulate before aggregate admission

[Input collection](../../../tools/acceptance/src/driver/inputs.rs) bounds each
private file but retains raw/trimmed copies across repeated inputs. Ignition
collection first clones storage-file entries, including unrelated entries, then
derives hash/source/decoded/line variants. The driver only afterward reaches the
16,384-pattern / 16 MiB metadata gate. Individual caps therefore leave aggregate
allocation and derivation unbounded before refusal.

Carry forward **C / L16.G** with its existing H06/D06 consumers. The collection
count/byte profile remains pending. Once defined, admit before retain, clone,
decode and derive, preserving every required variant and private-file custody.
Refusal must occur before capture/publication, with generic errors and no silently
omitted patterns. This repair benefits the current matcher and does not depend on
Aho-Corasick.

### OBS-D01 — SSH readiness accepts success after phase expiry

[Remote readiness](../../../tools/acceptance/src/command/ssh.rs) calls `ssh_true`
before checking its phase and immediately accepts success. Each attempt has its
own 12-second timeout; retry sleep also ignores remaining phase time. An expired
phase can start another attempt, and late success can reach VM `boot-ready.txt`
and a completed observation through the normal driver/finalizer chain.

**C** owns this H06/D06 follow-up through **C11.C/V**. Keep the existing SSH
arguments/host-key policy and local process owner. Check the phase before start
and before accepting success, bound each attempt/retry by it, and stop/reap/join
the owned client on expiry/cancellation. Local SSH cleanup does not prove remote
host work stopped. No late-success runtime reproduction was performed.

### OBS-S01 — redacted artifact keys overwrite a requested hash

The driver hashes repeatable `--artifact-file` inputs under literal path keys.
[Finalization](../../../tools/acceptance/src/driver/finalization.rs) then redacts
those keys and collects them into a `BTreeMap` without collision detection. Two
distinct neutral example paths ending in `token-one.bin` and `token-two.bin`,
with those exact known patterns, both become `[REDACTED].bin`. One digest is lost
before the structured sanitizer's key-collision guard sees the map. A quiet
successful command can pass the later leak scan and publication checks.

**C** owns this H06/D06 follow-up through **C11.C/V**. Keep artifact-reference
cardinality or refuse the transformed collision as an evidence failure before
completed publication. Prefer the existing fallible structured-scrubbing owner
over duplicate pre-scrubbing that discards information. Preserve the policy for
repeated identical original paths. This is evidence completeness loss supported
by source tracing, not a reproduced secret leak or native qualification failure.

### OBS-G01 — installed-probe capture is admitted after retention

[runBoundedDirEnv](../../../internal/acceptance/installed.go) gives stdout and
stderr unbounded `bytes.Buffer`s and uses `exec.CommandContext` without owned
group/drain completion or `WaitDelay`. A descendant retaining a pipe can keep
`Run` waiting after the direct child is killed. Lifecycle's 8 MiB stdout check
comes after capture; it cannot bound allocations and does not cap stderr. Actual
SSH, Git and psql consumers share this helper.

**B** owns the Go correction, handed off from **C11.C/V / H06**. Establish the
caller-appropriate stdout/stderr profile and enforce it during retention. Carry
the absolute deadline through wait, drain and cleanup; assess the existing
[`internal/acceptance/process.go`](../../../internal/acceptance/process.go)
owner before adding mechanics. Preserve native exit/expected-denial versus
transport/capture failures. `soda-installed-probes` is developer support; the
SQLite lifecycle fixture is not a current PostgreSQL qualification oracle.

### H06-F1 — analyzer failure can still produce complexity PASS

[The complexity gate](../../../scripts/check-complexity.sh) masks analyzer status
with `|| true` and decides PASS from stdout. Pinned gocyclo v0.6.0 uses exit 1 for
both a valid violation report and a fatal parse error; missing-path analysis may
log and continue with exit 0. Thus an operational/parse error with no report can
become PASS on the explicit-file hook path. The default path's stale absent root
can fail earlier; it does not repair the explicit-file path.

Keep the existing **H06-F1 / C11.C/V / C** correction. Require actual selected
inputs and completed analysis, distinguishing valid violation output from tool
failure. Removing `|| true` alone does not settle the semantics. The historical
injected tool-error → gate-PASS receipt remains evidence for its original subject;
only source and pinned analyzer semantics were rechecked here.

### OBS-R01 — diagnostic recall caps line count, not retained bytes

[RecallLog](../../../lib/soda-release-image/src/recall.rs) keeps 20 completed
lines but no byte cap on lines or its partial fragment. Each write clones the
fragment and concatenates input before splitting. A newline-free child stream
therefore grows retained memory and repeatedly copies it; a few enormous lines
also evade a line-count bound. The actual image runner sends stderr and ordinary
build output through this owner even when captured stdout has a separate cap.

**C / C08 follow-up / D03** owns the diagnostic resource profile. Select a finite
in-memory line/fragment byte budget, then bound before concatenation/copying while
preserving bounded recent failure context and the streaming build log. The exact
budget remains unresolved; acceptance's 16 MiB evidence limit does not establish
a build-log requirement. No OOM or false successful qualification was reproduced.
The completed D03-F1 concurrent-drain fix remains preserved.

## Prerequisites and acceptance for later authorized repairs

These are correction specifications within the existing task/lane plan, not a
second implementation schedule. Each row has one accountable owner. Source
findings alone do not authorize edits or native operations.

| Finding / owner | Prerequisite and exact correction scope | Acceptance on the real caller chain; not executed here |
| --- | --- | --- |
| B03.C / B | Settled existing 4 KiB/schema contract; outer operator body admission only | Exact cap succeeds; cap+one with valid padded prefix refuses; late read error refuses; principal/actions/error mapping remain |
| L16.G / C | Define aggregate collection count/byte profile; collector → admission before derived allocations | Repeated files and many Ignition variants reach exact limits/refuse overrun before retention/capture; all required variants covered; no secret-bearing refusal |
| OBS-D01 / C | Existing Phase and local process custody; readiness attempt/retry and direct VM outcome joins | No child start after expiry; delayed success/cancelled retry cannot produce ready/completed; hanging client is reaped within bounded cleanup; timely success/arguments remain |
| OBS-S01 / C | Existing artifact/hash and fallible scrub/publication contracts; finalizer map conversion | Two distinct redacted-colliding paths cannot publish completed with one hash lost; distinct paths preserve both; generic failure remains leak-safe |
| OBS-G01 / B | Define stdout/stderr profile; reconcile existing process owner with actual probe callers | Bound before retained copies; exact-cap/overflow on both streams; descendant-held pipe and cancellation complete within budget; capture failure cannot become PASS/expected denial |
| H06-F1 / C | Pinned analyzer report/error semantics and current explicit input selectors | Valid clean/violation results differ from parse/tool/missing-input failures through actual script and hook; no failed analysis produces PASS |
| OBS-R01 / C | Define diagnostic byte/fragment profile; existing RecallLog → Runner failure context | Newline-free chunks and oversized complete lines keep bounded memory/context; stderr path included; log-write/cancel errors propagate |

Checks should use existing subjects and the smallest sufficient focused fixtures.
One independent Luna medium review covers each consequential deadline, custody or
evidence correction; settled metadata/plumbing can use Luna low. The coordinator
owns explicit-path staging, dependency/selector joins and expensive checks.
Do not uncheck completed C08/L01/L12 work to represent these separate findings.

## Observation use and remaining limits

Until its matching correction is verified, do not treat complexity PASS as
completed analyzer proof, late VM readiness as deadline proof, redacted artifact
maps as complete selected-artifact coverage, or Go probe capture as bounded by
its post-read cap. Do not claim bounded aggregate secret collection or diagnostic
recall before their profiles and checks are established. Independent checks that
do not consume these mechanisms can continue under their own prerequisites.

Finalization's existing write/close/leak/publication failures still propagate.
The producer build receipt can precede final log completion, but the actual worker
consumer propagates runner/cleanup failure before accepting its result; no
successful qualification path from an orphan receipt was established. Progress
events describe log-arrival windows, not CPU profiling or a qualification oracle.

The infallible `Evidence::redact_string` fallback remains a latent error-visibility
question. The proposed overflow example did not survive the complete leak-scan
path; **OBS-S02 was withdrawn** and is not a correction task. Any new claim needs
a reachable end-to-end counterexample. Optional L16 matcher adoption remains
deferred and does not gate any finding above.

Source primaries independently challenged deadline/redaction/capture joins;
OBS-S01 was checked through final publication, and the coordinator's analyzer,
RecallLog and receipt-consumer questions received a separate Luna medium source
challenge. Local ignored receipts live in
`.artifacts/observation-reliability-20261007-06f1db91/`. This chapter is the durable
finding/contract record; detailed receipts remain outside tracked documentation.
No full H06/D06/D03 validity review, runtime failure reproduction or installed
qualification is claimed.
