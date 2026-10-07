# Dependency and architectural cost

Five source-supported cuts reduce unnecessary dependency reachability, feature
requests or interface scaffolding: move the staged SQLite seeder into its tests;
remove the host's release-build edge; narrow Identity's AEAD features; narrow the
installer's key-library features; and pass the existing command runner directly
to Forgejo snapshot extraction. No package merger or cross-language model
consolidation is justified by this review. Actual binary/image savings and complete
distribution notices remain questions for the selected artifacts.

Source: `1cb4bbd8774dca2f50a358291610c60fe31c8d6d`, tree
`b876d4d0a6fdc4a77ce9d8bfc83c5d9d4743ccba`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing
dirty guidance files and 39 dependency inputs retain their recorded bytes.
This extends the [caller map](library-integrations/README.md),
[adapter challenge](library-integrations/adapter-challenges.md) and
[representation review](data-representations.md). The [tasks](implementation-tasks.md)
and [lanes](implementation-lanes.md) remain the execution plan; completed adoption,
unresolved correctness and parked restructuring retain their separate statuses.

Three Luna medium primaries and independent cross-challenges inspected manifests,
locked/cached upstream feature definitions, current callers, local package joins
and shipping selectors. The coordinator ran one read-only Cargo metadata query:
`--format-version 1 --locked --offline --filter-platform x86_64-unknown-linux-gnu`,
using the installed stable compiler explicitly. It made no lock changes or builds.
No Go/Bun resolution, tests, builds, network, provider, database, VM or native
qualification ran. Exact unchanged Go/SDK evidence is reused at its recorded source
identity; old candidate graphs are not substituted for this workspace graph.

The subsequent [total maintenance assessment](integration-maintenance-result.md)
at `797e6ec8` places these feature/graph cuts alongside removed engines, remaining
caller representations, runtime/custody duties and operational costs. It does
not treat a dependency reduction as a measured overall maintenance saving.

## What the current selections cost

| Selection / current source evidence | Meaning and limit |
| --- | --- |
| Rust: 28 workspace packages; 38 direct external dependency names; 197 direct dependency edges across members | Product commands, system libraries, release/build and developer tools have different consumers. Counts include repeated edges and do not measure maintenance or shipped bytes |
| Linux-filtered full-workspace resolve: 259 nodes, 231 registry plus 28 local | Unified features include build/dev consumers. This is neither one binary's feature closure nor an installed image inventory |
| Cargo.lock: 18 duplicated registry names; current Linux graph: 16 duplicate families | Trace upstream parents before judging duplication. Platform-only lock entries need not enter the selected Linux build |
| Go: 47 explicit module requirements; unchanged selection of 90 external modules | Tool directives and test modules differ from production imports. `pgx/v5` has direct callers despite its indirect label; module selection is not a per-binary link map |
| SDK: local `../forgejo-ext/sdk` replacement at `c92db11c14b773c9cc20ccfa4b853b4c017e8717`, tree `1791fdbb1adc97e86b14eda36475d89147531439` | `v0.0.0` is not a source pin. Preserve exact sibling provenance, SDK authority and actual producer contracts |
| Bun: Lit 3.3.3 runtime; root TypeScript 7.0.2, Lit-check 5.9.3 and locked transitive 5.2.2 | Analyzer/compiler consumers explain separate TypeScript selections; coexistence alone does not select a consolidation |
| xterm 6.0.0 / addon-fit 0.11.0 declared for development | Their JavaScript/CSS/licenses **ship** through the separate terminal-asset integrity lock and release producer. A dev label does not remove them from shipping scope |

Most Rust protocol/syscall adapters already disable defaults and select needed
capabilities: rustix, Hyper/hyper-util, Tokio, Clap, ureq, ssh-key, time and several
curve profiles. Remaining defaults were checked against consumers rather than
removed wholesale. URL parsing includes actual authority/form admission; Serde
and RawValue serve distinct DTO/raw-byte profiles; flate2 serves gzip producers
and consumers; tempfile owns actual scratch lifetimes. Their presence alone
does not establish redundant work. The five bounded cuts below concern specific
edges, not replacement of these engines.

