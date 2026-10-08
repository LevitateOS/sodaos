# Data representations and serialization

The audit finds concrete repeated work in PostgreSQL values, SDK snapshots,
Acceptance evidence and development-record formatting. Other reductions need
specific input-profile decisions before visitors or dynamic trees can disappear.
The [existing tasks](implementation-tasks.md) and [lanes](implementation-lanes.md)
remain the execution plan; the packets below refine their scope and acceptance.
Completed L04 adoption and earlier correctness repairs remain completed.

Source: `463ce7724d91807a1e2c0e8b0bfc318ee844ed1f`, tree
`19a8a9e6a306e38f37c05ef9766bfe58b0d08d4d`, inspected 2026-10-07. Application
source remains `0b0734398f0350d75cc6fdb4dc8129251d8ab308`; the 12 pre-existing dirty
guidance files and 39 dependency inputs are unchanged. This extends the
[adapter challenge](library-integrations/adapter-challenges.md),
[caller maps](library-integrations/README.md) and
[workflow traces](workflow-traces.md) with representation-specific questions.
Older L04 profiles supply discovery/history; they are not permanent requirements
for an unreleased product. Fresh inspection addresses the actual producers,
defining codecs and consuming operations at the boundaries below.

## Authority and byte identity

Three questions must stay distinct. Which values and admission rules does the
current producer/consumer require? Which output needs deterministic generation?
Which already observed bytes have hash, signature or fingerprint authority?

Original `payload.json`, candidate/media/evidence inputs and OCI blobs/manifests
must be hashed from the observed byte vectors before interpretation. Verification
of an already signed manifest must not hash a decoded/re-encoded DTO. Trust-key
fingerprints still use original decoded DER, SSH fingerprints their admitted key
bytes, and certificate verification original TBS bytes. The owning
[release trust contract](../../architecture/release.md) remains authority.

A newly generated unsigned record can use a simpler serializer. Derive all hashes,
layers and signatures from that one emitted vector and preserve it after generation.
Determinism does not require the old Python/Go whitespace, HTML escapes or captured
manifest digest. Retire obsolete output goldens when changing that producer profile;
retain assertions about values, stable generation, trust custody and real consumers.
RawValue is useful for opaque unowned input and exact integer tokens; its presence
does not make every deferred field parse necessary.

The subsequent [test/evidence audit](test-evidence.md) at `a0bdba84` narrows
old emitter/error and Python layout goldens by actual current consumer profile.
It preserves original signed bytes, fingerprints, domain refusals and deterministic
current generation while separating fixture execution from native proof.

## Producer and representation coverage

Tracked discovery found 451 source/support/template files with 3,710 JSON-related
selectors. Those are seeds, not 451 fresh body reviews or 451 adapters. The scope
below covers defining representation mechanisms and named producer/caller joins;
unchanged caller and responsibility censuses retain their narrower evidence.
Selected APIs are Serde 1.0.229, serde_json 1.0.151 and exact Go 1.26.7 v1 JSON.
Experimental JSON v2 is outside the selected build. Native producer behavior was
not exercised; its uncertainty holds only the dependent profile/cutover.

