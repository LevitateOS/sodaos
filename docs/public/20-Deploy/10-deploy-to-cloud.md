# Deploy to a cloud or VM

Deploy a SodaOS QCOW2 on matching virtual hardware, establish private operator access and route project environments to your clients.

## Prepare the deployment

[Verify the release's QCOW2](05-verify-downloads.md) and use its matching
provisioning recipe. Select an x86-64 or AArch64 instance of the same architecture,
with enough storage for persistent project roots, repositories and service data.
Keep a usable provider/VM console independent of SSH.

Use a trusted LAN for a local VM or private Tailnet connectivity for remote
access. For teams, Scaleway is the first cloud-provider path; you operate the
server and its data in your own account.

## Close public ingress before boot

Create the provider security policy before starting the server. Deny public
inbound access to SSH, Cockpit, Soda, Forgejo and development services. Preserve
required outbound DNS, provisioning, dependency and Tailscale access and native
stateful response traffic. Keep the provider console for bootstrap; do not open
public SSH to work around private-network setup.

The provider security group, host listeners/firewall and Tailnet access policy
are independent boundaries. A host firewall change does not authorize public
cloud exposure.

## Import the disk and supply provisioning

1. Verify the compressed image, decompress using its documented format and
   verify the resulting disk as instructed by the release.
2. Import the QCOW2 as the boot disk using the platform's native image/snapshot
   import. Preserve the downloaded original; use a separate instance disk.
3. Allocate sufficient capacity and select supported native firmware/devices.
4. Prepare the release's per-instance operator configuration and deliver its
   Ignition through the documented platform/boot mechanism **before first boot**.
5. Start the instance and use its console to inspect provisioning and disk capacity.

Soda uses Fedora CoreOS and Ignition, not the predecessor's cloud-init account
model. A provider accepting a QCOW2 does not prove its generic user-data field
supplies Ignition. Use the matching release recipe and
[CoreOS provisioning guidance](https://docs.fedoraproject.org/en-US/fedora-coreos/producing-ign/)
for the selected platform. Do not boot multiple instances with copied private
host keys or already enrolled Tailnet/runner state.

Protect provisioning documents, password hashes and machine keys both locally
and in provider metadata. Deleting the local file does not remove provider-held
copies; apply native access and retention controls there too.

## Deploy on Scaleway

Use the release's Scaleway recipe for the boot/provisioning settings, alongside
Scaleway's own image and network controls:

1. Choose the matching Instance architecture, Availability Zone and storage capacity.
2. Upload the verified disk to the appropriate Object Storage region and follow
   [snapshot import](https://www.scaleway.com/en/docs/instances/how-to/snapshot-import-export-feature/).
   Wait for completion before selecting the snapshot as boot storage.
3. Create a dedicated [security group](https://www.scaleway.com/en/docs/instances/how-to/use-security-groups/)
   with public ingress closed and required outbound connectivity.
4. Create the Instance stopped, attach the intended boot/data volumes and review
   security group, architecture and boot settings.
5. Deliver the release recipe's private provisioning input, then start and use
   the [serial console](https://www.scaleway.com/en/docs/instances/how-to/use-serial-console/)
   to confirm first-boot results and operator access.

Scaleway owns exact import/storage parameters and billing. Review resource names
and costs before creating them. Do not substitute a different image format or
public bootstrap path merely because an import or provisioning step failed.

## Establish private access

From the operator console, complete
[Tailscale's initial sign-in](../30-Use-Soda/40-tailscale.md#initial-cloud-connection)
and any device approval. Join the client to the permitted Tailnet, then finish
[operator setup](25-operator-setup.md) for trusted browser origins and service binds.

Establish a separate approved subnet route to project IPs. Host enrollment alone
is insufficient. Check direct project SSH and intended service ports from a real
developer client while keeping public ingress closed.

## Validate before storing team data

Confirm console access, architecture, actual usable filesystem capacity, trusted
browser and SSH identities, developer login/join and project connectivity.
Record your backup/restore method and retain the console as an independent
access path. A running VM or green import status does not establish application
or data readiness.

If boot fails, inspect native console diagnostics, disk format/attachment,
firmware and provisioning delivery. Preserve partial disks and diagnostics;
do not overwrite an instance containing work as an import retry.