The current Linux duplicate families are `base16ct`, `block-buffer`, `const-oid`,
`cpufeatures`, `crypto-common`, `digest`, `getrandom`, `hmac`, `pem-rfc7468`,
`rand`, `rand_core`, `sec1`, `sha2`, `signature`, `syn` and `webpki-roots`.
RustCrypto 0.13/installer consumers and ssh-key's newer crypto API families explain
many splits; PostgreSQL protocol also selects the newer digest/SHA family.
Proc-macro generations explain syn 2/3, and TLS root wrappers and RNG/platform
consumers explain other edges. This is not a set of interchangeable Soda types.
Do not force one version by rewriting manifests: compatible upstream APIs and
caller admission must precede any version change.

The selected `ssh-key 0.7.0-rc.11`, limited to `std,ecdsa`, serves actual OpenSSH
certificate output with `valid_before = u64::MAX`; the earlier 0.6.7 proposal is
superseded. Preserve that producer admission. ureq 2.12.1's HTTPS graph uses
rustls/ring/webpki, not OpenSSL. Ring's packaged C/assembly contributes native
build tooling/cache requirements; this does not establish that every host or
release binary links the whole graph.

## Five scoped cost cuts

### COST-GO-SQLITE-FIXTURE-1 — a test helper imports a driver in production

[staged_seed.go](../../../internal/store/staged_seed.go) blank-imports SQLite to
seed an external staged Forgejo database. Its only callers are
[merge_native_setup_test.go](../../../internal/factory/control/merge_native_setup_test.go)
and [st15_demo_seed_test.go](../../../internal/factory/control/st15_demo_seed_test.go).
The production dashboard imports Store, so the helper's driver registration enters
its production import/compile/link closure even though product Store is PostgreSQL.
The [dashboard container](../../../system/containers/dashboard/Containerfile) ships
that binary. No linked-byte saving was measured.

Move the exact helper and SQLite import to existing factory/control `_test.go`
support and update those two callers. Do not introduce a forwarding fixture
package. Retain modernc.org/sqlite for the independent
[acceptance lifecycle probe](../../../internal/acceptance/lifecycle_state.go) and
its fixtures; this is not a whole-module deletion. Local working evidence's
`DEP-G-SQLITE-LINK-1` is this same packet, not another defect.

### COST-HOST-BUILD-EDGE-1 — native policy pulls in a build library

[host iconfig](../../../lib/host/src/iconfig/mod.rs) uses release-build only for
`files::require_native`; the unconditional [manifest edge](../../../lib/host/Cargo.toml)
also compiles that library's broader build/HTTPS dependencies. Its separate
release-deliver payload reader is live and retains image selection/conflict
admission. Delivery does not depend on build, so these edges can be separated.

Put the tiny matching-native predicate in its existing host configuration owner,
using the already available [foundation architecture map](../../../lib/release-inputs/src/reader.rs).
Then remove only host → release-build. Preserve Linux/native x86_64 refusal and
the exact Soda `x86_64` → OCI `amd64` mapping; do not admit `amd64` as a Soda input
alias. Keep payload load and saved-image conflict checks. This is a removal of
dependency reachability, not a new adapter package or release-model merger.

### COST-IDENTITY-AEAD-FEATURES-1 — duplicated entropy feature admission

[Identity crypto](../../../cmd/soda-identity/src/crypto.rs) uses Aead/Payload,
Aes256Gcm and KeyInit; Soda fills the nonce through direct getrandom and propagates
entropy failures. aes-gcm 0.11.1 defaults enable `aes,alloc,getrandom`.
Select `default-features = false, features = ["aes", "alloc"]` on this direct
edge. Retain direct getrandom, nonce/key/AAD policy and failure behavior. This
removes an unused upstream random-generation feature request, not all entropy
dependencies or necessarily a crate from the unified graph.

### COST-INSTALLER-KEY-FEATURES-1 — envelope features exceed the public-key caller

[certificate.rs](../../../cmd/soda-install/src/x509/certificate.rs) uses generic
ECDSA `VerifyingKey::from_sec1_bytes` and RSA `RsaPublicKey::from_pkcs1_der`;
[verify.rs](../../../cmd/soda-install/src/x509/verify.rs) consumes their prehash
verification paths. Soda's bounded PEM envelope reader retains original DER/TBS
and certificate admission. The selected curve crates' default PEM/PKCS8/ECDH or
random-key requests exceed these callers; their selected ECDSA verification
implementations still need the curve ECDSA gates.

