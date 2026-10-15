# Current Docker release responsibilities

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
revision. See [size concerns](../../decomposition/release-production.md#current-docker-release-concerns-2026-10-14),
[release slices](../../slices/release-and-installation.md#current-docker-release-model-2026-10-14),
[tasks](../../implementation-tasks.md#current-docker-release-handoff-2026-10-14)
and [schedule](../../implementation-lanes.md#current-docker-release-handoff-2026-10-14).
