# Install on premises

Install SodaOS on a computer you control or in a matching-architecture virtual machine using the release's ISO and provisioning instructions.

## Prepare the machine

Choose the x86-64 or AArch64 ISO matching the target, and
[verify the download](05-verify-downloads.md). Have:

- a target disk whose contents may be erased, plus independent backups of
  anything already on it;
- console access, a keyboard/display or VM console, and working network/DNS;
- CPU, memory and storage for retained projects, tools, databases and concurrent work;
- the operator's SSH public key and securely prepared initial host credentials;
- a non-overlapping private subnet for projects and a plan to route it to clients;
- the release's matching provisioning/deployment recipe and required trust material.

Installation uses the Fedora CoreOS provisioning model. It is network-assisted;
do not assume the ISO alone contains an offline dependency closure. Developers
are created later through Forgejo/Soda, not as human accounts on the host.

For a VM, choose matching CPU architecture, firmware and devices supported by
that release and VM platform. Keep console access after installation. A host-only
VM network is not automatically reachable from another computer. For disk-image
import instead, use [the QCOW2 guide](10-deploy-to-cloud.md).

## Prepare private provisioning

Follow the recipe accompanying the selected release to supply operator access
through native Butane/Ignition. Use only the operator's public SSH authentication
key, never their private key. Password hashes and any generated machine host
keys are sensitive; keep provisioning inputs/outputs in a restricted private
directory and out of Git, screenshots, shell tracing and shared logs.

[Butane](https://coreos.github.io/butane/) converts human-readable configuration
into Ignition. Follow [Fedora CoreOS provisioning](https://docs.fedoraproject.org/en-US/fedora-coreos/producing-ign/)
for its first-boot semantics. A generic cloud-init user file is not a substitute.
Keep the release-specific recipe with the verified media so their inputs agree.

## Write and boot the ISO

Use a raw disk-image writer to write the verified ISO to the intended removable
device. **Writing destroys that device's previous contents.** Check its identity
and capacity before confirming. In a VM, attach the ISO as optical media instead.

Boot the target from that media and retain the installation console. Follow the
release recipe and [CoreOS bare-metal installation guide](https://docs.fedoraproject.org/en-US/fedora-coreos/bare-metal/)
for the native installer and private provisioning delivery.

## Install deliberately

1. Verify the target architecture, disk identity and intended network.
2. Supply the prepared Ignition through the recipe's native installation path.
3. Review disk selection before confirming installation. **Installing can erase
   the selected disk permanently.** Do not select a disk based only on device order.
4. Wait for installation to complete, detach the media and boot the installed disk.
5. Use the operator console to inspect first-boot provisioning and required host
   extensions. Activate any pending deployment with the instructed explicit reboot.
6. Complete the release's matching Soda component installation, then
   [operator setup](25-operator-setup.md) for Forgejo, HTTPS and project routing.

A first-install command is not an upgrade or recovery command. If installation
reports existing or partial Soda state, preserve it and inspect the failure
rather than deleting its marker or rerunning disk installation over project data.

## Before inviting developers

Verify operator console/SSH access, trusted Soda/Forgejo HTTPS, separate protected
Cockpit access and the project's client route. Confirm storage capacity and your
backup plan. Use a normal account to complete [First connection](30-first-connection.md),
including real SSH to a project, not just a successful browser login.

WSL2 on x86-64 Windows is planned for a future release, with no WSL2 download.
This guide installs the full system on hardware or a VM.
