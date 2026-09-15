# Host capability strategy

Non-normative strategy for CoreOS host capabilities that strengthen Soda as a
dependable shared development appliance. Normative topology and trust remain in
[Architecture](../architecture/overview.md). Project runtime remains in
[Project OS](../reference/project-os.md).

## Three OS roles

| Role | Foundation | Responsibility |
| --- | --- | --- |
| Host OS | Fedora CoreOS | Boots the appliance, kernel, native services, container engines |
| Project OS | Rocky/Fedora profiles | Persistent developer accounts, tools, services, optional desktop |
| Runner OS | Dedicated Rocky headless job image | Fresh CI job environments, separate from projects |

Containers share the host kernel. Separate engines and authorities determine who
may manage them. Marketplace/appliance services, project nested workloads and CI
job containers are distinct lifetimes and credentials.

## Vocabulary

| Term | Meaning |
| --- | --- |
| Container image | Packaged filesystem and defaults used to create containers |
| Container | Runnable process environment sharing the host kernel |
| Container engine | Software such as Podman that manages images, containers, networks and storage |
| Pod | Group of containers sharing selected resources (often network); not a user DB |
| Volume / bind mount | Storage made available separately from a container's writable layer |
| Registry | Service for publishing and downloading images |
| systemd / Quadlet | Native service supervision; Quadlet integrates Podman resources into systemd |

## Recommended host investments

1. **Resource protection** for shared development and CI, leaving room for appliance
   services.
2. **Backups with tested restoration** for team work on the appliance.
3. **Machine health / first-boot report** through the existing operator console/CLI.
4. **Private connectivity that reaches developers' projects**, not only the host.
5. **Predictable maintenance** using native OS update mechanisms.
6. **Supported bootable storage policy** where Soda takes clearer ownership of
   first-boot disk layout without abandoning CoreOS foundations.

## Not automatic priorities

Features that merely look like “more OS” do not earn priority: desktop on the host,
arbitrary device brokering, or a second update engine parallel to CoreOS. KDE is a
**project** desktop, not a desktop installed on CoreOS.
