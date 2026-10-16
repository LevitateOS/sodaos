# Current Docker release responsibilities

## Installer integration scope

Bounded source pin: `92069938869499a671a52e84ab03c639bb0b09d9`, inspected on
2026-10-16 against parent `730fc20526ef7c65c5b6abe1445297dc91d3f5c0`.
This integration changes **18 paths: three added, fifteen modified**, with no
rename or deletion. The committed source has **2,907 paths**; that membership
count does not advance the historical whole-tree/body/review pins.

The previously reconciled Docker range below remains at `76daa366`.
`76daa366..730fc205` has 358 changed-path entries still outside this bounded
review, including components-v4 and the thirteen canonical commits
`86659d5e..730fc205`. Reading an installer caller or fixture does not reconcile
those complete intervening implementations. Existing unrelated inventory and
review gaps remain open. ARM64/UTM is outside this task.

| Change | Exact path | Retained defining owner and inspected duty |
| --- | --- | --- |
| M | `docs/architecture/release.md` | **H06/D07** — Immutable installer inventory, promoted exact tag/source/digest marker and installer-inclusive native qualification contract. |
| M | `docs/development/release.md` | **H06/D06/D09** — Prepare/stage/download/qualification/promotion joins; five downloaded assets and separately retained installer execution transcript. |
| M | `docs/guides/docker.md` | **H06/D10/D11** — Version-bound entrypoint, prerequisites, user-owned setup, same-release rerun and retained-state refusal. |
| M | `docs/public/20-Deploy/22-docker.md` | **H06/D10** — Public availability notice and intended installation path; no unqualified runnable release URL. |
| M | `lib/soda-release-deliver/README.md` | **H06/D07** — Renderer markers, noncircular identity ownership, transcript admission and qualification marker interface. |
| M | `lib/soda-release-deliver/src/docker/mod.rs` | **D05/D07** — `INSTALLER_SOURCE` and `installer_template` select exact archived template through the existing confined source reader. |
| M | `lib/soda-release-deliver/src/docker/registry_tests.rs` | **D06/D07/D09** — Archived rendering/substitution refusal, GitHub asset verification, every-phase installer binding, transcript identity and promotion marker fixtures. |
| M | `lib/soda-release-deliver/src/docker/source.rs` | **D05** — `SOURCE_FILES` requires the installer; `read_inputs_from` reuses bounded regular archive-member admission for candidate and prepared source names. |
| M | `lib/soda-release-deliver/src/docker/tests.rs` | **D05/D07** — The shared source-archive fixture includes a renderable installer template; other fixture bodies retain their previous scope. |
| M | `lib/soda-release-deliver/src/publish/assets.rs` | **D07** — Docker manifest cardinality/required asset; prepare renders and hashes installer; inventory reconstructs source-bound bytes to reject rehashed substitution. |
| A | `lib/soda-release-deliver/src/publish/installer.rs` | **D07** — `render` substitutes validated tag/source and metadata hash/size into archived template; no downloads or runtime effects. |
| M | `lib/soda-release-deliver/src/publish/mod.rs` | **D09/D07** — Promotion emits the exact Docker marker after selected qualification; `qualify_docker` verifies all canonical downloaded assets with GitHub. |
| M | `lib/soda-release-deliver/src/publish/qualification.rs` | **D06/D07** — `DOWNLOADS`, `NativeDocker` installer/transcript identities, selected asset checks and retained installation-transcript binding; collection remains observation admission. |
| M | `scripts/check-docker-packaging.sh` | **H06** — Adds Bash syntax admission beside existing Dockerfile/Compose checks. |
| M | `scripts/check-push.sh` | **H06** — Required core profile executes installer command fixtures; Rust profile includes release-deliver library tests. |
| A | `system/docker/install-docker.sh` | **D10/D11** — Stream-safe complete-script entry, exact immutable release/hash/platform/host admission, private deployment commit, preserved runtime identity and delegated setup/activation. `check_network` consumes N02 host routes/reservations; `setup_browser` calls existing O02/N01 owners without defining their native policy. |
| M | `tests/build/README.md` | **H06** — Documents subprocess-fixture scope and required push-profile inclusion. |
| A | `tests/build/docker_installer_test.go` | **H06/D10/D11** — Executes actual streamed script with substituted external commands; real JSON/hash/file/permission/rerun checks, failure-before-mutation, retained volume identity and activated-service/HTTPS refusal. |

### Boundaries, checks and completion dimensions

