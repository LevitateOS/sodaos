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
| L15 SDK input admission | B | B06, C01 | Complete at `c92db11c14` in the exact sibling checkout; retain meaningful per-dial credential transport |
| L16 Evidence matching | C | C11 | Consideration complete; optional adoption deferred. L16.G must bound aggregate secret collection before any cutover; preserve completed L01/L12 repairs |
| L17 Configuration evidence | C | C05/C09 | Native corpus collected; rust-ini 0.21.3 fails continuation admission, so CFG01 cutover is held; CFG02 retained independently |
| L18 Dead machinery removal | C | C05/C11, A/B handoffs | Complete in `eaed66a9`: N11/TMP02/DEAD01 retired after current reference and retained-duty checks |

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
Select pinned upstream ssh-key 0.7.0-rc.11 plus L03/L04 curve/hash/encoding
decisions. Keep separate
algorithm/options/line/count/size policies and canonical fingerprint inputs.
Reject ssh-key KeyData::Other; preserve the current Ed25519 length-only policy.
Parsing is not NIST point validation or a new Ed25519 torsion policy.

Acceptance: all eight raw/eight certificate families with product-specific
allowlists, no options, malformed mpints/points/encodings, revision application
and fingerprints. Library certificate parsing does not confer trust/expiry
authority. Replace parsing and canonical serialization, remove mpint/certificate
format engines and unsupported equivalence-only tests together.