Limit p224/p256/p384/p521 to `default-features = false` with
`arithmetic,ecdsa,std`; limit RSA to `std,u64_digit` with defaults disabled.
Keep the direct ecdsa `verifying,pkcs8,std`, elliptic-curve and x509-cert profiles
unchanged. Preserve every currently supported curve/algorithm and Soda's envelope
parser. Other legitimate edges can still unify PEM/PKCS8/random features; acceptance
must distinguish removed requests from actual per-target closure reductions.

### COST-BOOTSTRAP-SEAM-1 — two commands require a full production implementation

[run_build_inner](../../../lib/soda-release-image/src/build.rs) creates a local
RunnerProduction solely for
[extract_forgejo_snapshot](../../../lib/soda-release-image/src/forgejo.rs).
That helper validates the exact revision, creates its snapshot directory and
uses only two `execute` calls for Git archive and tar extraction. The bootstrap
implementation forwards execution to the existing Runner but stubs unrelated
production phases as `foreign production required`. Its invalid-revision test
also implements the broad trait just to reject dispatch.

Pass the already available [Runner](../../../lib/soda-release-image/src/build_runner.rs)
directly to this one helper/caller and remove the unused bootstrap implementation
and broad test stub. Runner is visible at this owner and retains the same shared
cancellation/log state. Keep revision admission before directory/process work,
exact command arguments, snapshot destinations and error propagation. The actual
release-stage Production contract and its build/delivery bridge remain intact;
no new execute trait, package, duplicate runner or framework is required.

| Packet / single accountable owner | Scope and prerequisites within existing plan | Acceptance for later implementation |
| --- | --- | --- |
| **COST-GO-SQLITE-FIXTURE-1 / B**, Go Store/factory native fixture | Exact helper relocation plus its two test callers; staged external database contract is settled. Retain product PostgreSQL and acceptance's separate probe | Dashboard production import graph loses this SQLite/helper edge; both native fixture seed paths still work; unrelated SQLite consumers remain admitted |
| **COST-HOST-BUILD-EDGE-1 / A**, host iconfig/L13 | Existing caller predicate plus host manifest edge; coordinator checks affected locked graph/compiler/cache and current native-payload tests | Host closure loses release-build and its build-only HTTPS edge; valid payload/empty-path behavior, unsupported OS/architecture and saved-image conflict refusal remain exact; no new model/package |
| **COST-IDENTITY-AEAD-FEATURES-1 / A**, Identity crypto/L02-L03 | Direct aes-gcm feature request; exact upstream API gates checked. Coordinator owns manifest/lock and affected target feature evidence | Locked Identity compile and focused crypto checks preserve seal/open, AAD, nonce/key generation and malformed-ciphertext refusal; this edge no longer requests aead/getrandom |
| **COST-INSTALLER-KEY-FEATURES-1 / C**, installer X.509/L06 | Five direct key-library feature profiles; exact SEC1/PKCS#1 and verify gates checked. Coordinator owns manifest/lock and affected target feature evidence | Locked installer compile plus existing focused X.509/crypto and actual Caddy fixtures preserve four curves, RSA, strict envelope, raw TBS/fingerprint and signature refusal; unused feature requests removed |
| **COST-BOOTSTRAP-SEAM-1 / C**, release image snapshot/L12-L18 | Helper signature, sole bootstrap caller and invalid-revision test; existing Runner visibility/execute contract checked. Preserve broad production and actual archive authority | Existing snapshot/build checks preserve exact Git/tar dispatch and failure paths through the same cancellation/log owner; bad revision still fails before filesystem/process work; unsupported bootstrap methods and test stub disappear |

Use Luna low for the settled fixture transfer, feature edits and bootstrap seam;
Luna medium for
the host caller/closure cut and one independent review per coherent packet.
Run focused checks first, then only prerequisite-supported native qualification.
The coordinator stages explicit paths and owns manifest/lock and resource-heavy
checks. These are reviewable planning packets, not implementation in this audit.

## Toolchain, distribution and notice duties

Rust selects floating `stable`, with no local package declaring rust-version.
The current metadata's largest declared registry requirement is Rust 1.89
(aes 0.9.3); time/ICU families declare 1.88 and 39 registry packages omit a
rust-version declaration. The earlier observed installed rustc/Cargo 1.99.0 is
not a pinned release-worker compiler. Declaration maxima are lower-bound metadata,
not a verified project MSRV. **A/C** retain affected Rust compiler/cache admission
under L00 and release qualification; **B** retains Go 1.26.7/module verification;
the browser producer retains Bun 1.4.2. Do not invent a global minimum or start
an upgrade merely to reduce duplicate counts.

