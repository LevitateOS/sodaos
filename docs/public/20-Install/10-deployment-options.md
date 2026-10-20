# Choose a deployment

SodaOS runs as an appliance installed on a dedicated x86_64 machine or compatible virtual machine, or as a privileged Docker runtime on a dedicated Linux host. Choose the model that fits who will administer the host and how clients will reach the private services.

Both options are intended for a private, trusted team. The appliance uses Fedora CoreOS as its host and offers a guided disk installer plus native host administration. Docker runs Soda's runtime inside a privileged outer container while the operator remains responsible for the Linux and Docker host. The outer container has substantial host authority; it is not a security boundary for mutually untrusted teams.

For an appliance, prepare a machine or VM with x86_64 CPU architecture, a console you can use during installation, a persistent destination disk, and network access to the release assets and registries during installation and first startup. You need the ability to select and erase that disk, set the native root password, and later reach the private browser address from an operator client. A VM also needs persistent virtual storage and console or equivalent recovery access. Exact hardware sizing and firmware combinations are not specified here; use a system that can run the selected release and retain its Project data.

Choose Docker when you already operate a dedicated x86_64 Linux server with rootful Docker Engine, Compose, cgroups v2 and host networking available. The host must allow IPv4 forwarding and provide persistent storage for `/etc`, `/var` and `/home`. You need administrative access to that host and a plan for protecting its runtime data. Rootless Docker and Docker Desktop are outside this deployment model.

In either model, select a private browser address reachable by intended clients and a separate Project subnet that does not conflict with LAN, VPN or container routes. Browser access, Git's advertised addresses and direct Project network access have separate settings. The [private networking and TLS guide](../30-Set-up/20-private-networking-and-tls.md) covers their setup and client verification.

Continue with [verify the release download](20-verify-downloads.md), then follow [on-premises installation](30-install-on-premises.md) for hardware or a VM. For a Docker deployment, use [Install with Docker](50-docker.md). Cloud VM operators should first prepare the VM as described in [Install in a VM or cloud](40-deploy-to-cloud.md).