The original 0.6.7 selection fails an actual producer boundary: OpenSSH uses
`valid_before = u64::MAX` for certificates without an expiry. Its timestamp
wrapper rejects that value. Official upstream
[PR 504](https://github.com/RustCrypto/SSH/pull/504) is merged, and the published
0.7.0-rc.11 parser stores and decodes both endpoints as protocol `u64` values.
Use only `std`/`ecdsa` format features; signing and random-key generation are
not dependencies of this adapter. The pinned release candidate requires Rust
1.85, within the inspected local toolchain. Its source, archive checksums,
licenses and locked offline graph are admitted. Keep this explicit prerelease
choice visible; installed/native-worker qualification remains separate.

Source scope is complete in `7b42671d`, with dependency selection in `22496cb3`,
using Luna medium for implementation and independent review. Host retains its
eight raw/eight certificate families and installer its seven raw operator-key
families. NIST/scalar checks, comments/options/line/count/size policy and raw
fingerprint recipes stay with callers. The original forever certificate round
trips byte-for-byte; certificate aliases or other serialization changes are
refused before any signed bytes can change. Redundant MPINT sign octets and
invalid UTF-8 are deliberate stricter admission, not a parallel compatibility
parser. All 697 host and 114 installer tests pass on the final source; the
locked offline native development build passes. Remaining installed evidence
belongs to the wider owner packets. Historical extraction and parked checkpoints
keep their recorded scope.

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

The pinned Caddy 2.10.2 Linux/amd64 binary generated the retained public root
with trust installation disabled, isolated state, no listeners and joined
shutdown. [Fixture provenance](../../../cmd/soda-install/src/x509/tests/fixtures/README.md)
records its exact production image, selected manifest, binary and original-DER
digests. This closes the producer/profile gate, independently of L05. The
root uses P-256/ECDSA-SHA256, typed CA BasicConstraints/KeyUsage and UTCTime
2026/2036; the fixture proves that profile, not an installed appliance.

The [owning installation guide](../../guides/installation.md#local-ca-fingerprint-admission)
defines the admitted root envelope and policy. The adapter deliberately narrows
legacy acceptance: canonical DER equality, positive nonzero serials of at most
20 INTEGER content octets, duplicate-OID refusal, understood critical
BasicConstraints/KeyUsage only, supported SPKI and signature parameters, and
strict DER signatures. PSS requires matching SHA-2/MGF1/hash-length salt.
`x509-cert`/`der` own names, times, extensions and framing; original DER and the
original TBS TLV remain fingerprint and signature inputs. Legacy time forms
outside the library profile are refused; no expiry/chain/name authority is
added. `urlx` stays with its remaining setup/origin callers until L11.

Source scope is complete in `7d063f15`, using Luna medium for implementation
and independent review. Custom DER, names/SAN/constraints, calendar and SPKI
engines and their unused grammar suites are deleted. Consequential tests retain
strict ECDSA DER on all four curves, original-TBS verification for RSA PKCS#1/PSS,
all four EC curves and Ed25519, weak/unsupported algorithm refusal, malformed
used extensions, duplicates, signature parameters, private/multiple/trailing PEM
and raw fingerprint/caller policy. All 114 installer tests and the final locked
offline native development build pass. Eight existing unrelated public-export
warnings remain; no installed/native-worker qualification is claimed. The
[development receipt](../../../.artifacts/l05-l06/verification.md) records scope,
producer provenance and failed diagnostics separately. CA did not wait for SSH;
A34's checkpoint is preserved; B27 is integrated and retained only as provenance.
Full inventory/count regeneration remains pending.

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

Source scope completed with Luna medium implementation and independent review:
four clients in `6b18ee1b`, candidate fixture in `78dec937`, identity listeners
in `551ab3ad`, and the coupled host upgrade/pump in `21387814`. L00's original
PG3/3 and transport4/4 boundary receipts were required before cutover; clients
were verified before server dispatch. A owns `lib/unix-http`, a small bounded
Hyper client adapter; endpoint/status/credential policy stays with its four
callers. Hyper1.12 exposes `max_header_size` on both client and server builders,
so direct library caps replace the investigation's earlier client-cap assumption.

The host admits 128 connections and 16 blocking callbacks, retains upgrade
read-ahead, and joins HTTP drivers, callback jobs and upgraded sessions. One
tungstenite owner uses an eight-frame child-output queue, 131072-byte line/frame
caps and a 262144-byte protocol write buffer. Its retained 5s write and 2s child
input budgets actually expire under stalled I/O; child teardown retains its 3s
grace before kill/reap. Tests prove same-write handshake/ping/text delivery,
short writes without duplicate final frames, active-upgrade accounting,
backpressure cancellation and direct-child reaping. The queue capacity and
returned-message retry branch are source-reviewed; no test directly forces
`WriteBufferFull` or a descendant retaining stdout.

Actual development checks pass: shared client5, broker client15, factory8,
identity43 library and three HTTP integration checks, release-tools97 library
and11 CLI checks, and the complete host suite (320 library,33 mux and the
remaining caller/oracle checks). Identity tests include blocked-provider I/O,
bounded callback admission and joined shutdown; fixture tests include active
stream cancellation and opened-file custody. The final affected four-package
native development build passes locked/offline without warnings. The host suite
retains six pre-existing unused-import warnings in its source-inclusion Muse
oracle and one deliberately ignored documentation test. Engine/caller census,
format checks and independent reviews are closed; the disposable PG fixture is
stopped. Receipts remain in `.artifacts/l08-l09/`. Installed services, appliance
images, artifact-worker/other-target qualification and native executor/provider
behavior remain unverified. Parked restructuring checkpoints and full tree/count
regeneration remain parked; this closure does not complete broader A07/C02
native acceptance.

### L10 External HTTP adapters

Scope: setup Forgejo client (N3) and host provider request recipes (N4). C owns
setup; A receives provider changes. Adopt already resolved ureq 2.12.1 with
purpose-specific Agent/redirect/proxy/TLS/compression and bounded body policy.
Keep distinct credential/status/execution contracts and uncertain key creation.
Setup HTTPS is independent of L09. Acceptance: real facade status/credential
cases, TLS rejection, body framing/caps and no secret-bearing argv/logs.
Declare the actual DNS/deadline guarantee. Acceptance curl remains an existing
engine with its L01 completion repair; do not replace that lifecycle by analogy.

N3 source transfer is complete in `719d1137`: one purpose-specific ureq Agent,
normal rustls trust, no redirect/environment proxy/compression, bounded success
and status-error bodies, neutral credential-safe errors and a whole-request I/O
deadline. Local fixtures prove framing, cap-plus-one, literal token authorization,
204 revocation, TLS transport rejection and a single budget across headers/body.
The OS resolver can exceed that deadline; setup's previous resolver was also
unbounded. This is a development check, not installed Forgejo qualification.

**N4 remains held.** C owns the transport fit proof and A receives any later host
cutover. The native Executor currently bounds the whole child, including DNS,
and joins/kills transfers while preserving its host-native authority marker and
injected test seam. ureq 2.12.1 explicitly cannot interrupt blocking DNS; a
worker detached on timeout would leave credential-bearing work alive. The
replacement prerequisite is a concrete owned resolver/transport fit that keeps
that contract. Acceptance requires resolver-inclusive deadline/cancellation,
Basic/Bearer custody, bounded framing, TLS/redirect/proxy policy and a single
uncertain key-create attempt. Curl remains the implementation until that proof
passes. Its form and timestamp adapters may change independently through L11.

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

The shared `lib/wire-time` adapter owns the strict four-digit year 1..9999,
uppercase T/Z, required timezone, two-digit fields, 1..9 fractional digits and
no-leap-second lexical gate; time owns calendar validation and checked conversion.
UTC range and nanosecond normalization are checked. Domain lifetime/expiry and
raw signed deadline text stay with callers. Filename formatters share its UTC
projection. Local polling uses Instant; release progress retains its absolute
Linux CLOCK_MONOTONIC origin shared through SODA_BUILD_START_NS.

URL adapters check controls, percent escapes, raw authority/delimiter/explicit
port presence and restricted raw paths before WHATWG normalization. Parsed hosts
supply IDNA/numeric-IP connection and loopback policy while stored authenticated
literals remain unchanged. Each owner retains its empty-userinfo/query/port rule.
N13 uses URL setters for credential/query/fragment removal and omits malformed or
non-UTF8 tokens; L01's bounded split withholding and failure publication remain.
Installer urlx is retired with its last setup display caller in `4d0f1128`.
IP adapters delegate grammar/display to std while retaining explicit IPv6 zones,
family-checked masks, private/CGNAT/ULA and Tailnet unicast policy.

Source adoption is complete through `4d0f1128`, `2d9066bd` and `380914ed`.
Luna medium settled unresolved HTTP/deadline and URL admission; Luna low handled
the settled HTTP, URL/percent, IP and time transfers. Root owned graph changes,
verification and commits. 1,572 selected test executions pass, alongside locked
offline native development builds for 16 selected packages. Four unchanged
opt-in PG tests and the existing pops doctest were ignored. A broader unrelated
release-assets staging test lacks prebuilt project-tools/bin/muse; it does not
qualify that staging workflow. The actual artifact worker must still meet time's
Rust1.88 minimum; local Rust1.99 proves only this development graph. Installed
provider/CA/browser/VM and shipping qualification remain separate. N4's failed
transport fit proof stays visible in L10; no task or parked checkpoint is skipped.

### L12 File, FD and process ownership

Source adoption is complete through `c5cca5e7`. C is the accountable execution
owner, with A's host/guest/identity handoffs; B's Go ownership remains unchanged.
Prerequisites were completed L01 deadline/evidence custody, L00's selected
dependency admission and L03's fail-closed entropy profile. Luna medium settled
file/descriptor custody, cancellation and independent review; Luna low transferred
settled ABI/PATH/temp plumbing. Root owned dependencies, serialized verification
and incremental commits. Bounded same-opened-file repairs landed in `86ce0875`
before library temp convenience; later process and release reads retain that order.

| Finding and remaining owner | Implemented adapter and retained policy |
| --- | --- |
| [FS01](../../research/library-reuse-investigation.md#fs01), C/A | Muse config, acceptance inputs, private provisioning and release image/worker metadata use one admitted FD with cap+1 reads; the host agent hash uses the same bounded FD. Installer key directories use typed component-by-component rustix traversal; guest file helpers use typed open/openat/locks. PG fixture roots use openat2 beneath/no-symlink resolution and FD-relative private password creation/readback. Retain regular/owner/mode/link/inode checks, FIFO nohang, the allowed Muse leaf-symlink profile, mount admission, append uncertainty, sync and exclusive publication |
| [N10](../../research/library-reuse-investigation.md#n10), A | rustix SCM_RIGHTS reception and accept use atomic CLOEXEC and unique OwnedFd custody. Rejected/excess/truncated descriptors close on every exit. Muse retains zero-or-three stdio admission, original-connection SO_PEERPIDFD, credentials and account/cgroup policy; guest connect retains its one-second readiness budget, socket inode/mode/PID/UID and retry policy |
| [PROC01](../../research/library-reuse-investigation.md#proc01) / [PROC02](../../research/library-reuse-investigation.md#proc02), C/A | Selected std Child capture owners use bounded fair nonblocking I/O, absolute configured budgets and required EOF. Image builds share live cancellation; worker failures stop the exact admitted systemd unit. Direct-child owners kill/reap their child and close their own pipes after a two-second drain grace, reporting inherited pipes as failure. Maintenance owns feeder shutdown/join; installer rejects incomplete stdin delivery; candidate owns ticker stop/join on all returns. Retain pinned executable/environment policy, secret suppression and the shared Linux monotonic origin |
| [TMP01](../../research/library-reuse-investigation.md#tmp01), C/A | Held tempfile owners use private creation modes and finish child/FD consumers before cleanup or handoff. Installer publication guards precede writes; identity holds enrollment scratch through pumps. Muse verifies size/hash before chmod/persist. Delivery deliberately keeps partial state evidence before writing. Candidate/lab clean authority scratch, and native worker cleanup errors fail the result. PG fixture transfers automatic roots only after success; backup publishes a private synced stage without replacing a timestamp run; restore owns a container-local exclusive stage and checks cleanup before reporting success. Registration keeps its account-traversable 0711 lifecycle |
| [PATH01](../../research/library-reuse-investigation.md#path01), C/A | std Path/PathBuf and narrowly scoped canonicalize replace generic Go lexical engines. Actual caller admission rejects unclean absolute/relative forms and ParentDir where required; filesystem confinement stays with the FD owner. Native-root and disk alias resolution retain subsequent authority checks |
| [FFI01](../../research/library-reuse-investigation.md#ffi01), C/A | Locked libc layouts/constants and std exec replace selected hand-declared ABI and foreign shell diagnostics. Activation NSS lookup uses bounded getpwnam_r growth. Meaningful NSS, SIGPIPE/reset/mask, PTY, uid/gid, umask and flock policy remains local |
| [WALK01](../../research/library-reuse-investigation.md#walk01), C | Muse uses sorted WalkDir with max_open16 and no directory/root symlink following. Native non-UTF8 names, allowed leaf-file symlinks, private destination modes, auth.json-file exclusion and existing auth.json-directory descent remain explicit caller policy |

The selected graph pins rustix 1.1.5, libc 0.2.190, tempfile 3.27.0 and walkdir
2.5.0 with only required feature edges. No broad filesystem/process service was
introduced. Existing already-bounded libc/stat/publication adapters remain where
a library transfer would not reduce policy-bearing code. The old generic PATH,
errno and shell-diagnostic engines and manual active ancillary parser are retired.
N11's separate daemon peer duplicate and TMP02's unused migrate temporary were
outside L12; L18 has since retired both. L10.N4's transport hold and L16's
optional matcher remain unchanged.

| Acceptance boundary | Executed local evidence |
| --- | --- |
| Same-file bounded admission and confinement | Cap/cap+1, growth and inode substitution; regular/FIFO/symlink profiles; PG root/leaf symlinks, ParentDir, private modes and same-FD password readback; retained installer publication checks |
| Descriptor ownership | Actual Unix socket checks for accepted/received CLOEXEC, original pidfd, excess/truncated descriptor cleanup and later validation failure cleanup; full host and guest integration suites |
| Process completion and cancellation | Late shared cancellation, live fair output, capture quotas/read errors, descendant-held pipes, incomplete child input, streamed archive EOF, blocked feeder cancellation/join and ticker error-path join |
| Temp publication and cleanup | Private custody, existing timestamp collision refusal and sync paths; mock copy/chown/restore failures clean the same container stage, and cleanup failure cannot return success; worker cleanup-error result checks |
| Native adapters and graph | VM fake SSH/QEMU argv/signal/missing-or-unexecutable status tests, affected CLI suites and locked offline development builds for 20 selected packages |

Final selected suites pass **1,687 test executions**: host 708, guest 213,
acceptance 133, identity 44, compose 11, Muse 20, maintenance 22, VM 6+18,
candidate 15+10, lab 7+11, installer 115, PG fixture 6, PG maintenance 5+2+3,
release image 70/tools 101/delivery 34/assets 85, activation 12/domain 15/setup 21.
Four unchanged opt-in identity PG tests were ignored. Final receipts are under
`.artifacts/l12/`: host-guest-integration-test, operator-process-final-test,
temp-custody-final-test, restore-final-test, release-final-test, vm-cli-final-test,
delivery-state-test and native-build logs. Earlier mixed-run failures are
superseded by the named final passing suites; the final warning check recompiles
host/PG fixture after removing newly unused plumbing. The eight existing
installer exports, two existing compose test-only helpers and six existing host
oracle imports remain warnings in their applicable builds/tests.

These are source, local syscall/process fixtures and native development checks.
They do not qualify real installed systemd/Podman, backup/restore jobs, provider
services, a booted VM, shipping artifacts or the appliance kernel/toolchain.
PG fixture openat2 fails closed if unsupported; appliance qualification must
verify that support alongside existing SO_PEERPIDFD requirements. Actual worker
Rust/toolchain qualification remains separate. Earlier completed structural work
and parked restructuring checkpoints remain distinguishable from those limits;
full tree/count regeneration waits for the later reconciliation step. L13/L14
source results are recorded below.

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

**Source completion (2026-10-07).** C completed CF-07/08/09, REL01/02/03 and
XML01 in `798e4e40`, `4ba9b1ec`, `d4f3c922`, `00c5cd63`, `65961956` and
`46d5c4cd`. Luna medium handled decoded EOF, budgets and shared ownership;
Luna low transferred the settled writers and XML/SVG profiles. Root retained
manifest/lock, command execution, review and commit ownership. The bounded
64-byte accessor lives in `lib/release-inputs/src/elf.rs`; build retains
ET_EXEC/ET_DYN and x86_64 admission, while host/Tea retain their machine gates.
The Muse feeder retains exact-size pread, its existing deadline and child/FD
custody. A short source makes the writer sticky before tar Builder drop, so
zero padding cannot manufacture a successful complete member.

Delivery owns the shared layer scanner, consumed through the acyclic
build-to-delivery dependency. MultiGzDecoder stays owned until decoded EOF,
including concatenated gzip members, CRC/ISIZE and post-tar zero padding.
Decoded nonzero trailing archives/junk and duplicate wanted terminal members
refuse. Raw descriptor size/hash, overlay/whiteout/ancestor policy and original
signed bytes remain caller responsibilities. New USTAR output is deterministic;
old Go writer hashes are historical observations, with no compatibility writer.

| Admitted format | Production bounds |
| --- | --- |
| OCI layers | 1 GiB compressed and decoded per layer; 512 MiB wanted member; 100,000 entries; 4 GiB unique compressed descriptors and 16 GiB decoded per image |
| Terminal distributions | 10,000,000 compressed bytes, 256 MiB decoded including skipped members/padding, 2,000,000 bytes per retained wanted member and 100,000 entries; bounded same-FD cache reads |
| Inline gzip | Image Ignition: 10 MiB compressed / 2 MiB decoded; acceptance: 2 MiB compressed / 1 MiB decoded, with encoded-source admission before expansion |
| Emblem | 64 KiB same-FD regular-file input and 256 XML nodes; no DTD/entity resolver; resolved SVG namespaces, finite absolute M/L/H/V/Z closed rings only |

Both actual pinned terminal distributions were fetched and their compressed and
wanted-member pins checked: xterm 6.0.0 has 1,418,562 compressed / 6,046,720
decoded bytes and 161 members; addon-fit 0.11.0 has 5,488 / 28,672 bytes and
9 members. Each of the five wanted members occurs exactly once. The new native
extractor processed those bytes through an owned local HTTP fixture and produced
all five pinned SHA-256 outputs. Small-budget tests cover skipped members and
post-tar padding without large expansion fixtures. Canonical emblem output is
byte-for-byte unchanged. These source/local checks do not qualify an installed
appliance, real import effects or shipping artifacts.

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

**Source completion (2026-10-07).** C completed FMT01/CLI02/CLI03/SYS01 in
`c23e07e7`, `3e3076c5`, `b7864443`, `f3fc2759`, `bdd7beac` and `c82125ee`.
Luna low implemented the settled Clap/humantime/literal-formatting profiles;
Luna medium handled inventory/deadline custody and independently reviewed CLI
admission. Root fixed compilation/runtime issues and verified actual executables.
The shared goflag and installer fmtx engines and line-oriented Cargo TOML scanner
are removed. Domain validation remains before secret loading, evidence creation
and execution. Generated help goes to stdout with exit zero. Parser failures
have neutral diagnostics; build syntax failures exit 2 and its existing domain
refusals retain exit 1. Scalar options are last-wins, repeated evidence inputs
append, explicit equals booleans preserve false, and exec's first positional
begins a literal tail including later flags and `--help`. Documented double-dash
flags remain; unproven Go single-dash-long aliases and foreign diagnostics retire.

Timeouts admit the positive humantime grammar, default 30m with a 24h maximum,
leading-plus and Greek-mu normalization. Human aliases/spacing admit; minus,
zero, overflow and unsupported subnanosecond/fractional precision refuse. This
profile deliberately replaces the old Go-duration oracle quirks.

One `cargo metadata --format-version 1 --no-deps --locked --offline` result is
bound to the admitted canonical snapshot and reused through compilation and
payload linking. Opaque member IDs, actual package/bin targets and local default
feature closure are checked; non-default required features refuse. Cargo's
hyphenated `required-features` field is exercised against actual Cargo output,
not only a handcrafted JSON fixture ([Cargo schema](https://doc.rust-lang.org/cargo/commands/cargo-metadata.html)).
Metadata capture has a 16 MiB stdout cap and one 120s operation deadline, with
existing cancellation, pipe-drain and direct-child reap ownership. A progressing
child is tested with a short injected budget. Ordinary build commands retain
their existing cancellation policy. The ten discovered commands and every
RUST_TOOLS target/destination are preserved; tools-owned members derive from
that table, runtime validation/build share one table, and Cargo builds now
select the intended `--bin`. Metadata grants no additional shipping authority.

L13/L14 verification totals 656 selected passing tests: foundation ELF 1, Muse
maintenance 23, build 48 plus 12 fixture tests, delivery 37 plus 14 fixture tests,
image 70, assets 89 plus 3 renderer tests, installer 112, acceptance 134,
release-tools 100 plus 12 actual CLI tests, and host ELF 1. Focused repeats after
subsequent fixes are not added to that total. Performed receipts are under
`.artifacts/l13-l14/`, including terminal-corpus receipts and actual workspace
metadata. Affected locked offline native development builds and final executable
checks pass: all ten selected packages (build-tools, release-build/deliver/image/
tools/assets, acceptance, installer, Muse maintenance and host) build. Only the
eight pre-existing installer export warnings remain. Native checks cover eleven
help cases without file effects, two safe invalid invocations and one completed
local printf capture preserving `--help`, a later owner-looking flag and `--`
in the exact invocation/output. These are fixture checks; installed/native-worker,
provider, VM, shipping and appliance-toolchain qualification remain separate.
The locked admission added eleven cached dependency nodes without changing
existing pins. Clap uses std/help/usage/error-context only; the highest declared
new MSRV is Rust 1.85, and cached manifests/licenses plus the actual affected
feature closure were checked. L15 input admission and L18 cleanup subsequently
completed at their defined source scopes. CFG01 fit, the held L10.N4 and
optional L16 remain separately scoped.

Final passing receipts are `muse-writer-test`, `oci-writer-test`,
`inventory-assets-test`, `assets-format-test` (canonical renderer),
`cli-format-test` (installer), `acceptance-final-test`, `release-cli-final-test`,
`format-oracles-test`, `host-elf-test`, `native-build` and the native CLI/terminal
corpus JSON receipts. Earlier compilation and mixed-suite failures were corrected
and are superseded by those final selected results; they grant no qualification.

### L15 SDK input admission

Scope: pinned SDK runtime/manifest readers and Soda callback/background clients.
B consolidates on existing Go cap+one read then strict decode. Keep duplicate/schema
policies, error secrecy and per-dial peer/admission/rebootstrap transport duties.
Gate external SDK implementation on its exact pinned source and matching scoped
authorization; a Soda planning change does not mutate the sibling repository.
Acceptance: cap+one/trailing whitespace, read errors and unchanged peer/credential
custody. Delete the faulty limiter pattern, retain the useful transport adapter.

Source completion (2026-10-07): the canonical Fountain checkout exactly matched
investigation pin `86a70e1155f1036fdcd38f2af49d4ca6defa8280` (tree
`fa3583c281047190d56f111af31714b5c7bb458e`). The explicitly dispatched L15
supplied the matching sibling scope. Commit
`c92db11c14b773c9cc20ccfa4b853b4c017e8717` (tree
`1791fdbb1adc97e86b14eda36475d89147531439`) repairs only SDK runtime requests
(64 KiB) and manifests (1 MiB), collecting cap+one before decoding and refusing
read failures. Exact-cap input, unknown-field rejection, one-object EOF,
duplicate-last-wins and manifest validation remain intact. Soda callback and
background readers already use cap+one; SDK callback/background code and Soda's
per-dial UID, shared admission and single rebootstrap/retry transport are retained.

Luna low performed the exact-source check, narrow edits and independent review.
Go 1.26.7 passed all 27 SDK tests (56 including subtests) and 48 Soda Forgejo
client tests (99 including subtests). An overlay of the original pinned readers
failed the new cap+one and long-whitespace cases at both boundaries, proving the
regression. No module/lock changes, native Forgejo build, credential fixture reads
or release qualification were needed. Historical investigation pins remain
historical; release provenance must capture the actual clean Fountain revision.
These source checks do not close B06's separate authority findings or native
installed qualification.

A separate source-level concern remains with B03.C: the operator endpoint wraps
its body in a 4 KiB LimitReader before calling the 1 MiB strict decoder, so the
outer limiter can still manufacture EOF. This caller is outside SDK01's
callback/background scope. Its pending repair must collect 4 KiB+one and refuse
overrun/read failures before the existing schema decoder, preserving operator
principal/actions and exact-limit success. This does not reopen completed
structural moves or claim that every Go input boundary is fixed.

### L16 Evidence matching

Consideration is complete against source `aa764269`, using two Luna medium
subagents and coordinator source checks. Defer the optional aho-corasick 1.1.5
cutover and retain the current matcher. Its repeated per-byte/per-pattern searches
are a source-derived cost, but no workload measurement establishes a worthwhile
replacement. The library still needs caller-owned streaming and resource policy;
its `memory_usage` describes retained memory, not construction peak. A pattern-byte
cap alone does not establish an automaton construction budget. No matcher,
dependency or application behavior changed; no new runtime checks or benchmarks
were executed. The original RED01 replacement recommendation is deferred here.

L01 remains complete, including writer finalization, absolute deadlines and
bounded transformed evidence. L12's same-FD cap+one repairs also remain complete.
Current `files::private_file` caps each input at 1 MiB; `jsonio::read_json_file`
caps JSON at 4 MiB and inline gzip decoding caps each decoded file at 1 MiB.
The existing evidence admission deduplicates up to 16,384 raw/escaped patterns
within 16 MiB, and preserves the 16 MiB output/pending/tee limits documented in
[native support](../native-support.md). These are per-input or admitted-set
bounds, not a bound on collection before admission.

**L16.G — aggregate input prerequisite, owner C:**
`driver/inputs.rs::read_secret_files` accumulates raw and trimmed copies for
repeated `--secret-file` arguments; `provisioning.rs::provisioning_secrets`
accumulates hash/source/decoded/line variants. `collect_all_secrets` joins these
vectors before `create_evidence` applies its pattern budget. Aggregate collection
is therefore still open, distinct from the completed same-FD read repairs.
Scope a follow-up to these collectors and their evidence-admission handoff;
establish count/byte accounting before retaining, cloning or deriving values.
Preserve complete raw/trimmed/escaped/Ignition coverage, private-file custody
and generic errors. Prerequisites are the existing L01/L12 bounds and an explicit
collection profile. Acceptance must cover repeated files, aggregate count/byte
limits, many derived Ignition variants, exact-limit admission and refusal before
capture/publication, without printing secrets or silently dropping patterns.
This pending repair is useful independently of matcher adoption.

**Optional cutover, owner C:** reconsider only after L16.G and a demonstrated
simplification or material caller cost justify the work. Coordinate exact
dependency/features/license/cache admission with the coordinator. Build and
share matchers once per admitted pattern set; establish construction-peak and
retained-memory budgets before construction, and sanitize build failures before
commands start. Use byte `LeftmostLongest` non-overlapping searches over the
whole buffered window with a withheld suffix; do not search a truncated safe
prefix or use the library's overlapping-search API. Commit a complete crossing
match or withhold from its start. Built-in streaming replacement supports only
`Standard`, which remains suitable for any-secret final leak finding.

Acceptance for that later cutover: split/overlapping-prefix/escaped/binary
patterns, crossing-window starts, tiny-write batching, bounded construction and
output, sticky write/final-flush failures and private/exclusive publication.
Remove repeated searching and sequential placeholder rewriting together;
explicitly prove the intended single-pass replacement behavior and JSON-key
collision refusal. No optional matcher work gates the completed L01 repairs,
unrelated adoption or integrated qualification of unchanged matcher behavior.

### L17 Configuration evidence

Owner C collected 21 synthetic native APP_DATA_PATH fixtures with Luna low.
The probe executes the pinned Forgejo provider, `EnvironmentToConfig` and
`loadServerFrom` through a Go test overlay; it copies no grammar and changes no
native source. Provider, server, environment and container startup source at
`86a70e1155f1036fdcd38f2af49d4ca6defa8280` are byte-identical to tag `v15.0.9`
(`19b9b9d216bbfb501c18514bd1a8c980246ca3f7`). SDK-only commit `c92db11c14`
does not change that native configuration authority. Go 1.26.7 executed the
corpus successfully. Exact inputs, probe and JSON outcomes are retained in the
configuration-decision commit's receipt; ignored scratch is not the only copy.

Observed native profile: section and key case are exact; duplicate keys and
sections select the last value. Whole-value quotes are stripped, backslashes
are preserved with `IgnoreContinuation:true`, and embedded quotes are retained.
Unquoted inline `#`/`;` comments are stripped with or without preceding spaces.
An explicit `[DEFAULT]` does not supply the server key. Same-section percent
interpolation resolves, while an unresolved token remains literal. Native
missing/empty paths default to the work-path data directory and relative paths
are joined to the work path. Soda still refuses absent/relative paths and keeps
its `/data` host mapping under the [operator contract](../../guides/operator-setup.md).
Those intentional policy differences are not parser equivalence failures.

The process-environment override wins even when the server section is absent.
Quote characters supplied in an actual process environment remain literal;
that fixture does not prove how Quadlet/container environment-file quoting is
processed. Production startup runs environment-to-ini before Forgejo reads
app.ini. Preserve the deployment override contract and qualify environment-file
admission separately from this native provider evidence. No installed app.ini,
credentials, running service or private provisioning state was inspected.

**Decision:** withdraw rust-ini 0.21.3 as the preferred replacement; it is not admitted.
Its exact registry archive was checksum-verified (SHA-256
`796e8d2b6696392a43bea58116b667fb4c29727dc5abd27d6acf338bb4f688c7`).
[Published source](https://docs.rs/crate/rust-ini/0.21.3/source/src/lib.rs)
`parse_str_until` discards backslash-LF and consumes the next line regardless of
`enabled_escape`; no continuation-disable option exists. The native fixture
`APP_DATA_PATH=/data/trailing\` followed by `NEXT=value` instead preserves
the backslash and the separate key. This candidate mismatch is inferred from
exact source, not claimed as an executed Rust comparison. Default escapes,
first-value selection and absent interpolation also need explicit admission.
Unspaced inline comments are a second source-derived mismatch: the default
candidate keeps them and its inline-comment feature only strips comments after
space/tab outside continuations; duplicate selection alone is a small adapter duty.
Do not add a preprocessing grammar to manufacture compatibility.

[Published manifest](https://docs.rs/crate/rust-ini/0.21.3/source/Cargo.toml)
confirms MIT, Rust 1.64, cfg-if ^1.0 and ordered-multimap ^0.7, with optional
unicase ^2.6. Its bundled lock is not Soda's dependency closure. With semantic
admission failed, no production dependency/lock mutation or candidate build was
needed. Public archive/index downloads were bounded to 256/128 KiB and 30
seconds; fixtures and raw/resolved/effective outputs are bounded synthetic data.

L17 evidence collection and the candidate disposition are complete; **CFG01
parser cutover remains held on semantic fit, not missing native evidence**.
C owns the remaining selection and implementation. Prerequisites: a maintained
parser/boundary that admits the frozen native corpus without recreating a full
foreign grammar; actual deployment override evidence; then its selected
license/feature/lock/compiler/offline closure. Scope: only the existing domain
command's effective-key and bounded file adapters, retaining writer quiescence,
marker confinement and no-guess policy. Acceptance: replay the native corpus,
prove bounded same-file input and override admission, preserve absent/relative
refusal and `/data` mapping, and remove the Python clone with its last caller.
The clone's lowercase/default/strict-duplicate/quote behavior is not an oracle.
This held cutover does not hold other packets. CFG02's byte-preserving locale
set/cap/hash/collision policy remains retained independently and needs no parser
cutover. No configuration production code changed in this evidence packet.

### L18 Dead machinery removal

Scope: N11 daemon peer wrapper (A), TMP02 migrate temporary (C), DEAD01 orphaned
Go testoci fixture package (B). C leads explicit handoffs. Recheck references at
the implementation revision, move meaningful tests to active owners where
needed, then delete without replacement. Preserve actual Muse peer policy and
testify/other dependencies with remaining consumers. Acceptance is reference
closure plus affected existing package/source checks, not a new absence harness.

Source completion (2026-10-07), `eaed66a9d7e521b8b920480000df2c2ca1318365`:
Luna low verified references against `3c64997c`, authored the A/C Rust and B Go
handoffs, and independently reviewed the final removal. N11's wrapper had only
module exports and two duplicate smoke tests. Both tests' credentials/pidfd
assertions already exist on the active Muse path; its peer-attestation test
passes. The wrapper, exports and duplicate tests are removed; actual Muse
SO_PEERCRED/original-connection SO_PEERPIDFD, OwnedFd lifetime and caller/cgroup
policy remain with `muse/socket.rs` and its real callers.

TMP02 no longer creates or deletes an unused random empty file. The migration
still reads and scrubs in memory before rewriting the original inode, preserving
byte-line/database-section policy, mode, ownership, diagnostics and error exits.
Only this command's getrandom manifest/lock edge was removed; its dependency
list is now empty. Shared getrandom/libc and Go testify retain live consumers.
DEAD01's single 128-line fixture file had no imports, callers, tests or explicit
build/install registration. It is deleted without replacement; active Rust OCI
fixtures remain with their actual import/install/build owners, and inert
comments no longer point to the dead Go helper.

Forty existing tests pass: one active Muse peer test, 31 remaining gmux smoke
subjects, two migrate unit tests, three real-command/service-wiring script tests
and three Go architecture tests. Eight bounded actual-command byte fixtures also
pass with mode 0640, the original dev/inode and a non-writable parent directory
as an unprivileged user. They cover non-UTF8, CRLF, unterminated lines, exact
sections, password spelling and blank/empty records; their inputs/results are
retained in the source commit receipt. Rust/Cargo 1.99.0 and Go 1.26.7 built the
three affected development entrypoints (`soda-host`, `soda-forgejo-tailnet`,
`soda-forgejo-migrate`) with locked offline dependencies. Current metadata lists
28 Cargo members and 29 Go packages. Format/diff checks pass. These checks do
not claim installed systemd/container behavior or release qualification.

The scoped R02 retirement maps and parked-seam assessment below are reconciled;
full historical inventory/count/table regeneration remains pending. L10.N4,
CFG01 fit, optional L16 and the separate B03.C operator input-cap concern keep
their existing owners and gates.

## Readiness gates

| Gate | Exact requirement | Holds only |
| --- | --- | --- |
| LA-G1 Dependency admission | Initial compiler/candidate closures and humantime metadata recorded above; requalify exact changed workspace features/license/cache and affected locked offline build on the selected worker | Each new adoption before its substantial cutover |
| LA-G2 Driver fit | tokio-postgres adapter deadline/cancel/discard/reconnect/exclusion proved locally; retain admitted auth/DSN/transport and actual Store/Tx integration checks | L08 cutover; not L07 or domain corrections |
| LA-G3 Transport fit | Hyper Unix client/server/backend/shutdown/read-ahead and WS owner fit proved locally; retain production framing/caps/slow-peer/lifetime and affected graph checks | L09 relevant server/upgrade cutover; not L10 HTTPS |
| LA-G4 Encoding/trust profiles | Actual producer semantics, original signed bytes, Caddy roots and critical-extension choice | Corresponding L04/L05/L06 boundary only |
| LA-G5 Native configuration | Native corpus collected; selected parser must pass continuation/quote/comment/effective-key and deployment override admission plus dependency closure | CFG01 parser cutover only |
| LA-G6 External SDK | L15 exact pin/scope and cap+one source repair proved at Fountain `c92db11c14`; native authority findings retain their own gates | No remaining L15 input-repair hold |
| LA-G7 Evidence input collection | L16.G must bound aggregate raw/trimmed/Ignition collection before evidence admission; automaton construction budgets remain unproved | Optional L16 matcher cutover only; completed L01/L12 scopes remain complete |

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
| [N4](../../research/library-reuse-investigation.md#n4) | HOLD: resolver-inclusive cancellation / Executor fit | L10 | C |
| [N5](../../research/library-reuse-investigation.md#n5) | REPLACE | L09 | A |
| [N6](../../research/library-reuse-investigation.md#n6) | REPLACE | L09 | A |
| [N7](../../research/library-reuse-investigation.md#n7) | REPLACE | L11 | A |
| [N8](../../research/library-reuse-investigation.md#n8) | REPLACE | L11 | A |
| [N9](../../research/library-reuse-investigation.md#n9) | REPLACE | L11 | A |
| [N10](../../research/library-reuse-investigation.md#n10) | CONSOLIDATE | L12 | C |
| [N11](../../research/library-reuse-investigation.md#n11) | DELETE complete | L18 `eaed66a9`; actual Muse retained | C |
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
| [TMP02](../../research/library-reuse-investigation.md#tmp02) | DELETE complete | L18 `eaed66a9`; original-inode scrub retained | C |
| [FS01](../../research/library-reuse-investigation.md#fs01) | CONSOLIDATE | L12 | C |
| [PATH01](../../research/library-reuse-investigation.md#path01) | DELETE | L12 | C |
| [PROC01](../../research/library-reuse-investigation.md#proc01) | CONSOLIDATE | L12 | C |
| [FFI01](../../research/library-reuse-investigation.md#ffi01) | REPLACE | L12 | C |
| [FMT01](../../research/library-reuse-investigation.md#fmt01) | DELETE | L14 | C |
| [CLI01](../../research/library-reuse-investigation.md#cli01) | RETAIN | L14 retained | C |
| [WALK01](../../research/library-reuse-investigation.md#walk01) | REPLACE | L12 | C |
| [CFG01](../../research/library-reuse-investigation.md#cfg01) | HOLD: candidate semantic fit | L17 native evidence complete; parser cutover held | C |
| [SQLITE01](../../research/library-reuse-investigation.md#sqlite01) | RETAIN | Existing Store/probe duties retained | B |
| [RED01](../../research/library-reuse-investigation.md#red01) | REPAIR complete in L01; matcher DEFER | L16 consideration complete; L16.G collection bound open; optional cutover deferred | C |
| [CLI03](../../research/library-reuse-investigation.md#cli03) | REPLACE | L14 | C |
| [REL01](../../research/library-reuse-investigation.md#rel01) | REPLACE | L13 | C |
| [REL02](../../research/library-reuse-investigation.md#rel02) | CONSOLIDATE | L13 | C |
| [REL03](../../research/library-reuse-investigation.md#rel03) | CONSOLIDATE | L13 | C |
| [CLI02](../../research/library-reuse-investigation.md#cli02) | REPLACE | L14 | C |
| [XML01](../../research/library-reuse-investigation.md#xml01) | REPLACE | L13 | C |
| [CFG02](../../research/library-reuse-investigation.md#cfg02) | RETAIN | L17 retained | C |
| [PROC02](../../research/library-reuse-investigation.md#proc02) | CONSOLIDATE | L12 | C |
| [SDK01](../../research/library-reuse-investigation.md#sdk01) | CONSOLIDATE complete | L15 at Fountain `c92db11c14` | B |
| [X50901](../../research/library-reuse-investigation.md#x50901) | REPLACE | L06 | C |
| [KEEP01](../../research/library-reuse-investigation.md#keep01) | RETAIN | Existing package duties retained | Coordinator |
| [SYS01](../../research/library-reuse-investigation.md#sys01) | REPLACE | L14 | C |
| [DEAD01](../../research/library-reuse-investigation.md#dead01) | DELETE complete | L18 `eaed66a9`; no replacement fixture | C |

## Parked checkpoints

Luna low reassessed the exclusive checkpoint deltas against canonical
`3c64997c`, then reconciled N11's removal at `eaed66a9`. This is the scoped R02
assessment of remaining seams, separate from the historical full-tree audit.
Parked A `fe23ec933fb0880bee07d2dc0b5b4fea3f9880e9`, B
`a62614a32feb827a06275263a0304570443fbccc`, and C
`b3c9e23b2b6b8c7a8e3b5cdb93af00321888fae8` remain at exactly those branch tips.
No checkpoint replay, merge, reset, restart or timer change was performed.

The 2026-10-07 cleanup removed the three inactive lane worktrees while retaining
`run/20261006-a`, `run/20261006-b` and `run/20261006-c` at those exact tips.
Ignored lane artifacts were moved intact to
`.artifacts/cleanup-20261007/parked-lanes/{a,b,c}/` in the canonical checkout.
Checkpoint assessment and retained duties below remain applicable; removing a
worktree did not integrate its branch or close its pending work.

| Checkpoint / exact comparison | Retained application duty and current defining owner | Disposition within existing tasks |
| --- | --- | --- |
| A34: eight exclusive commits, ten-path delta from merge-base `f253b96ddf235b6c5b6935730f0ebc713c8b3b45` | `tcontrol.rs` still owns project/run-binding/enroll-run bridges called by the companion and `dbackend`; policy/provider/enrollment/retry remain local. All seven moved tests already exist in `tests/tcontrol_oracle.rs` | C04.M retains policy/provider/enrollment and optional caller/test ownership seams; A07.M/V retains host/companion binding/lifetime. Library-local test placement is optional, not missing behavior. L09/L11 already own transport and selected time/IP/URL adapters. Preserve the checkpoint; any later split must consume current adapter bodies |
| B27: seven U1–U7 commits from `7089a6d089fc712ec35dcc168f737aad8a322154`; the full checkpoint/canonical merge-base is `c0caefb34971b0565440f3d346cf13cc3ba1828f`, whose 101-commit divergent range is not a B27 delta. Canonical U1–U6 patch IDs match and canonical U7 `72e4bb90` is an ancestor | Current `daemon/{admission,routes,response,http,websocket}.rs` owns actual policy and library adapters. Later L09 upgrade/pump/read-ahead and response tests supersede the snapshot representation | No B27 replay. The only dead peer duplicate is now removed by L18/N11; active Muse attestation is retained. Completed structural work remains complete at its original scope; native authority/qualification findings keep their own gates |
| C41: four exclusive commits, seven-path delta from merge-base `bb6be3a3f86dc385e53924af57965e972273b4e1` | `muse/args.rs` feeds execution; `connection.rs` supplies backend selection; resolve/nested retain caller/account/cgroup checks; `operate.rs` owns ordered mount/file/state cleanup called by stop. L12 owns lower-level FD/process/cancellation mechanics | A07.M owns retained caller/custody/cleanup seams; A07.V owns actual lifecycle evidence. Caller and cleanup splits are coherent optional seams; combining argv and broker connection selection lacks a demonstrated single owner. Existing Muse tests cover these helpers. Keep current boundaries pending a concrete owner benefit; do not replay the stale 17-path two-dot diff |

Acceptance of this assessment: every useful duty stays assigned to an existing
owner, superseded generic splits are excluded from the desired responsibility,
and the three checkpoint tips remain unchanged. No new coverage gap was
established by these module moves. If a seam is selected later, its prerequisite
is a diff against the then-current defining adapters; its acceptance is retained
caller/custody behavior and the relevant existing tests, with no old engine
restoration. Full source-to-slice recount and installed qualification remain
separate R02/R04 work.

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
