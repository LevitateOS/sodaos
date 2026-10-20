# Install on premises

Use the SodaOS installation media to install the appliance on a dedicated x86_64 computer or a VM's selected virtual disk. The guided installer reviews the destination and configuration before it writes anything; after installation, the first boot starts the appliance and operator setup continues separately.

Before starting, verify the release media, arrange a usable console, connect the machine to the installation network, and identify the exact disk to use. Back up any data on that disk. The installation downloads its matching authenticated root filesystem from the same release, and the installed host needs network access while it starts and acquires its required application images and Soda package.

Write the verified ISO to a USB device or attach it as virtual optical media, then boot the target from it. At the console, follow the guided `soda-install disk` flow. Review the network settings, hostname, root password and Project subnet. Choose a private subnet that does not overlap your LAN, VPN or reserved Soda networks. When the installer displays the destination, check the device identity and capacity against the machine's disk list.

**The next confirmation erases the selected destination disk.** Proceed only when the displayed disk is the one you intend to dedicate to SodaOS and its contents are backed up or disposable. If the device is wrong or you are unsure, go back or stop without confirming. The installer writes only after this explicit confirmation and reports installation failure rather than silently treating a partial write as success.

When installation reports completion, remove or detach the media and reboot. Log in at the installed host console with the root password you selected. The first boot obtains the release's required runtime components; keep the network available and allow startup to finish. If startup fails, retain the machine state and console access for diagnosis rather than repeating a disk install over it.

Continue with [operator setup](../30-Set-up/10-operator-setup.md) to create the configured Soda operator and activate private browser access. Then follow [private networking and TLS](../30-Set-up/20-private-networking-and-tls.md) to trust and verify access from a client. The installation itself ends at the first installed boot.
