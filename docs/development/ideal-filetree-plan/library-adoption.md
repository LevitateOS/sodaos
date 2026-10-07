# Library adoption within the existing implementation plan

The selected priority is to repair concrete correctness defects and replace
handwritten generic infrastructure before further decomposition of that
infrastructure. The [task list](implementation-tasks.md) and
[lane schedule](implementation-lanes.md) remain the execution plan. This chapter
defines their library-adoption packets; it is not another queue or architecture.
L00 initial preparation, L01, L02, L03, L04 and L07 are complete at their
recorded source scopes. L04 used medium profile decisions and independent review,
then low caller transfers; every selected engine is retired. Later cutovers
retain their scoped authorization and admission checks.

Planning reconciliation uses source `72e4bb9015b6d6a622b45638104c74851a137473`
and SDK `86a70e1155f1036fdcd38f2af49d4ca6defa8280`. The completed
[investigation](../../research/library-reuse-investigation.md) and
[coverage ledger](../../research/library-reuse-coverage.md) are non-normative
evidence. Historical structural and 80-slice audit baselines remain distinct;
this is a selective reconciliation, not a full tree/count regeneration or new
installed qualification.

Completed M/C/V entries remain completed at their recorded scope. Pending
generic-engine splits below are superseded by the selected replacement, while
unrelated domain/lifecycle duties remain open. Extracting a parser did not prove
it correct; replacing it does not reopen a completed move. Preserve established
Go/Rust owners, provider-specific contracts, PostgreSQL, full Forgejo design and
existing processes. No sidecars, language ports or compatibility facades follow
from these packets.

## Execution packets

L identifiers are subpackets of the existing A/B/C tasks. Each has one lead;
physical writers remain the exclusive owners in the lane schedule. A lead sends
cross-owner changes through named handoffs rather than editing another owner's
files. The coordinator owns manifests/locks and integration. Packet state is
recorded in the task list: L00 initial preparation, L01, L02, L03, L04 and L07
are complete at their defined source scopes. Later unchecked packets remain
undispatched subject to their exact gates below.

| Packet | Lead | Existing task joins | Sequence and required output |
| --- | --- | --- | --- |
| L00 Admission and boundary preparation | Coordinator | R00/R01, C01/C02/C08 | Finish initial preparation before L02; retain admission per selected dependency/contract |
| L01 Deadline and evidence repair | C | C11.C/V | Immediate; preserves the existing small lifecycle/evidence owners |
| L02 Trust-key and signature repair | C | C07/C10 | Complete after L00; existing curve/DER libraries and raw-byte contracts preserved |
| L03 Hash, curve and randomness owners | A | A03/A05/A06, C10/C11 | Complete: RNG01 in `a84447ff`; CF-01 and CF-02 in `52eee7ee` |
| L04 JSON and Base64 profiles | C | C01, A05/A07, C09/C10/C11 | Complete: caller profiles verified; every selected engine and shared dependency retired, ending in `229e9cce` |
| L05 SSH formats | A | A01/A07, C07 | Requires its CF-01/02/04 profiles from L03/L04 |
| L06 Local CA parsing | C | C05/C07 | L02 and required L04 PEM/Base64 profile; no L05 dependency |
| L07 Native SQL parameters | B | C02, B01 | Complete in `d12bf6d3`; can overlap driver preparation and precedes L08 cutover |
| L08 PostgreSQL driver | A | A05, C02, B01 schema join | L07 plus demonstrated L00 driver deadline/transaction fit |
| L09 Unix HTTP and WebSocket engines | A | C02/C03, A07, C08 fixture join | L00 transport/upgrade proofs; consume only required serialization profiles |
| L10 External HTTP adapters | C | C05, C03 | Setup HTTPS can start independently; provider changes hand off to A |
| L11 URL, IP and time adapters | A | C03/C05, A05/A07, C11 | Caller-specific admission; no global codec or syscall prerequisite |
| L12 File, FD and process ownership | C | C01/C05/C06/C07/C08/C11, A07 | Same-FD bounds and cancellation first; rooted custody precedes temporary convenience |
| L13 Archive and release formats | C | A04, C08/C09/C10 | Trailer/budget repairs first; independent format units can interleave |
| L14 CLI and target discovery | C | C05/C08/C09/C11 | Selected CLI grammar or Cargo metadata contract and L00 admission |
| L15 SDK input admission | B | B06, C01 | Exact pinned SDK boundary; retain meaningful per-dial credential transport |
| L16 Evidence matching | C | C11 | L01 completion, bounded secret-input custody from L12 and declared escape/resource profiles |
| L17 Configuration evidence | C | C05/C09 | CFG01 blocked on native corpus; CFG02 retained independently |
| L18 Dead machinery removal | C | C05/C11, A/B handoffs | Recheck last callers at implementation revision; no substitute or blanket dependency removal |

### L00 Admission and boundary preparation

Scope: the selected packet's direct dependencies, feature graph, concrete
Rust/Cargo compiler, licenses, runtime and affected binary/cache selectors.
The current Rust selector is floating stable. Pin/record the actual compiler in
the later implementation, qualify the affected locked offline build and inspect
the worker cache/environment; cached source alone is insufficient. Do this per
adoption, not as a late release sweep. Existing-dependency repairs can proceed
without waiting for every proposed crate.

Prove the consequential boundaries before substantial replacement: the PG
operation deadline/cancel/discard path; Hyper Unix I/O, bounded blocking backend
admission and shutdown; HTTP upgrade plus first frame in one write, preserving
read-ahead into std/tungstenite ownership; and the nonblocking WebSocket owner's
wakeup, queued writes and close/reap. Use the smallest affected subject, not a
new framework or full release run. A failed proof holds that cutover and revises
its adapter before large migration. R03/R04 still own integrated/native checks.

Acceptance/output: record the selected compiler/dependency/features/license/cache
closure and affected locked offline build, plus actual boundary-proof results and
limits against the implementation revision. Admit or hold that exact cutover;
no unrelated adoption inherits a global preparation gate.