`publish/installer.rs` owns rendering; `publish/assets.rs` owns rendered asset
hash/size and inventory. Docker metadata continues to bind Compose/image bytes,
without hashing its own installer. Qualification admits the downloaded installer
plus retained transcript; the promotion note marker binds tag/source/image and
is a consumer gate, not an independently collected installation observation.
The Bash adapter invokes Docker and existing Rust setup/activation executables;
there is no new daemon, crate, Go/Rust port or image payload owner.

Source/duty coverage is complete for this 18-path delta. The traced source join
is archived template → prepared immutable asset → downloaded consumer → retained
installation transcript → qualification → promotion marker. Full installed
workflow analysis/behavior remains pending. Root reports bounded code review and
93/93 delivery library tests, focused Go source/CI/installer tests (including
activated-state and cached-sudo cases), and Docker packaging checks passing;
this upkeep does not rerun or expand those receipts. Independent full-slice
A/C challenge remains at its earlier scope. Exact current targets are allocated
for the three additions, without endorsing a broader architecture revision.

[Concern inspection](../../decomposition/release-production.md#current-docker-release-concerns-2026-10-16),
[slices](../../slices/release-and-installation.md#current-docker-release-model-2026-10-16),
[tasks](../../implementation-tasks.md#current-docker-release-handoff-2026-10-16)
and [lanes](../../implementation-lanes.md#current-docker-release-handoff-2026-10-16)
retain public-image/runtime/Projects/nested-container/Factory/persistence proof
as separate gates. No install, build, VM, publication or qualification ran in this
planning refresh.

## Earlier reconciled Docker source scope

Scoped source reconciliation on 2026-10-14: `76daa366822e0a135df961d7c9c21713da016bd3`,
following merge `370767292d34dfe932b993de9c8e6c810ee9ea67` and its focused
CLI-fixture repair. The selected accumulated range `1d398279..76daa366` covers
**76 unique paths**, including all nine Docker release commits and four main-side
commits. Docker contributes **72 paths: 15 added, 57 modified**. The later CLI
repair changes an already-counted test, checking canonical checkout-root admission
with an isolated repository and no output creation; runtime code is unchanged.

The broader `68d82401..76daa366` range has 349 changed-path entries
(87 additions, 234 modifications, 26 renames and two deletions). Earlier Canlang,
Project support and frontend changes in that range remain outside this bounded
review, alongside previously recorded inventory gaps. The historical structural,
catalog, full-body and independent review pins are unchanged. The source tree has
**2,888 tracked paths**; this new H06 map is upkeep beyond that pin.

## Exact path and duty ledger

Multiple IDs identify distinct named duties, not shared authority. D01 owns
controller admission/completion; D02 acquisition; D03 production; D05 byte checks;
D06 native observations; D07 release admission; D09 publication effects; H06 CI/docs.

| Change | Exact path | Retained defining owner and responsibility |
| --- | --- | --- |
| M | `.github/workflows/ci.yml` | **H06/D07** — Required CI Docker packaging/admission checks; source checks do not qualify published runtime bytes. |
| M | `Cargo.lock` | **H06** — Existing Rust package graph wires shared reader/acceptance owners; no new crate or language owner. |
| M | `README.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `cmd/soda-project-maintain/src/release_operation.rs` | **P02/D11** — Native staging imports only digest blob refs; operation closes helper before unmount and rollback uses the actual booted deployment serial. |
| M | `cmd/soda-project-maintain/src/release_stage.rs` | **P02/D11** — Native staging imports only digest blob refs; operation closes helper before unmount and rollback uses the actual booted deployment serial. |
| M | `docs/README.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `docs/architecture/release.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `docs/development/native-support.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| D | `docs/development/project-runtime-upgrade-plan.md` | **H06** — Completed Project runtime upgrade plan retired; durable contracts remain in existing release/operator guides. |
| M | `docs/development/release.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `docs/guides/docker.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `docs/guides/installation.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `docs/guides/operator-setup.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `lib/release-inputs/src/reader.rs` | **D01/D02/D07** — Canonical completed build DTO and RPM/recipe readers; tools issues controller completion and delivery admits the exact build bytes. |
| A | `lib/release-inputs/src/reader/build.rs` | **D01/D02/D07** — Canonical completed build DTO and RPM/recipe readers; tools issues controller completion and delivery admits the exact build bytes. |
| A | `lib/release-inputs/src/reader/rpm.rs` | **D01/D02/D07** — Canonical completed build DTO and RPM/recipe readers; tools issues controller completion and delivery admits the exact build bytes. |
| M | `lib/release-inputs/src/reader/settings.rs` | **D01/D02/D07** — Canonical completed build DTO and RPM/recipe readers; tools issues controller completion and delivery admits the exact build bytes. |
| M | `lib/soda-release-build/README.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `lib/soda-release-build/src/lib.rs` | **D02/D03** — Frozen Fedora acquisition and production_docker consume the same admitted host config/payload, export OCI, capture packages/Compose and invoke delivery verification. |
| M | `lib/soda-release-build/src/production.rs` | **D02/D03** — Frozen Fedora acquisition and production_docker consume the same admitted host config/payload, export OCI, capture packages/Compose and invoke delivery verification. |
| M | `lib/soda-release-build/src/production/tests.rs` | **D03** — Production fixture checks same-candidate Docker packaging, frozen base/labels, no repeated compilation and refusal to record an invalid export. |
| A | `lib/soda-release-build/src/production_docker.rs` | **D02/D03** — Frozen Fedora acquisition and production_docker consume the same admitted host config/payload, export OCI, capture packages/Compose and invoke delivery verification. |
| M | `lib/soda-release-build/src/production_images.rs` | **D02/D03** — Frozen Fedora acquisition and production_docker consume the same admitted host config/payload, export OCI, capture packages/Compose and invoke delivery verification. |
| M | `lib/soda-release-build/src/production_inputs.rs` | **D02/D03** — Frozen Fedora acquisition and production_docker consume the same admitted host config/payload, export OCI, capture packages/Compose and invoke delivery verification. |
| M | `lib/soda-release-deliver/Cargo.toml` | **H06** — Existing Rust package graph wires shared reader/acceptance owners; no new crate or language owner. |
| M | `lib/soda-release-deliver/README.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `lib/soda-release-deliver/src/admission.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| M | `lib/soda-release-deliver/src/buildx/filesystem.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| M | `lib/soda-release-deliver/src/buildx/mod.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| M | `lib/soda-release-deliver/src/check.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| A | `lib/soda-release-deliver/src/docker/mod.rs` | **D05/D07** — record/verify compare original source archives, runtime files, embedded OCI blobs, frozen labels, packages and Compose against the same admitted candidate. |
| A | `lib/soda-release-deliver/src/docker/registry_tests.rs` | **D07/D09** — Scripted registry/GitHub and native-record fixtures check upload/readback recovery, qualification file/order/funding admission, exact-digest all-target promotion and status drift. |
| A | `lib/soda-release-deliver/src/docker/source.rs` | **D05/D07** — record/verify compare original source archives, runtime files, embedded OCI blobs, frozen labels, packages and Compose against the same admitted candidate. |
| A | `lib/soda-release-deliver/src/docker/tests.rs` | **D05/D07** — Real tar/OCI fixture checks source revision, candidate/runtime/embedded-blob/package/Compose bindings and production-only publication admission. |
| M | `lib/soda-release-deliver/src/lib.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| A | `lib/soda-release-deliver/src/model/docker.rs` | **D05/D07** — Distribution/BuildPurpose select outputs; DockerMetadata and FileIdentity bind source, candidate, payload, base, image/config/archive and Compose; production purpose constrains publication. |
| M | `lib/soda-release-deliver/src/model/mod.rs` | **D05/D07** — Distribution/BuildPurpose select outputs; DockerMetadata and FileIdentity bind source, candidate, payload, base, image/config/archive and Compose; production purpose constrains publication. |
| M | `lib/soda-release-deliver/src/oci/layers.rs` | **D05** — One open-file OCI scanner exposes verified labels/blob sizes/hashes and selected content; source-derived member limits retain aggregate budgets, overlay rules and EOF/trailer checks. |
| M | `lib/soda-release-deliver/src/oci/mod.rs` | **D05** — One open-file OCI scanner exposes verified labels/blob sizes/hashes and selected content; source-derived member limits retain aggregate budgets, overlay rules and EOF/trailer checks. |
| M | `lib/soda-release-deliver/src/oci/schema.rs` | **D05** — One open-file OCI scanner exposes verified labels/blob sizes/hashes and selected content; source-derived member limits retain aggregate budgets, overlay rules and EOF/trailer checks. |
| M | `lib/soda-release-deliver/src/prepare.rs` | **D05/D07** — Existing delivery admission, confined open-file access and canonical content/model exports serve Docker verification; no second OCI implementation. |
| M | `lib/soda-release-deliver/src/publish/assets.rs` | **D07** — Manifest selects appliance/docker/all; prepare requires exact completed production build, exports selected assets, and inventory/load rehash checksums and actual bytes. |
| M | `lib/soda-release-deliver/src/publish/mod.rs` | **D07/D09** — Exact main-push CI admission, selected immutable asset staging, downloaded appliance qualification, all-target promotion and read-only status/pointer drift observation. |
| A | `lib/soda-release-deliver/src/publish/qualification.rs` | **D06/D07** — Distinct installation/persistence/Factory records; collect rechecks downloaded and retained execution files, funded Factory, ordered timestamps; atomic stamp/verify binds original evidence. |
| A | `lib/soda-release-deliver/src/publish/registry.rs` | **D09** — Skopeo performs one digest-preserving version upload, anonymous observation/readback recovery and exact verified-digest promotion. |
| M | `lib/soda-release-deliver/src/publish/tests.rs` | **D07/D09** — Appliance fixture checks exact downloaded media, repeatable qualification, draft/transport status and recommendation drift. |
| M | `lib/soda-release-image/README.md` | **H06** — Current release, installation, native support and operator contracts/navigation; native proof remains distinct. |
| M | `lib/soda-release-image/src/build.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/build_source.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/foreign.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/host.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/packages.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/record.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-image/src/request.rs` | **D01/D03** — Source preflight and target admission select Docker after the common candidate; selected media packaging, package inventory and canonical build result remain at existing owners. |
| M | `lib/soda-release-tools/Cargo.toml` | **H06** — Existing Rust package graph wires shared reader/acceptance owners; no new crate or language owner. |
| M | `lib/soda-release-tools/src/build_cli.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/build_cli/tests.rs` | **D01** — Check selected target/purpose preflight and terminal controller completion/refusal. |
| M | `lib/soda-release-tools/src/build_spec.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/candidate_controller.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/lib.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/pipeline.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/release_cli.rs` | **D07/D09** — start/prepare/status/stage/verify-images/qualify/promote dispatch; require all selected target inputs before stamping either qualification. |
| A | `lib/soda-release-tools/src/release_start.rs` | **D01/D07** — choices validates explicit inputs; workflow_output confines fresh artifacts; run preflights source/tool/config/CI/offline inputs then delegates build and prepare, stopping before publication. |
| M | `lib/soda-release-tools/src/worker/execution.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/worker/runtime.rs` | **D01/D03** — Existing controller/worker/pipeline propagates production target/version and preflight; controller emits completed.json only after successful cleanup, terminal progress and interruption checks. |
| M | `lib/soda-release-tools/src/worker/tests.rs` | **D01** — Check admitted worker source visibility and distribution/version/preflight propagation. |
| M | `lib/soda-release-tools/tests/cli/soda_build.rs` | **D01** — Actual CLI fixture checks canonical checkout-root admission and no output creation; later correction isolates its Git fixture from dirty parent-checkout behavior. |
| A | `scripts/check-docker-packaging.sh` | **H06/D07** — Required CI Docker packaging/admission checks; source checks do not qualify published runtime bytes. |
| M | `system/docker/.dockerignore` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| M | `system/docker/Dockerfile` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| M | `system/docker/compose.yaml` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| A | `system/docker/network-scope.conf` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| A | `system/docker/project-scope.conf` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| M | `system/docker/soda-docker-init.service` | **D03/H01/D11** — Existing host payload packaged for Docker; Compose owns installed network/cgroup/volume/readiness choices, init owns atomic first configuration, scope drop-ins own shutdown ordering. |
| M | `system/host/services/soda-project-release-reconcile.service` | **P02/H01** — Existing release-reconcile oneshot retries boot readiness with a bounded start limit. |
| A | `tests/build/ci_admission_test.go` | **H06/D07** — Required CI Docker packaging/admission checks; source checks do not qualify published runtime bytes. |

## Boundary and evidence limits

`reader/build.rs` defines BuildResult/BuildCompletion; tools issues completion and
delivery admits it. `model/docker.rs` owns the Docker representation. Build uses
Podman and frozen inputs; registry credentials stay with the publisher's authfile.
The outer Docker deployment runs existing systemd/Podman applications and Projects;
there is no new VM, sidecar or language boundary. Qualification's reader consumes
native observations and does not collect the installation, browser/Project, funded
Factory or persistence journeys. That end-to-end collector remains pending.

This refresh performs source/duty and concern inspection only. It does not issue
new runtime/publication evidence or advance an old test receipt to the merged
revision. See [size concerns](../../decomposition/release-production.md#current-docker-release-concerns-2026-10-16),
[release slices](../../slices/release-and-installation.md#current-docker-release-model-2026-10-16),
[tasks](../../implementation-tasks.md#current-docker-release-handoff-2026-10-16)
and [schedule](../../implementation-lanes.md#current-docker-release-handoff-2026-10-16).