| Boundary and actual producer → consumer | Representation rules and disposition |
| --- | --- |
| Go Identity client → Rust Request/HTTP DTOs | Current tagged lower_snake names, quoted i64 and padded Base64 are real wire forms. The Go credential producer omits nil/empty values. No current producer emits case collisions or scalar null. Exact-key/non-null HTTP admission can retire compatibility remapping; operator Settings is a separate input decision |
| Rust Identity response → Go response decoder | Typed serializer emits unique declared members; Go uses unknown-field refusal and EOF. Keep matching scalar/byte forms. Provider and wire integer helpers have different null policies; identical-looking helpers are not automatically the same contract |
| Rust Identity Store → PostgreSQL JSONB → Store | DTO-to-string-to-Value writes and Value-to-string-to-DTO reads have no signed text authority. Upstream typed JSONB and rows can remove the detours. Schema/key checks also consume generic Row; include them in its retirement |
| Go Store DTOs → PostgreSQL columns → Go domain | Serialization at this persistence boundary is real. No query-string JSON translator was found. Keep domain/persistence ownership and intentional nullability; do not collapse wire, domain and SQL types just because fields resemble each other |
| Selected SDK NativeSnapshot → Soda SnapshotReader/validation | A typed answer is marshaled and decoded into a copied wire graph. Consume the SDK value directly while keeping revision bracketing, repository/hidden evidence and completeness/digest validation |
| Go strictjson and Rust host strict admission → owning DTOs | Both normalize root maps before typed decode; host also scans raw member subtrees. Sorted alias precedence is inherited implementation behavior. Original-byte typed decode needs an explicit alias decision while retaining duplicate/depth/object/EOF/UTF-8/caps and owner field policy |
| Go host clients → Rust terminal/factory/native request DTOs | Current clients emit their declared fields and scalar forms. Integer-token checks, Base64 byte frames and operation validation remain meaningful. Tolerant external readers keep their own extension/default policy; do not impose one universal strict schema |
| Guest state/factory records → domain checks and emission | Dynamic nesting, duplicate lookup/coercion and output order have actual consumers. Audit each projection before removing StateValue. Old Python ASCII formatting alone does not justify a second serializer; current numeric conversion and comparison duties must be traced separately |
| PTY bytes → JSON/Base64 lines → relay/browser | The current line protocol transports arbitrary bytes and typed control/dimension fields. Keep exact frame admission, queues and byte handling. A browser-only binary change leaves guest framing in place and is not selected |
| Muse/Compose → host launch socket and launch exit | Requests have different schemas; exit DTOs have related small default/unknown/duplicate behavior. Preserve current producer forms and socket framing. Small copied formatter helpers do not justify a new generic JSON crate |
| Muse-maintain config/release readers | Config currently consumes the first value; release admission requires a complete bounded payload. These are distinct duties. Changing suffix/null/alias behavior requires their own producer/input decision, not a shared decoder by name |
| Acceptance DTOs → scrub tree → evidence file | One bounded serialize/parse creates the tree used for scrubbing; a second compact encode/parse repeats work. Remove that second pass while retaining limits, numeric values, sorting, redaction collision refusal, leak scans and publication error ordering |
| QMP/native/project-state JSON → Acceptance dynamic projections | Unknown trees and raw number tokens can include values ordinary floating-point conversion cannot preserve. Keep opaque/raw representation where a real consumer needs it. The evidence-emitter cut does not authorize deleting the whole dynamic tree |
| Setup DashboardConfig → Go config, activation, welcome and installer access | One local file, different consumed fields and failure duties. Known scalar readers can use thin typed projections after a coordinated null/duplicate profile. Preserve presence-based refusal of `public_url`, including null, and the welcome command's safe non-authoritative banner |
| Browser API/layout producers and consumers | IDs are intentional decimal strings validated with BigInt/i64 bounds, not unsafe JS numbers. Bounded fatal-UTF-8 JSON reading and capability/shape checks remain. Layout's serialize/parse verifies final saved bytes and strips unsent locators; no rewrite selected merely to remove that bounded check |
| Candidate/lab tools → worker/trust/config readers | Concrete DTO fields and roles match their paired consumers. Two copied EnsureAsciiPretty state machines preserve Python formatting without a demonstrated consumer need. Standard deterministic pretty JSON can replace them without changing decoded keys, paths or public DER |
| Build live inputs/file/toolchain records → image/build callers | Controlled Rust producers emit canonical names. Raw-slot case-fold/last-wins codecs and wire graphs are candidates for a selected current profile. Keep original input hashes, field validation and actual domain differences; allocate one existing owner before sharing DTOs |
| Image Payload → pipeline DeliverPayload → verifier/importer | Fields overlap, but images Vec/BTreeMap and upgrade Vec/Option encode different representations. The conversion feeds actual domain validation. Resolve these contracts before sharing a wire model or narrowing borrowed validation; do not delete a domain boundary as a redundant DTO by inspection alone |
| Image Payload → installer `deliver::load` | Allocate direct local installer `Deserialize` records for the actual 12-field required PascalCase producer payload; preserve authenticated original bytes/hash and existing semantic validation. Keep this installer consumer separate from pipeline DeliverPayload/verifier/importer owners |
| Image config/native Ignition → edit/readback checks | Image config can keep unowned nested tokens opaque rather than recursively rebuilding them. First-non-null lookup and updating duplicate names are current operations. Ignition comparison needs explicit duplicate, integer and null-normalization rules before ordinary semantic JSON replaces the ordered tree |
| Tracked provisioning base.json → rendered Butane input | The sole current base has unique names and small integer values. A custom duplicate/raw-number Node and Python formatter add no required semantics to this controlled source. Ordinary JSON values preserve unmodeled future keys; retain private string validation, mutation and output custody |
| OCI layout/index/manifest/config → installer verification | Allocate direct local OCI v1.1.1 typed records using exact upstream field names and tolerating unknown extensions at every object level. Required fields and omitted/null/wrong-value behavior follow the settled rules in the [execution allocation](execution-findings.md#rank-2-costly-boundary-and-profile-decisions); preserve original blob authentication and descriptor/path/type/digest/layer budgets. Adjacent importer remains separately owned |
| Native lsblk/ip JSON → installer disk/address projections | Fixed commands request known fields, but selected native shape is not source-qualified. Typed projections are conditional on that profile. Disk refusal/destructive revalidation, kernel identity/mount/holder checks and address filtering stay with their domain owners |
| Release record generation → OCI packaging/signing | Deterministic standard emission can replace copied Go formatters at controlled output sites. Secret-input escaped variants and arbitrary external/raw JSON are separate consumers; do not delete their admission helpers as an emitter side effect |

## Concrete handoffs in the current task and lane plan

These are representation simplifications and profile decisions, not newly proven
runtime defects. No new correctness counterexample was established within this
recorded scope. Each row has one accountable lead; coordination does not create
another owner for a shared file. Luna medium settles/reviews unresolved contracts;
Luna low transfers repetitive callers once those contracts are settled. The
coordinator owns manifest/lock edits in a later authorized implementation.

New concrete defining scopes are Acceptance's
[evidence store](../../../tools/acceptance/src/evidence/store.rs), the
[candidate formatter](../../../tools/candidate-setup/src/config.rs) and
[lab formatter](../../../tools/lab-credentials/src/process.rs), and the
[provisioning document](../../../tools/release-assets/src/render/provisioning/document.rs).
Conditional admission scopes include [host JSON](../../../lib/host/src/json/mod.rs),
[activation](../../../cmd/soda-activate/src/activation.rs),
[welcome](../../../cmd/soda-console-welcome/src/config.rs),
[installer access](../../../cmd/soda-install/src/setup/access.rs) and the
[Go config reader](../../../internal/config/config.go).

| Packet / readiness / owner | Defined scope and prerequisites | Acceptance checks for later implementation |
| --- | --- | --- |
| **SIMP-I-PG-1**, specified; A/A05/L08 | Existing [typed-row packet](library-integrations/adapter-challenges.md#simp-i-pg-1-typed-query-parameters-and-rows), including store_schema and query tests. Use `Row::try_get` and selected `Json<T>`; leave execution ownership intact | No generic Field/Row conversion or JSON string detour; actual NULL/int4-range/JSONB DTO cases and schema checks pass; preserve exclusive Tx, cancellation/deadline/drain/discard/join and commit uncertainty |
| **SIMP-SNAPSHOT-1**, specified; B/B06 | Existing [SDK wire-value packet](library-integrations/adapter-challenges.md#simp-snapshot-1-consume-the-sdks-typed-wire-values). Move real reader, validation and callers/tests together | Copied wire graph and marshal/decode alignment disappear; revision/visibility/completeness/digest/repository checks still use the real reader. SDK transport capability gate is independent |
| **SIMP-I-JSON-1 / SIMP-GJSON-1**, profile decision; A/Identity HTTP and B/Go admission respectively | Refine the existing separate packets. Identity HTTP current producer supports exact tagged forms; do not apply that decision automatically to Settings. Retire sorted alias precedence only at callers whose input contract permits it | Delete Value/remapping or RawMessage normalization after selection; current producers round-trip with duplicate/depth/unknown/EOF/UTF-8/caps, quoted i64/Base64 and sanitized failure preserved. Tailscale case-insensitive field presence remains its external profile |
| **REP-HOST-STRICT-1**, profile decision; A/host admission, A07/L04 | In `json::decode_strict_as`, replace root BTreeMap/raw member recapture/re-emission with recursive unique/depth preflight followed by owner DTO decode of original bytes. Decide sorted-versus-source alias precedence per actual host DTO first | Root map and `to_vec` normalization disappear; preserve duplicate decoded names at every depth, exact depth counting, root object/EOF/UTF-8/body caps, owner unknown/null/number/byte admission and operation refusal |
| **REP-ACC-EVIDENCE-ROUNDTRIP-1**, specified; C/D06/L01 | Remove only the second compact serialization/parse in Evidence::encode_scrubbed_json. Keep the bounded first Serialize/tree conversion, then scrub/sort and emit through upstream pretty serialization | Preserve depth and pre-growth/final byte budgets, raw valid number values, deterministic sorted keys, key-redaction collision refusal, escaped/literal leak scans, LF and exclusive publication/failure joins. Reuse existing typed/dynamic evidence subjects; no old Go byte-equality gate |
| **REP-FMT-1**, specified current producer change; C/H06/C10/L04 | Replace both candidate/lab EnsureAsciiPretty implementations with selected standard Serde pretty output. Keep local DTOs and caller newline choices; no shared formatter crate | Same inputs give stable new bytes; worker/trust/config readers receive identical fields and Unicode/control/path/key values; integer widths and decoded-DER fingerprints remain. Replace formatter-only goldens with parsed-field, determinism and consumer checks; retire formatter machinery |
| **REP-CFG-1**, profile decision; C/C05 | Coordinate current dashboard names/null/duplicate rules with B's whole-config reader. Replace activation/welcome/access scalar RawValue/Value paths with command-local consumed-field DTOs; unknown current optional fields remain ignored by projections | Keep positive operator i64, presence-based public_url refusal including null, URL/listener/private-origin validation, read custody/caps, safe welcome banner and no activation effect on invalid input. Forgejo INI/CFG01 is a separate decision |
| **SIMP-ASSET-NODE-1**, specified; C/release-assets/H06 | Replace provisioning Node/Python formatter with serde_json::Value and standard deterministic emission for the sole tracked base source. Keep entry/private-data mutations | Static base and inserted fields retain their JSON values, private strings round-trip, integer mode remains exact, unknown future source keys survive, output custody/security checks remain and Butane receives the current document. No Python output equality requirement |
| **SIMP-REL-EMIT-1**, output-site decisions; C/D03/D07/L04 | Replace Go formatters only at named controlled unsigned record writers. First account for any remaining secret-escape/external/raw-data consumers of shared helpers | Stable field/map order and caller newline; recompute layer/config/manifest/candidate hashes from actual new bytes; preserve the emitted vector through packaging/signing and original signed-input verification. No captured historical digest gate |
| **SIMP-REL-WIRE-1**, source complete in `3ccbc127`; C/D03/D05/D07 | The existing `lib/release-inputs/src/reader/stream.rs` owns required exact PascalCase records, `deny_unknown_fields`, shared validation and sorted `BTreeMap` emission; build retains bounded read/write I/O, and existing Rust build/image consumers import the canonical records directly. The deleted image `model/live_inputs.rs` leaf is not recreated. Payload wire/domain contracts remain separate | 27 source/graph paths; 54 focused tests across nine selectors and all three release-tools binaries in a locked offline check passed; independent medium review PASS and mechanical ACK. Only two existing-crate graph edges were added, with no package/version changes. No native/installed qualification. See current [execution allocation](execution-findings.md) |
| **SIMP-REL-ORDERED-1**, source complete in `feb95e50`; C/D03 | The two consumers now use different representations; the recursive `OrderedValue` tree and `ImageConfig` wrapper are removed. Image-config edits use direct `BTreeMap<String, Box<RawValue>>` for opaque nested values/tokens, edit owned root fields, emit sorted root JSON and recompute staged/signing hashes; selected primitive emission uses upstream `to_raw_value`. Exact-key then first case-folded setting lookup, missing/null behavior and fast-mode policy remain. The iterative scan preserves a 140-container unknown subtree plus `1e400` and greater-than-u64 tokens after actual freeze; capture is capped at 16 MiB before append/retention. Ignition comparison uses ordinary `serde_json::Value`, recursively prunes null object members and retains array order; tests accept depth 127 and refuse 128. Existing gzip/base64/EOF checks and raw `live.ign` custody remain | Seven paths, 196 additions/381 removals including tests; 10 focused tests across four selectors with zero ignored, locked offline check of all three release-tools binaries, and independent medium final review PASS. No crate graph changes, pinned image specimen or native/installed qualification. Upstream image roots are unique and sorted; this does not claim general duplicate-equivalence behavior. See current [execution allocation](execution-findings.md) |
| **SIMP-INSTALL-OCI-1**, selected installer-local profile; C/C10/D11 | Plan direct local `Deserialize` records in `cmd/soda-install/src/deliver.rs` for the required PascalCase image payload; preserve authenticated original bytes/hash and existing semantic validation. Plan local OCI v1.1.1 typed metadata/layout records with exact upstream names, recursively tolerated unknown fields and the settled null/default rules in the [execution allocation](execution-findings.md#rank-2-costly-boundary-and-profile-decisions). Preserve strict UTF-8 and keep this installer consumer separate from the pipeline DeliverPayload/verifier/importer owners | Plan a `model::Payload` Serialize → `deliver::load` regression through one acyclic test-only installer → existing soda-release-image edge; root owns dev-dependency/lock and graph integration. Keep `seal_candidate_payload` custody source-inspected. No shared DTO, package or production facade; adjacent image-import, Muse and release-deliver decoder cuts remain separate |

No packet authorizes a new universal JSON binder, cross-language schema generator,
compatibility reader or error framework. Keep application validators at the actual
authority/state owner. Removing machinery includes its last production callers,
helpers and obsolete fixtures; moved or newly equivalent machinery counts against
the claimed simplification.

## Review and completion limits

Luna medium primaries inspected release/install, native/guest/Acceptance and
Identity/Go Store/SDK mechanisms; the coordinator inspected browser/dashboard and
small producers. Independent medium challenges corrected blanket retention of
old formatting goldens, premature shared-payload/ordered-tree claims, missing null
and integer cases, and labeling five conditional release candidates as ready.
Concrete source-supported cuts and conditional decisions are distinguished above.

Source coverage is complete for the recorded defining mechanisms and caller
questions, with discovery/census explicitly separate. Producer/workflow contracts
are established where the source owns both ends; native lsblk/ip/OCI/Ignition and
operator admission decisions retain their bounded uncertainty. Target allocation
and handoff checks are specified for the listed packets; the full desired tree
and historical 80-slice validity review are not regenerated or advanced.

This is documentation and source inspection only. No application, manifest, lock,
runtime or dependency changes, tests, builds, DB/provider/VM operations or native
qualification ran. [Observation findings](observation-reliability.md), existing
workflow correctness findings and optional L16 retain their current scopes.
