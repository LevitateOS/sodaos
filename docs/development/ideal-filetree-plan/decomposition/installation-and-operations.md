# Installation and operations

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

Pending work follows [library-adoption findings](../library-adoption.md#finding-allocation)
and their [execution packets](../library-adoption.md#execution-packets). The
historical source spans below remain evidence; completed structural extractions
remain completed. Replace each selected generic engine at its actual caller
boundary, retaining Soda admission, descriptor custody and operation ordering.
Check dependency resolution, offline cache, artifact toolchains and raw-byte
fixtures at the [readiness gates](../library-adoption.md#readiness-gates) before
large cutovers; do not add a second compatibility implementation.

## rust/soda-activate/src/main.rs

Observed size: 1374 lines, including tests where embedded. Make main a thin entry; use the already present CliArgs, Paths/Sys, origin/IP admission, activation operation and Forgejo environment rewrite boundaries. Group existing unit tests by those concerns. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-activate/src/main.rs`
- `cmd/soda-activate/src/cli.rs`
- `cmd/soda-activate/src/origin.rs`
- `cmd/soda-activate/src/system.rs`
- `cmd/soda-activate/src/activation.rs`
- `cmd/soda-activate/src/forgejo_env.rs`
- `cmd/soda-activate/src/tests/mod.rs`
- `cmd/soda-activate/src/tests/fixtures.rs`
- `cmd/soda-activate/src/tests/activation.rs`
- `cmd/soda-activate/src/tests/origin.rs`
- `cmd/soda-activate/src/tests/cli.rs`

Evidence: rust/soda-activate/src/main.rs:75-186 CliArgs and parse_args; rust/soda-activate/src/main.rs:188-279 Paths, ActivateError and Sys/RealSys; rust/soda-activate/src/main.rs:289-513 IP/origin admission; rust/soda-activate/src/main.rs:515-775 activate; rust/soda-activate/src/main.rs:794-886 rewrite_forgejo_env; rust/soda-activate/src/main.rs:888-1374 fixtures and activation/origin/CLI tests.

## rust/soda-candidate-setup/src/main.rs

Observed size: 1558 lines, including tests where embedded. Extract the existing setup stages and subprocess/status helpers; keep their current order and task-owned cleanup in one small orchestration. Fixture authority stays separate from release keys. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/candidate-setup/src/main.rs`
- `tools/candidate-setup/src/process.rs`
- `tools/candidate-setup/src/preflight.rs`
- `tools/candidate-setup/src/storage.rs`
- `tools/candidate-setup/src/controller.rs`
- `tools/candidate-setup/src/worker_tools.rs`
- `tools/candidate-setup/src/worker_caches.rs`
- `tools/candidate-setup/src/selinux.rs`
- `tools/candidate-setup/src/fixture_authority.rs`
- `tools/candidate-setup/src/config.rs`
- `tools/candidate-setup/src/tests.rs`

Evidence: rust/soda-candidate-setup/src/main.rs:399-470 worker/trust/config JSON emission; rust/soda-candidate-setup/src/main.rs:496-603 active-build refusal, storage migration and SELinux helpers; rust/soda-candidate-setup/src/main.rs:605-745 input/native/toolchain admission and lease; rust/soda-candidate-setup/src/main.rs:755-818 build/admit controller and wrapper; rust/soda-candidate-setup/src/main.rs:820-949 worker directories and tool installation; rust/soda-candidate-setup/src/main.rs:950-1084 cache warming; rust/soda-candidate-setup/src/main.rs:1086-1166 worker SELinux installation; rust/soda-candidate-setup/src/main.rs:1191-1309 restricted worker configuration and fixture authority; rust/soda-candidate-setup/src/main.rs:1375-1558 helper unit tests.

Open detail: The long run_setup body currently interleaves local variables and ordered system mutations. Extract stage functions without a generic workflow/state machine; current privileged path is source-inspected, not executed here.

Library boundary: JSON01 delegates worker/trust/config documents to locked
serde/serde_json; RNG01, TMP01 and FS01 delegate randomness, temporary allocation
and typed rooted-file mechanics to getrandom, tempfile and rustix. The setup
stages retain lease ownership, fixture-only authority, private modes and the
ordered privileged installation/cleanup sequence. These are library-backed
stage helpers, not new generic engines to extract.

## rust/soda-candidate-setup/tests/cli.rs

Observed size: 463 lines, including tests where embedded. Keep this coherent pre-mutation admission suite together after extracting existing fixture/fake-tool/lease helpers. It explicitly does not prove the later privileged setup flow.

- `tools/candidate-setup/tests/cli.rs`
- `tools/candidate-setup/tests/support/mod.rs`

Evidence: rust/soda-candidate-setup/tests/cli.rs:1-5 pre-mutation-only scope comment; rust/soda-candidate-setup/tests/cli.rs:20-147 TempDir, command isolation, fake tools and HeldLock; rust/soda-candidate-setup/tests/cli.rs:149-463 source/toolchain/lease/active-build refusal tests.

## rust/soda-console-welcome/src/main.rs

Observed size: 790 lines, including tests where embedded. Keep operator observation/banner composition, configuration admission and display-origin policy distinct. JSON01 replaces the custom Python-like lexer with serde/serde_json at the config boundary. Retain read-only behavior and the actual display fields; incidental NaN/Infinity acceptance is a documented contract decision, not a reason to preserve another lexer. Keep tests as cfg(test) descendants of their owning module.

- `cmd/soda-console-welcome/src/main.rs`
- `cmd/soda-console-welcome/src/welcome.rs`
- `cmd/soda-console-welcome/src/config.rs`
- `cmd/soda-console-welcome/src/origin.rs`
- `cmd/soda-console-welcome/src/tests.rs`

Evidence: rust/soda-console-welcome/src/main.rs:19-182 root gate, observed uplinks, banner and subprocess helpers; rust/soda-console-welcome/src/main.rs:184-240 render_config; rust/soda-console-welcome/src/main.rs:245-578 JsonParser and top-object extraction; rust/soda-console-welcome/src/main.rs:581-707 listen/origin validation; rust/soda-console-welcome/src/main.rs:713-733 duplicate-key/string/NaN/Infinity tests.

Open detail: The config owner specifies duplicate fields, wrong types and missing display values under the JSON01 profile. The origin owner uses N7's url adapter with its own display-origin admission. Retire obsolete lexer/error-parity cases with the complete caller cutover; keep banner and refusal observations.

## rust/soda-forgejo-domain/src/main.rs

Observed size: 1021 lines, including tests where embedded. Separate the existing native-writer control verbs from deployment app.ini/env parsing and host marker mapping. Retain native offline-marker name and quiescence/inhibition semantics. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-forgejo-domain/src/main.rs`
- `cmd/soda-forgejo-domain/src/cli.rs`
- `cmd/soda-forgejo-domain/src/system.rs`
- `cmd/soda-forgejo-domain/src/config.rs`
- `cmd/soda-forgejo-domain/src/domain.rs`
- `cmd/soda-forgejo-domain/src/tests/mod.rs`
- `cmd/soda-forgejo-domain/src/tests/fixtures.rs`
- `cmd/soda-forgejo-domain/src/tests/config.rs`
- `cmd/soda-forgejo-domain/src/tests/domain.rs`
- `cmd/soda-forgejo-domain/src/tests/cli.rs`

Evidence: rust/soda-forgejo-domain/src/main.rs:80-187 CLI, Paths and Sys; rust/soda-forgejo-domain/src/main.rs:237-493 INI interpolation, AppDataPath and marker mapping; rust/soda-forgejo-domain/src/main.rs:495-625 stop/inhibit/status/lift/start; rust/soda-forgejo-domain/src/main.rs:628-1021 fake system and configuration/control tests.

Library boundary: CFG01 keeps parser selection blocked on the native Forgejo
configuration contract. Qualify rust-ini against the admitted Forgejo version's
effective APP_DATA_PATH/interpolation behavior before replacing the port. This
is distinct from the narrow locale catalog scanner. Native writer quiescence,
offline markers, inhibition and lift/start ordering remain domain policy.

## rust/soda-image-import/src/main.rs

Observed size: 2172 lines, including tests where embedded. Keep platform, payload, OCI metadata/layout verification and Podman import as the existing operation boundaries. CF-01 replaces the streaming SHA engine with sha2; JSON01 replaces lexer/binding mechanics with serde/serde_json and a narrow input-policy visitor. Keep whole-layout verification before any import and tests with their current subjects.

- `cmd/soda-image-import/src/main.rs`
- `cmd/soda-image-import/src/context.rs`
- `cmd/soda-image-import/src/platform.rs`
- `cmd/soda-image-import/src/json_binding.rs` — Input-policy adapter over serde, without a second lexer.
- `cmd/soda-image-import/src/payload.rs`
- `cmd/soda-image-import/src/oci/mod.rs`
- `cmd/soda-image-import/src/oci/metadata.rs`
- `cmd/soda-image-import/src/oci/layout.rs`
- `cmd/soda-image-import/src/oci/inspection.rs`
- `cmd/soda-image-import/src/import.rs`
- `cmd/soda-image-import/src/tests/mod.rs`
- `cmd/soda-image-import/src/tests/fixtures.rs`
- `cmd/soda-image-import/src/tests/payload.rs`
- `cmd/soda-image-import/src/tests/oci.rs`
- `cmd/soda-image-import/src/tests/import.rs`
- `cmd/soda-image-import/src/tests/primitives.rs`

Evidence: rust/soda-image-import/src/main.rs:49-120 entry admission and import cancellation context; rust/soda-image-import/src/main.rs:122-205 native platform and identifier shapes; rust/soda-image-import/src/main.rs:207-521 streaming SHA and JSON binding; rust/soda-image-import/src/main.rs:524-692 Payload decoding/validation/load; rust/soda-image-import/src/main.rs:693-1135 OCI metadata/layout/content inspection; rust/soda-image-import/src/main.rs:1136-1298 verified native import; rust/soda-image-import/src/main.rs:1301-2172 fixture, payload, layout and import tests.

Open detail: Retire the custom sha256 module after its callers use sha2. FS01 preserves bounded, no-follow, descriptor-relative layout/blob admission. REL02's surviving OCI inspection owner requires the documented caller/type/error comparison; do not infer equivalence from similar layouts. Digest checking still hashes the admitted original bytes before native import.

## rust/soda-install/src/candidate.rs

Observed size: 453 lines, including tests where embedded. Keep the 216-line candidate-media authentication and destination assembly together; extract its substantial OCI/template fixtures and tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/candidate.rs`
- `cmd/soda-install/src/candidate/tests.rs`

Evidence: rust/soda-install/src/candidate.rs:18-67 candidate media/payload/image authentication; rust/soda-install/src/candidate.rs:113-215 host-file rewrite and candidate_destination; rust/soda-install/src/candidate.rs:217-453 fixtures and requirement/destination tests.

## rust/soda-install/src/console.rs

Observed size: 700 lines, including tests where embedded. Separate raw TTY read/write/echo handling from the established network editor/review interaction. Move the existing PTY test support into a test-only child module still shared by installer tests. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/console/mod.rs`
- `cmd/soda-install/src/console/terminal.rs`
- `cmd/soda-install/src/console/network.rs`
- `cmd/soda-install/src/console/test_support.rs`
- `cmd/soda-install/src/console/tests.rs`

Evidence: rust/soda-install/src/console.rs:21-217 terminal IO, line/secret input and echo restoration; rust/soda-install/src/console.rs:219-391 nmtui and live-network interaction; rust/soda-install/src/console.rs:396-482 byte trimming and errno formatting; rust/soda-install/src/console.rs:485-593 existing shared test_support PTY helpers; rust/soda-install/src/console.rs:595-700 console and network tests.

Library boundary: N10/FS01 use rustix for typed descriptor and terminal mechanics,
with std I/O/process helpers. The console retains secret echo restoration,
terminal custody and the native nmtui interaction. Existing PTY support is
cfg(test) infrastructure, not a production library to preserve or extract anew.

## rust/soda-install/src/deliver.rs

Observed size: 524 lines, including tests where embedded. Keep the 288-line installer payload reader and content-binding entry together; extract payload and OCI fixtures/tests. This is installer verification, not release publication. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/deliver.rs`
- `cmd/soda-install/src/deliver/tests.rs`

Evidence: rust/soda-install/src/deliver.rs:28-240 Payload shapes, validation, decoding and load; rust/soda-install/src/deliver.rs:243-286 revision/layout identity binding and verify_content; rust/soda-install/src/deliver.rs:290-524 validation/load/OCI-binding tests.

## rust/soda-install/src/disks.rs

Observed size: 588 lines, including tests where embedded. Keep the 403-line disk inventory/identity safety operation intact; its sysfs, live-media and holders checks jointly determine one disk selection. Extract tests rather than split this safety sequence by size. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/disks.rs`
- `cmd/soda-install/src/disks/tests.rs`

Evidence: rust/soda-install/src/disks.rs:173-315 sysfs/live-media/removable/holders reads; rust/soda-install/src/disks.rs:317-388 scan_disks and same_disk; rust/soda-install/src/disks.rs:405-588 inventory, live-media and identity-change tests.

## rust/soda-install/src/enroll/arm.rs

Observed size: 636 lines, including tests where embedded. Retain the current 397-line enrollment address selection/intent/arming refusal path; extract descendant tests for admission, expiry and preserving existing state. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/arm.rs`
- `cmd/soda-install/src/enroll/arm_tests.rs`

Evidence: rust/soda-install/src/enroll/arm.rs:31-185 interface/state/window input; rust/soda-install/src/enroll/arm.rs:202-315 select address, fingerprint and explicit intent; rust/soda-install/src/enroll/arm.rs:329-396 guard_existing_enrollment_state; rust/soda-install/src/enroll/arm.rs:398-636 address/state/unit admission tests.

## rust/soda-install/src/enroll/keys.rs

Observed size: 791 lines, including tests where embedded. Separate descriptor-owned directory/stat helpers from the cohesive authorized_keys append/create/confirm transaction. Preserve one-write uncertainty and concurrent-native-editor protection. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/keys/mod.rs`
- `cmd/soda-install/src/enroll/keys/directory.rs`
- `cmd/soda-install/src/enroll/keys/authorized_keys.rs`
- `cmd/soda-install/src/enroll/keys/tests.rs`

Evidence: rust/soda-install/src/enroll/keys.rs:40-145 descriptor, ownership, lock and key-stat helpers; rust/soda-install/src/enroll/keys.rs:149-308 existing-key append and exclusive creation; rust/soda-install/src/enroll/keys.rs:361-465 append_enrollment_key_with_writer and confirmations; rust/soda-install/src/enroll/keys.rs:467-491 safe directory and native root-home admission; rust/soda-install/src/enroll/keys.rs:494-791 preservation/race/partial-write tests.

Library boundary: FS01 replaces raw FD/ABI mechanics with rustix/libc at the
directory owner; CF-03 supplies the admitted canonical public key. Keep the
same-inode/ownership checks, native-editor race protection, exclusive create,
one-write uncertainty and confirmation transaction together.

## rust/soda-install/src/enroll/mod.rs

Observed size: 519 lines, including tests where embedded. Leave the enrollment entry module small; separate generated native SSH/systemd configuration from socket peer/cgroup provenance. Move already shared test keys, temporary-directory helpers and environment lock into a test-only sibling. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/enroll/mod.rs`
- `cmd/soda-install/src/enroll/native_config.rs`
- `cmd/soda-install/src/enroll/peer.rs`
- `cmd/soda-install/src/enroll/tests.rs`
- `cmd/soda-install/src/enroll/test_support.rs`

Evidence: rust/soda-install/src/enroll/mod.rs:19-27 fixed enrollment paths and limits; rust/soda-install/src/enroll/mod.rs:39-81 fixed restricted sshd configuration; rust/soda-install/src/enroll/mod.rs:131-197 start/socket/template unit configuration; rust/soda-install/src/enroll/mod.rs:199-252 root peer credentials and native unit provenance; rust/soda-install/src/enroll/mod.rs:255-355 existing shared test-only constants and helpers; rust/soda-install/src/enroll/mod.rs:357-519 configuration/provenance tests.

## rust/soda-install/src/execute.rs

Observed size: 695 lines, including tests where embedded. Keep the existing 393-line disk-attempt execution, retry and diagnostic landing flow together; extract tests. The fresh inventory, intentional-removable confirmation and started marker remain ordered before disk writing. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/execute.rs`
- `cmd/soda-install/src/execute/tests.rs`

Evidence: rust/soda-install/src/execute.rs:69-112 failure/retry flow; rust/soda-install/src/execute.rs:151-211 diagnostic console landing; rust/soda-install/src/execute.rs:261-389 verified disk attempt and execute_disk write boundary; rust/soda-install/src/execute.rs:394-695 disk execution/refusal/landing tests.

## rust/soda-install/src/inputs.rs

Observed size: 454 lines, including tests where embedded. Keep the 211-line private provisioning input validation/template extension together; extract input vectors and destination assertions. No new schema or validation rule is proposed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/inputs.rs`
- `cmd/soda-install/src/inputs/tests.rs`

Evidence: rust/soda-install/src/inputs.rs:13-60 hostname/subnet/password admission; rust/soda-install/src/inputs.rs:62-160 template decoding and path collisions; rust/soda-install/src/inputs.rs:162-208 destination construction; rust/soda-install/src/inputs.rs:212-454 vectors and provisioning template tests.

Installer JSON01 completed in `595fb604`; `jsongo.rs` is deleted. Dynamic
Ignition uses Serde values with a small Go-compatible formatter, while factory
defaults and payload/media records use concrete DTOs. Sorted output, embedded
machine bytes, file-content newlines and raw authenticated hashes retain their
existing duties. All 128 installer checks and the development build passed
with medium review; the historical split remains distinguishable from that
completed engine replacement.

## rust/soda-install/src/netip.rs

Observed size: 836 lines, including tests where embedded. N8 replaces the IPv4/IPv6 grammar and formatting engine with std::net::IpAddr. Retain one small adapter for admitted zones, prefix masking/containment and purpose-specific address selection. Do not further split the legacy grammar/error port into permanent target modules. Keep address/prefix policy tests with the adapter.

- `cmd/soda-install/src/netip/mod.rs`
- `cmd/soda-install/src/netip/tests/mod.rs`
- `cmd/soda-install/src/netip/tests/address.rs`
- `cmd/soda-install/src/netip/tests/prefix.rs`

Evidence: rust/soda-install/src/netip.rs:1-9 documented Go byte/error fidelity; rust/soda-install/src/netip.rs:69-331 IPv4/IPv6 byte parsing; rust/soda-install/src/netip.rs:333-462 address classification/string/mask helpers; rust/soda-install/src/netip.rs:463-543 Prefix parse/mask/contains; rust/soda-install/src/netip.rs:546-836 oracle address/prefix and error-text tests.

Open detail: N8 explicitly narrows obsolete arbitrary byte-zone and Go error-text emulation. Retain documented zone/prefix requirements and the installer callers' admissibility rules; std classification alone does not decide which address may be selected for enrollment or setup. These acceptance decisions accompany the complete engine replacement.

## rust/soda-install/src/oci.rs

Observed size: 1175 lines, including tests where embedded. Keep fd-relative layout/blob loading, index/manifest/config admission and image/rootfs/attribution inspection distinct. JSON01 delegates codec mechanics to serde/serde_json with explicit hard/soft input profiles; FS01 delegates typed descriptor mechanics to rustix. The existing OCI fixture stays test-only, and REL02 governs any proven inspector reuse.

- `cmd/soda-install/src/oci/mod.rs`
- `cmd/soda-install/src/oci/layout.rs`
- `cmd/soda-install/src/oci/metadata.rs`
- `cmd/soda-install/src/oci/inspection.rs`
- `cmd/soda-install/src/oci/test_support.rs`
- `cmd/soda-install/src/oci/tests.rs`

Evidence: rust/soda-install/src/oci.rs:27-258 layout types and descriptor-confined file/blob loading; rust/soda-install/src/oci.rs:260-617 soft/hard descriptor and manifest/config decoding; rust/soda-install/src/oci.rs:618-863 rootfs/attribution/image/layout inspection; rust/soda-install/src/oci.rs:866-1020 existing test fixture module; rust/soda-install/src/oci.rs:1023-1175 identity/count/substitution tests.

In `595fb604`, schema-specific raw-slot Serde visitors replace the OCI binder.
They retain ordered alias selection, caller null rules and integer lexemes
including `-0`, without a generic object-field lookup engine. File/blob custody
and original digest verification remain here; FS01 and REL02 remain pending.

## rust/soda-install/src/setup.rs

Observed size: 1140 lines, including tests where embedded. Separate existing address and token prompting, reserved setup/activation attempt, configured-access guidance and public CA parsing/fingerprint logic. Group the 517 lines of unit/PTY tests by flow and access concern. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/setup/mod.rs`
- `cmd/soda-install/src/setup/address.rs`
- `cmd/soda-install/src/setup/configure.rs`
- `cmd/soda-install/src/setup/access.rs`
- `cmd/soda-install/src/setup/local_ca.rs`
- `cmd/soda-install/src/setup/tests/mod.rs`
- `cmd/soda-install/src/setup/tests/fixtures.rs`
- `cmd/soda-install/src/setup/tests/configure.rs`
- `cmd/soda-install/src/setup/tests/access.rs`

Evidence: rust/soda-install/src/setup.rs:18-156 setup address decoding/origin selection; rust/soda-install/src/setup.rs:209-435 prompting, live-address recheck and setup/activation operation; rust/soda-install/src/setup.rs:472-600 installed origin/unit admission and client trust guidance; rust/soda-install/src/setup.rs:602-619 public CA fingerprint verification; rust/soda-install/src/setup.rs:623-1140 CA/configuration/rollback/secret-transcript tests.

Library boundary: X50901 uses x509-cert plus the existing RustCrypto signature
verifiers for the sole local-CA display caller. Keep the 16 KiB regular-file
bound, one CERTIFICATE PEM, CA/self-signature admission and SHA256 fingerprint
of the admitted raw DER. Dates, chain trust and principals are not newly implied
by this display operation. The signature check consumes the original signed
TBS bytes rather than a re-encoded certificate.

## rust/soda-install/src/sshkey.rs

Observed size: 904 lines, including tests where embedded. CF-03 replaces the full key/certificate wire reader and serializer with ssh-key 0.6.7; CF-04 supplies named base64 0.22.1 profiles. Keep a small installer adapter and authorized_keys text/admission policy with one library key representation. The complete ordinary/certificate algorithm matrix is an acceptance gate, not separate hand-written wire modules to extract.

- `cmd/soda-install/src/sshkey/mod.rs`
- `cmd/soda-install/src/sshkey/authorized_keys.rs`
- `cmd/soda-install/src/sshkey/tests.rs`

Evidence: rust/soda-install/src/sshkey.rs:15-113 Go-compatible Base64 decode/encode; rust/soda-install/src/sshkey.rs:115-568 SSH wire integers, keys, certificates and marshaling; rust/soda-install/src/sshkey.rs:570-721 authorized_keys parsing, operator key policy and fingerprint; rust/soda-install/src/sshkey.rs:724-904 key vectors, unsupported types, canonicalization and codec tests.

Open detail: Retire the historical base64 and wire leaves after the actual callers migrate. Retain the installer key allowlist, option/comment/newline and input-size policy; reject library Other explicitly. Use the existing RustCrypto PublicKey adapters for real ECDSA curve validation while retaining the uncompressed-point gate. Parsing a certificate does not authorize it or request Certificate::validate. Review canonical mpint/UTF8/trailing-data and padding changes through CF-03/04 fixtures, rather than retain a parallel Go-compatible parser.

## rust/soda-install/src/urlx.rs

Observed size: 730 lines, including tests where embedded. N7 replaces manifest/origin URL parsing with url 2.5.8 and explicit raw lexical/admission guards. X50901 removes the obsolete URI-SAN parser caller. Retire the complete urlx port once both caller changes land; there is no desired generic URL-parser leaf here.

The historical urlx module and its exact foreign-error vectors are retirement scope. Keep manifest/origin policy fixtures with their actual caller.

Evidence: rust/soda-install/src/urlx.rs:1-9 documented net/url byte/error semantics; rust/soda-install/src/urlx.rs:105-355 unescape, host/authority/scheme and parse; rust/soda-install/src/urlx.rs:356-370 hostname; rust/soda-install/src/urlx.rs:374-730 oracle URL and exact-error tests.

Open detail: WHATWG normalization does not authorize a host or rewrite signed URL literals. Preserve required HTTPS, userinfo/query/control/backslash/escape restrictions and the original authenticated text. Certificate URI representation belongs to the ASN.1/X509 library; do not route it through the web-URL adapter.

## rust/soda-install/src/wizard.rs

Observed size: 627 lines, including tests where embedded. Separate the established wizard state/navigation loop from individual input steps and the final disk-erasure review/confirmation. Preserve back-navigation resets and password-to-hash handling. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/wizard/mod.rs`
- `cmd/soda-install/src/wizard/steps.rs`
- `cmd/soda-install/src/wizard/review.rs`
- `cmd/soda-install/src/wizard/tests.rs`

Evidence: rust/soda-install/src/wizard.rs:25-284 navigation and network/disk/hostname/password/subnet steps; rust/soda-install/src/wizard.rs:285-366 final review and erase confirmation; rust/soda-install/src/wizard.rs:368-447 step dispatch and collect_disk_install_choices; rust/soda-install/src/wizard.rs:450-627 selection and complete PTY flow tests.

## rust/soda-install/src/x509.rs

Observed size: 3624 lines, including tests where embedded. X50901 plans replacement of the complete parser with x509-cert 0.2.5's strict profile and a narrow local self-signed-CA adapter. CF-06's custom ECDSA signature DER reader is now retired through ecdsa::Signature::from_der on all four supported curves; existing RustCrypto libraries verify the original signed TBS. The prior DER/calendar/name/extension module split is historical extraction evidence, not a target generic parser hierarchy.

- `cmd/soda-install/src/x509/mod.rs`
- `cmd/soda-install/src/x509/verify.rs`
- `cmd/soda-install/src/x509/tests/mod.rs`
- `cmd/soda-install/src/x509/tests/fixtures.rs`
- `cmd/soda-install/src/x509/tests/structure.rs`
- `cmd/soda-install/src/x509/tests/public_key.rs`
- `cmd/soda-install/src/x509/tests/algorithms.rs`
- `cmd/soda-install/src/x509/tests/verify.rs`

Evidence: rust/soda-install/src/x509.rs:1-27 scope, exact Go parser/error semantics, rejected x509-cert profile and existing crypto delegates; rust/soda-install/src/x509.rs:38-378 DER reader and tag/OID constants; rust/soda-install/src/x509.rs:388-506 algorithm enums/public key/certificate types; rust/soda-install/src/x509.rs:512-781 validity time and ASN1 name/algorithm parsing; rust/soda-install/src/x509.rs:783-1481 extension and name-constraint parsing; rust/soda-install/src/x509.rs:1488-1862 signature algorithm and public-key parsing; rust/soda-install/src/x509.rs:1870-2046 ParseCertificate sequence; rust/soda-install/src/x509.rs:2052-2336 CheckSignatureFrom and existing signature verification delegates; rust/soda-install/src/x509.rs:2339-3624 structural/key/time/extension/signature vectors and DER fixture builders.

Open detail: Preserve raw admitted DER for the fingerprint and the exact signed TBS bytes for verification; do not hash or verify re-encoded data. Retain CA/self-signature policy and weak-signature refusal. X50901 explicitly narrows noncanonical DER, long serials, legacy UTC forms and trailing DER at the fixture gate; the display caller already collapses parser failures to one neutral error. It does not consult expiry, chains or the recorded unhandled-critical-extension list. Define the critical-extension admission decision in that gate, rather than add a new trust verifier. ASN.1 time, SAN/name/constraint parsing and netip/urlx dependencies retire with the complete L06 owner cutover; the separate CF-06 signature repair is complete.

## rust/soda-rotate-lab-creds/src/main.rs

Observed size: 831 lines, including tests where embedded. Separate read-only lab inventory/runbooks from the explicitly acknowledged fixture-key rotation operation and process/status helpers. Retain class-specific acknowledgement and no-secret-output behavior. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `tools/lab-credentials/src/main.rs`
- `tools/lab-credentials/src/process.rs`
- `tools/lab-credentials/src/inventory.rs`
- `tools/lab-credentials/src/fixture_authority.rs`
- `tools/lab-credentials/src/runbooks.rs`
- `tools/lab-credentials/src/tests.rs`

Evidence: rust/soda-rotate-lab-creds/src/main.rs:168-334 process capture and trust/config output; rust/soda-rotate-lab-creds/src/main.rs:337-499 runbooks and credential file metadata inventory; rust/soda-rotate-lab-creds/src/main.rs:537-622 fixture-only authority rotation; rust/soda-rotate-lab-creds/src/main.rs:624-708 command dispatch and cleanup; rust/soda-rotate-lab-creds/src/main.rs:711-831 serialization/metadata/helper tests.

## rust/soda-setup/src/main.rs

Observed size: 1731 lines, including tests where embedded. Keep CLI/origin admission, Forgejo bootstrap transport, configuration DTOs, PostgreSQL/grant secret policy and setup/revocation orchestration distinct. JSON01 uses serde/serde_json for decoding/emission; N3 uses the resolved ureq 2.12.1 transport. RNG01 and CF-04 replace random/Base64 mechanics with getrandom and explicit base64 profiles. Keep native ownership and publication/rollback order local.

- `cmd/soda-setup/src/main.rs`
- `cmd/soda-setup/src/cli.rs`
- `cmd/soda-setup/src/origin.rs`
- `cmd/soda-setup/src/forgejo.rs`
- `cmd/soda-setup/src/json.rs` — Go-compatible Serde output formatter for the configuration producer; Forgejo response admission lives with its typed DTO in `forgejo.rs`.
- `cmd/soda-setup/src/config.rs`
- `cmd/soda-setup/src/secrets.rs`
- `cmd/soda-setup/src/setup.rs`
- `cmd/soda-setup/src/system.rs`
- `cmd/soda-setup/src/tests/mod.rs`
- `cmd/soda-setup/src/tests/fixtures.rs`
- `cmd/soda-setup/src/tests/setup.rs`
- `cmd/soda-setup/src/tests/postgres.rs`
- `cmd/soda-setup/src/tests/admission.rs`
- `cmd/soda-setup/src/tests/encoding.rs`

Evidence: rust/soda-setup/src/main.rs:79-246 entry and Go-style CLI parsing; rust/soda-setup/src/main.rs:247-333 origin and bootstrap credential admission; rust/soda-setup/src/main.rs:334-563 Forgejo user/token HTTP path; rust/soda-setup/src/main.rs:567-843 local JSON parse and string emission; rust/soda-setup/src/main.rs:845-905 dashboard config encoding; rust/soda-setup/src/main.rs:907-1067 randomness/encoding/PostgreSQL secret generation/reuse; rust/soda-setup/src/main.rs:1090-1187 setup publication/revocation and exclusive output; rust/soda-setup/src/main.rs:1190-1731 HTTP fixtures, secret/admission/encoding tests.

JSON01 completed Setup's typed response and configuration transfer in
`ae09f634`, and CF-04 completed its Base64 transfer in `26493cf2`. The parser
and codec engines in the historical ranges are retired. The response cap,
scalar admission, configuration field order, escaping and newline remain local
policy, with four encoding checks and the development build recorded.

Open detail: JSON01 keeps distinct caller profiles instead of one global acceptance configuration. N3 retains status/body bounds, deadline/error secrecy and bootstrap uncertainty outside ureq. CLI01 retains the small admitted selector instead of a foreign flag emulator. Preserve private token custody, exclusive outputs, native account ownership and revocation after uncertain setup effects; library adoption does not reorder those operations.
