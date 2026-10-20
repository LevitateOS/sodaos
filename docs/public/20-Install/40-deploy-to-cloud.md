# Install in a VM or cloud

Prepare a private x86_64 VM with persistent storage, console access and network reachability; install SodaOS from its release ISO. The VM follows the same disk review and erase-confirmation flow as a physical machine.

This ISO path supports a compatible x86_64 cloud or private-cloud VM using the same reviewed disk-install flow as a physical machine. Provider network, storage and console settings remain the VM operator's responsibility.

## Prepare the virtual machine

Create a VM whose virtual CPU architecture is x86_64 and attach a persistent disk dedicated to SodaOS. Provide enough capacity for the operating system, retained Projects, application data and backups; exact minimum sizing is not specified here. Arrange console access that lets you select the boot device, follow the text installer and recover if startup needs attention. Attach the verified SodaOS ISO as virtual optical media, or use the VM platform's equivalent supported ISO boot method.

Configure a private network interface with an address your operator client can reach. Ensure the VM can reach the release assets and registries required during installation and first startup. Plan a Project subnet that does not overlap the VM's LAN, VPN or other routed networks. Keep the virtual disk persistent across ordinary VM stops and host maintenance, and arrange independent backups for SodaOS data.

## Install from ISO

Boot the VM from the ISO and follow [on-premises installation](30-install-on-premises.md). Review the exact virtual destination disk carefully: the explicit erase confirmation destroys its existing contents. After the installed first boot, complete [operator setup](../30-Set-up/10-operator-setup.md) and [private networking and TLS](../30-Set-up/20-private-networking-and-tls.md).

## Provider responsibilities

SodaOS does not provide provider-specific machine images or an image-import workflow here. The installation path is the same ISO disk installation on hardware or a VM; cloud network rules, persistent-disk configuration and console access remain the VM operator's responsibility.
