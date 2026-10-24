# Native build and installation

How to prepare builders, produce candidates, install components and activate a
software factory appliance with supporting persistent human projects.
Release model: [Release architecture](../architecture/release.md).
Tool effects: [Native support](../development/native-support.md).
Media details: [Installation media](media.md).

For Docker Engine deployment, follow [Install a released image](docker.md#install-a-released-image)
to download the versioned Compose file, pin its GHCR digest and configure the runtime.

After installation and private operator setup, the [public Factory journey](../public/20-Factory/10-software-factory.md)
explains how authorized issues progress to verified results. The
[factory operator interface](../reference/factory.md) documents the existing
command surface.

The [deployment capacity guide](../public/60-Install/10-deployment-options.md#select-capacity-for-your-workload)
owns the measured basic and small-team allocations and storage planning.

The October 24 development measurement used a dedicated appliance VM with four
vCPUs, 8 GiB RAM, a 96 GiB disk and no swap. A one-hour window contained 720
five-second samples with no collection failures. Two native Builder units ran
simultaneously across separate Projects and distinct bound Codex leases for
235 seconds, alongside a real browser terminal and a retained private Canlang
application. Peak sampled host memory in use was 3.58 GiB, with at least 4.16 GiB
available; peak five-second average CPU use was 2.38 cores. These sampled values
do not bound shorter spikes or larger workloads.

The retained filesystem reached 13.76 GiB used with 81.68 GiB available. The
application's sampled cgroup usage reached 551 MiB; the two active Projects
reached 2.33 GiB and 704 MiB respectively, including their processes and cache.
Container PID 1 RSS does not include descendant workers, and cgroup peaks must
not be summed as independent host requirements. The 96 GiB disk allocation
includes headroom rather than a measured minimum installation footprint.

This measurement used the T5 installed payload with separately measured Go,
native Host and Project-maintenance development components. It establishes the
development sizing procedure; the complete frozen release must qualify the
stated profiles through R52. The Docker Factory window used the same four-vCPU,
8 GiB host allocation and one active issue; its application and terminal probes
ran in a separate window. Their combined Docker workload and Docker two-issue
support remain to be measured.

Commands that write disks, mutate providers, publish artifacts or destroy fixtures
require explicit task approval. This guide is not that approval.

## Install from release media

Use the selected release's verified x86-64 installation media for a PC or VM.
Write it to the intended USB device or attach it as VM optical media, then boot
its console. The installer authenticates the matching host release and renders
the machine's provisioning from the reviewed choices. The installed host downloads
its six pinned application images on first boot, then Fountain acquires the
required Soda package. Keep internet access available through startup. Tailnet
remains absent until an administrator enables it. End users do not prepare
Butane/Ignition, build application images or generate password hashes.

The guided `soda-install disk` flow reviews network settings, exact destination
disk, hostname, native root password and Project subnet before an explicit disk
erase confirmation. Check the subnet against LAN/VPN routes; the installer also
rejects overlap with Soda's reserved networks. Preserve independent backups and
console access. After successful installation, remove the media, reboot and log
in with the chosen root password.

Run `/usr/libexec/soda/soda-install configure` from a root terminal on the installed
host. It guides private-address selection, native Forgejo bootstrap and local HTTPS
activation. Root password SSH is available after the guided installation; operator
key enrollment is optional. Complete client certificate trust and a real browser
Project terminal as described in [Operator setup](operator-setup.md).

Use [the public installation guide](../public/60-Install/40-install-on-premises.md)
for the complete end-user path. The builder instructions below are for release
developers, not prerequisites for installing a downloaded release.

## Prepare the builder

Use a native `x86_64` builder under the
[platform scope](../architecture/release.md#architectures). Install the installed Go,
Rust and Bun toolchains and native Podman. From the repository root:

```sh
bun install --frozen-lockfile
```

Produce a candidate with the admitted `soda-build` controller (see
[native support](../development/native-support.md)). Verify without rebuilding:

```sh
bash scripts/check-native.sh ARCH /ABS/PATH/TO/stage
```

Use an exact committed Soda revision and a fresh output directory per attempt.
Select the independent Fountain release metadata explicitly. Pass its matching
source checkout with `--forgejo-source` for SDK compatibility and source binding;
Soda consumes the released Fountain image and adds its presentation layer.
Fountain acquires and verifies the required Soda package before core readiness.
Follow the admitted inputs and commands in the native support guide.

## Install developer candidates

For an admitted developer candidate on an existing host, use its component install
path for application images, host content and Forgejo customization. This is
separate from the release-media disk flow and its first-start component acquisition.
Prefer the same immutable candidate for install and later update.

Existing-state maintenance updates affected components without replaying first
install as a service upgrade and without using `--rm` / `--replace` as repair
against preserved project roots.

## Reachability and activation

1. Choose a private appliance address the intended client can reach.
2. Complete guided configure, or the deliberate setup/activate path for custom
   configuration in [Operator setup](operator-setup.md).
3. Trust the appliance public root certificate on intended clients when using
   local TLS, then verify actual browser login and a Project terminal.
4. Establish a separate client route to Project addresses when direct SSH or
   native service access is required.

The dashboard mounts only its config and credential files, the host and
identity helper sockets, its private data and the shared extension IPC
directory. Forgejo joins only that IPC directory through supplemental group
2100. Keep SELinux enabled so the private mounts retain separate labels.

Browser origins, Git advertisement and project routing are separate configuration
facts.

## Local CA fingerprint admission

The installer reads the public local root from one regular file without following
symlinks, with a 16 KiB input limit. It accepts one `CERTIFICATE` PEM envelope,
standard padded Base64 and surrounding whitespace. It refuses private keys,
multiple certificates and additional content.

Before displaying the SHA-256 fingerprint of the original DER, it requires
canonical DER, a positive nonzero serial using at most 20 INTEGER content octets,
matching signature identifiers, CA BasicConstraints, and certificate-signing permission
when KeyUsage is present. It refuses duplicate extensions and critical extensions
other than BasicConstraints and KeyUsage. Supported RSA, named-curve ECDSA and
Ed25519 keys must have valid algorithm parameters; the self-signature must verify
with the supported SHA-2 or Ed25519 profile.

This admission identifies the public root for explicit client trust. The operator
still decides whether to trust it; the display operation performs no validity-window,
chain or hostname verification. Invalid input prevents ready-to-open trust guidance.

## Operator services

Stock Cockpit listens on all interfaces with the operator credential. Project
units follow the appliance service definitions under `system/host/services/`.
The optional Tailnet runtime starts only after administrator enablement and
authenticated reconciliation; disabling it stops its host and Project companions. Local CI execution is
unavailable. Preserve credentials,
project state, backups and failed evidence unless cleanup is explicitly approved.