#### Initial preparation result (2026-10-07)

Complete against application source `4b02122b`; the sequence reconciliation
is `fcbcccb4`. No production source, Cargo manifest, workspace lock, toolchain
selector or installed payload changed. The coordinator ran isolated probes on
x86_64 Linux with rustc **1.99.0** (`b940084d7`, 2026-09-28), Cargo
**1.99.0** (`5f94df478`, 2026-08-27), GCC **14.3.1** and Perl **5.40.2**.
The floating `stable` selector remains visible; this records the actual compiler
rather than claiming a release-worker pin.

The [exact inventory](../../../.artifacts/l00/dependency-inventory/inventory.md)
and [package-level closure](../../../.artifacts/l00/dependency-inventory/resolved-graphs.json)
record direct versions/features, active transitive features, license expressions,
declared MSRVs, cache/archive checksums and affected package/bin selectors.
Isolated manifests/locks and receipts are retained under `.artifacts/l00` on
`/home`; their fingerprints and commands are recorded with this completion
commit. They are preparation artifacts, not new shipping packages.

| Isolated locked graph | Registry packages / active Linux nodes | Source + archive + lock checksum coverage | Result |
| --- | --- | --- | --- |
| Selected library candidates | 171 / 155 | 171/171 | Offline compilation and two typed-API probes pass |
| PG deadline adapter | 85 / 62 | 85/85 | Three real/socket/transaction probes pass |
| Hyper client/server + WS | 46 / 42 | 46/46 | Four Unix I/O/upgrade/owner probes pass |

Every active node has license metadata. Some omit a declared MSRV; compilation
on the recorded compiler supplies the local evidence. The highest declared
requirements are 1.88 for library candidates, 1.87 for PG and 1.85 for transport.
The ureq TLS graph uses rustls/webpki roots and bundled ring C/assembly, requiring
a C compiler for this packaged Linux graph. PostgreSQL and transport add an
in-process Tokio runtime. No new daemon or OpenSSL runtime is selected.
Artifact distribution still follows [licensing](../../research/licensing.md).

| Finding/boundary | Exact demonstrated subject and result | Adoption disposition and remaining owner |
| --- | --- | --- |
| CF-05/CF-06 → L02 | L00 P-256 SPKI roundtrip/off-curve refusal; canonical/redundant DER on P-224/P-256/P-384/P-521 typed signatures; **2/2** pass. Current release-image/deliver/install graphs plus soda-build/soda-candidate compile locked offline | **Complete** in `743dde17`: C's shared `soda-build-tools` trust adapter serves release-image/deliver; the installer uses strict typed signatures. Original DER/TBS, uncompressed-only policy, role separation and existing bounds are retained; actual regression and package evidence is recorded under L02 below. |
| PG01 → L08 | Fresh PostgreSQL **17.11** private Unix socket, NoTls/trust fixture: one absolute 5s connect/query/cancel/ack/discard/join/reconnect budget; actual SQLSTATE 57014 and new backend PID; 100ms silent-socket expiry; full transaction guard excludes competitor; **3/3** pass | Select **tokio-postgres 0.7.18** futures behind the A-owned small deadline facade. Hold a synchronous postgres-only Client migration: private blocking calls expose no operation deadline hook. L07 is complete in `d12bf6d3`; admitted DSN/auth/TLS policy, typed values/errors and actual Store/Tx integration remain L08 gates |
| N1/N2/N5/N6 → L09 | Hyper **1.12.0** `http1,client,server`, hyper-util **0.1.21** `tokio`, Tokio **1.53.2**; real Unix client/server, one admitted blocking backend plus rejected excess request and bounded shutdown joins | API/runtime fit proved. A owns actual listener/client/backend policy; C owns candidate fixture. Production request/header/body/admission bounds, cancellation and affected offline graphs remain L09 acceptance |
| N5 → L09 upgrade/session | Validated HTTP upgrade with first masked frame in the same write; assert all eight frame bytes in Hyper read-ahead, restore into tungstenite **0.30.0** `handshake`; one nonblocking owner, idle blocking/wakeup, short writes/WouldBlock without duplicate frame, automatic Pong, bounded close/child reap/slot release; included in **4/4** transport pass | Preserve one bounded production owner/queue and upgrade task custody. Slow-peer caps, real NativeAttach/error paths and integrated shutdown remain A's L09 acceptance |
| CLI03 → L14 | Exact humantime **2.4.0** source checked: no features, MIT OR Apache-2.0, declared Rust 1.60; candidate graph compiles offline | Metadata uncertainty closed. C still retains positive-duration/precision/overflow and command-tail policy at actual caller transfer |

At that L00 checkpoint, affected baseline package checks and development builds passed using
`cargo check` and `cargo build` with `--locked --offline -p soda-release-image
-p soda-release-deliver -p soda-install`, followed by `--locked --offline -p
soda-release-tools --bin soda-build --bin soda-candidate`.
The installer emits 17 existing unused-import warnings; no new probe warning or
compile failure remains. All nine preparation tests passed. The exact PG fixture
was stopped and its postmaster PID file is absent. Independent Luna medium
review passed the final proof subjects, receipts and affected plan changes;
Luna low handled the bounded inventory.

At that checkpoint, initial preparation was closed and L02 had not yet started.
L02 has since completed; LA-G1 continues at
each actual cutover for the changed workspace graph, selected compiler and
worker cache; local probe cache is not artifact-worker qualification. LA-G2/3
now have demonstrated adapter fit, while actual production acceptance remains
with L08/L09. LA-G4 caller profiles/Caddy corpus, LA-G5 native configuration and
LA-G6 external SDK scope remain explicit dependent holds. R03/R04 still own
integrated source and separately scoped native qualification. No readiness claim
is added for those future implementations.