All 231 selected registry nodes declare license expressions. Eighteen of 28 local
manifests omit a package license field, while root LICENSE grants Apache-2.0 for
the covered original code. No individual crate publication path was found; metadata
hygiene remains conditional on such distribution. SPDX/cache metadata does not
prove notice completeness or license compatibility for a final distributable.

**H05-Q4 / C**, existing release/attribution gate, now includes the exact
avatar-notice staging question. Root NOTICE and the licensing research claim
`system/licenses/avatar-dependencies.txt` is included in new native bundles.
[rootfs_file_map](../../../lib/soda-release-image/src/prepare.rs) explicitly stages
root LICENSE/NOTICE, while [copyNotices](../../../scripts/build-soda-extension.ts)
stages font/Lit notices; neither map names the avatar text. The text is present
in source, and this observation does not prove every artifact omits it. Working
evidence's `DEP-NOTICE-AVATAR-1` refines **H05-Q4/D03**, without a duplicate legal
finding. Identify the claimed distributable and actual content first; acceptance
is exact-version notices at every claimed destination or a corrected narrower
statement, plus the existing actual-artifact attribution inventory.

Retain the established root license/NOTICE, coder/websocket's **ISC** text,
Lit's BSD-3-Clause and exact terminal MIT copies, fonts/Octicons/Tea notices,
Tailscale container attribution, and the actual Fountain fork/SDK source and
per-file obligations. The release producer archives `source.tar` and
`forgejo-source.tar` before generated compilation/SDK relocation and binds their
hashes to the payload. Those source selectors are not substitutes for auditing
the selected binaries, browser outputs, OS packages and containers. Existing
[licensing research](../../research/licensing.md) retains its qualified scope;
no new licensing clearance is asserted here.

Measure cost at each actual destination: resolved features → compiled packages →
linked binaries → copied assets → container/native package inventories → source
and notices. Release compilation/payload selectors and fixed container copy
tables determine what ships. Host/project base pins do not pin every repository
package installation; recorded RPM inventories describe the resolved image.
Python packages installed by external OS tooling are not Soda-authored Python
programs and do not violate the established language policy. **C**, release
inventory owner, requires one selected candidate's exact outputs before accepting
a numeric footprint, reproducibility or complete notice claim. Acceptance records
compiler/profile, target features and actual destination inventories/bytes;
no candidate build or byte reduction was demonstrated in this pass.

## Retained boundaries and earlier simplifications

Small package size is not evidence of forwarding-only architecture. Unix HTTP
owns bounded body/deadline/driver custody for four real consumers; wire-time
owns strict wire precision/range/UTC recipes; release-inputs owns shared input,
platform and optional trust-key admission with original DER. Retain these duties
at their current owners. SHA modules already delegate primitives to sha2 and
retain different stream/hex/fingerprint recipes; no shared hash framework is
selected. The release production bridge maps stage-specific operations, outputs,
errors and cancellation; its existence alone does not justify another facade.

Go's server/domain/coordinator/Store/client responsibilities and Rust's system,
privileged execution and credential/runtime custody remain distinct. Similar Go/
Rust identity DTOs cross process/authority boundaries and are not automatically
duplicate canonical domain owners. Build image, delivery payload and produced
image models describe different stages and artifact hashes. **SIMP-REL-WIRE-1**
still requires matching producer/null/unknown-field/ownership profiles before
consolidation; do not create a third model. **SIMP-I-PG-1** already selects typed
PostgreSQL rows/parameters/Json instead of generic wrappers. **SIMP-SNAPSHOT-1**
and **SIMP-SDK-1** keep representation and transport-authority decisions separate.
**SIMP-BROWSER-1** remains a parked callback/lifetime decision; no new competing
canonical state owner was established in the inspected joins.

Caller/dependency/feature and shipping-selector review is complete for this
chapter's stated scope. Per-target compile/link measurements, actual artifact
notice closure and conditional producer/ownership questions remain explicit.
Existing correctness findings, optional matcher/configuration gates and parked
restructuring are preserved; neither historical slice validity nor the complete
desired tree is regenerated by this documentation refresh.
