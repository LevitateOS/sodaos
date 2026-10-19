# Install on premises

Install SodaOS on an x86-64 PC or a virtual machine using the release installation media and its guided console.

The same appliance provides software factory work and persistent human projects.
The installer supplies its matching Soda components and prepares provisioning
from your choices. You do not build the software, assemble application images,
write Butane/Ignition or generate password hashes to install it.

## Prepare the machine

Choose the installation media from [SodaOS releases](https://github.com/LevitateOS/sodaos/releases/latest)
and [verify the download](05-verify-downloads.md). Have:

- an x86-64 target and a disk whose contents may be erased, with independent
  backups of anything already on it;
- keyboard/display or a VM console, plus a client computer for browser setup;
- storage, memory and CPU capacity for your retained projects and intended work;
- a private network the appliance and client can reach;
- a non-overlapping private subnet for Projects, separate from LAN/VPN routes.

Use the hardware and capacity requirements accompanying the selected release.
Keep console access throughout installation and first boot. A VM's host-only or
NAT network is not automatically reachable from another computer; choose its
network mode deliberately.

## Boot the installation media

For a PC, use a raw disk-image writer to write the verified media to the intended
USB device. **Writing erases that device's previous contents.** Check its identity
and capacity first. Boot the PC from that device.

For a VM, attach the verified media as an optical disk and create a separate
installation disk. Select supported x86-64 firmware and devices for the release.
Boot into the installation console.

## Review and install

The console checks the included installation payload before asking for machine
choices. If you need to start the console explicitly from the live root terminal,
run `soda-install disk`.

1. Review the network settings. DHCP is the default; use the offered network
   editor for a static address, gateway or DNS, then review the resulting addresses.
2. Select the destination by its displayed device, capacity, model and serial.
   Do not identify it only by device order.
3. Accept or change the hostname and enter the native root password twice.
   The password needs at least 12 characters and grants host administration.
4. Review the Project subnet. The default is `10.89.0.0/24`; check it against your
   LAN/VPN routes. Soda also reserves `10.88.0.0/16` and `10.90.0.0/24`.
5. Check the final review, then type its exact `ERASE` confirmation for the selected
   device. **Installation erases the destination disk.** Back, restart and cancel
   remain available before writing.
6. Wait for the completion screen, remove or detach the installation media and
   choose reboot. Boot the installed disk and log in as root with your password.

The installer prepares destination provisioning and copies the reviewed network
settings. Installation and first startup acquire the matching release payloads
and pinned application images; keep internet access available. No separate Soda
component-installation command is needed.

If writing fails or is interrupted, the disk may be partly written. Preserve the
failure and inspect the current boot before deciding what to do; do not blindly
retry disk installation over an existing machine or Project data.

## Open Soda in your browser

Continue with [Operator setup](25-operator-setup.md). It guides you through the
private address, native Forgejo administrator and client certificate trust.
Root password SSH is available after this installation; importing an operator
SSH key is optional. Developers use their own Forgejo accounts and Project
accounts, rather than the host's root account.

Before inviting developers, verify trusted browser access, a Project Join and a
real browser terminal. Verify Project SSH separately if you need it; displaying
an address does not establish a route from your client.
