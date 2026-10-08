# Resource bounds before allocation and expansion

At the inspected snapshot, four producer-side gaps remained: complete lease
enumeration, complete connection list construction, provider version-probe
capture and factory dependant traversal. The earlier aggregate secret-collection,
installed-probe and diagnostic-ring findings were also open. The ledger and
findings below preserve that source identity; later dispositions belong to the
[finding allocation](execution-findings.md).

Source: `88a40b5df6c146fa568e46bc97e6bcb39cab16e5`, tree
`22829c7af775066f65e9dcd4620e64d1ffdaa745`, inspected 2026-10-07. The intervening
authority documentation commit `4b308d13` changes no application source;
application source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`.
The 12 pre-existing dirty guidance files and 39 dependency inputs retain their
recorded bytes. This extends [observation reliability](observation-reliability.md),
[representation review](data-representations.md), [concurrency custody](concurrency-and-termination.md)
and [authority/state tracing](authority-and-state.md). The [existing tasks](implementation-tasks.md)
and [lanes](implementation-lanes.md) remain the execution plan.

This is source inspection and documentation. No tests, builds, database/provider/VM,
network or native qualification ran. Findings establish missing application bounds,
not an observed exhaustion event, hostile producer or production population size.

The later [file/descriptor/process custody audit](file-and-process-custody.md)
at `1d8c4e11` adds Identity settings admission before its existing cap and
Project snapshot stream-bound enforcement. These have separate IDs and owners;
the four producer-side packets here retain their original scope.

The selected A05 settings, provider-probe and lease/connection list corrections
are now source complete. Their final follow-up removes the unscreened relational
state result column from connection queries; the existing guarded JSON projection
retains relational state equality and strict typed decoding. The complete-list
API and existing 512 KiB response budget remain, with visible all-or-error refusal
and complete internal authority scans. The current 26-check receipt and its
limits are recorded in the [finding allocation](execution-findings.md). This
selects no population ceiling or continuation API and claims no global heap,
installed/native or live-provider bound. Factory total-visited/depth/query
admission remains the separate RES-GO-ACCEPTANCE-DEPENDANTS-1 profile decision.

## Boundary ledger

For each representation, distinguish input admission, allocation/append, expansion,
retention and emission. Count bytes and objects across the actual operation where
they coexist; a per-file, per-frame or serialized-output ceiling does not establish
one heap ceiling. A deadline bounds elapsed work only where it reaches that work;
it does not bound bytes produced during the interval. Refusal needs an owned error
and cleanup path, including an explicit incomplete result for partial traversal.

| Actual producer → consumer | Where the current bound takes effect | Refusal and remaining exposure |
| --- | --- | --- |
| Private files / Ignition → acceptance secret collector → evidence patterns | Private files and Ignition document: 1 MiB each via bounded reads. Inline gzip: 2 MiB compressed / 1 MiB decoded per item. Only **after** raw/trimmed/source/decoded/line variants are retained does `create_evidence` admit 16,384 patterns / 16 MiB aggregate raw plus escaped bytes. Its conservative sixfold escape preflight precedes escaped-copy allocation | Collection or pattern admission errors return before action capture/publication. Many files/items/variants can allocate before that aggregate gate: **L16.G** |
| Process bytes → RedactingWriter pending / URL / produced / tee buffers → file | Cumulative input and pending length are checked against 16 MiB before pending append; produced and URL-pending appends use checked limits; cumulative emitted bytes are checked before file write/tee append | Sticky write/flush/close errors propagate through capture joins to failed finalization. These logical buffers and cloned pattern sets coexist. URL parsing/sanitization constructs a temporary transformed string/vector **before** its length is checked for final append; bounded input does not prove that temporary is at most 16 MiB |
| Application DTO → structured evidence trees → compact/pretty output → observation | BoundedJsonBuffer refuses above 16 MiB before Serde append. Validation/depth/size estimates and output gates surround scrub/sort/render stages; several encoded strings and decoded trees can coexist | Evidence errors set failed evidence/outcome before pending/final publication. Encoded-byte bounds do not independently cap tree-node overhead, caller-prebuilt DTOs or aggregate retained files. **REP-ACC-EVIDENCE-ROUNDTRIP-1** already removes the redundant second compact encode/parse; preserve admission, scrubbing and finalization |
| QMP peer / Identity HTTP peer → application buffers | QMP 4 MiB before line append. Identity headers 8 KiB; cumulative body 512 KiB before application Vec extension; typed route admission repeats the body boundary. Listener permits 64 active connections (one accepted stream can wait for a permit) and 8 backend callbacks | Oversize messages/bodies fail the operation; HTTP returns the owned error. Hyper has already materialized each incoming frame before the application check. These limits do not bound SQL result cardinality |
| Credential/record files and protocol output → host/guest/terminal callers | Configured secrets 64 KiB, provider credentials 256 KiB, terminal records 4 KiB: bounded read with one-byte oversize detection before validation. Native terminal input 32 MiB plus oversize byte; native output frame 131,072 bytes before line append; eight queued output frames; protocol write 262,144 bytes. Go terminal WebSocket read 32,768 bytes, one queued control, 128 peers | Refusal returns through the owning protocol/process cleanup. Copies/decoded forms remain additional to wire bytes. Muse's aggregate worker/retained-handle gap stays **CON-M02**; finite frame sizes do not repair it |
| Factory log → tail API → browser renderer | Tail slice 24 KiB minus envelope, trailing window 256 KiB, cursor horizon 256 MiB. Browser pending counter covers 256 KiB of **raw frame bytes** awaiting xterm callbacks; diagnostic marker writes are separately produced | Tail/body/cursor or raw pending overflow refuses/terminates the attachment. Markers are not charged to this counter; current producer advances the cursor after truncation/gap, so a repeated marker-only stream was not established. Selected xterm 6.0.0's 50,000,000-byte discard watermark is a separate upstream safeguard |
| Browser HTTP / storage → UTF-8 / JSON / layout | `readSodaJSON` checks Content-Length then cumulative 64 KiB before decoded-text append, uses fatal UTF-8 decoding and parses JSON afterward. Layout keeps at most 64 entries/panes with storage admission. Snapshot requests admit seven families, 50 IDs/items, eight refs and 4,096-byte cursor; transport admits 64 KiB response bytes before typed decode/remarshal | Failed HTTP reads cancel/release the reader and surface error; invalid storage is rejected by its owner. Bounded encoded input can produce multiple decoded representations. Snapshot response byte admission is not an independent response-item count guarantee; no accepted oversized producer list was established |
| Go Store/factory views → API/browser | Existing views cap issues at 1,000, projects at 129 (128 plus overflow detection) and publications at 64. Identity client's 512 KiB response limit acts after Rust list production | These consumer/view safeguards remain. Complete connection production and acceptance-head/dependant scans have separate gaps below |
| Release process → retained capture / attached log / RecallLog | Runner capture is 16 MiB when no output writer is supplied; attached output instead streams to the log. Metadata deadline 120s, post-exit drain grace 2s. RecallLog keeps 20 completed lines but has no byte/unfinished-fragment cap | Runner capture/drain errors fail their caller. Streaming a complete log does not imply a disk quota. RecallLog clones/concatenates/splits before trimming; newline-free fragments and failure-reason copies remain **OBS-R01** |
| OCI compressed descriptors → decoded tar → requested content | Compressed layer/image: 1/4 GiB; decoded layer/image: 1/16 GiB; requested member: 512 MiB; layer/entry counts: 100,000. Bounded streaming readers reject excess consumption | Scanner errors fail release verification. Shared scan drains decoded tar padding to EOF while MultiGzDecoder is alive, then callers drain/hash the raw descriptor. Complete gzip trailer validation is present; historical D05 Q3 is not reopened. Streaming ceilings are not equivalent retained Vec allocations |
| Installer reachable OCI descriptors → retained JSON / entry metadata | Individual retained JSON blob at most 4 MiB; aggregate retained JSON 32 MiB. The next bounded blob is read/parsed **before** aggregate admission, allowing one bounded item of body-byte overshoot; its temporary decoded tree adds overhead outside that byte budget. Large blobs hash through 64 KiB scratch | Oversize/mismatch fails inspection. Loader follows validated reachable index/manifest descriptors, not unrelated directory entries. Descriptor JSON contributes to the aggregate budget; no unbounded arbitrary-file enumeration finding is supported |

Defining sources: [secret collection](../../../tools/acceptance/src/driver/inputs.rs),
[inline decoding](../../../tools/acceptance/src/trust/inline_data.rs),
[redactor](../../../tools/acceptance/src/evidence/redaction.rs),
[structured evidence](../../../tools/acceptance/src/evidence/store.rs),
[finalization](../../../tools/acceptance/src/driver/finalization.rs),
[Identity HTTP](../../../cmd/soda-identity/src/http.rs),
[terminal framing](../../../lib/host/src/terminal/native.rs),
[factory renderer](../../../frontend/spaces/sodaspaces-factory-screen.ts),
[layout](../../../frontend/spaces/sodaspaces-layout.ts),
[release runner](../../../lib/soda-release-image/src/build_runner.rs),
[OCI decoded scanner](../../../lib/soda-release-deliver/src/oci/layers.rs)
and [installer loader](../../../cmd/soda-install/src/oci/layout.rs).

## Four new correction packets

### RES-I-LEASE-ENUM-1 — scoped operations load every lease

[Store::leases](../../../cmd/soda-identity/src/store_leases.rs) queries all lease
rows into Vec<Row>, then decodes another complete Vec<Lease>. Revocation/retirement
need only scoped leases but filter after global loading; reconcile/sweep legitimately
need to consider all. Controller list serialization and success-response body
cloning add complete representations. The Store's 30s deadline is not a row/byte cap.

Filter scoped work in SQL and use bounded, stable iteration for global internal
work. Define mutation-safe continuation and partial-progress/error retention;
an unbounded error list can undo bounded paging. The public `/leases` complete-list
contract needs a supported inventory/continuation decision before changing its
interface. Do not invent public pagination or an arbitrary population ceiling.

### RES-I-PROBE-OUTPUT-1 — provider wait has no stdout admission

[run_capture](../../../cmd/soda-identity/src/providers/mod.rs) reads stdout to an
unbounded Vec, then constructs a String. Codex/Muse check the configured binary
digest before checking its exact version: these are pinned local CLI producers,
not arbitrary HTTP inputs. No oversized pinned producer was demonstrated.

The reader also ignores read errors; a successful child can expose an incomplete
prefix to the exact-version comparison. A matching prefix could be accepted.
After wait expiry, direct-child kill/wait is followed by an unbounded reader join;
a descendant-held pipe could outlast the wait. Neither OS failure nor hang was run.
Bound before append, propagate read/cap errors before comparison, and define reader
cleanup ownership. Keep fixed sanitized errors, binary pinning and existing timeout.

### RES-GO-ACCEPTANCE-DEPENDANTS-1 — traversal retains whole populations

[AcceptanceDependants](../../../internal/store/issue_controls.go) loads all
acceptance heads, performs one decision query per head and retains all matching
dependants. [assessCascade](../../../internal/factory/control/readiness.go) propagates
scan errors; [dispatch reconsideration](../../../internal/factory/control/dispatch.go)
currently ignores errors as best-effort work. Traversal's visited map also grows
with distinct reached nodes. Per-decision source limits do not bound these aggregates.

Process stable pages/results without simultaneous full head/dependant collections.
Paging alone does not bound a returned complete slice or the visited set; a strict
total-memory contract requires supported graph size/retention or externalized
traversal state first. Preserve deterministic sort/dedup and exhaustive eventual
reconsideration. Do not impose an arbitrary head/node cap or silently promote
best-effort dispatch into confirmed completion.

### RES-GO-IDENTITY-CONNECTION-LIST-1 — client refusal comes after production

[Store::connections/available](../../../cmd/soda-identity/src/store_connections.rs)
loads all matching rows, decodes all connections and additionally filters available
ones into another vector. [Routes](../../../cmd/soda-identity/src/http_routes.rs)
serialize the whole result and clone the response body. The Go client's 512 KiB
cap protects that client after server allocations. The browser-facing API currently
consumes the complete list; concurrency admission does not limit its cardinality.

Settle the supported inventory/output and caller completeness contract. Enforce
meaningful admission before duplicate full construction, or use explicit end-to-end
continuation if callers can consume it. This ID names its Go consumer profile;
the accountable producer owner is A, with a B API/browser handoff.

| Packet / single accountable owner | Scope and prerequisites within current plan | Acceptance checks for later implementation |
| --- | --- | --- |
| **RES-I-LEASE-ENUM-1 / A**, Identity Store/controller A05/L08 | Scoped SQL and global internal iteration; choose stable progress/error policy. Public list change depends on its caller contract; preserve grant revocation, terminal fence and custody | Bound page/result copies; consider every supported lease without deletion/mutation omissions; bound error retention or expose partial progress; incomplete scan cannot claim completed revocation/reconciliation |
| **RES-I-PROBE-OUTPUT-1 / A**, provider process owner A05/L12 | Select probe byte ceiling; cap before append, join/read-error handling and explicit cleanup allowance. Preserve pinned expected versions and fixed errors | Exact cap succeeds, cap+one/read failure fails before comparison; stalled output/retained pipe has owned cleanup within an explicit allowance after child wait; no unbounded capture, leaked reader or successful incomplete evidence |
| **RES-GO-ACCEPTANCE-DEPENDANTS-1 / B**, Go Store/factory control B03 | Stable paged consumption, sort/dedup and partial-result policy for correctness-critical versus best-effort callers; settle graph profile if total traversal memory is required | Bound intermediate retention; exhaustive eventual traversal for supported graph; incomplete critical scan returns failure; preserved dispatch attribution; demonstrate separately how result/visited retention is bounded |
| **RES-GO-IDENTITY-CONNECTION-LIST-1 / A**, Identity list producer; B caller handoff | Inventory/output contract before selecting admission or continuation; retain owner/project filtering and available-connection authority | Refuse before complete duplicate production or expose/consume explicit pages; cap+one visible to caller; no silent truncation, missing inventory or changed authorization |

Use Luna medium for these unresolved custody, continuation and admission contracts
and independent review; Luna low can perform settled repetitive transfers afterward.
Coordinate both Identity Store packets through one writer. No general queue,
pagination or resource-budget framework is selected.

## Earlier findings and conditional profiles

| Existing item / owner | Precise remaining cut, prerequisite and acceptance |
| --- | --- |
| **L16.G / C**, acceptance collector/admission | Charge aggregate input/count and raw, trimmed, source, decoded, line and escaped variants **before** clone/decode/retain. Set supported secret profile while preserving all required variants and leak-safe refusal. Repeated files/many inline items must refuse at cap+one before derivation; exact-cap variants still redact; refusal precedes capture/publication. This mandatory repair is independent of optional L16 matcher selection |
| **OBS-G01 / B**, installed Go probes | stdout/stderr bytes.Buffer capture is unbounded despite CommandContext; no process-group/WaitDelay drain contract. Select capture/cleanup profile at the existing probe owner; overflow/read/cleanup errors fail observation, never PASS or expected denial. Preserve bounded diagnostic content and descendant custody |
| **OBS-R01 / C**, release RecallLog | Choose retained diagnostic bytes/fragment budget and check before concat/split/reason copies. Large complete and newline-free lines stay bounded while the attached log receives complete output; preserve writer failures and useful failure reasons |
| **B03.C / B**, operator request admission | Existing 4 KiB LimitReader can synthesize false EOF. Cap-plus-one outer admission must precede strict decode; exact cap accepted, overflow refused, no duplicate ID/effect or successful truncated request |
| **CON-M02 / A**, Muse listener | Bound workers and reclaim completed handles during uptime, separately from frame bytes. Preserve request/FD/peer admission, timeout and final joins; silent peers/churn cannot exceed the chosen aggregate custody budget |

CoreOS curl writes header blocks to scratch outside redacting capture. Its body
cap, five redirects and 60s timeout do not establish a Rust-side header-byte cap;
[capture_fetch_with_writers](../../../tools/acceptance/src/coreos.rs) later uses
read_to_string without pre-read admission. **C** owns the conditional header/status
profile: settle supported/native header behavior, then bounded same-file read and
overflow refusal before parse. L01 pump/writer finalization remains completed;
no oversized native header response or publication defect was demonstrated.

External Production metadata is similarly conditional:
[verify_build_meta](../../../lib/soda-release-image/src/media_container.rs) reads
all `meta.json` and parses a HashMap before verifying five named image outputs.
No in-repository producer for that schema was found. **C** owns exact producer
schema/cardinality evidence, then pre-read/count admission preserving all five
hash checks. Missing native evidence holds this parser/profile decision only.

For tighter heap requirements, **C** must first define simultaneous evidence writer,
decoded-tree and URL-temporary profiles; encoded 16 MiB limits remain valid without
a global heap claim. Installer reachable-descriptor counts or earlier 32 MiB
aggregate admission are optional profile refinements, not a newly proven unbounded
directory scan. **A** can stream Codex's currently whole-file executable hash using
the existing Muse digest-reader pattern, preserving exact digest admission without
inventing an executable-size maximum.

Three Luna medium primaries and independent cross-challenges covered host/Identity,
Go/browser and evidence/release; the coordinator checked collector, URL temporary,
probe read-error and rendered-marker boundaries. Challenge narrowed OCI entry
claims and rejected a speculative marker-only unbounded stream. Recorded scope,
owners, prerequisites, refusal paths and later acceptance checks are complete;
native sizes and held product profiles remain explicit. Historical full-slice
validity, parked restructuring checkpoints and completed repairs are unchanged.
