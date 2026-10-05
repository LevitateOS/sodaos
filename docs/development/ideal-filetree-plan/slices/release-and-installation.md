# Release and installation

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## D01 Builder admission and controllers

Admit exact clean source, a restricted controller/worker configuration and a fresh attempt before running isolated build code.

- **Entrypoints:** soda-build; soda-candidate interactive/non-interactive wrapper; soda-candidate-setup builder provisioning.
- **Owned data:** Root-owned admitted executable and worker configuration; Canonical Soda/Fountain revisions; Worker storage roots, leases and per-attempt runtime/output.
- **Authority:** Root admits the dispatcher; source execution uses soda-build-worker; Build identity has no operator-home or production release-key custody; Public dispatch refuses production mode.
- **Dependencies:** [D02](#d02-pinned-input-acquisition); systemd; Pinned Go/Rust/Bun tools.
- **Source files:** [rust/soda-release-tools/src/build_cli.rs:132](../../../../rust/soda-release-tools/src/build_cli.rs#L132); [rust/soda-release-tools/src/worker.rs:308](../../../../rust/soda-release-tools/src/worker.rs#L308); [docs/development/native-support.md:70](../../native-support.md#L70); [rust/soda-candidate-setup/README.md:1](../../../../rust/soda-candidate-setup/README.md#L1).
- **Tests:** [rust/soda-release-tools/tests/cli.rs:105](../../../../rust/soda-release-tools/tests/cli.rs#L105) — CLI fixture admission/refusal matrix; no successful privileged build; [rust/soda-release-tools/src/worker.rs:1071](../../../../rust/soda-release-tools/src/worker.rs#L1071) — Unit: worker state paths must stay under admitted storage root.
- **Unclear boundaries:** Builder provisioning changes host policy; ordinary controllers consume that policy. Wrapper interaction is not independent admission/signing authority.
- **Evidence status:** Current development controller path; no production-run availability implied.

## D02 Pinned input acquisition

Resolve current upstream inputs, acquire verified bytes and record the exact identities consumed by a candidate.

- **Entrypoints:** soda-build parent per-attempt input resolution; soda-artifacts fetch-coreos|fetch-coreos-iso; soda-fetch-terminal|soda-fetch-tea|soda-fetch-muse.
- **Owned data:** Controller-resolved live-inputs JSON; Asset locks/manifests and verified caches; CoreOS stream identities, signatures/keyring and verified ISO/base outputs.
- **Authority:** Controller resolves live inputs; isolated build stage consumes them and refuses missing resolution; Pinned checksums/signatures constrain acquired bytes.
- **Dependencies:** [D01](#d01-builder-admission-and-controllers); Upstream HTTPS registries/streams; Native signature verification and decompression tools.
- **Source files:** [rust/soda-release-tools/src/artifacts.rs:12](../../../../rust/soda-release-tools/src/artifacts.rs#L12); [rust/soda-release-build/src/coreos_stream.rs:179](../../../../rust/soda-release-build/src/coreos_stream.rs#L179); [rust/soda-release-build/src/coreos_iso.rs:42](../../../../rust/soda-release-build/src/coreos_iso.rs#L42); [rust/soda-asset-fetchers/Cargo.toml:10](../../../../rust/soda-asset-fetchers/Cargo.toml#L10).
- **Tests:** [rust/soda-asset-fetchers/src/terminal.rs:202](../../../../rust/soda-asset-fetchers/src/terminal.rs#L202) — Local HTTP fixture: exact archive members/digests, verified cache reuse and changed-cache refetch; [rust/soda-release-build/src/coreos.rs:428](../../../../rust/soda-release-build/src/coreos.rs#L428) — Stub transport: redirect/bounds/no-overwrite oracle cases; no live upstream/TLS qualification.
- **Unclear boundaries:** Pinned acquired bytes are distinct from live resolution. Mutable RPM repositories and floating Butane/Assembler remain recorded inputs, not promised fully locked/reproducible builds.
- **Evidence status:** Current source acquisition contracts; no network execution.

## D03 Candidate production

Compile shipping programs and package the immutable application/host candidate from admitted source and resolved inputs.

- **Entrypoints:** soda-build --development --target candidate; Worker stage -> release-tools pipeline -> release-build/release-image.
- **Owned data:** Fresh work/artifacts/logs/evidence directories; Program binaries, staged assets and source archives; Payload/candidate records, six application images (dashboard, Forgejo, extension, proxy, Project OS and Tailnet) and host OCI image; Observed package/toolchain provenance.
- **Authority:** Isolated build worker within admitted source/output/cache boundaries; Candidate identities do not authorize publication, installation or production activation.
- **Dependencies:** [D01](#d01-builder-admission-and-controllers); [D02](#d02-pinned-input-acquisition); Podman; Go/Rust/Bun; Fountain source.
- **Source files:** [rust/soda-release-tools/src/pipeline.rs:550](../../../../rust/soda-release-tools/src/pipeline.rs#L550); [rust/soda-release-image/src/build.rs:522](../../../../rust/soda-release-image/src/build.rs#L522); [rust/soda-release-build/src/production.rs:682](../../../../rust/soda-release-build/src/production.rs#L682); [docs/development/native-support.md:146](../../native-support.md#L146); [rust/soda-release-image/src/model.rs:15](../../../../rust/soda-release-image/src/model.rs#L15).
- **Tests:** [rust/soda-release-build/src/production.rs:1006](../../../../rust/soda-release-build/src/production.rs#L1006) — Oracle-style fake production commands: failure stops before later images/exports; [rust/soda-release-build/src/production.rs:1079](../../../../rust/soda-release-build/src/production.rs#L1079) — Fixture command recipes and ELF verification, not a real candidate build.
- **Unclear boundaries:** Artifact checking is D05; media is D04. Product source suites consume established contracts and are not release qualification.
- **Evidence status:** Current development candidate producer; source fixtures inspected without execution.

## D04 Authenticated installation media

Turn the candidate into authenticated native CoreOS installation media bound to exact ISO/rootfs and provisioning inputs.

- **Entrypoints:** soda-build --development --target media; release-image prepare_build_media/assemble_media.
- **Owned data:** destination.ign/live.ign; Packaging inventories and fixture media trust/key files; ISO/rootfs/media.json, hash-named URL and readback evidence; Disposable Assembler containers/scratch.
- **Authority:** Explicit fixture-only media authority; never production signing custody; Fresh media output and explicit reachable rootfs URL; Candidate-only target skips media admission/execution.
- **Dependencies:** [D02](#d02-pinned-input-acquisition); [D03](#d03-candidate-production); Butane; CoreOS Assembler; Sigstore/native OCI verification.
- **Source files:** [rust/soda-release-image/src/build_media.rs:23](../../../../rust/soda-release-image/src/build_media.rs#L23); [rust/soda-release-image/src/media.rs:585](../../../../rust/soda-release-image/src/media.rs#L585); [docs/development/native-support.md:126](../../native-support.md#L126).
- **Tests:** [rust/soda-release-image/src/build_media.rs:185](../../../../rust/soda-release-image/src/build_media.rs#L185) — Stub production: candidate target skips media setup/finishing; [rust/soda-release-image/src/media.rs:1159](../../../../rust/soda-release-image/src/media.rs#L1159) — Oracle unit: credential/query/loopback media URL refusals; no media build/boot.
- **Unclear boundaries:** Fixture authentication is not production admission or public release delivery. Successful media assembly is distinct from D06 installed qualification.
- **Evidence status:** Current development media path; native installed proof not claimed.

## D05 Artifact verification

Inspect an emitted candidate and verify architecture/revision, archive and payload identity bindings without rebuilding it.

- **Entrypoints:** soda-candidate-check; soda-artifacts inspect-oci; scripts/check-native.sh ARCH CANDIDATE_DIR.
- **Owned data:** Existing candidate/payload JSON and OCI archives; Expected native architecture and Soda/Fountain revisions; Inspection observations; no installed runtime state.
- **Authority:** Read/inspect emitted artifacts under explicit candidate bindings; No build, installation, publication or host-trust mutation.
- **Dependencies:** [D03](#d03-candidate-production); D04 when verifying media; Native artifact verification tools.
- **Source files:** [rust/soda-release-tools/src/check_cli.rs:67](../../../../rust/soda-release-tools/src/check_cli.rs#L67); [rust/soda-release-deliver/src/check.rs:1](../../../../rust/soda-release-deliver/src/check.rs#L1); [scripts/check-native.sh:1](../../../../scripts/check-native.sh#L1).
- **Tests:** [rust/soda-release-deliver/tests/oracle.rs:487](../../../../rust/soda-release-deliver/tests/oracle.rs#L487) — Fixture: wrong architecture/revision rejected before archive checks; correct bindings reach absent-archive refusal; [rust/soda-release-tools/src/check_cli.rs:102](../../../../rust/soda-release-tools/src/check_cli.rs#L102) — Unit flag binding and missing/unknown argument matrix.
- **Unclear boundaries:** Archive/source correctness is not native installation, recovery or product-journey qualification. Qualification record acceptance belongs D07, observation generation D06.
- **Evidence status:** Current inspection entrypoints; no checks executed.

## D06 Installed qualification

Collect retained matching-native observations for exact candidate installation and guest state, invoking existing product-owned journeys without inventing scenarios.

- **Entrypoints:** soda-acceptance exec|native|vm|probe-ssh|report; soda-test-vm; Explicitly selected tests/installed drivers.
- **Owned data:** Pinned SSH/VM fixture identity; Private evidence files, observations and retained hashes; Installed host/deployment/service observations bound to candidate.
- **Authority:** Selected checks determine effects; VM/disk/provider mutations require their own grants; x86_64 native evidence; source tests/emulation are different proof classes.
- **Dependencies:** [D04](#d04-authenticated-installation-media); [D05](#d05-artifact-verification); [D11](#d11-host-installation-and-payload-application); KVM/QEMU; Pinned SSH; Product-owned installed journey drivers.
- **Source files:** [rust/soda-acceptance/src/driver.rs:1](../../../../rust/soda-acceptance/src/driver.rs#L1); [rust/soda-acceptance/src/evidence.rs:226](../../../../rust/soda-acceptance/src/evidence.rs#L226); [docs/architecture/release.md:37](../../../architecture/release.md#L37); [docs/development/native-support.md:195](../../native-support.md#L195).
- **Tests:** [rust/soda-acceptance/src/evidence.rs:839](../../../../rust/soda-acceptance/src/evidence.rs#L839) — Unit filesystem fixtures: failed/leaking/occupied evidence cannot publish success-shaped observations; [tests/installed/host.sh:8](../../../../tests/installed/host.sh#L8) — Native driver source: host/SELinux/deployment/unit observations; no execution or update/recovery qualification claim.
- **Unclear boundaries:** Driver plumbing and product tests are separate owners. The current controller has no P9/P10 production qualification path; complete admission wiring is not asserted.
- **Evidence status:** Existing support drivers; production qualification path unavailable and installed evidence not produced here.

## D07 Release admission and preparation

Validate trust and candidate/media/qualification bindings before creating local release or channel OCI documents.

- **Entrypoints:** release-deliver prepare/admission/document APIs; no current operator delivery CLI.
- **Owned data:** Public integrity-controlled trust; Candidate/media records and qualification evidence; Fresh local release/channel OCI layouts and digest references; Proposed policy/registries configuration.
- **Authority:** Candidate qualification and exact identity admission precede delivery; Preparation neither signs/publishes nor installs proposed host policy.
- **Dependencies:** [D05](#d05-artifact-verification); [D06](#d06-installed-qualification); release-deliver OCI/document verification.
- **Source files:** [rust/soda-release-deliver/src/prepare.rs:78](../../../../rust/soda-release-deliver/src/prepare.rs#L78); [rust/soda-release-deliver/src/admission.rs:133](../../../../rust/soda-release-deliver/src/admission.rs#L133); [docs/development/native-support.md:215](../../native-support.md#L215).
- **Tests:** [rust/soda-release-deliver/tests/oracle.rs:237](../../../../rust/soda-release-deliver/tests/oracle.rs#L237) — Golden oracle: candidate admission state and stable channel's native-evidence requirement; [rust/soda-release-deliver/src/prepare.rs:277](../../../../rust/soda-release-deliver/src/prepare.rs#L277) — Unit: invalid trust refused before candidate/media work.
- **Unclear boundaries:** Retired Go delivery worker is not an active ideal entrypoint. Proposed policy remains output, not runtime installation authority; production invocation is unresolved.
- **Evidence status:** Current Rust library behavior; no operator delivery entrypoint or completed production release path.

## D08 Signing custody

Admit an exact digest under protected release authority, sign a private local snapshot and verify the resulting signed directory.

- **Entrypoints:** release-deliver native::sign; no current operator delivery CLI.
- **Owned data:** Private permit, signer key/passphrase files; Integrity-controlled trust and exact repository/digest; Fresh signing snapshot/signatures and verification receipt.
- **Authority:** Protected release custody separated from build/test identities; Permit binds repository, digest and expiry; secrets use restricted files; Signing alone performs no registry publication.
- **Dependencies:** [D07](#d07-release-admission-and-preparation); Native Sigstore/skopeo primitives; Restricted signer storage.
- **Source files:** [rust/soda-release-deliver/src/native.rs:551](../../../../rust/soda-release-deliver/src/native.rs#L551); [docs/development/native-support.md:240](../../native-support.md#L240); [docs/development/native-support.md:252](../../native-support.md#L252).
- **Tests:** [rust/soda-release-deliver/src/native.rs:581](../../../../rust/soda-release-deliver/src/native.rs#L581) — Mock runner: locked skopeo version requirement; [rust/soda-release-deliver/src/native.rs:596](../../../../rust/soda-release-deliver/src/native.rs#L596) — Unit filesystem: absolute/restricted secret-file admission; not real cryptographic signing.
- **Unclear boundaries:** Fixture media signing in D04 has separate authority. Permit production and signer deployment are not established by library/source tests.
- **Evidence status:** Existing library operation; production custody/invocation and native signing proof unverified in this pass.

## D09 Publication and effect observation

Publish admitted signed payloads/promotions while preserving a durable record of uncertain and completed registry effects.

- **Entrypoints:** release-deliver publish and observe APIs; no current operator delivery CLI.
- **Owned data:** Private registry auth and protected permit; Per-repository ledger with idle/pending/complete phase and highwater state; Signed inputs, fresh outputs and public readback observations.
- **Authority:** Explicit registry-write/promotion authority for exact repository/digest/history; Observe existing ledger effects without replaying uploads; preserve uncertain/pending state.
- **Dependencies:** [D08](#d08-signing-custody); [D07](#d07-release-admission-and-preparation); Native skopeo/registry; Restricted publisher ledger.
- **Source files:** [rust/soda-release-deliver/src/publish.rs:17](../../../../rust/soda-release-deliver/src/publish.rs#L17); [rust/soda-release-deliver/src/publish.rs:454](../../../../rust/soda-release-deliver/src/publish.rs#L454); [docs/development/native-support.md:241](../../native-support.md#L241).
- **Tests:** [rust/soda-release-deliver/src/publish.rs:498](../../../../rust/soda-release-deliver/src/publish.rs#L498) — Unit: valid ledger phases/repositories and invalid-state refusals; [rust/soda-release-deliver/src/publish.rs:521](../../../../rust/soda-release-deliver/src/publish.rs#L521) — Unit: exact prior-channel versus explicit absent history rules; no registry writes.
- **Unclear boundaries:** Registry publication is distinct from build success, signing and appliance activation. Observing previous effects supplies no fresh installation approval.
- **Evidence status:** Existing Rust library/state model; production publication entrypoint unavailable and registry execution not verified.

## D10 Verified distribution consumption

Discover an authorized channel, verify and download complete release contents, and retain observed release-authority state.

- **Entrypoints:** release-deliver fetch/init_state APIs; no current operator delivery CLI.
- **Owned data:** Public trust and signed channel/release documents; Existing restricted durable fetch state/highwater; Fresh verified output and completeness receipt.
- **Authority:** Native signature verification and monotonic observed-authority admission; Explicit new-state bootstrap; no silent reset/adoption; Anonymous public commissioning target; fetch does not install or activate.
- **Dependencies:** [D09](#d09-publication-and-effect-observation); Native skopeo/OCI verification; Integrity-controlled public trust.
- **Source files:** [rust/soda-release-deliver/src/fetch.rs:42](../../../../rust/soda-release-deliver/src/fetch.rs#L42); [rust/soda-release-deliver/src/fetch.rs:397](../../../../rust/soda-release-deliver/src/fetch.rs#L397); [docs/development/native-support.md:243](../../native-support.md#L243).
- **Tests:** [rust/soda-release-deliver/src/fetch.rs:439](../../../../rust/soda-release-deliver/src/fetch.rs#L439) — Unit filesystem: exclusive state lock and persisted state update; [rust/soda-release-deliver/tests/oracle.rs:664](../../../../rust/soda-release-deliver/tests/oracle.rs#L664) — Fake runner: malformed fetched manifest/config causes refusal; no live registry proof.
- **Unclear boundaries:** Download receipt is distinct from D11 disk/payload mutation and O04 native update. Client/operator invocation remains unresolved without a delivery CLI.
- **Evidence status:** Existing library consumption/state behavior; no live fetch or installation claim.

## D11 Host installation and payload application

Perform explicit native disk installation and import validated local candidate images into ordinary Podman storage on the installed appliance.

- **Entrypoints:** soda-install disk|configure; enrollment verbs hand off separate enrollment responsibilities; soda-image-import.service -> /usr/libexec/soda/soda-image-import.
- **Owned data:** Verified media identity, destination Ignition and chosen disk; Installation-attempt marker; /usr/share/soda/release.json and /usr/share/soda/images; Installed stamp and ordinary Podman image state.
- **Authority:** Explicit native root/console operation; disk action restricted to live CoreOS; Whole local layout validated before imports; qualified upgrade paths are currently refused.
- **Dependencies:** [D04](#d04-authenticated-installation-media); [D05](#d05-artifact-verification); CoreOS installer; Podman; [O01](operator-administration.md#o01-first-boot-database-provisioning); [O02](operator-administration.md#o02-operator-identity-bootstrap); [N01](networking.md#n01-private-origins-tls-and-activation).
- **Source files:** [rust/soda-install/src/main.rs:1](../../../../rust/soda-install/src/main.rs#L1); [rust/soda-install/src/execute.rs:319](../../../../rust/soda-install/src/execute.rs#L319); [appliance/host-image/soda-image-import.service:8](../../../../appliance/host-image/soda-image-import.service#L8); [rust/soda-image-import/src/main.rs:654](../../../../rust/soda-image-import/src/main.rs#L654).
- **Tests:** [rust/soda-install/src/execute.rs:442](../../../../rust/soda-install/src/execute.rs#L442) — Fake runner: hotplug/cancellation/marker gates and one CoreOS install attempt; [rust/soda-image-import/src/main.rs:2004](../../../../rust/soda-image-import/src/main.rs#L2004) — Local fixture: invalid layout causes zero Podman imports.
- **Unclear boundaries:** Console installation and boot-time image import are distinct lifecycle seams inside this slice. HTTPS activation is N01; ordinary host updating is O04, not implied by payload import.
- **Evidence status:** Existing installed command/service paths; tests do not establish native installation/update qualification.


