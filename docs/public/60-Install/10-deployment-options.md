# Choose a deployment

Choose the appliance or Docker path that matches your host responsibilities and private network. Both provide SodaOS for a trusted team, with different host and lifecycle ownership.

Both deployment choices keep host administration and private client reachability with the operator; choose between them based on the host and runtime lifecycle you will maintain. SodaOS targets a trusted team, not public or mutually untrusted tenant hosting.

## Appliance requirements

The appliance uses Fedora CoreOS as its host and offers a guided disk installer plus native host administration. Prepare a machine or VM with x86_64 CPU architecture, a console you can use during installation, a persistent destination disk, and network access to release assets and registries during installation and first startup.

You need the ability to select and erase that disk, set the native root password, and later reach the private browser address from an operator client. A VM also needs persistent virtual storage and console or equivalent recovery access.

## Docker requirements

Choose Docker when you already operate a dedicated x86_64 Linux server with rootful Docker Engine, Compose, cgroups v2 and host networking available. The host must allow IPv4 forwarding and provide persistent storage for `/etc`, `/var` and `/home`. You need administrative access to that host and a plan for protecting its runtime data. Rootless Docker and Docker Desktop are outside this deployment model.

Docker runs Soda's runtime inside a privileged outer container while the operator remains responsible for the Linux and Docker host. The outer container has substantial host authority; it is not a security boundary for mutually untrusted teams. The [Docker guide](50-docker.md) lists the required host tools, ports and installer options.

## Select capacity for your workload

Use the following measured allocations to plan a dedicated host. Match the
workload and concurrency to the selected release's qualification material;
installation or a successful boot alone does not establish capacity.

| Profile | CPU | RAM | Persistent disk | Measured workload |
| --- | --- | --- | --- | --- |
| Basic appliance | 4 CPU cores or vCPUs | 8 GiB | 96 GiB | One active Factory issue, a browser terminal and one small private application. |
| Small-team appliance | 4 CPU cores or vCPUs | 8 GiB | 96 GiB | Two native Factory issues executing across separate Projects, a browser terminal and one small private application. |
| Docker starting allocation | 4 CPU cores or vCPUs | 8 GiB | 96 GiB | One native Factory issue; the application and browser terminal were measured in a separate window. |

These are the smallest allocations exercised for these workloads, rather than
measurements of every smaller configuration. The two-issue profile was measured with two
independently authorized Codex connections, each admitting one native Factory
execution at a time; Factory workers now run on Muse, whose leases are
concurrent. Raising appliance capacity does not increase a
connection's concurrency. The Docker windows do not establish the combined
Factory, terminal and application profile or two-issue concurrency. Match any
such Docker workload to its own release qualification.

The application in these profiles uses the normal 1 GiB memory, two-CPU and
128-process limits. Those limits cap an individual application; they do not
reserve memory or CPU for the rest of the host. Larger builds, repositories,
application data or additional concurrent services need additional capacity.
For Docker, the allocation also needs to cover Linux, Docker and nested
containers. Add capacity for unrelated workloads on the outer host.

Allow storage for repositories, Project roots and homes, application data and
container images. Before an update, check free space on the filesystem holding
`/var` and container storage. Keep the current deployment and selected target,
including downloaded archives and their unpacked images, until verification and
fallback are complete; compressed download sizes alone do not size that reserve.

Size a separate backup destination for the complete stopped recovery set. A raw
copy of a 96 GiB virtual disk needs 96 GiB plus every additional retained disk;
sparse-copy storage depends on actual allocation and the destination's support.
For Docker, measure and capture all three retained volumes and the matching
deployment files together. A backup on the same host consumes additional space
and does not protect against losing that host. Follow
[backups and restoration](../50-Operate/50-backups-and-restoration.md).

## Private network planning

In either model, select a private browser address reachable by intended clients and a separate Project subnet that does not conflict with LAN, VPN or container routes. Browser access, Git's advertised addresses and direct Project network access have separate settings. The [private networking and TLS guide](../40-Configure/50-private-networking-and-tls.md) covers their setup and client verification.

## Continue installation

Continue with [verify the release download](20-verify-downloads.md), then follow [on-premises installation](40-install-on-premises.md) for hardware or a VM. For a Docker deployment, use [Install with Docker](50-docker.md). Cloud VM operators should first prepare the VM as described in [Install in a VM or cloud](30-deploy-to-cloud.md).
