# Installation and operations

[Decomposition scope and baseline](README.md). This page groups historical
source reviews; it does not retain obsolete implementations in the target.

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

## rust/soda-candidate-setup/tests/cli.rs

Observed size: 463 lines, including tests where embedded. Keep this coherent pre-mutation admission suite together after extracting existing fixture/fake-tool/lease helpers. It explicitly does not prove the later privileged setup flow.

- `tools/candidate-setup/tests/cli.rs`
- `tools/candidate-setup/tests/support/mod.rs`

Evidence: rust/soda-candidate-setup/tests/cli.rs:1-5 pre-mutation-only scope comment; rust/soda-candidate-setup/tests/cli.rs:20-147 TempDir, command isolation, fake tools and HeldLock; rust/soda-candidate-setup/tests/cli.rs:149-463 source/toolchain/lease/active-build refusal tests.

## rust/soda-console-welcome/src/main.rs

Observed size: 790 lines, including tests where embedded. Separate operator observation/banner composition from its top-level config parser and display-origin validation. Keep read-only behavior and exactly the existing permissive Python-like JSON cases. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-console-welcome/src/main.rs`
- `cmd/soda-console-welcome/src/welcome.rs`
- `cmd/soda-console-welcome/src/config.rs`
- `cmd/soda-console-welcome/src/origin.rs`
- `cmd/soda-console-welcome/src/tests.rs`

Evidence: rust/soda-console-welcome/src/main.rs:19-182 root gate, observed uplinks, banner and subprocess helpers; rust/soda-console-welcome/src/main.rs:184-240 render_config; rust/soda-console-welcome/src/main.rs:245-578 JsonParser and top-object extraction; rust/soda-console-welcome/src/main.rs:581-707 listen/origin validation; rust/soda-console-welcome/src/main.rs:713-733 duplicate-key/string/NaN/Infinity tests.

Open detail: This parser accepts NaN and Infinity while other Rust JSON readers use different rules. Consolidating parsers or replacing its dependency-free implementation needs contract reconciliation, not a file move.

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

## rust/soda-image-import/src/main.rs

Observed size: 2172 lines, including tests where embedded. Separate already marked platform, hash, Go-compatible JSON binding, payload, OCI metadata/layout verification and Podman import sections. Keep whole-layout verification before any import and group the 872 test lines by current concern. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-image-import/src/main.rs`
- `cmd/soda-image-import/src/context.rs`
- `cmd/soda-image-import/src/platform.rs`
- `cmd/soda-image-import/src/sha256.rs`
- `cmd/soda-image-import/src/json_binding.rs`
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

Open detail: This duplicates installer release/OCI verification with somewhat different result shapes. A shared release/OCI library would require a caller/type/error comparison beyond a filetree split; do not create it solely from visual similarity.

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

## rust/soda-install/src/netip.rs

Observed size: 836 lines, including tests where embedded. Separate existing address parse, byte-preserving address formatting/classification and prefix operations, with grouped oracle vectors. Keep one canonical Addr/Prefix representation and parent-private helper access. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/netip/mod.rs`
- `cmd/soda-install/src/netip/address.rs`
- `cmd/soda-install/src/netip/address_format.rs`
- `cmd/soda-install/src/netip/prefix.rs`
- `cmd/soda-install/src/netip/tests/mod.rs`
- `cmd/soda-install/src/netip/tests/address.rs`
- `cmd/soda-install/src/netip/tests/prefix.rs`

Evidence: rust/soda-install/src/netip.rs:1-9 documented Go byte/error fidelity; rust/soda-install/src/netip.rs:69-331 IPv4/IPv6 byte parsing; rust/soda-install/src/netip.rs:333-462 address classification/string/mask helpers; rust/soda-install/src/netip.rs:463-543 Prefix parse/mask/contains; rust/soda-install/src/netip.rs:546-836 oracle address/prefix and error-text tests.

Open detail: Std IpAddr replacement is not equivalent to the existing arbitrary byte-zone/error contract. Upstream replacement is a separate behavior decision, not presumed by relocation.

## rust/soda-install/src/oci.rs

Observed size: 1175 lines, including tests where embedded. Separate fd-relative layout/blob loading, index/manifest/config binding and image/rootfs/attribution inspection. Preserve hard versus soft JSON decoding distinctions. Move the existing shared OCI fixture into test-only support. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/oci/mod.rs`
- `cmd/soda-install/src/oci/layout.rs`
- `cmd/soda-install/src/oci/metadata.rs`
- `cmd/soda-install/src/oci/inspection.rs`
- `cmd/soda-install/src/oci/test_support.rs`
- `cmd/soda-install/src/oci/tests.rs`

Evidence: rust/soda-install/src/oci.rs:27-258 layout types and descriptor-confined file/blob loading; rust/soda-install/src/oci.rs:260-617 soft/hard descriptor and manifest/config decoding; rust/soda-install/src/oci.rs:618-863 rootfs/attribution/image/layout inspection; rust/soda-install/src/oci.rs:866-1020 existing test fixture module; rust/soda-install/src/oci.rs:1023-1175 identity/count/substitution tests.

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

## rust/soda-install/src/sshkey.rs

