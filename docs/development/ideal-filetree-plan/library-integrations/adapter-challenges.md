# Adapter challenges against selected upstream APIs

This chapter refines the existing [tasks](../implementation-tasks.md),
[lanes](../implementation-lanes.md) and [adoption packets](../library-adoption.md).
It selects concrete simplifications and records where a caller or upstream
contract must change first. Completed adoption and correctness repairs keep
their recorded status. A retained responsibility does not require retaining its
present wrapper or decomposition.

Source: `2dc3bce9215b6769172ad20a28678cf2771d15db`, tree
`6b87d8f2dc6715ebd724fe7cf968cb170852e5a4`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`. The 12 pre-existing dirty
guidance files and 39 dependency inputs retain their recorded bytes. The
[caller census](README.md) remains valid at its unchanged source and question;
the new challenge reads defining mechanisms and their actual callers against
selected upstream source. It does not claim a fresh body review of every census
selector or every transitive implementation path.

The subsequent [representation audit](../data-representations.md) at `463ce772`
reassesses retained visitors and formatting against actual producers. Old L04
profiles/goldens do not impose permanent unreleased compatibility; release wire,
ordered-tree and formatter cuts now have explicit scoped dispositions there.

The subsequent [dependency/architecture cost audit](../dependency-and-architecture-cost.md)
at `1cb4bbd8` selects five caller-specific dependency/feature/interface cuts.
Its bootstrap Runner cut removes one overbroad trait implementation; the actual
release production bridge and separate authority/state owners remain retained.

## Selected API coverage and dispositions

The Rust partition covers all **38 direct external dependency names**. Exact
versions, enabled features, public types, reverse callers and fixtures remain in
the [crypto](crypto-profiles.md) and [native](native-engines.md) maps. The table
groups related mechanisms; it does not count every primitive call as an adapter.
Selected sources were read from the local Cargo registry, including the
transitive API owners named below. No upgrade is proposed.

| Selected Rust APIs | Actual boundary and challenged disposition |
| --- | --- |
| aes-gcm 0.11.1; getrandom 0.4.3 | Identity sealing and entropy callers already use upstream AEAD/OS entropy. Retain nonce/key custody, purpose and failure policy; no replacement crypto engine identified |
| ecdsa 0.16.9; elliptic-curve 0.13.8; p224/p256 0.13.2; p384 0.13.1; p521 0.13.3; rsa 0.9.10; ed25519-dalek 2.2.0; signature 2.2.0 | Trust, SSH and CA callers delegate typed keys and verification. Curve/point/algorithm admission, canonical public-key representation and original signed bytes remain application contracts; do not replace them with upstream defaults |
| sha2 0.10.9; base64 0.22.1; ssh-key 0.7.0-rc.11 | Hash, encoding and SSH syntax/verification are upstream. Raw fingerprints, padded/no-newline producer profiles and authorized key-purpose restrictions remain; transitive sha2/signature versions do not change direct caller APIs |
| x509-cert 0.2.5; selected der 0.7.10 / pem-rfc7468 0.7.0 | DER/X.509 are upstream; the remaining PEM line/body decoder can shrink under **SIMP-PEM-1**. Its upstream PEM feature is currently disabled |
| serde 1.0.229; serde_json 1.0.151 | Typed codecs are upstream. Identity's Value/remapping pipeline is challenged under **SIMP-I-JSON-1**. Release/acceptance raw or ordered visitors have separate signed-byte and producer contracts; their presence alone does not justify universal deletion |
| percent-encoding 2.3.2; url 2.5.8; time 0.3.55 | URL/calendar grammar is delegated. Invalid-percent preflight, authenticated literal preservation, exact wire grammar, precision and expiry policy remain; permissive upstream parsing is not equivalent admission |
| bytes 1.12.1; http-body-util 0.1.3; hyper 1.12.0; hyper-util 0.1.21; tokio 1.53.2 | Unix HTTP clients/listeners use upstream HTTP framing and runtime APIs. `BodyExt::collect` does not supply the current pre-growth body cap. Retain body admission, one deadline, driver cleanup and domain dispatch; synchronous alternatives are assessed below |
| tokio-postgres 0.7.18; selected postgres-types 0.2.14 | Upstream `Row::try_get`, `ToSql`, `FromSql` and `types::Json<T>` can replace the generic Field/Row and JSON string detours under **SIMP-I-PG-1**. Cancellation and transaction custody remain separate |
| tungstenite 0.30.0; ureq 2.12.1 | RFC WebSocket framing and external HTTP are upstream. Host upgrade/pump authority, queues, cleanup, credential/response admission and uncertain-operation policy remain. Native curl's resolver/custody replacement is still held only by L10.N4 |
| libc 0.2.190; rustix 1.1.5 | Kernel operations are upstream, including owned descriptors. **SIMP-FD-1** removes a concrete integer-to-typed flag bridge. The direct libc-use census is not a census of Soda wrappers; PTY, process, signal, poll, mount and peer custody require their own duties |
| tempfile 3.27.0; walkdir 2.5.0 | All 18 mapped tempfile uses have parent/mode/lifetime duties; no forwarding-only tempfile wrapper was found. Walkdir exposes native paths; Muse copy/config ordering, secret exclusion and symlink admission remain |
| flate2 1.1.10; tar 0.4.46 | Decompression, headers, checksums and PAX/GNU handling are upstream. Retain full gzip trailer/EOF drain after tar termination and exact declared-FD length admission: archive Builder copies until EOF. Completed L13 remains completed |
| clap 4.6.7; humantime 2.4.0 | CLI/duration parsing is upstream. Duration empty/minus prechecks can disappear; admitted leading-plus/Greek-mu normalization, zero refusal and the separate 24-hour option cap remain. Command tails, secret/action policy and error presentation stay with real commands |
| roxmltree 0.21.1; svgtypes 0.16.1 | DOM/namespace/path parsing is upstream. DTD/node admission, finite supported absolute rings and deterministic raster/output duties remain; no second XML/path grammar replacement is selected |

All **33 Go/browser map entries** receive a disposition, including support-only
entries. Selected external modules were read from the local module cache and
browser packages from their locked installations. Standard-library claims use
the exact cached **Go 1.26.7** source selected by module/shipping inputs, not the
host wrapper's Go 1.27.0. The SDK is the clean sibling checkout
`c92db11c14b773c9cc20ccfa4b853b4c017e8717`; nominal `v0.0.0` does not identify its API.

| Selected Go/browser APIs | Actual boundary and challenged disposition |
| --- | --- |
| forgejo.org/extension-sdk at c92db11c | Background operation methods already exist. Per-dial peer verification, typed refusal and shared admission/push ownership are the actual missing capabilities (**SIMP-SDK-1**). Typed snapshot values already permit **SIMP-SNAPSHOT-1** independently |
| coder/websocket v1.8.15; Go net/http 1.26.7 | Libraries own framing, Dial/Accept, readers/writers, limits and connections. `wsjson.Read` admits both message kinds and ordinary JSON; it cannot replace text-only strict terminal admission. Keep the bounded authorized pump. HTTP contexts fit the existing synchronous Go consumers |
| dicebear-go/v10 v10.7.0; x/crypto/ssh v0.55.0 | Upstream owns artwork and key syntax. Artwork version/seed/size, one public key/no options/no trailing object and raw fingerprint policy remain |
| oauth2 v0.34.0; Tailscale client/v2 v2.10.1 | Upstream owns token and key request protocols. Actual operation credentials, endpoint/scope, capped response and capability-field presence remain. SDK `Keys.Create` eagerly reads and ordinary-decodes its response; transport admission cannot be erased just because Key has boolean fields. A body/string copy can use a byte reader |
| x/sys/unix v0.47.0 | Upstream owns kernel operations. Cancellable nonblocking locks, owned descendant reaping and peer policy remain; no upstream cancellable flock API was found |
| pgx/v5 v5.10.0; modernc SQLite v1.58.0; testify v1.12.1 | pgx explicitly supports the product's database/sql driver and typed parameters/Scan. No Go SQL translator was found. SQLite belongs to developer fixtures, and testify to tests; neither is a production adapter engine |
| Go encoding/json 1.26.7 | Token/typed decoding are upstream, but v1 has no duplicate-key rejection option. Repeated RawMessage parsing and map normalization can shrink under **SIMP-GJSON-1**. Experimental build-tagged JSON v2 is not the selected API |
| Go encoding/base64, crypto/*, net/url and time 1.26.7 | Retain bounded producer admission, credentials/literals and expiry/cancellation state. Base64 Strict still ignores CR/LF. Selected crypto/rand.Read fails fatally on entropy error; its ignored error is not evidence of a predictable fallback |
| Lit 3.3.3; xterm 6.0.0; FitAddon 0.11.0 | Upstream owns rendering, terminal parsing/byte writes and fitting. Callback-bag ownership is a parked reassessment (**SIMP-BROWSER-1**). Browser's bounded fatal-UTF-8 response reader has an actual policy; Response.json does not replace it |
| jsdom 30.0.1; Playwright 1.63.0; TypeScript root 7.0.2 / checker 5.9.3; lit-analyzer 2.0.3; oxfmt 0.68.0; oxlint 1.81.0 | Developer APIs/compiler/analyzer/CLI consumers, with no production adapter engine. Lit checking uses TypeScript Program and LitAnalyzer rather than recreating their parsers; no LitAnalyzer defect follows from the separate Go complexity-gate finding |
| @types/bun 1.4.2; @types/node 24.10.1; @types/jsdom 30.0.0; dependency-only Go modules | Type declarations or graph support, not additional production integrations. The caller map records their selected identities and actual consumers |
| gocyclo v0.6.0; errcheck v1.9.0; staticcheck v0.8.1; gofumpt v0.9.1 | Selected Go developer tools, not product adapter engines; current scripts/tool directives already use upstream interfaces. H06-F1 concerns gocyclo/check-complexity.sh failure classification |

Source-owned `soda-wire-time` and standard IP/monotonic helpers retain the finite
wire and expiry duties above. The six [application library joins](README.md#application-libraries-and-composition)
are application ownership, not upstream limitations. Release build/deliver/image
models and pipeline conversions remain subject to D03/D05/D07/D09 workflow
contracts. Their different names or package boundaries do not prove that each
conversion is necessary; no unsupported wholesale model merger is selected here.

## Concrete refinements within the existing execution plan

These are maintainability recommendations, not newly established runtime defects.
Each packet has one accountable lead; independent review and coordinator-owned
manifest/lock work follow the existing lane rules. A conditional prerequisite
holds its own deletion only. No completed L04/L06/L08/L12 work is reopened by
labeling a later simplification as an unfinished original adoption.

### SIMP-I-PG-1: typed query parameters and rows

**Owner A, A05/L08.** Change private query results/parameters and actual
`store_connections`, `store_events`, `store_executions`, `store_grants` and
`store_leases` and `store_schema` callers together, including Store's query
entrypoints and their existing tests. [pg_query.rs](../../../../cmd/soda-identity/src/pg_query.rs)
currently turns every upstream row into generic Field values. JSONB becomes a
string and is parsed again by Store; writes serialize DTOs and the parameter
adapter parses them again. Use upstream typed rows and `Json<T>` at concrete SQL
sites; remove Field/Row and JSON string conversion with their last consumers.

Prerequisite: settle concrete nullable/type/range behavior with the existing SQL
schema and selected serde_json feature. Acceptance: no generic row type-switch or
JSON serialize/reparse path remains; actual null, int4 range and JSONB DTO checks
still exercise their callers. Preserve exclusive transaction access, one operation
deadline, cancel/drain/discard/reconnect and driver joins, and commit uncertainty.
Those contracts do not require this representation. Luna medium settles/reviews
the typed boundary; Luna low transfers repetitive callers after it is settled.

### SIMP-PEM-1: delegate the admitted PEM decoder

**Owner C, C05/L06.** Replace [pemx.rs](../../../../cmd/soda-install/src/pemx.rs)'s
manual boundary/body/Base64 processing at its actual
[local CA caller](../../../../cmd/soda-install/src/setup/local_ca.rs) with the
selected der PEM API exposed through x509-cert's `pem` feature. The pinned
[Caddy fixture](../../../../cmd/soda-install/src/x509/tests/fixtures/caddy-2.10.2-root.pem)
is one 631-byte, standard wrapped padded block; internal SP/HTAB acceptance has no
demonstrated producer requirement.

Prerequisite: coordinator enables the existing selected feature, and the owner
reconciles that narrower body profile with the
[installation contract](../../../guides/installation.md). Acceptance: the fixture
and documented surrounding-whitespace cases decode; the 16 KiB cap, exact
CERTIFICATE label, one-block/no-extras guard reject preambles/private keys/extra
objects before decoding. Upstream permits a preamble, so that guard remains.
Preserve original DER fingerprints, original-TBS verification and CA/algorithm
admission. Luna medium reviews the feature/profile and retained guard; Luna low
handles the settled decoder transfer.

### SIMP-I-JSON-1: retire an unnecessary Identity compatibility profile

**Owner A, A05/L04, with B producer coordination.**
[strict.rs](../../../../cmd/soda-identity/src/strict.rs) performs a recursive
duplicate/depth scan, builds a Value tree, remaps casing through field tables and
then decodes DTOs. L04 deliberately preserved Go casing/null semantics; removing
them is an explicit profile change, not correction of an omitted implementation.
The actual Go Identity client emits declared lower_snake tags, string integers
and Base64 bytes. Settings is separately loaded operator input; no production Go
Settings emitter was established, and Compose launch JSON is a different wire.

Prerequisite: adopt exact declared-key admission separately for HTTP Request and
startup Settings where their actual contracts permit it. Then use typed serde
unknown-field refusal plus the retained recursive duplicate scanner, and delete
Value/remapping/parallel field tables. Acceptance: retain HTTP's 512 KiB cap,
Settings' 1 MiB decode limit, UTF-8, object/EOF, decoded-key duplicates, depth 100,
nested unknown-field refusal, required string-integer/Base64 representations and
invalid-request behavior. Do not impose this profile on signed or other producer
JSON. Luna medium settles/reviews admission; Luna low transfers settled callers.

### SIMP-FD-1: accept upstream typed flags at the real callers

**Owner A, A03/L12.** [Project terminal sys.rs](../../../../cmd/soda-project-terminal/src/sys.rs)
accepts libc integer flags/mode in private `open_at` and immediately converts
them to rustix `OFlags`/`Mode`. Change that signature and its real account, keys,
cgroup, terminal creation/preparation, subscription and filesystem callers to
the upstream types. Delete the roundtrip without creating another flag facade.
Prerequisite: enumerate those private call sites under one writer. Acceptance:
the libc flag conversion disappears, and component admission, forced NOFOLLOW/
CLOEXEC, OwnedFd/parent custody and errno behavior stay at the same boundary.
Luna low performs mechanical transfer; Luna medium reviews custody.

### SIMP-SNAPSHOT-1: consume the SDK's typed wire values

**Owner B, B06.** [snapshot_transport.go](../../../../internal/forgejo/snapshot_transport.go)
serializes an already typed SDK answer and decodes it into a copied Soda wire
graph. Change SnapshotReader and its callers to consume the SDK wire values with
the existing revision-bound domain validation. Remove copied snapshot/issue/pull
wire DTOs and this in-process JSON roundtrip; map only actual domain differences.
Prerequisite: update the real reader and validation/tests together; no upstream
change is needed. Acceptance: compiler-checked wire fields, no alignment-by-JSON,
unchanged equal-idle bracketing, requested-family completeness, digests, hidden
evidence and repository binding. Luna medium settles/reviews the boundary;
Luna low transfers repetitive fields/callers.

### SIMP-GJSON-1: one Token preflight, then typed decode

**Owner B, C01/L04 handoff.**
[strictjson/decode.go](../../../../internal/strictjson/decode.go) reparses nested
RawMessages and marshals a map before typed decoding. Use an upstream Token walk
for object/EOF, recursive decoded-key duplicates and depth, then typed-decode the
admitted original bytes. V1 DisallowUnknownFields alone cannot enforce duplicates.
Prerequisite: explicitly retire sorted-map precedence for competing differently
cased aliases where admitted; retain it only if an actual caller contract
requires it. None was established here. Decoding original bytes changes the
current alias winner to v1's source order. The Token preflight and typed decode
are two traversals, avoiding the current repeated subtree parses and reserialization.

Acceptance: remove recursive RawMessage/tree normalization; retain byte/read-error
bounds, UTF-8, one object/EOF, duplicates, depth and unknown-field policy. Do not
globally force exact-case Go keys: actual Tailscale native `HaveNodeKey` handling
uses case-insensitive presence and redecodes original producer bytes. B03.C's
outer request-size concern remains a separate correctness finding. Luna medium
settles/reviews profiles; Luna low transfers settled admission plumbing.

### SIMP-SDK-1: move mirrored service-client machinery upstream

**Owner B, B06/L15; upstream SDK capability is the prerequisite.** The selected
SDK has the required operation methods, but `backgroundClient.post` does not
verify each new socket peer, errors expose formatted status rather than a typed
bounded refusal, and push/refresh is not bound to one shared client admission.
Its credential file path check also precedes opening, whereas Soda admits the
actual opened descriptor. Current
[background transport](../../../../internal/forgejo/background_transport.go)
and admission ownership implement real security and uncertain-operation policy.

Have the upstream service client supply per-dial peer checks, typed sanitized
status/reason, shared admission with authorized 401 refresh and client-bound push
headers, plus the required credential-custody API. Then delete the mirrored Soda
HTTP/admission engine; retain domain operation/bracket/retry decisions, not another
generic facade. Acceptance: same-descriptor bounded credentials, every-dial peer
pin, shared snapshot/publish admission, one permitted rebind, bounded complete
responses and no uncertain mutation replay. Match exact native capability
evidence before cutover. Luna medium reviews the API/security boundary. This
audit authorizes no sibling edit and does not hold SIMP-SNAPSHOT-1.

## Synchronous, string and ownership interfaces

The [Unix HTTP request helper](../../../../lib/unix-http/src/lib.rs) creates a
current-thread Tokio runtime per call. Its four actual consumers are synchronous
Identity HostClient, host BrokerClient, host Tailnet Control and the factory CLI.
A persistent owned Client could amortize construction for the three reusable
clients, but keeps `block_on` and adds shared scheduler/concurrency/shutdown
ownership; the one-shot CLI has little benefit. Async propagation crosses their
domain/provider interfaces and bounded blocking dispatch. No such change is
selected without demonstrated cost and a concrete ownership improvement.

Identity Store already owns one persistent runtime; it has no PG command channel.
Controller/Store/Tx and provider calls are synchronous behind domain state locks,
while the async listener uses bounded `spawn_blocking`. Propagating async would
change transaction/lock/dispatch ownership across that workflow. A PG worker
channel adds queues, replies, cancellation, transaction pinning and joins.
Neither is a proven local simplification. Upstream's borrowed transaction API
also needs its cancellation/uncertainty fit established before replacing the
existing guard. The typed-row cut is independent of these execution choices.

Terminal strings were challenged against xterm `write(Uint8Array)` and the
libraries' binary WebSocket APIs. Guest IPC still uses line-framed JSON/Base64;
changing only browser messages would retain that machinery and add another
boundary. No binary protocol migration is selected. General error conversion is
likewise not automatically removable: sanitized operation uncertainty, public
refusal and contextual command errors have real consumers. SDK formatted status
strings are the concrete upstream interface defect identified above.

**SIMP-BROWSER-1 — owner A, A07/A08/R02, parked reassessment.** SodaTerminal's
`actionsInput`, `screenInput` and `attachmentInput` repeatedly construct callback
bags around one private owner; Lit/xterm do not require that representation.
Before further splits, assess direct typed Terminal/FitAddon access through the
existing coherent attachment/screen owner and a small view interface. This is
not yet a demonstrated net deletion: useful helper/test seams could be lost.
Prerequisite: name retained duties and show fewer interfaces/state owners without
a new controller framework. Acceptance: preserve fixture renderer injection,
authorization/generation/visibility cleanup, detach versus End, output queue
release on xterm's write callback, fit bounds and generated/shipped assets.
Luna medium assesses/reviews; no implementation packet is selected yet.

The source-wide Go census finds no caller of `RevokeCurrentToken`; its definition
is a bounded L18 last-reference cleanup candidate. Actual setup revocation is
Rust-owned. Confirm current references before deletion and retain the live actor
REST client. This does not create another compatibility obligation.

## Evidence, review and remaining gates

Luna medium primaries covered crypto/profiles, native transport/storage and
format/descriptor mechanisms; the coordinator covered Go/browser callers.
Independent medium challenges checked consequential cuts and retention, and
Luna low checked scoped metadata. Review corrected weak
tempfile deletion advice, duration-policy location, SDK capability assumptions,
JSON producer identity and claims that async callers already existed. The caller
census and fresh defining-body/upstream inspections remain distinct evidence.

This completes selected-family adapter challenge at the recorded scope, with
conditional profile/upstream decisions and the browser reassessment visible.
It does not establish a new complete workflow validity review, native producer
equivalence or runtime performance. Earlier [workflow findings](../workflow-traces.md),
[observation findings](../observation-reliability.md), L10.N4, CFG01 semantic fit,
Q5/native qualification and optional L16 keep their recorded scopes; no global
hold or extra compatibility layer follows. There were no application, manifest,
lock, dependency or runtime configuration changes and no executed tests, builds,
database, provider, VM or native operations.
