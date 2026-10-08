# Release production

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

Pending work follows [library-adoption findings](../library-adoption.md#finding-allocation)
and their [execution packets](../library-adoption.md#execution-packets). Historical
source spans and completed C08/C09 structural work remain evidence, not new
extraction tasks. Replace selected generic engines through the existing release
owners; retain admission, descriptor custody, authenticated raw bytes,
publication order and native tooling authority. Dependency/cache/toolchain and
fixture checks precede large cutovers at the
[readiness gates](../library-adoption.md#readiness-gates).

## internal/release/build/coreos_stream.go

Observed size: 518 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: resolveStreamBuild at 232; ResolveCoreOSISO at 359; ResolveCoreOSQEMU at 376; ResolveTailnetInputs at 68; latestTailnetRelease at 95; latestTailnetBaseTag at 122; fetchCappedJSON at 190; resolveRegistryDigests at 289; WriteLiveInputs at 438; ReadLiveInputs at 450; ValidLiveInputs at 462.

## internal/release/build/oci.go

Observed size: 711 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: InspectOCI at 175; InspectOCIContent at 475; openOCIArchive at 46; readOCIArchiveEntries at 144; readOCIIndex at 545; whiteoutTarget at 220; scanOCILayer at 307; scanOCIArchiveLayers at 399; parseOCIManifest at 603; validateOCIRootFS at 628; validateOCIAttribution at 644.

## internal/release/build/production.go

Observed size: 591 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-build`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Production at 24; step at 35; validate at 42; SodaCommands at 52; CompileRust at 77; Compile at 108; Assets at 127; Dependencies at 156; assetSteps at 178; admitResolvedInputRecord at 276; pullResolvedInput at 285; ResolveInputs at 349; buildImage at 376; exportAppImages at 448; exportImages at 538.

## internal/release/deliver/model.go

Observed size: 570 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-deliver`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: Hash at 29; Digest at 30; admitTrustRoleKeys at 73; Role at 107; ValidCandidateContent at 156; requiredCandidateContent at 179; MediaBinding at 230; References at 334; EmptyState at 368; AdmitChannel at 483; AdmitRelease at 535.

## internal/release/image/assemble.go

Observed size: 736 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-image`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: assembleMedia at 585; admitMediaInputs at 273; fetchAssemblerConfig at 72; verifyAssemblerLayers at 119; prepareAssembler at 168; collectMediaInventory at 288; verifyMediaInventory at 327; authenticatePackagingInputs at 337; buildMediaContainer at 375; assembleNativeMedia at 387; verifyBuildMeta at 429; customizeInstallerISO at 483; verifyMediaReadback at 508; sealMedia at 545; VerifyLiveIgnition at 651; VerifyRootfsChunks at 711.

## internal/release/image/build.go

Observed size: 671 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-image`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: ValidateTarget at 65; Build at 148; runBuild at 571; verifyCheckoutSource at 198; extractBuildSnapshot at 300; setupBuildWorkspace at 315; compileSodaCommands at 405; compileRustTools at 463; compileShippingTools at 476; runBuildCommand at 623; linkPreparedAssets at 658; failureReason at 184.

## rust/soda-asset-fetchers/src/muse.rs

Observed size: 521 lines, including tests where embedded. Keep the 222-line pinned Muse fetch/stage path together; most of this file is test fixtures and negative cases. Extract the descendant tests rather than split the small fetcher into extra production modules. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/fetch/muse.rs`
- `tools/release-assets/src/fetch/muse/tests.rs`

Evidence: rust/soda-asset-fetchers/src/muse.rs:68-120 load_release; rust/soda-asset-fetchers/src/muse.rs:151-220 streaming stage and fetch_muse; rust/soda-asset-fetchers/src/muse.rs:224-521 manifest, integrity, output and transport tests.

## rust/soda-asset-fetchers/src/tea.rs

Observed size: 542 lines, including tests where embedded. Keep the 163-line Tea fetch/license staging path intact; extract its 377 lines of fixtures and integrity/exclusivity cases. No new fetcher framework is needed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

Library boundary: CF-08 retains the small bounded ELF architecture predicate
and consolidates its duplicate mechanics with the explicit Tea admission profile.
Do not replace this header check with a full object parser. The existing HTTP,
tar/gzip and digest engines already use libraries; keep pinned license, integrity
and exclusive-output policy with the fetcher.

- `tools/release-assets/src/fetch/tea.rs`
- `tools/release-assets/src/fetch/tea/tests.rs`

Evidence: rust/soda-asset-fetchers/src/tea.rs:75-162 upstream tag/checksum/ELF/license fetch and staging; rust/soda-asset-fetchers/src/tea.rs:165-228 local test fixtures; rust/soda-asset-fetchers/src/tea.rs:230-542 integrity, output and upstream refusal tests.

## rust/soda-forgejo-locales/src/locales.rs

Observed size: 568 lines, including tests where embedded. Keep the 319-line catalog admission/merge and locked-input operation intact after extracting tests. CFG02 retains the bounded duplicate-name/section-set scanner and original-byte merge: ini_core would leave its multiline/trim grammar in another adapter. Narrow obsolete Python quirks only through an explicit catalog contract change. Keep tests with the real merge owner; do not introduce a generic INI framework.

- `tools/release-assets/src/locales/merge.rs`
- `tools/release-assets/src/locales/merge/tests.rs`

Evidence: rust/soda-forgejo-locales/src/locales.rs:115-201 parse_ini and byte-preserving merge; rust/soda-forgejo-locales/src/locales.rs:214-318 locked/native input and exclusive output; rust/soda-forgejo-locales/src/locales.rs:321-568 parser/merge/source-integrity tests.

## rust/soda-forgejo-locales/tests/cli.rs

Observed size: 462 lines, including tests where embedded. Group current help/flag admission separately from local catalog merge/output behavior and lock-document admission. Reuse the existing local TempDir/command helpers; keep tests that reject a lock before network access.

- `tools/release-assets/tests/locales_cli.rs`
- `tools/release-assets/tests/locales_native_merge.rs`
- `tools/release-assets/tests/locales_locked_input.rs`
- `tools/release-assets/tests/locales_support/mod.rs`

Evidence: rust/soda-forgejo-locales/tests/cli.rs:23-71 shared fixture and binary invocation helpers; rust/soda-forgejo-locales/tests/cli.rs:73-184 native merge, defaults and size bound; rust/soda-forgejo-locales/tests/cli.rs:185-253 CLI argument/help checks; rust/soda-forgejo-locales/tests/cli.rs:254-330 lock source/document refusal; rust/soda-forgejo-locales/tests/cli.rs:332-462 merge namespace/exclusive-output refusals.

## rust/soda-stage-render/src/provisioning.rs

Observed size: 614 lines, including tests where embedded. Keep document admission, private-input/host-public-key/output custody and render orchestration distinct. JSON01 replaces custom Python-style JSON construction with serde/serde_json at the document owner. Keep secret-file handling, native ssh-keygen authority and exclusive destination admission; PROC02 retains bounded concurrent output drains and failure cleanup.

- `tools/release-assets/src/render/provisioning/mod.rs`
- `tools/release-assets/src/render/provisioning/document.rs`
- `tools/release-assets/src/render/provisioning/private_files.rs`
- `tools/release-assets/src/render/provisioning/render.rs`
- `tools/release-assets/src/render/provisioning/tests.rs`

Evidence: rust/soda-stage-render/src/provisioning.rs:79-144 regular/private file and hostname input checks; rust/soda-stage-render/src/provisioning.rs:146-331 Python JSON emission, file entries and public template; rust/soda-stage-render/src/provisioning.rs:336-469 public host-key derivation and exclusive output; rust/soda-stage-render/src/provisioning.rs:471-564 RenderInputs and render; rust/soda-stage-render/src/provisioning.rs:566-614 hostname and exact JSON tests.

## rust/soda-stage-render/src/stage.rs

Observed size: 518 lines, including tests where embedded. Separate confined destination copying/tree normalization from current payload/terminal locks and host/Forgejo branding adaptation. Retain one run sequence and the existing current generated browser inputs. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/release-assets/src/render/stage/mod.rs`
- `tools/release-assets/src/render/stage/files.rs`
- `tools/release-assets/src/render/stage/payload.rs`
- `tools/release-assets/src/render/stage/branding.rs`
- `tools/release-assets/src/render/stage/tests.rs`

Evidence: rust/soda-stage-render/src/stage.rs:36-161 native platform, directory/copy/normalization helpers; rust/soda-stage-render/src/stage.rs:162-262 payload manifest/terminal pins/favicon/build-source resolution; rust/soda-stage-render/src/stage.rs:264-478 context admission and appliance/Forgejo/terminal staging; rust/soda-stage-render/src/stage.rs:482-518 ICO layout and source-token tests.

Open detail: Source-root detection and payload/asset path strings must change together with the project-wide tree. Existing runtime binary destinations remain the installed contract.

## rust/soda-stage-render/src/terminal_logo.rs

Observed size: 630 lines, including tests where embedded. XML01 replaces the XML cursor with roxmltree 0.21.1 and the path tokenizer with svgtypes 0.16.1 PathParser. Keep emblem admission distinct from polygon sampling and terminal-text output/check orchestration. Preserve the canonical rendered text rather than the custom parser's incomplete XML/SVG lexical grammar.

- `tools/release-assets/src/render/terminal_logo/mod.rs`
- `tools/release-assets/src/render/terminal_logo/svg.rs` — Namespace-aware element/path admission over roxmltree/svgtypes, with DTD/external-entity handling disabled and bounded input.
- `tools/release-assets/src/render/terminal_logo/geometry.rs` — Existing admitted polygon sampling and ASCII/color rendering, without a path lexer or full SVG renderer.
- `tools/release-assets/src/render/terminal_logo/tests.rs`

Library boundary: Keep the explicit viewBox, two fills, evenodd rule and absolute M/L/H/V/Z command subset; reject groups/transforms and unsupported commands. Resolve element names by namespace, not prefix spelling. Admit valid upstream SVG number syntax deliberately at the fixture gate. This remains a restricted emblem renderer.

Evidence: rust/soda-stage-render/src/terminal_logo.rs:21-119 path tokenization and polygon extraction; rust/soda-stage-render/src/terminal_logo.rs:121-176 evenodd raster and terminal coloring; rust/soda-stage-render/src/terminal_logo.rs:178-462 restricted SVG cursor/parser; rust/soda-stage-render/src/terminal_logo.rs:464-516 render_svg and canonical output/check; rust/soda-stage-render/src/terminal_logo.rs:519-630 tokenizer/geometry/SVG/output tests.

## rust/soda-stage-render/tests/cli.rs

Observed size: 768 lines, including tests where embedded. Split the three CLI suites at their existing labeled sections and share only the current TempDir/command/copy helpers. Keep golden provisioning files and canonical branding comparisons as their existing byte assertions.

- `tools/release-assets/tests/render_staging.rs`
- `tools/release-assets/tests/render_provisioning.rs`
- `tools/release-assets/tests/render_terminal_logo.rs`
- `tools/release-assets/tests/render_support/mod.rs`

Evidence: rust/soda-stage-render/tests/cli.rs:22-118 common fixture, root lookup, binary and command helpers; rust/soda-stage-render/tests/cli.rs:120-284 soda-stage CLI suite; rust/soda-stage-render/tests/cli.rs:286-670 soda-render-provisioning CLI suite; rust/soda-stage-render/tests/cli.rs:672-768 soda-render-terminal-logo CLI suite.

Open detail: repo_root currently assumes the crate sits two levels below the checkout root at lines 64-69. Fixture paths include the old payload manifest path and require coordinated updates.

## tools/soda-candidate/display.go

Observed size: 415 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: renderer at 114; buildLines at 250; startTicker at 303; parseEvent at 31; parseArtifactEvent at 93; hostArtifactPath at 81; printWhyPanelLocked at 356; exitMeaning at 373; wallDur at 384.

## tools/soda-candidate/main_test.go

Observed size: 619 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: TestValidateArchFlagAdmitsOnlyX8664 at 16; TestResolveOptionsDefersWorkerAdmissionToController at 552; TestOverviewStartsWhenValid at 113; TestOverviewBlocksBadStartWithoutLosingAnswers at 133; TestServeAndFileRootfs at 254; TestFileBuiltRootfsCoversNonLoopbackMedia at 292; TestParseControllerEvents at 27; TestRunningPhaseShowsLiveElapsed at 379; TestFailedRunPrintsWhyPanelWithHostPaths at 397; TestCheckCleanTreeRefusesUntrackedFiles at 319; TestCopyFileRefusesOccupiedPickup at 349; TestReadyRunTouchesNoWorkerState at 573.

## tools/soda-candidate/prompts.go

Observed size: 425 lines, including tests where embedded.

Disposition: retire this predecessor and tests belonging only to it at the
decided cutover. Its surviving responsibilities belong to `lib/soda-release-tools`.
No decomposition leaves for this obsolete implementation appear in the
target tree. Preserve its actual behavior through the retained owner and
update real callers, payloads and verification together.

Evidence: prompter at 17; line at 56; choice at 76; askAbsolute at 333; askOut at 380; overviewFields at 130; baseFields at 140; mediaFields at 151; editMode at 160; overview at 240; renderOverview at 272; dispatchOverviewCmd at 288; startIfValid at 299.

## rust/soda-release-build/src/coreos.rs

Observed size: 476 lines, including tests where embedded. Keep the existing ordered QEMU authentication chain, its CoreOSImage/VerifiedBase records, bounded HTTPS copy and three small unit cases together. Extract the command drain used by both xz expansion and gpgv verification into a private CoreOS child; its timeout/nonblocking/output-limit mechanics are a real shared concern. This reduces the parent without inventing a general executor or changing the download, checksum, signature, expansion and retained-record order.

- `lib/soda-release-build/src/coreos.rs` — Existing records, native/signer/keyring admission, exclusive bounded HTTPS download, compressed checksum and detached-signature binding, QEMU expansion/hash and verified-base record; retain the current three tests here.
- `lib/soda-release-build/src/coreos/process.rs` — Existing run_bounded and set_nonblocking implementation used by xz and gpgv; child visibility only as required by those callers.

Evidence: 15-64: CoreOSImage/VerifiedBase and emit/marshal; 65-85: https_url/valid_signer/admit_coreos_fetch; 86-165: download_http/download/download_verified_archive; coreos_iso.rs:4,26-38 uses the same download/signature closure; 166-268: run_bounded/set_nonblocking; 269-315 and 365-402 call it for xz and gpgv; 316-364: fetch_coreos/fetch_coreos_with; 404-476: URL, exclusive bounded-download and record-shape unit cases

Open detail: Do not substitute BuildExecution or image run_build_command merely because all spawn processes: this drain carries different fixed limits/timeouts. Its helper is private, not a new public or service boundary.

## rust/soda-release-build/src/coreos_stream.rs

Observed size: 856 lines, including tests where embedded. Separate admitted live-input records and validation, Tailnet release/base selection, CoreOS stream selection and registry image-index resolution. Reuse the existing http module for the already shared capped metadata/text fetch rather than create another transport package. Keep test fixtures shared only where production unit tests already consume them; retain actual resolver and validation cases with their owner.

- `lib/soda-release-build/src/coreos_stream.rs` — Stable CoreOS stream endpoint, stream document and release-location parsing, stream triple validation, ISO/QEMU/full resolution entrypoints and their existing stub cases.
- `lib/release-inputs/src/reader/stream.rs` — Canonical `CoreOSImage`, `ResolvedCoreOS`, `TailnetInputs` and `LiveInputs` records plus wire validation. Use required exact PascalCase fields, reject unknown record fields, represent full string-key maps with `BTreeMap`, and serialize fresh maps in sorted-key order. Keep the actual resolver allowances: ISO/QEMU may retain extra architecture entries when `x86_64` is present; `Container` requires exactly one `x86_64` entry.
- `lib/soda-release-build/src/live_inputs.rs` — Retain bounded read, fresh exclusive `0644` write with LF, and validation invocation; its duplicate wire records, custom decoders, and validator remapping are removed. Build-side network resolution, hashing, and admitted-file lifecycle remain in their current Rust owners.
- `lib/soda-release-build/src/coreos.rs` — Use the canonical `CoreOSImage` in verified download/hash inputs; keep signer, signature and verified-base policy local.
- `lib/soda-release-build/src/coreos_stream.rs` — Use canonical records in the existing stream, ISO/QEMU and registry resolution path; retain upstream fetch/decode and selection policy.
- `lib/soda-release-build/src/coreos_iso.rs` — Use the canonical `CoreOSImage` while preserving bounded download, signature verification and ISO output custody.
- `lib/soda-release-build/src/tailnet_inputs.rs` — Produce the canonical `TailnetInputs` without changing version/base resolution.
- `lib/soda-release-build/src/production_inputs.rs` — Read the shared Tailnet record through the retained bounded live-input reader.
- `lib/soda-release-build/src/test_support.rs` — Build the existing canonical production fixture for resolver and pipeline tests.
- `lib/soda-release-tools/src/pipeline.rs` — Remove `coreos_image_of`, `sorted_pairs`, `sorted_images`, `resolved_coreos_of`, `tailnet_of` and `live_inputs_of`; no clone/sort DTO bridge remains.
- `lib/soda-release-tools/src/worker/runtime.rs` — Construct and record the canonical live-input value at the existing controller-side resolution call.
- `lib/soda-release-build/src/tailnet_inputs.rs` — Tailnet endpoints, newest stable archive selection, checksum/base-tag selection, resolve_tailnet_inputs and its existing release-selection case.
- `lib/soda-release-build/src/coreos_registry.rs` — Existing registry image-index request, bounded body decode and matching x86_64 digest selection.
- `lib/soda-release-build/src/http.rs` — Existing transport plus current fetch_capped_json/fetch_capped_text helpers reused by CoreOS and Tailnet callers.
- `lib/soda-release-build/src/test_support.rs` — Current cfg(test) fixture_live_inputs used by coreos_stream tests and production.rs:843; keep fixture ownership test-only.

Evidence: 24-65: current endpoint accessors; 67-257: TailnetInputs/ResolvedCoreOS/LiveInputs plus decode/read/write/validation; 258-301: fetch_capped_json/fetch_capped_text; existing http.rs is 296 lines including its current unit tests; 302-438: Tailnet resolution, release scan and newest base tag; 439-559: stream document/location/triple parsing; 560-645: resolve_registry_digests_with; 646-706: ISO/QEMU/full CoreOS resolution; 707-856: fixture_live_inputs, stream/index fixtures and four existing cases; production.rs:843-844 imports fixture/read/write

Open detail: SIMP-REL-WIRE-1 source cut is complete in `3ccbc127`: the actual `write_live_inputs` → `RealProduction::read_live_inputs` path and tests cover the producer/consumer contract. The live-input writer retains I/O; the actual resolver is in the existing Rust build owner. The deleted image `model/live_inputs.rs` leaf is not recreated: consumers import canonical shared records directly where needed, and image-only policy remains with its current owners. No native qualification or change to original signed/raw-byte handling is implied. SIMP-REL-ORDERED-1 source cut is complete in `feb95e50`: the two actual consumers now use the selected direct representations, preserving opaque nested values, existing settings policy, Ignition null-object normalization/array order and raw `live.ign` custody. Its scoped tests and independent medium review pass; no pinned image specimen or native/installed qualification is claimed. SIMP-INSTALL-OCI-1 installer source cut is allocated and pending implementation/checks; the settled profile and limits are in the [finding allocation](../execution-findings.md#rank-2-costly-boundary-and-profile-decisions).

## rust/soda-release-build/src/files.rs

Observed size: 621 lines, including tests where embedded. Keep artifact inventory/output, descriptor-relative custody and bounded JSON input distinct. JSON01 replaces the first-value/trailing-data scanner with serde/serde_json's complete-input check; FS01 uses rustix for typed confined-file mechanics. Preserve regular-file/inode admission, the 4 MiB bound and digest of the admitted original bytes before decoding.

- `lib/soda-release-build/src/files.rs` — File inventory emit/decode, architecture/revision/digest admission, regular-file hash, fresh/private output admission, exclusive write/chmod and absolute-path helper.
- `lib/soda-release-build/src/confined_files.rs` — Root/FileMeta, openat traversal, no-follow lstat/open, inode identity checks and hash_at; shared with OCI layout reads and strict JSON input.
- `lib/soda-release-build/src/json_input.rs` — Bounded read_json/read_json_at, exact-byte digest and strict input-policy visitor over serde; no custom lexical or trailing-data engine.
- `lib/soda-release-build/src/files/tests.rs` — Existing validator/hash/directory/exclusive-write/JSON/root-escape cases; retain unit scope and real Root reads.

Evidence: 18-83: File inventory and admission; 84-109: hash_file/hash_at;110-165: output admission/write/chmod;476-483: abs_path; 166-221: read_json/read_json_at;222-388: Root/FileMeta and descriptor-relative helper closure; 389-475: first_json_end/skip_ws/skip_string/skip_number/skip_literal/skip_value, used only to classify trailing input; 484-621: six existing unit cases; oci_layout.rs:5-6,45-54,116 onward consumes Root and FileMeta

Open detail: Update current callers directly. JSON01 specifies duplicate/fold/null/unknown-field policy before decoding into a map can discard evidence. Preserve the JSON size, inode and raw-byte guarantees together; a library Value alone is insufficient for strict duplicate admission.

## rust/soda-release-build/src/json_go.rs

Observed size: 451 lines, including tests where embedded. JSON01 replaces the custom JSON grammar and recursive byte emitter with locked serde/serde_json. Keep narrow strict/lenient DTO profiles where actual callers need duplicate, folded-name, null and unknown-field rules. Preserve original authenticated input bytes and the explicitly required producer representation; do not recreate a general Go JSON API around serde.

- `lib/soda-release-build/src/json_emit.rs` — Serde producer formatting for required compact/pretty Go output; concrete records and sorted maps remain with their callers.

Evidence: 14-122: FieldError/Fields lenient extraction;125-312: Strict/json_kind/type and unknown-field diagnostics; 315-389: Emit/sorted_object/marshal_indent/emit_indent/emit_value; 390-451: four existing unit cases; coreos.rs,coreos_stream.rs,files.rs,forgejo.rs,production.rs import these real semantics

Open detail: Build and image JSON01 caller transfers are complete; their distinct input profiles remain with concrete record adapters. Delivery has retired jsonx; `02a788be` closes its ordered/raw policy merge and exact strict-depth preservation with focused producer/admission checks and review. JSON01 assigns no global decoder. Required field/map ordering and signed producer bytes have focused fixtures. Obsolete byte/error parity does not justify another permanent serializer abstraction.

Build JSON01 completed in `d52d8ca8`. `json_go.rs` and its engine tests were
deleted with the last strict/lenient callers; named DTO visitors retain final
matching-field selection before conversion. `json_input.rs` keeps confined
4 MiB reads, full-input admission, exact-byte hashes and integer-token policy.
The historical ranges above remain allocation evidence. All 48 library checks,
12 integration oracles and the development build passed with medium review;
image and delivery retain their separate pending profiles.

## rust/soda-release-build/src/oci.rs

Observed size: 1286 lines, including tests where embedded. Keep one OCI inspection owner. Separate bounded outer archive/blob admission, index/manifest/config identity validation, single-layer whiteout/path semantics and multi-layer compressed-content resolution. Share Descriptor/Blob/Image with the existing oci_layout caller rather than copying them. Preserve two-pass archive verification, requested-file regularity, whiteout/opaque/ancestor behavior, trailer drain and conservative zstd refusal. Unit fixtures already shared with production and layout tests stay test-only.

- `lib/soda-release-build/src/oci.rs` — Image/Descriptor/Blob/LoadBlobs and existing inspect_oci/inspect_oci_content entrypoints orchestrating the real archive, manifest and layer owners.
- `lib/soda-release-build/src/oci/archive.rs` — Archive input admission, tar entry uniqueness/path/type gates, bounded blob copy/hash and outer archive collection.
- `lib/soda-release-build/src/oci/manifest.rs` — Layout/index/descriptor/manifest/config parsing, local bounded blob resolution, rootfs/layer/media/platform/source/base validation and image identity construction.
- `lib/soda-release-build/src/oci/layers.rs` — Requested member and layer path gates, whiteout/opaque/non-directory ancestor tracking, per-member hashing and one-layer tar scan.
- `lib/soda-release-build/src/oci/content.rs` — Descriptor-to-archive indexes, hash tee, gzip drain/zstd block handling, archive-layer collection and reverse overlay member resolution.
- `lib/soda-release-build/src/oci/tests.rs` — Existing identity/platform, content-member, scanner and gzip/zstd cases with their tar helper; preserve unit access to actual layer implementations.
- `lib/soda-release-build/src/test_support.rs` — Existing cfg(test) fixture_oci_bytes/FIXTURE_REVISION used by OCI, oci_layout.rs tests and production.rs:845; one shared fixture owner.

Evidence: 26-84: Image/Descriptor/Blob/LayerMember/LoadBlobs;85-294: archive/blob admission and read_oci_archive_entries; 295-548: read_oci_index/parse_oci_manifest/fetch_oci_blob/config/rootfs/attribution/inspect_oci_image;549-573: archive identity entrypoint; 574-782: requested_oci_paths/clean_layer_name/whiteout_target/record_* and scan_oci_layer; 783-952: layer_archive_indexes/HashReader/scan_layer_reader/scan_archive_layer/scan_oci_archive_layers/resolve_oci_members;953-1001: content inspection entrypoint; 1002-1286: shared OCI fixture and four current test groups; oci_layout.rs:5-6 reuses inspect_oci_image/read_oci_blob/read_oci_index/Blob/Image/LoadBlobs

Open detail: REL02 consolidates the overlapping scanner into the existing acyclic build-to-deliver owner after comparing actual callers and fixing deliver's decompressed trailer-drain guarantee. Existing tar/flate2/sha2 libraries own format mechanics. Keep requested-path/regularity, whiteout/opaque/ancestor semantics, raw digest/size admission, conservative zstd refusal and bounded complete decoder drain; do not preserve two scanners or create another OCI package.

## rust/soda-release-build/src/production.rs

Observed size: 1179 lines, including tests where embedded. Split the existing Production methods by compiler/dependency recipes, public asset staging, resolved image admission and application image production/export. Keep one Production value and its private inputs across resolve_inputs and images, preserving one recorded attempt and execution order. Keep the substantial scripted fixture and all production-sequence/refusal cases in one unit descendant; no new orchestrator or generalized plugin layer follows.

- `lib/soda-release-build/src/production.rs` — Existing Production state, hooks, validation/step delegation, images entrypoint and Forgejo method delegation; shared ProducedImage result.
- `lib/soda-release-build/src/production_compile.rs` — Current Go/Cargo compile recipes, ELF/output mode checks, pinned Bun and dependency admission and Soda command discovery.
- `lib/soda-release-build/src/production_assets.rs` — Existing one-time frontend/terminal/locales/upstream-tools staging and project helper compilation sequence.
- `lib/soda-release-build/src/production_inputs.rs` — ResolvedInput record, frozen selection, pull/digest admission and app-inputs record, recipe/unit references, live Tailnet inputs and resolve_inputs.
- `lib/soda-release-build/src/production_images.rs` — Current Podman build/export recipe, Rocky base/project/dashboard/Forgejo/extension/proxy/Tailnet role sequence and lexical relative paths used by those recipes.
- `lib/soda-release-build/src/production/tests.rs` — Current production_fixture and seven scripted sequence/failure/refusal/compile/discovery cases; unit access to actual private Production state.

Evidence: 15-83: hooks/Production/state/validation;85-138: compile_rust/compile;167-204: pinned Bun/dependencies;805-838: soda_commands; 139-166,205-325: assets/asset_steps and current package/binary/path strings; 326-480,734-759,777-804: frozen/pulled/recorded inputs and recipe parsers;354-368: images public entrypoint; 481-714: build/export application image roles;715-749: Forgejo delegation and current result records;760-776: lexical_rel; 840-1179: fixture plus oracle_production_sequence/failure_stops/refusals/asset_destinations_refuse_early/compile_recipes/soda_commands/image_repo_parsing

Open detail: Preserve completed package/caller changes. SYS01 replaces hand-parsed Cargo target discovery with cargo metadata --format-version 1 --no-deps over the admitted checkout and authoritative bin/required-features records. Shipping inventory and feature selection remain release policy; metadata is not a build, resolver-isolation proof or permission for network access. CF-08 consolidates only tiny bounded ELF predicates, retaining their distinct admission profiles.

## rust/soda-release-build/src/progress.rs

Observed size: 857 lines, including embedded tests. The current [D03 audit](../reviews/D03.md)
and A's independent source/caller challenge identify this mirrored progress,
BuildExecution and SharedBuffer closure as obsolete: it has only its own tests
and the two exclusive progress/exit oracle cases as consumers. Retire it and its
exclusive `clock.rs` dependency at the implementation cutover. It has no desired
target leaf; do not split it into a second live executor or preserve it solely
to satisfy predecessor parity tests.

Evidence: 14-353: BuildProgress and transitions/origin/summary/finish, error joins, new_build_progress/build_exit_code; 354-383: SharedBuffer;384-654: BuildExecution/environment/tool selection/execute/capture; 657-857: progress_fixture/fake_clock and six existing cases; tests/oracle.rs:426 onward uses progress bytes and shared output; image/build.rs:36-170,1122-1244 has a distinct runner/environment/execution implementation; tools/src/progress.rs and exitcode.rs duplicate timing/exit concerns

Current production already reaches the release-tools progress owner and the
release-image Runner through `pipeline.rs:475-538`. Preserve those actual owners,
their cancellation/log/reap contracts, the active build crate/Error base and
other oracle duties. The previous unfinished-dispatch description is superseded.
Remove exclusive obsolete module declarations, oracle imports/cases and stale
module documentation with the cutover. Shared Error fields require their own
exact type/consumer assessment; this disposition is not whole-crate retirement.

## rust/soda-release-build/tests/oracle.rs

Observed historical size: 534 lines. Retain frozen vectors and OCI fixture/layout
bytes for active assertions, dividing the surviving battery into three concern
modules under one integration-test entrypoint. Reuse data_path/scratch and byte
comparison helpers once. Each retained module invokes the actual Rust owner.
Retire only the two cases/imports belonging exclusively to the obsolete progress
closure (`oracle_progress_bytes` and `oracle_exit_codes`, current lines 433-491).
No Go implementation or temporary generator is required for retained execution.

- `lib/soda-release-build/tests/oracle.rs` — Existing integration entrypoint, oracle_vectors import and shared data_path/scratch; declares the concern modules.
- `lib/soda-release-build/tests/oracle/oci.rs` — Existing frozen archive/layout identity/content/rejection cases and artifact validator vectors.
- `lib/soda-release-build/tests/oracle/inputs.rs` — Existing live-input/verified-base/resolved-input byte cases and strict bounded-JSON read assertions.
- `lib/soda-release-build/tests/oracle/production.rs` — Existing Forgejo argv/script/toolchain and scripted real Production sequence oracle.

Evidence: 1-53: frozen-vector imports/data_path/scratch;55-155: OCI archive/content/layout cases; 156-248: live-input/base/resolved-input serialization;249-284: Forgejo args/script/toolchain and ELF fixture; 285-425: scripted Production sequence;426-484: progress bytes and exit classes; 485-509: miscellaneous artifact/URL validators;510-534: read_json strictness; oracle_vectors.rs and tests/data remain existing frozen fixture leaves

Open detail: After an intentional CLI/path/package change, update the corresponding vector from the agreed current output contract. Deleting a Go predecessor does not justify deleting these Rust assertions or claiming their old captured recipes are still the new installed layout.

## rust/soda-release-deliver/src/buildx.rs

Observed size: 595 lines, including tests where embedded. Separate the inspected build identity from filesystem custody. Keep the fd, no-follow checks, same-inode observations, bounded reads and exact byte hashing in one filesystem module; do not spread the unsafe access sequence across generic helpers.

- `lib/soda-release-deliver/src/buildx/mod.rs` — Inspected image and pinned Forgejo toolchain types, validation/decoding/emission, shape reexports, and existing build-JSON error/byte adapters.
- `lib/soda-release-deliver/src/buildx/filesystem.rs` — Root owns the fd and every confined open/stat/read/hash/JSON read, plus fresh/private/create-new output admission and writes.
- `lib/soda-release-deliver/src/buildx/tests.rs` — Existing toolchain validation and file-helper error tests.

Evidence: 18-166: compiler pins, Image, ForgejoToolchain and APK provenance; 167-386: os_error, Root and confined readers/hash; 387-526: output admission, create-new write, decode_build_json/read_json_at; 527-537: unknown_field/decode_bytes_value; 538-595: two existing tests.

Open detail: Keep crate::buildx public paths through the module root; Root fields and fd/stat/open helpers remain inside filesystem. Reexport only current callers' existing functions. prepare.rs, admission.rs, document.rs and oci.rs continue to use these same custody operations. cfg(test) descendant tests retain real filesystem assertions.

## rust/soda-release-deliver/src/fetch.rs

Observed size: 467 lines, including tests where embedded. Durable state custody and authenticated image verification are separate existing responsibilities. Keep partial authenticated highwater results and error propagation intact; the caller still persists the returned state before propagating image verification failure.

- `lib/soda-release-deliver/src/fetch/mod.rs` — Discovery/document transport and fetch request/orchestration: admit, save authenticated progress before later failures, and emit verification-only receipt.
- `lib/soda-release-deliver/src/fetch/state.rs` — StateLock/drop, flock admission, fresh private temporary write, fsync/rename/directory sync.
- `lib/soda-release-deliver/src/fetch/verification.rs` — Architecture selection, verified image metadata/config binding, image copy checks, mixed-architecture identity tracker and highest authenticated state.
- `lib/soda-release-deliver/src/fetch/tests.rs` — Current lock/save serialization and request-refusal tests with their private-directory fixture.

Evidence: 19-41 discover; 42-50 init_state; 51-129 StateLock/lock_state/create_temp/save_state; 130-141 fetch_document; 142-358 resolve_verification_architectures through verify_releases; 359-418 admit_fetch_request/complete_fetch/fetch; 419-467 tests.

Open detail: Preserve crate::fetch::{discover,fetch_document,lock_state,save_state,verify_releases} visibility for publish.rs. Publish and Fetch share this existing lock/write code, not a new process or persistence abstraction. verification retains ReleaseIdentityTracker and refs_seen within its closure.

## rust/soda-release-deliver/src/jsonx.rs

Historical source size: 623 lines, including tests. Checkpoint `c5fef89a` removes jsonx/decode/emit/tests and their generic Binder/Soft/Emit surfaces. Their pending engine splits are superseded by the caller transfer.

- `lib/soda-release-deliver/src/json_serde.rs` — Strict object/decoded duplicate/depth/cap admission and raw scalar policy over Serde. The strict depth-100 and arbitrary-number preflight preservation is verified in `02a788be`.
- Existing model/payload/OCI/fetch/publication record owners — Concrete exact-field DTOs, last-winner raw slots where lenient callers require them, and original signed/hash custody.
- `lib/soda-release-deliver/src/document.rs` — OCI document custody and actual Go two-space/newline producer formatter. Five frozen byte goldens and the captured Channel manifest digest remain verification requirements.
- `lib/soda-release-deliver/src/native/policy.rs` — Generated trust requirements and proposed policy merging. Ordered/raw-number scope values and generated trust requirements are verified in `02a788be`; strict depth 100 admission remains ahead of merging.

Historical Evidence: 1-18 cap/error; 22-96 Base64; 97-352 duplicate traversal and Binder/Soft; 353-558 Emitter/Emit; 559-573 marshal; 574-623 tests. These ranges record completed structural work, not remaining engine ownership.

## rust/soda-release-deliver/src/model.rs

Observed size: 1312 lines, including tests where embedded. The file already names distinct release-authority documents. Keep validation, decoding and emission with each actual type rather than create separate model/validator/codec layers. Channel highwater and its admission remain together; release serial admission stays with Release. Permit remains meaningful code in the public root.

- `lib/soda-release-deliver/src/model/mod.rs` — Public model exports and the existing protected Permit type, validate/decode/Emit implementation.
- `lib/soda-release-deliver/src/model/trust.rs` — Trust envelope/timing, library-backed P-256 PEM/SPKI/curve admission, signer-role separation and raw-DER fingerprinting, role/reference lookup, typed decode/emission.
- `lib/soda-release-deliver/src/model/candidate.rs` — Candidate host/source/toolchain/content binding, asset/path rules, decode/emission and existing path_clean.
- `lib/soda-release-deliver/src/model/release.rs` — MediaFile/MediaBinding and media admission, Release validation/provenance/evidence/reference/decode/emission, release serial admission and its exact helpers.
- `lib/soda-release-deliver/src/model/channel.rs` — Channel/Seen/Highwater shapes and decode/emission/state validation; channel reference/identity/timing/progression helpers and admit_channel.
- `lib/soda-release-deliver/src/model/tests.rs` — Current content/path and trust-reference unit tests plus their full_content fixture.

Library boundary: CF-05 is implemented through the C-owned shared adapter in `lib/release-inputs/src/trust_key.rs`, using p256's typed SPKI/DER APIs. Both image and delivery reject off-curve keys and preserve admitted original DER. Their role limits, reference lookup and minimum sequence remain Soda authority. JSON01 supplies DTO codec profiles; it does not merge Permit, Channel highwater or Release serial authority.

Evidence: 24-299 Trust and all PEM/DER/key-role helpers; 300-535 Candidate, exact content set, path_clean and code/asset binding; 536-615 media; 616-798 Release; 799-1021 Channel/Seen/Highwater; 1022-1125 channel admission; 1126-1179 release admission; 1180-1231 Permit; 1232-1312 tests.

Open detail: Retain crate::model paths through explicit reexports, including pub(crate) path_clean and valid_media_* used by OCI/admission. Channel admission owns validate_release_reference (used by validate_channel_releases); release admission owns admit_channel_ref/admit_release_digest. Cross-type calls use the same Trust/Payload/Candidate/Highwater, never new adapter types.

## rust/soda-release-deliver/src/native.rs

Observed size: 614 lines, including tests where embedded. Separate trust-policy construction and signer custody from the existing native runner/copy operation. Keep signing admission before effects and the final fresh verification in its current order. The 49-line native test module can stay inline in mod.rs rather than gain another tiny leaf.

- `lib/soda-release-deliver/src/native/mod.rs` — Runner/Native command execution, locked version check, private-file/write-json custody, verified copy and manifest check, existing focused native tests.
- `lib/soda-release-deliver/src/native/policy.rs` — Requirement shape/emission, exact role policy construction/merge/conflict refusal, local policy and registry Sigstore configuration.
- `lib/soda-release-deliver/src/native/sign.rs` — SecretFiles decode, protected permit/key admission, snapshot source, signed document admission, signature emission and round-trip verification.

Library boundary: JSON01 delegates native policy emission to serde/serde_json. Keep pinned native signer/verifier tools, secret-file custody, role admission, copy uncertainty and fresh verification with this owner. Reuse does not introduce a signing service or transfer publication authority.

Evidence: 18-69 Runner/Native/check_native; 71-106 private_file/write_json; 107-370 Requirement/policy/merge/registries; 371-434 verify source/copy/digest; 435-565 SecretFiles/sign closure; 566-614 version/private-file tests.

Open detail: Preserve crate::native public and crate-visible paths used by fetch/publish. Keep policy_for/local_policy/registry_config visible only to their native parent/siblings as needed. Moving TOOL_LOCK into src/native/mod.rs changes include_str! from ../tools.json to ../../tools.json; preserve the same tracked lock bytes, skopeo env clearing, command timeout and output cap.

## rust/soda-release-deliver/src/oci.rs

Observed size: 1109 lines, including tests where embedded. Split wire identity, outer archive custody and rootfs layer resolution. Keep whiteouts, ancestor replacement, unsupported zstd blocking and topmost-member resolution in one layer owner; these are one algorithm. The layout loader and public orchestration stay together and consume the same schema/archive/layer internals.

- `lib/soda-release-deliver/src/oci/mod.rs` — OCI constants, OciLayout/shared Blob, three existing public inspect entrypoints, content identity orchestration, confined LayoutLoader/image-set closure, existing focused tests.
- `lib/soda-release-deliver/src/oci/schema.rs` — Descriptor/OciManifest/OciConfig and exact index/manifest/config decoding; bounded blob hashing, layer/rootfs/attribution checks and inspected image identity.
- `lib/soda-release-deliver/src/oci/archive.rs` — Regular archive admission, safe unique bounded archive-entry collection, archive index selection and outer layer-archive traversal.
- `lib/soda-release-deliver/src/oci/layers.rs` — LayerMember, exact requested paths, clean layer names, whiteout/opaque/ancestor replacement rules, member hashes, descriptor index, TeeHasher/drain/layer scan and reverse member resolution.

Evidence: 29-67 OciLayout/Blob/Descriptor/OciManifest/LayerMember; 68-421 exact index/manifest/config/blob/image inspection; 422-518 outer archive admission/read/index; 519-528 inspect_oci; 529-855 member path/whiteout/hash/layer resolution; 856-928 content orchestration/outer scan; 929-1080 layout input/Root/loader/image set; 1081-1109 current tests.

Open detail: REL02 selects this existing surviving inspection owner only after the build/image/installer caller comparison. Fix bounded decompression through EOF/trailer before reuse. tar/flate2/sha2 remain format engines; LayerMember whiteout/ancestor/topmost selection remains Soda policy. Keep current public inspection entrypoints and bounded private helpers, without a new public archive abstraction.

## rust/soda-release-deliver/src/payload.rs

Observed size: 406 lines, including tests where embedded. The production owner is 366 lines and is one delivered payload contract. Extract the existing 40-line tests only; retain shared optional decode helpers and load with their current owner to avoid a new generic codec layer.

- `lib/soda-release-deliver/src/payload.rs` — Current Image/Payload wire types, validity checks/decode/emission, typed optional field adapters, repository prefix validation and confined load operation.
- `lib/soda-release-deliver/src/payload/tests.rs` — Current repository-prefix and CoreOS-version tests.

Evidence: 24-213 Image/Payload and validation/decode; 214-262 decode_opt_* / decode_string_map (used by model/buildx/native/publish); 263-346 Emit/repository prefix; 347-366 load via Root/read_json_at; 367-406 tests.

Open detail: Preserve crate::payload public names, build decoding last-wins behavior versus strict document callers, and exact load errors/digest checks. Use #[cfg(test)] mod tests; src/payload.rs + src/payload/tests.rs is an ordinary Rust module/descendant pair.

SIMP-INSTALL-OCI-1 stays with [`cmd/soda-install/src/deliver.rs`](../../../cmd/soda-install/src/deliver.rs), [`cmd/soda-install/src/oci/metadata.rs`](../../../cmd/soda-install/src/oci/metadata.rs) and [`cmd/soda-install/src/oci/layout.rs`](../../../cmd/soda-install/src/oci/layout.rs). The release-image producer [`model/payload.rs`](../../../lib/soda-release-image/src/model/payload.rs) owns its required PascalCase payload; `payload_stage::seal_candidate_payload` owns pre-write validation and private `0600` LF custody. Plan direct installer-local typed deserialization and semantic validation, preserving authenticated original payload bytes/hash and bounded input. Plan OCI v1.1.1 typed records with exact upstream names, unknown extensions tolerated at each object level, and the required/optional-field rules in the [execution allocation](../execution-findings.md#rank-2-costly-boundary-and-profile-decisions). Strict UTF-8 precedes all parsing; typed derive skips unknown fields, and `IgnoredAny` plus complete EOF replaces only discarded-`Value` syntax checking in `copy_oci_blob`. Retain 4 MiB per retained JSON blob and 32 MiB aggregate retained JSON caps; larger layer blobs continue streaming/hash admission under separate descriptor/layer size/count policy. Allocate a `model::Payload` Serialize → `deliver::load` regression through one acyclic test-only edge to the existing release-image crate; root owns dev-dependency/lock and graph integration. Keep sealer custody source-inspected because invoking it enters native staging and `complete::complete`. Exclude adjacent image-import, Muse and release-deliver decoder cuts.

## rust/soda-release-deliver/src/publish.rs

Observed size: 533 lines, including tests where embedded. Ledger state and channel promotion are distinct duties within one publication operation. Keep the ordered publication state machine in mod.rs: lock, hold pending uncertainty, admit/signature-check, record pending, immutable upload, optional channel promotion, observe and record complete. Existing 47-line tests remain inline.

- `lib/soda-release-deliver/src/publish/mod.rs` — Upload/observed publication, ledger admission, observe-only completion, signed snapshot admission, immutable commit, final receipt and publish sequencing; current focused tests.
- `lib/soda-release-deliver/src/publish/ledger.rs` — Ledger type/decode/validation/emission, phase rule and explicit private init_ledger.
- `lib/soda-release-deliver/src/publish/channel.rs` — Protected channel history/offer admission, anonymous tag observation, prior-channel verification and tag promotion.

Library boundary: JSON01 replaces ledger codec mechanics, and FS01/TMP01 replace typed file/temporary allocation mechanics. Keep lock custody, pending uncertainty, immutable upload, optional promotion, observation and durable completion in the same operation. REL01 delegates document.rs's new delivery-document tar production to tar Builder with explicit deterministic metadata. Existing authenticated archive bytes remain untouched during reading; new descriptor hashes bind newly emitted bytes. Historical Go checksum-field bytes are not an independent requirement.

Evidence: 19-102 Ledger/init_ledger/phase; 103-219 upload/observe/admit/observe-only; 220-257 channel history/offer; 258-325 admit_signed/commit_immutable; 326-428 tag/prior-channel/promotion; 429-485 finalization/publish; 486-533 tests.

Open detail: Preserve crate::publish::{Ledger,init_ledger,publish} and private helper order. Channel functions use the actual fetch::verify_releases and native::verify_copy, ledger uses the same fetch::lock_state/save_state, and no blind publication retry or new recovery machinery appears.

## rust/soda-release-deliver/tests/oracle.rs

Observed size: 680 lines, including tests where embedded. Group oracle assertions by pure authority documents, delivered artifact bytes, and durable fetch state. Retain the single existing golden JSON fixture without mechanical splitting; no live Go subprocess is needed to consume retained byte/outcome evidence.

- `lib/soda-release-deliver/tests/oracle/main.rs` — Cargo-discovered oracle test root: golden decode/result/time helpers, model byte-roundtrip/validation/channel/release/policy/strict-decode cases, shared temp/private-dir and current small formatting/hash helpers.
- `lib/soda-release-deliver/tests/oracle/artifacts.rs` — Document layout round-trip/copy fixture, plain/gzip OCI/content tests, payload load errors, candidate binding-before-archives and qualification/fixture-refusal cases.
- `lib/soda-release-deliver/tests/oracle/fetch_state.rs` — Real private state/ledger init observations and the existing scripted Runner/fetch wiring refusal case.

Evidence: 24-83 shared retained golden readers; 84-313 model/channel/release/policy parity; 314-431 document/OCI fixture; 432-463 strict/load; 464-486 state/ledger; 487-588 candidate/qualification; 589-628 helpers; 629-680 ScriptRunner/fetch refusal.

Open detail: Preserve completed suite wiring and tests/goldens/deliver.json. Keep raw signed-byte, digest, highwater and publication/refusal observations. Update obsolete parser/error/format parity fixtures deliberately with each library profile; scripted fetch proves wiring/refusal and qualification JSON tests admission, neither establishes installed or live publication proof.

## rust/soda-release-image/src/build.rs

Observed size: 1446 lines, including tests where embedded. Separate source admission/snapshot, command/recall/log execution, host-context preparation, shipping compilation inventory and candidate sealing from the single ordered build orchestration. The two host-candidate bodies duplicate the same observation-build/package-hash/seal/final-build/readback sequence; use the existing callback-based implementation once while retaining phase labels. Keep the narrow RunnerProduction bootstrap with source extraction: its intentionally refusing foreign methods are not a completed build/deliver adapter.

- `lib/soda-release-image/src/build.rs` — Existing ProductionInputs/build/run_build/run_build_inner, ordered phase orchestration, production factory, prepare/execute/finalize joins and record-result boundary.
- `lib/soda-release-image/src/build_runner.rs` — Cancel/SharedFile/Runner/LogCloser, recall/media log attachment, child environment/tool resolution and run_build_command.
- `lib/soda-release-image/src/build_source.rs` — Canonical clean checkout/revision/compiler/output admission, directories/source archive/workspace/log lifecycle and current Forgejo extraction bootstrap.
- `lib/soda-release-image/src/build_context/mod.rs` — Freeze selected base-image config with the direct opaque RawValue root and retained settings lookup policy; prepare host context and existing link_prepared_assets.
- `lib/soda-release-image/src/build_compile.rs` — Go command discovery, Rust installed-command/tool tables, shipping tool compilation/copy/hardlink and exact tools.json record.
- `lib/soda-release-image/src/build_candidate.rs` — One current observation-build/package-hash/payload-seal/inventory/final-build/readback implementation, shared by callback/progress callers.
- `lib/soda-release-image/src/build_runner/tests.rs` — Existing command environment/capture/failure case against the real execution function.
- `lib/soda-release-image/src/build_context/tests.rs` — Existing prepared-assets/no-command assertion with its current panic-on-command fixture, preserving that specific boundary.

Library boundary: SYS01 makes admitted Cargo metadata authoritative for declared bin targets, while release inventory selects what ships. PROC02 keeps std process execution with concurrent bounded drains, live cancellation, exact-unit stop/reap and descriptor ownership; these are lifecycle corrections, not another generic executor. CF-08 retains the small bounded ELF predicate.

Evidence: 36-170: Cancel/SharedFile/Runner/LogCloser;171-204: ProductionInputs/build;205-378: source/input/output/snapshot/workspace admission; 379-490: freeze_base_image_config/prepare_build_host_context;491-521: prepare_build_production; 522-710: compile_soda_commands/record_tool_files/RUST_TOOLS/compile_rust_tools/compile_shipping_tools; 711-771 and1059-1121: same host-candidate sealing/build sequence with next callback versus Progress;772-864,976-1058: orchestration/finalization; 865-975: narrow RunnerProduction bootstrap for forgejo::extract_forgejo_snapshot;1122-1244: environment/tool resolution/run_build_command;1246-1264: link_prepared_assets; 1265-1326: real child command test;1327-1446: no-command prepared-assets test and specific stub

Open detail at f7: `soda-release-tools/src/pipeline.rs` is the composition owner wiring release-build and release-deliver behind the image traits; `build_cli.rs:304–313,343–347` calls the actual worker and image pipeline. The image library does not need direct dependencies on those implementations. The current `compile_shipping_tools` still selects the deleted Go `./tools/soda-artifacts` producer and compiles only the acceptance driver, omitting its remote companion. D03 retains corrections in the existing Rust tools/compiler inventory and acceptance owner. Source wiring does not establish installed build or packaging proof.

## rust/soda-release-image/src/media.rs

Observed size: 1204 lines, including tests where embedded. Separate upstream assembler preparation, authenticated packaging inventory, packaging-container/meta verification and installer customization/readback. Keep Media/MediaLock/MediaAuthority/MediaInputs, admitted-input sequencing, final media binding and assemble_media orchestration together. Preserve exact authority/digest/inventory admission, native container cleanup and ISO/rootfs readback before sealing; no new media service or alternate signing path is implied.

- `lib/soda-release-image/src/media.rs` — Current media/lock/authority records and public-base URL gate, authority/candidate/compression admission, final seal_media binding and one assemble_media sequence.
- `lib/soda-release-image/src/media_assembler.rs` — Upstream config revision fetch, frozen buildroot arg, assembler digest/wrapper/layer verification, prepare_assembler and builder_id.
- `lib/soda-release-image/src/media_authentication.rs` — Existing sign/verify delegation and auxiliary input inventory collection/binding/recheck before native packaging.
- `lib/soda-release-image/src/media_container.rs` — Current run-owned packaging-container stop/build, native assembly and MediaMeta parse/identity verification.
- `lib/soda-release-image/src/media_installer.rs` — Rootfs placement, coreos-installer ISO customization/verification, initrd readback/chunk verification and prepare_and_verify_media.
- `lib/soda-release-image/src/media/tests.rs` — Existing public URL and buildroot selection tests against the actual owning functions.

Library boundary: CF-09 uses flate2's multistream gzip decoder with a cap-plus-one read and complete EOF/trailer validation for live configuration. Keep public media URL policy, native assembler admission, exact inventory/digest binding, container cleanup and readback before sealing. The decoder does not replace the authenticated packaging sequence.

Evidence: 20-174: callback aliases, MediaLock/MediaAuthority/Media and media_base_url;175-398: assembler preparation/identity; 399-439,533-639: sign_media_input/inventory/authenticate_packaging_inputs;440-532: MediaInputs and authority/candidate/compression admission; 640-740: owned-container cleanup/build/native assembly;741-819: MediaMeta/image/meta validation; 820-1035: setup_media_rootfs/verify_customized_iso/customize_installer_iso/verify_media_readback/prepare_and_verify_media; 1036-1153: seal_media/assemble_media;1154-1204: URL and assembler buildroot tests; build_media.rs consumes prepare_assembler/assemble_media

Current detail at f7: the existing tools pipeline adapter supplies release-deliver signing/verification behind the image Production trait. Preserve that actual caller and its custody boundary during decomposition. This source wiring does not establish native packaging/signing success.

## rust/soda-release-image/src/model.rs

Observed size: 1475 lines, including tests where embedded. Keep artifact records, URL/IP admission, payload, candidate/Forgejo provenance, trust/permit/secret-file authority and admitted live inputs distinct. JSON01 delegates codec mechanics, N7/N8 use url/std::net with purpose guards, and CF-05 uses p256/spki/der for complete key admission. At the existing tools/image/build/deliver cutover, reuse proven equivalent DTO/validator owners; completed structural extraction does not prove equivalence or justify new generic engines.

- `lib/soda-release-image/src/model.rs` — Current format/source/schema and image-role constants plus shared digest/revision/architecture/repository/CoreOS identity primitives; schema remains derived from the real store owner.
- `lib/soda-release-image/src/model/url.rs` — Purpose-specific HTTPS/media/loopback admission over url and std::net, preserving authenticated literal text without a URL/IP grammar port.
- `lib/soda-release-image/src/model/images.rs` — Current Image/ProducedImage records and their JSON interface for foreign OCI/production operations.
- `lib/soda-release-image/src/model/payload.rs` — Current PayloadImage/Payload parse/emit/load and exact identity/base/independent-image/no-upgrade admission closure.
- `lib/soda-release-image/src/model/candidate.rs` — Current ForgejoToolchain/package provenance and Candidate parsing/emission/payload/source/host/content binding.
- `lib/soda-release-image/src/model/trust.rs` — Trust role/timing/minimum-sequence policy, library-validated P-256 key/raw-DER fingerprint, Permit and SecretFiles models; no custom DER/SPKI reader.
- Deleted by SIMP-REL-WIRE-1 (`3ccbc127`): `lib/soda-release-image/src/model/live_inputs.rs`. Its duplicate wire records and decoder are gone; actual image consumers import canonical shared records directly where needed. Image-domain base selection and consumer checks remain with their existing owners.
- `lib/soda-release-image/src/model/tests.rs` — Existing primitive/URL/payload cases remain unit scoped; attach each case to its actual concerned module without exporting parser helpers for tests.
- `lib/soda-release-image/src/jsonio.rs` — Serde admission/formatting and field-token policy for concrete image DTOs; no syntax engine or universal field binder.
- Deleted by SIMP-REL-ORDERED-1 (`feb95e50`): `lib/soda-release-image/src/ordered_json.rs`. Its recursive `OrderedValue` tree and `ImageConfig` wrapper are removed after the last consumers moved to direct `BTreeMap<String, Box<RawValue>>` image-config handling and ordinary `serde_json::Value` Ignition comparison. The image consumer retains opaque nested bytes/tokens and current settings lookup; Ignition comparison retains recursive null-object pruning and array order. No new crate graph edge or dependency/version change. No pinned `image.json` specimen establishes exact schema/casing or native behavior; C10.M-profiles remains open for INSTALL-OCI-1, and C10.V/R04 remain open.

Evidence: 12-104: current constants, identity/digest/architecture/repository gates;105-290: URL/IPv4/IPv6/loopback implementation; 291-365: Image/ProducedImage;366-608: PayloadImage/Payload JSON/load/identity/base/image/upgrade validation; 609-695: ForgejoToolchain/APK provenance;696-939: Candidate and exact payload/source/host/content binding; 940-1176: Trust and private PEM/DER/SPKI/role-key admission;1177-1244: Permit/SecretFiles; 1245-1427: admitted CoreOS/Tailnet inputs;1428-1475: three unit groups; actual consumers are build,complete,host,inspect,layout,payload_stage,prepare,record,media and foreign trait signatures

Open detail: SIMP-REL-WIRE-1 is complete in `3ccbc127`; `lib/release-inputs/src/reader/stream.rs` owns the shared live-input records. The distinct image upgrade-from and delivery nil-to-null policy remains outside this transfer, as do the different ProducedImage records. CF-05's two custom SPKI readers are retired behind the shared admitted-key adapter; trust policy stays in these model owners. JSON01/N7/N8 do not authorize aliases, FFI/RPC, signed-literal normalization or another schema owner.

## rust/soda-release-image/src/prepare.rs

Observed size: 501 lines, including tests where embedded. Keep the cohesive public-only host context staging operation and explicit source-to-installed file map together. The production body is 376 lines; the excess comes from a 125-line unit case largely implementing the existing foreign trait. Extract that test as a descendant instead of manufacturing separate writer/map/base orchestration packages. Preserve current live-versus-admitted base paths, public-file modes/vendor normalization, exact revision and inventory semantics.

- `lib/soda-release-image/src/prepare.rs` — Base/live/file resolution, PreparedWriter/public copies and rootfs_file_map, one prepare/prepare_resolved/finish_prepare sequence and exact context inventory.
- `lib/soda-release-image/src/prepare/tests.rs` — Existing base-input architecture/revision gate case and its original foreign trait stub; keep private finish_base_inputs access.

Evidence: 20-68: Base and live/admitted base loaders;69-184: PreparedWriter/public-file normalization/links/build record; 185-254: rootfs_file_map;255-340: base input/revision/provisioning admission and prepare/prepare_resolved/finish_prepare; 341-376: inventory modes/hash/link record;377-501: oracle_base_inputs_require_exact_revision and current stub; build.rs:451-490 consumes prepare/prepare_resolved;build.rs:748,1095 consumes inventory

Open detail: All current appliance source paths and installed libexec/service destinations must follow the chosen system tree. These are production inventories, not obsolete Go logic; retain their staged bytes and update path-reading test fixtures directly.

## rust/soda-release-image/tests/oracle.rs

Observed size: 684 lines, including tests where embedded. Retain the frozen Go outputs as Rust test data and split the heterogeneous battery into three existing responsibility groups under one integration entrypoint. Shared byte/error helpers remain in the entrypoint. Keep the refusal-only Production stub local to staging; do not turn its deliberately unreachable operations into another general production adapter. Large base64 literals stay beside the cases they qualify.

- `lib/soda-release-image/tests/oracle.rs` — Existing single integration-test entrypoint, frozen-output note and b64/check_ok/check_err helpers.
- `lib/soda-release-image/tests/oracle/media.rs` — Existing public media URL, compression, native media log-event and rootfs-chunk assertions.
- `lib/soda-release-image/tests/oracle/host.rs` — Existing local quadlet, live Ignition, package/RPM input and candidate-live-config byte/refusal assertions.
- `lib/soda-release-image/tests/oracle/staging.rs` — Existing asset-name/failure-reason/stage-layout cases and the stage-only refusing Production stub.

Evidence: 1-23: frozen-output helpers;24-115: media public-base URL cases;116-218: quadlet/live Ignition byte cases; 219-236 and292-324: package/RPM admission;237-291: compression;325-392: media event bytes; 393-442: extension asset/failure reason;443-579: stage-only Production stub;580-612: stage layout refusals; 613-648: rootfs chunk verification;649-684: candidate-live-config frozen byte/refusal cases

Open detail: The source contains frozen expected bytes, not a runtime Go dependency. Keep authenticated raw-byte/digest and actual rendering/refusal observations. Where a library adoption deliberately changes parser acceptance or newly produced formatting, update the agreed fixture rather than preserve the generic engine solely for historical parity.

## rust/soda-release-tools/src/artifacts.rs

Observed size: 506 lines, including tests where embedded. Production is 330 lines and cohesive. Extract its 176-line tests, retaining the exact command/validation order and current explicit unimplemented pipeline boundaries. No separate filesystem library or new subprocess wrapper is justified by this size.

- `lib/soda-release-tools/src/artifacts.rs` — One artifact command owner: flag admission, exact native tools/output/Butane execution and OCI/CoreOS boundary dispatch.
- `lib/soda-release-tools/src/artifacts/tests.rs` — Existing unit tests with private destination/Butane/CLI/OCI/CoreOS inputs and refusal observations.

Library boundary: CLI02 uses clap 4.6.7 builder parsing while this owner retains action/native-tool/output admission and order. FS01 uses typed rooted-file mechanics where needed; a new filesystem package or subprocess framework is still unjustified.

Evidence: 55-89 flags; 90-161 executable lookup/private output admission; 162-235 Butane admission/execution/new-file completion; 236-302 CoreOS/OCI admission and current boundary errors; 303-330 action dispatch/main; 331-506 tests.

Open detail: Keep crate::artifacts public paths and its current String errors. cfg(test) descendants use current private helpers, without widening production APIs. Current CoreOS fetch/OCI inspection boundary stubs remain evidence of pending wiring, not a completed pipeline claim.

## rust/soda-release-tools/src/build_cli.rs

Observed size: 508 lines, including tests where embedded. The 371-line production file is one controller entry workflow; its flags, source binding and branch/exit sequence should remain readable together. Extract tests instead of distributing these steps into independent tiny files.

- `lib/soda-release-tools/src/build_cli.rs` — Existing soda-build flags/admission, source/VCS binding, environment/signals/progress, parent/worker branch and exit handling.
- `lib/soda-release-tools/src/build_cli/tests.rs` — Current exact flags/worker dispatch, progress/source/environment boundary tests.

Library boundary: CLI02 delegates flags/help/value parsing to clap 4.6.7. Keep the actual parent/worker branch, admitted source/environment and exit mapping here. D01-F4/PROC02 require live cancellation propagation and bounded drains; parser reuse does not establish those lifecycle guarantees.

Evidence at f7: 1–173 flags and dispatch admission; 174–202 environment/signals/progress; 203–292 source revision and checkout binding; 293–314 artifact output and actual parent dispatch; 315–371 run/main; cfg(test)372 and complete module373–508. Tests remain descendants of the real controller entry.

Open detail: Preserve crate::build_cli paths and the existing build.rs VCS stamp. Cargo manifest depth is still two directories below repository root after rust/soda-release-tools -> lib/soda-release-tools, so build.rs ../.. root lookup remains valid. Current parent dispatch calls the actual Rust worker and the worker-stage branch calls the real image pipeline; source wiring is present, while D01-F4 identifies missing ongoing controller-signal propagation and native qualification remains separate.

## rust/soda-release-tools/src/candidate.rs

Observed size: 522 lines, including tests where embedded. Separate the actual answer/flag contract from executing an admitted candidate run. Keep Options plus all current validation together; keep fixture startup, controller wait, stop and filing order in the run owner.

- `lib/soda-release-tools/src/candidate/mod.rs` — Checkout cleanliness/fresh-output preflight, monotonic origin, summary, terminal option resolution, controller/fixture lifecycle, failure/exit handling.
- `lib/soda-release-tools/src/candidate/options.rs` — Typed Options/defaults over clap builder, plus actual mode/architecture/answered path/media admission; retire generic Go flag specs/parsing.
- `lib/soda-release-tools/src/candidate/tests.rs` — Current base_options fixture and option/answer/preflight unit tests.

Library boundary: CLI02 replaces the general flag engine with clap builder, preserving explicit boolean-value and positional-tail custody where documented commands require it. Keep checkout/fresh-output preflight, prompt callbacks and fixture/controller stop/join/copy order as candidate policy.

Evidence: 14-203 Options/flag_specs/ARCH_DEFAULT/usage/native_arch/parse_options/valid_out_leaf/validate_*; 204-279 checkout/Git/fresh-out preflight; 280-327 monotonic/describe/resolve_options/ready_run; 328-405 ExitError/run_candidate/CandidateError/main; 406-522 tests.

Open detail: Reexport current crate::candidate Options and option functions so candidate_prompts/fixture/controller callers stay stable. Preserve #[cfg(test)] pub mod tests and base_options, which candidate_prompts' existing test module uses; do not silently sever that test-only caller. Keep existing callback to prompter_overview, not a new run interface.

## rust/soda-release-tools/src/candidate_display.rs

Observed size: 686 lines, including tests where embedded. Event parsing/path translation is an existing stateless seam. Keep the entire mutable renderer lifecycle, both impl blocks, ticker ownership and its private state in one file; do not split drawing from lifecycle into stand-in UI APIs.

- `lib/soda-release-tools/src/candidate_display/mod.rs` — Phase/RendererInner/Renderer/TickerHandle, both Renderer impls, every locked feed/note/draw/finish method, phase/display/time/terminal helpers and constants.
- `lib/soda-release-tools/src/candidate_display/events.rs` — Controller Event and exact START/DONE/FAILED/CANCELLED/artifact parsing, duration fields and current sandbox-to-host artifact path translation.
- `lib/soda-release-tools/src/candidate_display/tests.rs` — Current captured-writer event/parser/render/failure/panel/path/viewport cases, including private renderer state checks.

Library boundary: PROC02/CLI02 retain the small Soda renderer/event protocol, uses std::io::IsTerminal and typed rustix terminal-width mechanics, and requires ticker RAII stop/join on relay failures. N9 replaces calendar/duration emulation with the selected time/std profiles; it does not split mutable renderer lifecycle into another UI API.

Evidence: 11-103 Event/parse_event and all parser helpers/host_artifact_path; 104-271 renderer fields and first impl; 272-352 phase mutation/line rendering; 353-435 ticker/finish/why panel impl; 436-477 exit/time/truncate/terminal helpers; 478-686 tests.

Open detail: Keep crate::candidate_display exports consumed by candidate_controller and candidate. tests stays a cfg(test) descendant of the root and retains direct private RendererInner/Phase access; Renderer is not made public beyond its current API. Preserve the event wire, lock order, first failed cause and existing ticker stop/join behavior. D01-F5 requires label normalization at the existing parser/phase boundary: padded START and DONE currently retain different leading spaces and fail to close one phase; historical parity assertions do not mandate retaining that defect.

## rust/soda-release-tools/src/candidate_fixture.rs

Observed size: 403 lines, including tests where embedded. Keep the selected loopback-only development media pickup and stop-handle/file custody together. N5 replaces the complete hand-written HTTP engine with the same in-process Hyper boundary selected by N1; N7 supplies URL/percent adapters. This remains a candidate-process fixture, without a sidecar or product server redesign.

- `lib/soda-release-tools/src/candidate_fixture.rs` — Development rootfs/loopback admission, bounded Hyper serving, existing stop/join and fresh-destination copy custody.
- `lib/soda-release-tools/src/candidate_fixture/tests.rs` — Current fixture URL/address matrices, HTTP/rootfs filing, busy-port behavior and occupied/missing output refusal tests.

Evidence: 11-97 rootfs defaults/URL host/port/fixture predicates; 98-215 FixtureServer/serve_fixture/request/percent decode/response; 216-281 copy_built_rootfs/chown_name/copy_file; 282-403 tests.

L09's fixture cutover completed in `78dec937`. The retained in-process Hyper
server admits 16 connections, 64KiB heads and 64 headers with a 5s header timer.
It streams 64KiB chunks from one opened regular file with no symlink traversal;
there is no arbitrary whole-download timeout for gigabyte rootfs images. The
owned stop handle also joins on early-return drop. Actual tests cover active
64MiB transfer cancellation/join, incomplete heads, symlink refusal and existing
copy custody. `fixture_wanted`, busy-port no-op and candidate-controller paths
are retained. N7 URL adoption remains a separate pending packet.

## rust/soda-release-tools/src/candidate_prompts.rs

Observed size: 656 lines, including tests where embedded. Separate the actual default-answer/controller-argv calculations from the interactive Prompter owner. Keep its full private implementation, field order, renderer and edit methods together; overview_rows continues to call Prompter::show_fast_compress within that owner.

- `lib/soda-release-tools/src/candidate_prompts/mod.rs` — Prompter fields and entire impl, private editing/line/choice/render/overview state, overview_rows/output status/path observations, real-terminal entry.
- `lib/soda-release-tools/src/candidate_prompts/defaults.rs` — Existing constants/mode labels/default path/timestamp suggestion/controller arguments and default_overview answer initialization.
- `lib/soda-release-tools/src/candidate_prompts/tests.rs` — Existing Shared capture/scripted prompt fixtures, controller args/defaults/answer edits/refusal/output suggestion assertions.

Library boundary: N9 delegates calendar/timestamp formatting to time 0.3.55 with an explicit producer profile. Keep default-answer suggestions, controller argv and the whole private interactive Prompter together; a TUI framework or generic calendar parser is not a target.

Evidence: 7-115 constants/mode/default_path/file_exists/suggest_out/utc_stamp/controller_args; 116-377 full Prompter impl and existing injected exists callback; 378-423 overview_rows/out_status/parent_dir_exists/path_absent; 424-442 default_overview; 443-451 terminal entry; 452-656 tests.

Open detail: Preserve crate::candidate_prompts public exports, constructor existing file_exists callback and cfg(test) fixture dependency crate::candidate::tests::base_options. Only current private default_overview/file_exists access crosses to defaults (pub(super)); no detached Prompter method or new state/callback abstraction is introduced.

## rust/soda-release-tools/src/progress.rs

Observed size: 407 lines, including tests where embedded. Production is 318 lines and one ordered progress lifecycle. Extract the existing 89-line tests; keep event emission, inherited timing origin/log admission, phase/section completion and summary/exit in the same owner.

- `lib/soda-release-tools/src/progress.rs` — One BuildProgress timing/event/log owner with current private clock/phase/reason/finish state and all methods.
- `lib/soda-release-tools/src/progress/tests.rs` — Current environment restoration/locking, fixed clock/captured output and phase/failure/origin assertions.

Library boundary: N9 uses std monotonic timing and the chosen time profile for calendar output. Preserve the inherited cross-process timing-origin contract, injected clock and event wire rather than replace them with unrelated process-local Instants. This narrow progress lifecycle remains one Soda owner.

Evidence: 12-47 monotonic/duration/failure/finish text; 48-318 BuildProgress with full impl, log/events/phase/next/end/finish; 319-407 existing environment/clock fixtures and tests.

Open detail: Keep crate::progress names and the START/DONE/FAILED/CANCELLED wire consumed by candidate_display. Retain the real injected clock/capture API and test environment locking/restoration; no new progress sink/service or event protocol is introduced.

## rust/soda-release-tools/src/worker.rs

Observed size: 1516 lines, including tests. Keep configuration custody, fresh runtime ownership and actual transient-unit execution cohesive within the existing Rust controller package.

- `lib/soda-release-tools/src/worker/mod.rs` — Worker/constants, uid/user/identity helpers, pure worker description, exact result decode/validation/rebinding, live-input naming/resolution and controller attempt dispatch.
- `lib/soda-release-tools/src/worker/config.rs` — WorkerConfig, path/trusted-executable/private-file custody, task/config/storage admission and bounded config read/load.
- `lib/soda-release-tools/src/worker/runtime.rs` — Fresh attempt runtime claim/nonce/private mode/chown and exact direct-child release.
- `lib/soda-release-tools/src/worker/execution.rs` — Whole existing execution boundary comments and identity/bind/env argv admission, paired concurrent pipe drains, systemd-run wait and exact-unit cancellation/kill/reap/cleanup errors.
- `lib/soda-release-tools/src/worker/tests.rs` — Actual current config/runtime/request/name/result/argv/live-input/early-progress refusal tests and one existing fixture closure.

Library boundary: JSON01 delegates WorkerConfig/result codec mechanics to serde/serde_json; RNG01/TMP01/FS01 use getrandom/tempfile/rustix without transferring runtime custody. PROC02 retains live controller cancellation, concurrent bounded drains, exact transient-unit stop/reap and cleanup uncertainty. Native systemd/container tooling remains the execution authority.

Evidence at f7: constants15–20; WorkerConfig22–34; Worker35–46 and identity47–100; config101–328; runtime329–408; pure description/result409–599; execution comments600–606 and complete functions607–836; live-input/controller dispatch837–947; cfg(test)948 and complete tests949–1516. Extraction includes attributes and comments, not function-body fragments.

Keep crate::worker public caller paths through defining reexports, with execution internal to the same package and no new process/API. `trusted_executable` has one config owner, used by execution; Worker has one mod owner. Private child helpers use parent-only imports/visibility, and tests remain descendants of the real subject rather than copied production code. Existing run_worker concurrently drains both pipes before polling exit; preserve that lifecycle. D01-F4 corrects the actual controller attempt's constant-false cancellation predicate, retaining existing exact-unit stop/reap behavior. Actual execution is wired; neither file movement nor source assertions establish installed isolation.

## rust/soda-release-tools/tests/cli.rs

Observed size: 472 lines, including tests where embedded. Organize tests by the three actual existing command identities, sharing only the current temporary directory and command-result fixtures. Keep each exact usage golden and every code/stdout/stderr assertion beside the command it verifies.

- `lib/soda-release-tools/tests/cli/main.rs` — Cargo-discovered CLI suite root with existing TempDir/drop/counter and binary command result capture.
- `lib/soda-release-tools/tests/cli/soda_build.rs` — Current build binary lookup/usage golden, help/flag/bool/admission matrix and deep refusal.
- `lib/soda-release-tools/tests/cli/soda_candidate.rs` — Current candidate binary lookup/usage golden/help/flags/answer/preflight refusals.
- `lib/soda-release-tools/tests/cli/soda_artifacts.rs` — Current artifacts binary lookup/action/native/Butane/archive-shape refusals.

Evidence: 1-60 TempDir/bin/run fixtures; 61-176 build CLI cases; 177-327 candidate cases; 328-434 artifact cases; 435-472 build deep refusal.

Open detail: Preserve completed suite discovery and CARGO_BIN_EXE_soda-build/candidate/artifacts identities. CLI02 updates obsolete foreign usage/flag/error goldens to the documented clap contract while retaining actual admission, boolean-value, positional-tail and branch/refusal observations. Existing pipeline-boundary refusal cases remain pending behavior evidence, not completed success-path proof.