Observed size: 904 lines, including tests where embedded. Separate Base64 codec, existing SSH key/certificate wire parse/marshal and authorized_keys text/options policy. Preserve wire truncation and canonicalization rules; keep key representation singular. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/sshkey/mod.rs`
- `cmd/soda-install/src/sshkey/base64.rs`
- `cmd/soda-install/src/sshkey/wire.rs`
- `cmd/soda-install/src/sshkey/authorized_keys.rs`
- `cmd/soda-install/src/sshkey/tests.rs`

Evidence: rust/soda-install/src/sshkey.rs:15-113 Go-compatible Base64 decode/encode; rust/soda-install/src/sshkey.rs:115-568 SSH wire integers, keys, certificates and marshaling; rust/soda-install/src/sshkey.rs:570-721 authorized_keys parsing, operator key policy and fingerprint; rust/soda-install/src/sshkey.rs:724-904 key vectors, unsupported types, canonicalization and codec tests.

Open detail: Replacing wire/key readers with a library requires checking certificate/security-key/trailing-byte and error-taxonomy behavior already exercised here. This plan proposes module boundaries only.

## rust/soda-install/src/urlx.rs

Observed size: 730 lines, including tests where embedded. Keep the coherent 373-line byte-preserving URL parser intact; extract its extensive oracle/error vectors as a descendant test module. Do not split its mutually dependent authority/unescape parser into arbitrary pieces. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/urlx.rs`
- `cmd/soda-install/src/urlx/tests.rs`

Evidence: rust/soda-install/src/urlx.rs:1-9 documented net/url byte/error semantics; rust/soda-install/src/urlx.rs:105-355 unescape, host/authority/scheme and parse; rust/soda-install/src/urlx.rs:356-370 hostname; rust/soda-install/src/urlx.rs:374-730 oracle URL and exact-error tests.

Open detail: Standard or third-party URL parsing cannot be assumed equivalent to decoded non-UTF8 fields and the existing scheme-specific host rules.

## rust/soda-install/src/wizard.rs

Observed size: 627 lines, including tests where embedded. Separate the established wizard state/navigation loop from individual input steps and the final disk-erasure review/confirmation. Preserve back-navigation resets and password-to-hash handling. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/wizard/mod.rs`
- `cmd/soda-install/src/wizard/steps.rs`
- `cmd/soda-install/src/wizard/review.rs`
- `cmd/soda-install/src/wizard/tests.rs`

Evidence: rust/soda-install/src/wizard.rs:25-284 navigation and network/disk/hostname/password/subnet steps; rust/soda-install/src/wizard.rs:285-366 final review and erase confirmation; rust/soda-install/src/wizard.rs:368-447 step dispatch and collect_disk_install_choices; rust/soda-install/src/wizard.rs:450-627 selection and complete PTY flow tests.

## rust/soda-install/src/x509.rs

Observed size: 3624 lines, including tests where embedded. Use the explicit DER, algorithm/type, validity, name/constraint, extension, key, ParseCertificate and CheckSignatureFrom sections as private modules. Group the 1286 test lines by those same protocols and keep existing DER fixture builders test-only. Signature math continues to use the existing crypto crates; no new cryptographic implementation is proposed. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-install/src/x509/mod.rs`
- `cmd/soda-install/src/x509/der.rs`
- `cmd/soda-install/src/x509/algorithms.rs`
- `cmd/soda-install/src/x509/types.rs`
- `cmd/soda-install/src/x509/time.rs`
- `cmd/soda-install/src/x509/names.rs`
- `cmd/soda-install/src/x509/name_constraints.rs`
- `cmd/soda-install/src/x509/extensions.rs`
- `cmd/soda-install/src/x509/public_key.rs`
- `cmd/soda-install/src/x509/certificate.rs`
- `cmd/soda-install/src/x509/verify.rs`
- `cmd/soda-install/src/x509/tests/mod.rs`
- `cmd/soda-install/src/x509/tests/fixtures.rs`
- `cmd/soda-install/src/x509/tests/structure.rs`
- `cmd/soda-install/src/x509/tests/public_key.rs`
- `cmd/soda-install/src/x509/tests/algorithms.rs`
- `cmd/soda-install/src/x509/tests/time.rs`
- `cmd/soda-install/src/x509/tests/extensions.rs`
- `cmd/soda-install/src/x509/tests/verify.rs`

Evidence: rust/soda-install/src/x509.rs:1-27 scope, exact Go parser/error semantics, rejected x509-cert profile and existing crypto delegates; rust/soda-install/src/x509.rs:38-378 DER reader and tag/OID constants; rust/soda-install/src/x509.rs:388-506 algorithm enums/public key/certificate types; rust/soda-install/src/x509.rs:512-781 validity time and ASN1 name/algorithm parsing; rust/soda-install/src/x509.rs:783-1481 extension and name-constraint parsing; rust/soda-install/src/x509.rs:1488-1862 signature algorithm and public-key parsing; rust/soda-install/src/x509.rs:1870-2046 ParseCertificate sequence; rust/soda-install/src/x509.rs:2052-2336 CheckSignatureFrom and existing signature verification delegates; rust/soda-install/src/x509.rs:2339-3624 structural/key/time/extension/signature vectors and DER fixture builders.

Open detail: The source explicitly documents why x509-cert did not match accepted serial/UTCTime behavior. Removing this port or narrowing certificate validation needs an owner decision and primary-source/caller validation. Moving modules must preserve parser error precedence and uses of netip/urlx; no public visibility should be added solely for tests.

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

Observed size: 1731 lines, including tests where embedded. Separate existing CLI/origin admission, Forgejo bootstrap HTTP, local JSON decoding/config emission, PostgreSQL/grant secrets and setup/revocation orchestration. Keep native ownership helpers local and group the 541 lines of existing tests by those concerns. Keep extracted unit tests as cfg(test) descendants of their owning module; do not make production helpers public for test access.

- `cmd/soda-setup/src/main.rs`
- `cmd/soda-setup/src/cli.rs`
- `cmd/soda-setup/src/origin.rs`
- `cmd/soda-setup/src/forgejo.rs`
- `cmd/soda-setup/src/json.rs`
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

Open detail: The local setup decoder, console parser and shared soda-json do not establish a single common acceptance/error contract. Keep local behavior until source/caller semantics are reconciled.

