# Project system definitions

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

R02 @HEAD `cd111721` (C09): Containerfile rebound with +8 bytes drift — STALE, bannered; project-init verified byte-identical, intervals kept.

<a id="coverage-c74a47de5bc0"></a>

## [system/project/Containerfile](../../../../../system/project/Containerfile)

> R02 STALE: intervals below reference the audited `project-os/Containerfile` blob; the moved file differs (import rebinds / drift) — pending re-audit.

Native Project image/config responsibility; no image build, activation or live readiness proof performed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1–7 | Declared native Project base/profile and creation identity |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 8–22 | Native development foundation and signed GH CLI package input |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 23 | Pinned Project-local native Podman Compose tool |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 24–30 | Checksum-verified native shared mise runtime package installation |
| [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) / active | 31–34 | Public native Muse wrapper/Compose registration/upstream execution binaries |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 35 | Native developer Tea CLI |
| [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) / active | 36 | Tea license attribution |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 37 | Native Project rootfs assembly |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 38–39 | Verified managed terminal helper staging |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 40 | Project-local human account helper staging |
| [P06](../../slices/projects.md#p06-factory-role-accounts) / active | 41 | Project-local factory-role helper staging |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 42–50 | Final native userspace/profile/init readiness admission |

<a id="coverage-a619746c8031"></a>

## [system/project/rootfs/usr/libexec/soda/project-init](../../../../../system/project/rootfs/usr/libexec/soda/project-init)

Native Project image/config responsibility; no image build, activation or live readiness proof performed.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 1–3 | Native persistent Project init admission |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 4–23 | Nested engine native networking/sysctl directory setup |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 24–25 | Shared development group/files |
| [P11](../../slices/projects.md#p11-nested-services-and-volumes) / active | 26 | Project-local Podman directories |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 27 | Managed developer key directory |
| [P03](../../slices/projects.md#p03-human-membership-and-accounts) / active | 28 | Native identity/account markers |
| [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) / active | 29 | Managed terminal runtime directory |
| [P10](../../slices/projects.md#p10-shared-tools-and-packages) / active | 30 | Shared mise tools/cache directories |
| [P04](../../slices/projects.md#p04-development-ssh-access) / active | 31 | Per-Project native SSH host key generation |
| [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) / active | 32 | Native ready marker |

