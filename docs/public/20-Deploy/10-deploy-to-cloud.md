# Deploy in a VM or cloud

Use the release installation media in an x86-64 virtual machine, with a separate destination disk, console access and private client connectivity.

A VM follows the same [guided installation](20-install-on-premises.md) as a PC.
You do not need to build application images or prepare an Ignition document for
that path. Keep capacity for retained Projects, Forgejo data and intended factory
work. Continue with [Operator setup](25-operator-setup.md) after first boot.

## Prepare a virtual machine

[Verify the installation media](05-verify-downloads.md). Choose supported x86-64
virtual hardware and firmware, create the destination disk, attach the media as
an optical disk and retain the platform's console.

Choose networking that lets the intended client reach the appliance's private
address. With NAT, use the VM platform's deliberate private forwarding or access
path; a guest address alone does not prove client access. A host-only network
may be appropriate for one laptop, but it does not connect other team members.

Boot the media, follow the installer and detach it after successful installation.
Do not start a second instance by copying an already configured disk containing
private machine keys, accounts or enrolled provider state.

## Choose a cloud deployment path

A cloud platform must provide a supported boot mechanism, independent console
access and a private network path for the operator. Use only the deployment path
and hardware documented for the selected release and platform. A provider's
ability to import a generic disk does not establish that it supports the release's
boot or provisioning requirements.

If that release publishes a platform disk image and matching import instructions,
verify and use them together. Disk-image import is a separate path from the guided
media installer; do not substitute generic cloud-init for CoreOS Ignition or
assemble another image from application container archives.

## Keep public ingress closed

Set the provider's network policy before first boot. Keep SSH, Cockpit, Soda,
Forgejo and Project services on their intended private network. Preserve console
access and any required outbound DNS, dependency or Tailscale connectivity.

Provider policy, host listeners/firewall and Tailnet access rules are independent
boundaries. Do not open public SSH or publish the unfinished Forgejo installer to
work around an unavailable private route.

For remote access through Tailscale, establish the appliance and client's approved
Tailnet membership before [private browser setup](25-operator-setup.md).
Project subnet routing is separate; configure it only for the direct Project
access your clients need. See [Tailscale](../30-Use-Soda/40-tailscale.md).

## Verify the installed result

Confirm console access, usable disk capacity, trusted private browser and SSH
identity, and a real Project Join/browser terminal before storing team data.
Verify direct Project SSH separately when required. Keep the console available
and establish your backup procedure.

For a failed boot, inspect native console diagnostics, selected firmware/devices,
media identity and network path. Preserve partial disks and data; an import or
installation retry must not overwrite a machine containing work.
