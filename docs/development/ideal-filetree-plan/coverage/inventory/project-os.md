# Tracked files: project-os

[Inventory index](README.md) · [Coverage snapshot and limits](../README.md).

## Coverage inventory: project-os

R02 reconciliation @HEAD `406d5ac7` (post-C09): all rows rebound
`project-os/*`→`system/project/*` (every destination existence-verified;
old root verified absent). Audit baseline paths retained in this note.

| Current tracked path | Kind / lifecycle | Slice mapping |
| --- | --- | --- |
| [system/project/Containerfile](../../../../../system/project/Containerfile) | configuration / active | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution), [I09](../../slices/identity-brokering.md#i09-provider-execution-integration), [P02](../../slices/projects.md#p02-profile-and-runtime-readiness), [P03](../../slices/projects.md#p03-human-membership-and-accounts), [P06](../../slices/projects.md#p06-factory-role-accounts), [P10](../../slices/projects.md#p10-shared-tools-and-packages), [P11](../../slices/projects.md#p11-nested-services-and-volumes), [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) — [interval map](../maps/project-system-definitions.md#coverage-c74a47de5bc0) |
| [system/project/licenses/tea-LICENSE](../../../../../system/project/licenses/tea-LICENSE) | asset / active | [H05](../../slices/shared-supporting-slices.md#h05-branding-avatars-and-attribution) |
| [system/project/muse-release.json](../../../../../system/project/muse-release.json) | configuration / active | [D02](../../slices/release-and-installation.md#d02-pinned-input-acquisition) |
| [system/project/rootfs/etc/containers/containers.conf](../../../../../system/project/rootfs/etc/containers/containers.conf) | configuration / active | [P11](../../slices/projects.md#p11-nested-services-and-volumes) |
| [system/project/rootfs/etc/containers/storage.conf](../../../../../system/project/rootfs/etc/containers/storage.conf) | configuration / active | [P11](../../slices/projects.md#p11-nested-services-and-volumes) |
| [system/project/rootfs/etc/mise/config.toml](../../../../../system/project/rootfs/etc/mise/config.toml) | configuration / active | [P10](../../slices/projects.md#p10-shared-tools-and-packages) |
| [system/project/rootfs/etc/profile.d/soda-mise.sh](../../../../../system/project/rootfs/etc/profile.d/soda-mise.sh) | configuration / active | [P10](../../slices/projects.md#p10-shared-tools-and-packages) |
| [system/project/rootfs/etc/profile.d/soda-podman.sh](../../../../../system/project/rootfs/etc/profile.d/soda-podman.sh) | configuration / active | [P11](../../slices/projects.md#p11-nested-services-and-volumes) |
| [system/project/rootfs/etc/ssh/sshd_config.d/10-soda.conf](../../../../../system/project/rootfs/etc/ssh/sshd_config.d/10-soda.conf) | configuration / active | [P04](../../slices/projects.md#p04-development-ssh-access) |
| [system/project/rootfs/etc/sudoers.d/soda-project](../../../../../system/project/rootfs/etc/sudoers.d/soda-project) | configuration / active | [P03](../../slices/projects.md#p03-human-membership-and-accounts) |
| [system/project/rootfs/etc/systemd/system/soda-podman.service](../../../../../system/project/rootfs/etc/systemd/system/soda-podman.service) | configuration / active | [P11](../../slices/projects.md#p11-nested-services-and-volumes) |
| [system/project/rootfs/etc/systemd/system/soda-podman.socket](../../../../../system/project/rootfs/etc/systemd/system/soda-podman.socket) | configuration / active | [P11](../../slices/projects.md#p11-nested-services-and-volumes) |
| [system/project/rootfs/etc/systemd/system/soda-project-init.service](../../../../../system/project/rootfs/etc/systemd/system/soda-project-init.service) | configuration / active | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness) |
| [system/project/rootfs/etc/yum.repos.d/gh-cli.repo](../../../../../system/project/rootfs/etc/yum.repos.d/gh-cli.repo) | configuration / active | [P10](../../slices/projects.md#p10-shared-tools-and-packages) |
| [system/project/rootfs/usr/libexec/soda/project-init](../../../../../system/project/rootfs/usr/libexec/soda/project-init) | configuration / active | [P02](../../slices/projects.md#p02-profile-and-runtime-readiness), [P03](../../slices/projects.md#p03-human-membership-and-accounts), [P04](../../slices/projects.md#p04-development-ssh-access), [P10](../../slices/projects.md#p10-shared-tools-and-packages), [P11](../../slices/projects.md#p11-nested-services-and-volumes), [S04](../../slices/spaces-and-terminals.md#s04-human-terminal-lifecycle) — [interval map](../maps/project-system-definitions.md#coverage-a619746c8031) |

