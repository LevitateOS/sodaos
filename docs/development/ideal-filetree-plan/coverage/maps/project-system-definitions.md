# Project system definitions

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-c74a47de5bc0"></a>
<a id="coverage-2db3366af448"></a>

## [system/project/Containerfile](../../../../../system/project/Containerfile)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–7, 37, 42–50; current configuration/service block | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Declared native Project base/profile and creation identity; Native Project rootfs assembly; Final native userspace/profile/init readiness admission — Current named units/source consumers; retained normalized source evidence records each selector |
| 8–22, 24–30, 35; current configuration/service block | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Native development foundation and signed GH CLI package input; Checksum-verified native shared mise runtime package installation; Native developer Tea CLI — Current named units/source consumers; retained normalized source evidence records each selector |
| 23; current configuration/service block | [P11](../../slices/projects.md#p11-nested-services-and-volumes) | retained | Pinned Project-local native Podman Compose tool — system/project/Containerfile:23-23; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 31–34; current configuration/service block | [I09](../../slices/identity-brokering.md#i09-provider-execution-integration) | retained | Public native Muse wrapper/Compose registration/upstream execution binaries — system/project/Containerfile:31-34; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 36; current configuration/service block | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) | retained | Tea license attribution — system/project/Containerfile:36-36; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 38–39; current configuration/service block | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Verified managed terminal helper staging — system/project/Containerfile:38-39; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 40; current configuration/service block | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Project-local human account helper staging — system/project/Containerfile:40-40; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 41; current configuration/service block | [P06](../../slices/projects.md#p06-factory-role-accounts) | retained | Project-local factory-role helper staging — system/project/Containerfile:41-41; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |

<a id="coverage-a619746c8031"></a>
<a id="coverage-514d0954f107"></a>

## [system/project/rootfs/usr/libexec/soda/project-init](../../../../../system/project/rootfs/usr/libexec/soda/project-init)

Current configuration/service body read and reconciled to line-bounded responsibility units

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–3, 32; current configuration/service block | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) | retained | Native persistent Project init admission; Native ready marker — system/project/rootfs/usr/libexec/soda/project-init:1-3; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/project/rootfs/usr/libexec/soda/project-init:32-32; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 4–23, 26; current configuration/service block | [P11](../../slices/projects.md#p11-nested-services-and-volumes) | retained | Nested engine native networking/sysctl directory setup; Project-local Podman directories — system/project/rootfs/usr/libexec/soda/project-init:4-23; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/project/rootfs/usr/libexec/soda/project-init:26-26; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 24–25, 30; current configuration/service block | [P10](../../slices/projects.md#p10-shared-tools-and-packages) | retained | Shared development group/files; Shared mise tools/cache directories — system/project/rootfs/usr/libexec/soda/project-init:24-25; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/project/rootfs/usr/libexec/soda/project-init:30-30; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 27, 31; current configuration/service block | [P04](../../slices/projects.md#p04-development-ssh-access) | retained | Managed developer key directory; Per-Project native SSH host key generation — system/project/rootfs/usr/libexec/soda/project-init:27-27; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file; system/project/rootfs/usr/libexec/soda/project-init:31-31; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 28; current configuration/service block | [P03](../../slices/projects.md#p03-human-membership-and-accounts) | retained | Native identity/account markers — system/project/rootfs/usr/libexec/soda/project-init:28-28; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
| 29; current configuration/service block | [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) | retained | Managed terminal runtime directory — system/project/rootfs/usr/libexec/soda/project-init:29-29; current body read; consumer/selector is the service, rootfs copy or extension manifest named by the file |