For L07, the unchanged L00 PostgreSQL and HTTP proof receipts were reused at
baseline `921667ff` and reviewed without rerunning their harnesses. The PG proof
still selects the Tokio deadline facade and holds the synchronous-only driver;
actual Store integration remains L08. L07 itself completed in `d12bf6d3` with
native parameters across the Go and Rust product callers and no dependency or
schema change. Fresh PostgreSQL 17.11 caller tests and focused builds passed;
they establish this source slice only, not L08/L09 integration or native
qualification. The exact evidence and limits are recorded in
[the L00/L07 verification note](../../../.artifacts/l00-l07/verification.md)
and [SQL review](../../../.artifacts/l00-l07/sql-review.md).

### L01 Deadline and evidence repair

Scope/callers: acceptance `process/phase.rs`, `qmp.rs`, VM lifecycle,
`coreos.rs::fetch_capped` and `evidence/redaction.rs` through driver execution and
finalization. N14 keeps Phase/QMP as small std-backed policy; use a finite child
deadline with an unbounded parent and one absolute exchange deadline including
connect, buffered events, partial reads/writes and cancellation. RED01 first
repairs writer completion: join pumps, propagate errors, close both writers,
then inspect safe buffers. N13's immediate repair removes slashless sensitive
URL components and conservatively omits ambiguous malformed/binary spans. Its
planned url 2.5.8 parser replacement remains in L11's C-owned acceptance handoff.
C is the sole physical writer for the shared evidence file. The
[evidence guide](../native-support.md#evidence-records) owns resource limits and
capture completion policy.

Acceptance: parent/child deadline and cancellation direction; stalled/trickling
QMP, buffered event storm and bounded VM cleanup; newline-free metadata;
sticky pump/close errors; split/malformed/non-UTF8 URLs; bounded expanded output
and pending/tee buffers; no successful publication after incomplete evidence.
This does not depend on L16 automaton adoption or a new QMP runtime.

Immediate repair complete: Phase::child remains at `9fda53fe`; CoreOS capture
completion is `3873614f`, bounded redaction and capture failure propagation are
`d80aec93`, and absolute QMP deadlines plus VM pump ownership are `4c87f5e9`.
All 126 acceptance library tests pass, including local socket/shell regressions
for the acceptance boundaries above; all three acceptance binaries compile with
locked offline dependencies. Independent Luna medium review covered the changed
secrecy, deadline and ownership paths. This is local source verification, without
native QEMU or installed qualification. L11 parser adoption and L16's optional
matcher change remain open; no dependency or manifest change was needed here.

### L02 Trust-key and signature repair

Scope: release-image and release-deliver SPKI intake, installer
`x509/verify.rs` signature decoding and their trust/CA callers. CF-05 selects
existing p256 0.13.2 public PKCS8/SPKI APIs (spki/der transitively);
CF-06 selects ecdsa 0.16.9 typed DER signatures on all four installer curves.
The demonstrated P-256 tuple is `default-features=false` with
`arithmetic,ecdsa,pem,std`; generic ecdsa uses `verifying,pkcs8,std`.
Do not add direct spki/der dependencies unless the chosen adapter imports them.
Complete both trust-key intake owners and all supported signature curves. Keep
uncompressed-point admission, role separation, input bounds, raw
DER fingerprints and raw TBS verification. This is independent of full SSH/CA
parser adoption.

Acceptance: admitted producer keys, off-curve rejection, redundant INTEGER
rejection, wrong key/tampered signature and unchanged raw-byte fingerprints.
Delete the replaced coordinate-only/lenient decoding bodies and revise only
tests enforcing their unsupported permissiveness.

Status: **complete**, implemented in commit `743dde17` by C at Luna low. The
shared adapter is `lib/release-inputs/src/trust_key.rs`; release-image and
release-deliver use typed strict P-256 SPKI admission with original-DER
fingerprints and role separation. Installer signature verification uses strict
typed DER on P-224/P-256/P-384/P-521 and preserves raw TBS bytes. Independent
Luna medium source review found no blocker. The coordinator completed integration
checks and committed the repair.

| Completed acceptance evidence | Result |
| --- | --- |
| Trust foundation with `trust-key` | 24 passed |
| Release-image and release-deliver models | 5 + 5 passed |
| Installer X.509 | 26 passed, including valid signatures on all four curves, strict DER on all four, wrong-P256-key refusal and raw-TBS tamper detection |

These checks cover admitted producer keys, off-curve refusal, redundant INTEGER
rejection, wrong-key/tampered-signature behavior and unchanged raw-byte
fingerprints. The L00 2/2 API probe remains historical admission evidence, not a
substitute for these caller tests.

### L03 Hash, curve and randomness owners

Scope: the seven CF-01 SHA-256 definitions, host NIST validation (CF-02), and
RNG01 identity/host/terminal/Muse randomness callers. A owns host/identity/guest
primitive changes; C receives image-import, maintenance and acceptance changes
and owns the remaining entropy callers. Reuse
sha2 0.10.9, existing p256/p384/p521 and getrandom 0.4.3. Fail entropy acquisition
closed before other primitive work. Retain canonical grant/artifact/input
recipes, nonce/identifier formats and safe errors; entropy is not hashing.

Acceptance: canonical fingerprints/bindings and streaming boundaries; all three
host NIST curves with explicit uncompressed policy; acquisition failure cannot
return a usable predictable token or zero revision. Migrate one crate owner and
all its callers, then delete its engine. WS-only SHA-1 deletion belongs to L09.

Progress: **complete at the source scope**. RNG01 is complete in `a84447ff`: A changed host and
identity; C changed the remaining callers. All ten production packages now use
`getrandom::fill`; PID/time fallbacks and zero-revision defaults were removed,
and all four policy-revision callers propagate acquisition errors. `52eee7ee`
replaces all seven CF-01 SHA-256 definitions with sha2 and the CF-02 field
arithmetic with typed p256/p384/p521 point validation. Callers retain raw
fingerprints, sorted input recipes, read-error propagation and explicit
uncompressed-point admission. No selected SHA/NIST engine remains.

The primitive refresh passed eight focused SHA tests across six owners, all
319 host library tests, and locked offline development builds for all six
affected packages. Host coverage includes all three curves, wrong-curve and
off-curve refusal, coordinate bounds, point forms and SK-ECDSA admission.
Independent Luna medium review closed the initial coverage gaps; implementation
used Luna low. See the [primitive review](../../../.artifacts/l03-l04/primitive-review.md)
and coordinator receipts under `.artifacts/l03-l04/receipts`. These are local
source/development checks; installed and native-worker qualification remain open.

| RNG01 completed evidence | Result |
| --- | --- |
| Installer enrollment keys; worker runtime | 9; 19 passed |
| Host policy oracle | 57 passed |
| Identity entropy and crypto | 3 + 4 passed; the seal test overlaps |
| Host library entropy | 2 passed; its oracle also includes the revision test |
| Setup callers | 14 passed |
| Other affected caller package/CLI checks | 60 passed across candidate setup (26), lab credentials (19), identity-compose (10), PG fixture (3), and Forgejo migration (2) |

All-default locked-offline development builds passed for 12 affected packages,
including library and binary targets, host plus `forgejo-tailnet`, and all four
release-tools binaries. The installer build had 16 existing unused-import
warnings; one obsolete parser import was removed. Socket and worker executable
fixtures passed in exact local runs outside sandbox network/UID restrictions.
These results verify local development behavior. Installed appliance and
artifact-worker qualification remain open. Source and inventory workers used
Luna low; independent Luna medium source review found no blocker. Root owns
manifests, checks and Git.

Admission details and package selectors are in the [L02/L03 inventory](../../../.artifacts/l02-l03/inventory.md)
and [verification record](../../../.artifacts/l02-l03/verification.md), with
the exact commands and coordinator receipts. That x86_64 Linux refresh used
rustc/Cargo 1.99.0; selected versions were already cached, no registry version
changed, and the metadata matches the final manifest features.

### L04 JSON and Base64 profiles

Scope: JSON01's complete defining-family/caller table and CF-04's independent
codec table in the investigation, including shared `lib/json`, host/identity,
installer/setup/Muse, guest wire and release/acceptance owners. Serde 1.0.229 /
serde_json 1.0.151 own Rust syntax/binding/emission; base64 0.22.1 owns codecs.
C leads shared profiles; A/B implement their physical callers.

Declare decoded duplicates, depth/full-input limits, aliases/unknowns,
null/missing/numbers/bytes and deterministic output only where required. Define
distinct padding/CRLF/unused-bit/raw-fingerprint Base64 profiles. Preserve raw
signed inputs without reserialization. Go encoding/json admission remains
JSON02. Test consequential profile examples and current producer/consumer
round trips, then delete each replaced lexer/binder/emitter/codec owner. No
universal permissive decoder or foreign-diagnostic compatibility layer.

Progress: **source scope complete**. CF-04 completed in `26493cf2`; JSON01
finished with guest `1350250a`, Acceptance `747959ed`, bounded borrowed child
capture `3e1504ac`, and shared-crate retirement `229e9cce`. `lib/json`, its
workspace member, lock entry and every source/dependency consumer are removed.
No forwarding crate or replacement general parser remains. Luna medium settled
[JSON caller profiles](../../../.artifacts/l03-l04/json-profiles.md),
[host profiles](../../../.artifacts/l03-l04/host-json-profile.md) and
[encoding profiles](../../../.artifacts/l03-l04/encoding-profiles.md), and
independently reviewed the transfers; Luna low implemented settled callers.

Completed owner packets retain their distinct input and producer policies:

| Owner scope | Completed source and local verification |
| --- | --- |
| Identity/Setup, Compose, Muse and maintenance | `ae09f634`, `22c858fc`, `26493cf2`, `bf88640b`: strict request and typed caller admission, explicit Base64 profiles, exact producer checks, affected builds and review |
| Host | Earlier DTO/structural checkpoints remain recorded; `3bf7e75b` completes all engine callers and deletion. 317 library tests, 286 integration checks, build and review passed. Lower counts reflect deleted engine tests, not skipped application checks |
| Installer and release-build | `595fb604`, `d52d8ca8`: 128 installer tests; build 48 library plus 12 oracles; affected builds and review. Raw integer tokens, original hashes and required Go producer bytes retained |
| Activation, candidate/lab, import, console/factory | `eba27412`, `0cd73f16`, `c8997d1d`: 54, 14 and 12 checks respectively, builds and review; Python/Go property order, escaping, newline and custody remain caller-owned |
| Release-image | `9ee0a75b`: 62 library tests, 12 integration oracles, final stable-sort producer regression, build and review; typed first-exact/folded admission plus ordered duplicate/raw-number metadata and Ignition comparison |
| Delivery and release-tools | `c5fef89a`, `02a788be`, `b35ae13c`: delivery 34 plus 14 oracles, tools 94 plus 11 CLI checks, builds and review; ordered/raw policy merge and strict depth 100 closed, typed image/tools boundary retained |
| Release-assets | `b67d9d97`: 83 library and 31 integration checks, build and review; exact-last raw slots, ordered Butane document, original downloaded hashes and Python output |
| Guest | `1350250a`: 19 account, 67 factory and 93 terminal unit tests; 33 integration checks, build and review. Exact control-frame shapes, decoded duplicate policy, ordinary dictionary insertion position and Python producer bytes retained |
| Acceptance | `747959ed`: 131 tests and three binary builds, static/dynamic review; L01 absolute deadlines, pump/error/close custody, unterminated curl metadata, split redaction and failed-evidence publication checks remain passing |

Dynamic guest, image, provisioning and Acceptance trees admit at most **127
nested containers**; the root container counts as one and scalar roots as zero.
The guard refuses the 128th container before recursive conversion. This matches
ordinary pinned Serde admission and intentionally tightens the former parser's
incidental 10,000-container allowance. Deliver retains its separate strict
100-depth rule. Nested RawValue captures borrow original input rather than
retaining a full subtree copy at each level. Owner boundary and byte-preservation
checks passed after the final borrowed-capture change; evidence producers also
use a capped 16 MiB serialization sink before structural scrub and publication.
Signed bytes and fingerprints always use admitted original bytes.

The final locked offline graph builds guest, Acceptance, image, assets and tools
without warnings. The source/manifest/resolved graph census finds zero references
or edges to the retired crate. Existing producer oracles include five frozen
Go marshal goldens and the actual Channel manifest digest
`sha256:5203d05966d2a64d48b1f30f3ace177cf2402dec02c00553bee56633c088e986`.
The [verification record](../../../.artifacts/l03-l04/verification.md) preserves
receipts and failed diagnostic attempts separately. These are source and local
development checks, not installed appliance or native artifact-worker
qualification. Historical moves and audit findings retain their recorded
scope; unrelated correctness tasks and full tree regeneration remain parked.

### L05 SSH formats

Scope: host `ssh/`, installer `sshkey/authorized_keys.rs` and wire helpers,
through account/key-revision/enrollment callers. A leads host; C owns installer.
Select ssh-key 0.6.7 plus L03/L04 curve/hash/encoding decisions. Keep separate
algorithm/options/line/count/size policies and canonical fingerprint inputs.
Reject ssh-key KeyData::Other; preserve the current Ed25519 length-only policy.
Parsing is not NIST point validation or a new Ed25519 torsion policy.

Acceptance: all eight raw/eight certificate families with product-specific
allowlists, no options, malformed mpints/points/encodings, revision application
and fingerprints. Library certificate parsing does not confer trust/expiry
authority. Replace parsing and canonical serialization, remove mpint/certificate
format engines and unsupported equivalence-only tests together.

### L06 Local CA parsing

Scope: installer `setup/local_ca.rs → pemx → x509` through local CA trust
guidance. Select x509-cert 0.2.5 plus existing signature libraries after L02 and
the relevant L04 encoding profile. Keep the 16 KiB same-file bound, one public
certificate envelope, CA/basicConstraints/keyUsage and algorithm gates, raw DER
fingerprint and original TBS bytes. L05 is not a prerequisite.

Gate: prove pinned Caddy-root fixtures and define critical-extension admission
before deleting validators; strict DER serial/time behavior is an explicit
contract choice. Acceptance covers actual roots, signature/used-extension
failures, duplicates/trailing content and input custody. Remove the broad custom
certificate/DER/SAN/calendar/URL machinery after its last real caller is gone.

### L07 Native SQL parameters

Scope: Go Store and Rust identity placeholder translators and every traced
PostgreSQL query caller in SQL01. B owns Go SQL; A receives Rust changes.
Convert to native $n parameters, retain parameter order, transaction intent and
affected-row authority, then delete both translators. Use existing real query
subjects; exclude external Forgejo SQLite fixtures. No ORM or SQL parser.
This can run alongside L08 driver preparation.

Acceptance: the actual Go/Rust query subjects bind native $n with parameter order,
NULL/value types, affected-row decisions and transaction behavior preserved.
Remove both translators and recheck their caller/reference closure; fixture
SQLite behavior is outside this conversion.

**Complete in `d12bf6d3`.** Go uses direct `*sql.DB` and `*sql.Tx` calls;
Rust retains its parameter-encoding and query/transaction helpers while passing
native SQL through. The external SQLite seed remains unchanged. Luna low
implemented the conversion and independent Luna medium review found no blocking
SQL issue. The fresh PG17.11 fixture passed 68 Go store tests without skips,
eight Rust broker/enrollment integration tests, and the Rust bytea unit test;
Go architecture checks and affected development builds passed. No manifest,
lockfile, dependency or schema changed. Rust's per-operation transaction lock
hazard remains an L08 concern. See the linked verification note for exact
commands and scope.

### L08 PostgreSQL driver

Scope: identity `pg.rs`, `pg_dsn.rs`, `pg_query.rs`, Store/Tx and all store/controller
callers. A leads; B retains canonical Go schema ownership. L00 selects
**tokio-postgres 0.7.18** Config/typed bindings/rows/Transaction with an internal
Tokio 1.53.2 deadline facade inside the existing process. The synchronous
postgres 0.19.14-only path is held because its private blocking methods expose
no operation-deadline hook. Preserve the synchronous repository-facing boundary,
controller serialization, schema/CAS/sealing/events and safe errors; no broader
async conversion follows from the internal adapter.

Requires L07 and L00's demonstrated operation deadline/cancel/reconnect behavior
including relevant connection/auth/network/mutex waits. connect_timeout plus
statement_timeout is insufficient. A driver Transaction/connection guard spans
the whole closure. Acceptance: local TCP/Unix auth, typed NULL/bytes/bools/ints,
affected rows, rollback/commit/exclusion and recovery after error/cancellation.
Delete startup/auth/query/frame/DSN/text-conversion engines. The current Tx API
hazard was latent; this plan does not claim proven live transaction corruption.

Source scope completed in `6b18ee1b` using Luna medium. Four explicit fresh
PG17.11 tests prove SCRAM over Unix/TCP, typed nullable rows and scalar/JSONB
bindings, cancellation/discard/new session, transaction exclusion and recovery
after swallowed statement errors. Actual broker7/enrollment1 and affected HTTP
and client checks pass, with a warning-free locked offline build. Store keeps
the existing 30s budget; cancellation and driver join use a reserved slice of
that same deadline. The connection guard spans the full closure, and a failed
Tx cannot reconnect or report a successful commit. Production-generated Unix
DSN parsing is checked; pinned image defaults use Unix trust and loopback
SCRAM, while the disposable fixture proves SCRAM on both. No auth policy was
changed; installed/customized HBA and artifact-worker qualification remain
unverified. Detailed development receipts stay in `.artifacts/l08-l09/`.

### L09 Unix HTTP and WebSocket engines

Scope: host daemon/gmux/dbackend, four Rust Unix HTTP client engines, identity
listeners and candidate fixture. A owns host/identity/guest; C receives candidate
and factory/setup tool changes. Select Hyper 1.12.0 + hyper-util 0.1.21 + Tokio
1.53.2 complete framing and tungstenite 0.30.0, after L00's boundary proofs.

Migrate clients against existing peers; then identity ordinary HTTP, host ordinary
routes and the coupled host upgrade/pump. Fixture adoption can follow availability
of the shared adapter independently. Keep actual listener/socket authority,
route/body/header limits, synchronous facade and no replay of uncertain POSTs.
One nonblocking WS owner has a bounded child channel and readiness wakeup;
queued writes, automatic control replies, expiry, close/reap and inflight belong
to that owner. Preserve upgrade read-ahead. Stream fixture files inside the same
candidate process, with owned stop/join on early returns.

Acceptance: legal and malformed framing, cap/deadline exhaustion, a blocked
backend without IO starvation, request+first-frame together, concurrent output
and ping, slow peer/bounded queues, error-path custody and active shutdown.
Delete whole framing/frame/handshake engines and detached expiry, not just their
parsers. A repository-wide rustix or JSON migration is not a prerequisite.

### L10 External HTTP adapters

Scope: setup Forgejo client (N3) and host provider request recipes (N4). C owns
setup; A receives provider changes. Adopt already resolved ureq 2.12.1 with
purpose-specific Agent/redirect/proxy/TLS/compression and bounded body policy.
Keep distinct credential/status/execution contracts and uncertain key creation.
Setup HTTPS is independent of L09. Acceptance: real facade status/credential
cases, TLS rejection, body framing/caps and no secret-bearing argv/logs.
Declare the actual DNS/deadline guarantee. Acceptance curl remains an existing
engine with its L01 completion repair; do not replace that lifecycle by analogy.

### L11 URL, IP and time adapters

Scope: N7/N8/N9's complete caller tables, including host/guest/identity and C's
installer/setup/acceptance tool handoffs. C also owns the N13 evidence URL parser
replacement after L01's completed confidentiality repair. Reuse url 2.5.8 / percent-encoding
2.3.2, std IP and time 0.3.55. Keep raw lexical/admission guards, zones/prefixes,
strict wire timestamps, signed text and shared Linux monotonic origin.
Acceptance: admitted URL/loopback/IP forms, rejected controls/credentials/escapes,
embedded-IPv4 IPv6, timestamp grammar/expiry and no unsafe normalization.
Delete grammar/calendar engines; installer certificate-only URL/calendar removal
follows L06. N12's small purpose predicates remain retained.

### L12 File, FD and process ownership

Scope: FS01/TMP01/PATH01/PROC01/PROC02/FFI01/WALK01/N10 defining/caller tables,
including installer publication, private provisioning/evidence inputs, host/guest
FDs and existing release process runners. C leads operator/release changes;
A owns host/guest and B Go callers. Use selected rustix 1.1.5, existing libc 0.2.190,
std process/path, tempfile 3.27.0 and walkdir 2.5.0 where the actual contract fits.

Same-FD bounded read and confinement precede tempfile convenience. Preserve
creation modes, no-follow/inode/owner/link checks, synchronization and exclusive
publication; process cleanup names its actual child/group/systemd authority.
Keep the audited SO_PEERPIDFD call and meaningful NSS/signal/PTY policy.
Acceptance: same admitted bounded file, partial-write cleanup, late cancellation,
descendant-held pipes, bounded live capture, descriptor close/CLOEXEC and ticker
join. Delete generic PATH/errno/shell diagnostics and duplicate mechanics only
after equivalent callers move; no all-policy process/filesystem framework.

### L13 Archive and release formats

Scope: CF-07/08/09, REL01/02/03 and XML01, through Muse staging, delivery/build
OCI admission, terminal asset fetching and emblem rendering. C owns these;
A receives host's bounded ELF predicate. Reuse tar 0.4.46, flate2 1.1.10 and
sha2; select roxmltree 0.21.1/svgtypes 0.16.1 for token parsing.

Repair decoded EOF/trailer ownership and member/aggregate budgets first. Keep
deterministic metadata, exact-size FD reads, deadlines, original signed blob
verification, descriptor hashes/whiteouts and wanted-member policy. Consolidate
OCI low-level scanning into the existing delivery owner with an acyclic build
dependency. Terminal policy remains separate. Keep small 64-byte ELF purpose
gates and Soda's narrow renderer.

Acceptance: corrupt/truncated trailers, decoded budget exhaustion, OCI precedence,
duplicate wanted members, deterministic new tar output and old raw signatures;
namespace/DTD/unsupported SVG cases and canonical emblem output. Delete manual
writers/tokenizers and duplicate scanners; no legacy writer for old hash identity.

### L14 CLI and target discovery

Scope: shared release goflag and callers, acceptance driver options/duration,
installer fmtx literal callers, release-image sys target discovery. C selects
clap 4.6.7, humantime 2.4.0, Rust formatting and existing Cargo metadata.
Keep action/owner validation before effects, repeat/scalar/bool and literal
command-tail behavior, explicit positive-duration grammar/precision and safe
diagnostics. L00 verified exact humantime metadata and offline availability;
the actual caller's grammar and feature/build checks remain this packet's duty.

Use bounded cargo metadata --format-version 1 --no-deps with admitted manifest
identity, opaque IDs and intended bin/features/lock policy. Keep RUST_TOOLS
shipping destinations; metadata is not authority to install every target.
Acceptance: actual CLI invocations, tails/false/repeats/overflow, unchanged shipping
inventory and invalid/required-feature manifests. Delete generic flag/duration/
format/TOML emulators. CLI01's tiny selectors remain custom.

### L15 SDK input admission

Scope: pinned SDK runtime/manifest readers and Soda callback/background clients.
B consolidates on existing Go cap+one read then strict decode. Keep duplicate/schema
policies, error secrecy and per-dial peer/admission/rebootstrap transport duties.
Gate external SDK implementation on its exact pinned source and matching scoped
authorization; a Soda planning change does not mutate the sibling repository.
Acceptance: cap+one/trailing whitespace, read errors and unchanged peer/credential
custody. Delete the faulty limiter pattern, retain the useful transport adapter.

### L16 Evidence matching

Scope: acceptance evidence matcher, structured redaction and final leak scans.
C selects aho-corasick 1.1.5 after L01 and actual bounded secret reads in L12.
Declare deduplicated raw/escaped pattern and construction/output/pending budgets.
Preserve byte LeftmostLongest replacement with a bounded overlap adapter;
search the whole window so crossing matches cannot leak. Builtin streaming
replacement does not support that match mode. Retain Standard final leak scans,
sticky errors and private/exclusive publication.

Acceptance: split/overlapping/escaped/binary patterns, crossing-window starts,
tiny writes, resource bounds and close failures. Replace repeated matching and
sequential placeholder rewriting without weakening secrecy or cleanup. This
adoption does not hold the immediate CoreOS writer-finalization repair.

### L17 Configuration evidence

CFG01 is blocked on native Forgejo effective APP_DATA_PATH fixtures: quoting,
backslashes/case/duplicates/default/interpolation/environment precedence and
absent/relative refusal. C collects that bounded evidence and confirms or revises
preferred rust-ini 0.21.3 plus the effective-key adapter before implementation.
No guessed data path or expanded configuration engine. CFG02's byte-preserving
locale set admission remains retained; its original-byte/cap/hash/collision
policy is a different responsibility and needs no library cutover.

Acceptance/output: retain corpus inputs and native effective-key results, then
confirm or revise the selected dependency and small adapter against them, with
absent/relative paths refused. If evidence is unavailable, keep CFG01 blocked;
CFG02 admission and unrelated tasks remain independently ready.

### L18 Dead machinery removal

Scope: N11 daemon peer wrapper (A), TMP02 migrate temporary (C), DEAD01 orphaned
Go testoci fixture package (B). C leads explicit handoffs. Recheck references at
the implementation revision, move meaningful tests to active owners where
needed, then delete without replacement. Preserve actual Muse peer policy and
testify/other dependencies with remaining consumers. Acceptance is reference
closure plus affected existing package/source checks, not a new absence harness.

## Readiness gates

| Gate | Exact requirement | Holds only |
| --- | --- | --- |
| LA-G1 Dependency admission | Initial compiler/candidate closures and humantime metadata recorded above; requalify exact changed workspace features/license/cache and affected locked offline build on the selected worker | Each new adoption before its substantial cutover |
| LA-G2 Driver fit | tokio-postgres adapter deadline/cancel/discard/reconnect/exclusion proved locally; retain admitted auth/DSN/transport and actual Store/Tx integration checks | L08 cutover; not L07 or domain corrections |
| LA-G3 Transport fit | Hyper Unix client/server/backend/shutdown/read-ahead and WS owner fit proved locally; retain production framing/caps/slow-peer/lifetime and affected graph checks | L09 relevant server/upgrade cutover; not L10 HTTPS |
| LA-G4 Encoding/trust profiles | Actual producer semantics, original signed bytes, Caddy roots and critical-extension choice | Corresponding L04/L05/L06 boundary only |
| LA-G5 Native configuration | CFG01 native effective-config corpus | L17 parser selection only |
| LA-G6 External SDK | Pinned boundary and matching scope for sibling implementation/native evidence | L15 external change only |

Existing Q1–Q8 remain applicable to their domain corrections. Q9's old choice
to maintain custom chunk decoding is superseded by the complete-driver direction
in L09; its replacement still has framing/bounds acceptance. Readiness here is
a planning disposition, not implementation or operational authorization.

## Finding allocation

Every investigation finding has one primary packet/lead below. A phased finding
does not create a second owner. Linked research supplies defining files, complete
caller/profile tables and evidence limits; packet scope includes those tables.

| Finding | Disposition | Primary packet | Lead |
| --- | --- | --- | --- |
| [N1](../../research/library-reuse-investigation.md#n1) | REPLACE | L09 | A |
| [N2](../../research/library-reuse-investigation.md#n2) | REPLACE | L09 | A |
| [N3](../../research/library-reuse-investigation.md#n3) | REPLACE | L10 | C |
| [N4](../../research/library-reuse-investigation.md#n4) | CONSOLIDATE | L10 | C |
| [N5](../../research/library-reuse-investigation.md#n5) | REPLACE | L09 | A |
| [N6](../../research/library-reuse-investigation.md#n6) | REPLACE | L09 | A |
| [N7](../../research/library-reuse-investigation.md#n7) | REPLACE | L11 | A |
| [N8](../../research/library-reuse-investigation.md#n8) | REPLACE | L11 | A |
| [N9](../../research/library-reuse-investigation.md#n9) | REPLACE | L11 | A |
| [N10](../../research/library-reuse-investigation.md#n10) | CONSOLIDATE | L12 | C |
| [N11](../../research/library-reuse-investigation.md#n11) | DELETE | L18 | C |
| [N12](../../research/library-reuse-investigation.md#n12) | RETAIN | L11 retained | A |
| [N13](../../research/library-reuse-investigation.md#n13) | REPLACE | L01 repair complete; L11 parser adoption | C |
| [N14](../../research/library-reuse-investigation.md#n14) | RETAIN | L01 | C |
| [CF-01](../../research/library-reuse-investigation.md#cf-01) | REPLACE | L03 | A |
| [CF-02](../../research/library-reuse-investigation.md#cf-02) | REPLACE | L03 | A |
| [CF-03](../../research/library-reuse-investigation.md#cf-03) | REPLACE | L05 | A |
| [CF-04](../../research/library-reuse-investigation.md#cf-04) | REPLACE | L04 | C |
| [CF-05](../../research/library-reuse-investigation.md#cf-05) | REPLACE | L02 | C |
| [CF-06](../../research/library-reuse-investigation.md#cf-06) | REPLACE | L02 | C |
| [CF-07](../../research/library-reuse-investigation.md#cf-07) | REPLACE | L13 | C |
| [CF-08](../../research/library-reuse-investigation.md#cf-08) | CONSOLIDATE | L13 | C |
| [CF-09](../../research/library-reuse-investigation.md#cf-09) | CONSOLIDATE | L13 | C |
| [PG01](../../research/library-reuse-investigation.md#pg01) | REPLACE | L08 | A |
| [SQL01](../../research/library-reuse-investigation.md#sql01) | DELETE | L07 | B |
| [JSON01](../../research/library-reuse-investigation.md#json01) | REPLACE | L04 | C |
| [JSON02](../../research/library-reuse-investigation.md#json02) | RETAIN | L04 retained | C |
| [RNG01](../../research/library-reuse-investigation.md#rng01) | REPLACE | L03 | A |
| [TMP01](../../research/library-reuse-investigation.md#tmp01) | REPLACE | L12 | C |
| [TMP02](../../research/library-reuse-investigation.md#tmp02) | DELETE | L18 | C |
| [FS01](../../research/library-reuse-investigation.md#fs01) | CONSOLIDATE | L12 | C |
| [PATH01](../../research/library-reuse-investigation.md#path01) | DELETE | L12 | C |
| [PROC01](../../research/library-reuse-investigation.md#proc01) | CONSOLIDATE | L12 | C |
| [FFI01](../../research/library-reuse-investigation.md#ffi01) | REPLACE | L12 | C |
| [FMT01](../../research/library-reuse-investigation.md#fmt01) | DELETE | L14 | C |
| [CLI01](../../research/library-reuse-investigation.md#cli01) | RETAIN | L14 retained | C |
| [WALK01](../../research/library-reuse-investigation.md#walk01) | REPLACE | L12 | C |
| [CFG01](../../research/library-reuse-investigation.md#cfg01) | BLOCKED | L17 blocked | C |
| [SQLITE01](../../research/library-reuse-investigation.md#sqlite01) | RETAIN | Existing Store/probe duties retained | B |
| [RED01](../../research/library-reuse-investigation.md#red01) | REPLACE | L16 (L01 repair prerequisite) | C |
| [CLI03](../../research/library-reuse-investigation.md#cli03) | REPLACE | L14 | C |
| [REL01](../../research/library-reuse-investigation.md#rel01) | REPLACE | L13 | C |
| [REL02](../../research/library-reuse-investigation.md#rel02) | CONSOLIDATE | L13 | C |
| [REL03](../../research/library-reuse-investigation.md#rel03) | CONSOLIDATE | L13 | C |
| [CLI02](../../research/library-reuse-investigation.md#cli02) | REPLACE | L14 | C |
| [XML01](../../research/library-reuse-investigation.md#xml01) | REPLACE | L13 | C |
| [CFG02](../../research/library-reuse-investigation.md#cfg02) | RETAIN | L17 retained | C |
| [PROC02](../../research/library-reuse-investigation.md#proc02) | CONSOLIDATE | L12 | C |
| [SDK01](../../research/library-reuse-investigation.md#sdk01) | CONSOLIDATE | L15 | B |
| [X50901](../../research/library-reuse-investigation.md#x50901) | REPLACE | L06 | C |
| [KEEP01](../../research/library-reuse-investigation.md#keep01) | RETAIN | Existing package duties retained | Coordinator |
| [SYS01](../../research/library-reuse-investigation.md#sys01) | REPLACE | L14 | C |
| [DEAD01](../../research/library-reuse-investigation.md#dead01) | DELETE | L18 | C |

## Parked checkpoints

Canonical application source remains the investigation pin above; planning
commits change documentation only. Parked A
`fe23ec933fb0880bee07d2dc0b5b4fea3f9880e9`, B
`a62614a32feb827a06275263a0304570443fbccc`, and C
`b3c9e23b2b6b8c7a8e3b5cdb93af00321888fae8` retain their branches/work and command
handles. No reset, merge, replay, restart, timer or implementation dispatch is
part of this reconciliation. Planning uses their source diffs as observations,
not as merged coverage or replacement qualification.

| Checkpoint | Useful retained scope | Superseded or held scope | Planning disposition |
| --- | --- | --- | --- |
| A | Eight checkpoint-only A34 commits move Tailnet control/project caller/test seams; these application policy seams remain useful | The snapshot predates later canonical consolidation: its two-dot comparison also exposes older protocol, predicate, forwarding and test wiring, not additional A34 work to carry forward | Preserve branch; assess the A34 delta from merge-base `f253b96d` against current L09/L11 adapters. Do not restore stale snapshot bodies |
| B | B27 admission/routes/HTTP/WS/peer/test extraction is already patch-equivalent in canonical history | Two-dot differences also include canonical's later host-native consolidation; they do not establish unmerged B27 work | Preserve checkpoint; do not replay already integrated extraction. Its generic bodies now have L09/L18 follow-ups |
| C | Four C41 commits change seven Muse paths (286 insertions/273 deletions); argument/caller/cleanup seams retain meaningful state/authority and cleanup-order duties | Its 17-path two-dot comparison includes pre-B27 daemon/test wiring because canonical advanced after divergence; those differences are not C-authored reinlining work | Preserve branch; evaluate the seven-path C41 delta from merge-base `bb6be3a3` under L12 and existing duties. Preserve current canonical protocol/test wiring |

These are source/diff and caller assessments, not approval to merge or proof of
replacement/native behavior. Recheck selected commits against the actual
implementation revision before any separately scoped integration.

## Completion of a selected change

R00 selects one bounded owner change with its exact files, recipients and gates.
R01 integrates callers, dependency/bin selections and deletion coherently. Keep
adoption/correctness separate from unrelated M folder moves, and keep performed
V/native evidence separate from completed structural work. Independent challenge
must check retained limits/authority/lifecycle and the removed implementation.
Affected checks/builds include the actual shipping entrypoint and offline graph;
reserve broader native/release operations for ready integrated subjects.

After integration, update these packets and affected ownership/decomposition/tree
entries in place through maintenance. Historical counts remain historical until
their full reconciliation. No packet closes merely because a queue empties or
old equivalence tests pass.
