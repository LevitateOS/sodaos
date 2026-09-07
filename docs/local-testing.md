# Local test host and retained access

This describes the **last recorded** isolated `soda-test` installation, not a live
status check or permission to mutate it. Current source has neither standalone
Soda frontend; the guest still has historical `8b823db` React preview/HTMX defaults.
The Sodaspaces UI is not installed or implemented. See [handoff/evidence](implementation-status.md).

## Target and state to preserve

- Builder/client: `linux-infra.dimensionlab.net` / recorded `192.168.2.253`, native
  x86_64 Rocky Linux. No Soda appliance installation on the builder is authorized.
- Guest: Fedora CoreOS `44.20260817.3.2`, direct QEMU/KVM, 4 vCPU, 8 GiB RAM.
- Persistent 64 GiB sparse overlay: `.artifacts/test-vm/disk.qcow2`, backed by
  `.artifacts/downloads/fedora-coreos.qcow2`. **Keep both; never replace the base.**
- Management SSH: builder loopback `127.0.0.1:22220`, with pinned host keys.
- Private inputs/browser homes/evidence: `.artifacts/test-vm/`, directories 0700,
  sensitive files 0600. Preserve all four projects listed in the handoff, their
  accounts, keys, dirty work, workloads/volumes and later writes.
- Existing `scripts/test-vm.sh` is a wrapper for this prepared instance, not an
  installer or disposable-fixture provisioner. Do not adopt its disk/base/state
  into the [new outside support tools](native-support.md).

## Browser access

When the existing builder-to-guest tunnels are available, the recorded origins are:

| Service | Origin | Identity / private builder input |
| --- | --- | --- |
| Historical Soda | `https://localhost:24443/` (`/app/` preview) | Forgejo `operator`; `.artifacts/test-vm/forgejo-operator-password` |
| Native Forgejo | `https://localhost:24444/` | Same native Forgejo identity, not host root |
| Cockpit | `https://localhost:29090/` | Native `root`; `.artifacts/test-vm/root-password` |

For laptop browser access through the recorded builder, an explicitly selected
SSH transport is:

```sh
ssh -N -o ExitOnForwardFailure=yes \
  -L 24443:127.0.0.1:24443 -L 24444:127.0.0.1:24444 \
  -L 29090:127.0.0.1:29090 vince@192.168.2.253
```

Do not open duplicate occupied tunnels. Keep the exact configured localhost
origins: both Soda and Forgejo are visited during historical OAuth. This is not
project routing or authorization for browser mutations. Read credentials privately,
never in tool output/chat/logs. The builder's infrastructure Forgejo is unrelated.

The public test CA is `.artifacts/test-vm/cockpit-ca.pem`. Trust only that public
certificate in the selected isolated browser/profile; no TLS bypass or reuse of a
personal credential-bearing profile. Trust is not installed automatically on a laptop.
The CA's private key and provisioning hashes remain secrets.

## VM wrappers and action limits

Inspect the subcommand and actual target before use:

| Wrapper | Effect |
| --- | --- |
| `scripts/test-vm.sh status` | Local process/status observation |
| `scripts/test-vm.sh ssh` | Native operator shell; commands may mutate the guest |
| `scripts/test-vm.sh console` | Guest console attachment, potentially interactive |
| `scripts/test-vm.sh web-tunnel` | Soda 24443 / Forgejo 24444 forwarding |
| `scripts/test-vm.sh tunnel` | Cockpit 29090 / legacy loopback bootstrap 23000 forwarding |
| `scripts/test-vm.sh start` | Starts the existing guest/disk; requires applicable scope |

Port 23000 is diagnostic/bootstrap access, not the configured normal Forgejo origin.
Existing fixture and reboot approvals were used. A new start/stop/reboot, fixture,
installation, provider operation or host-network change requires its own scope.
Never rerun first-install/bootstrap or clear markers/configuration to repair this guest.

## Developer routing and historical checks

Project subnet `10.89.0.0/24` was routed **from infra** through the private Layer-3
SSH tunnel `tun8417` and exact run-owned firewall rules, not through a LAN/Tailnet
change. Direct SSH/PTY/SCP/SFTP, native Git/shared tools, HTTP/SQL and scoped lifecycle
results are in the handoff. This does not route the laptop or establish automatic
route/workload restart. Project addresses changed across lifecycle operations; old
examples, agents and recorded bindings are not current reachability evidence.

Preserve the run-owned tunnel, probe files, failed workload resources, private Git
inputs and snapshots. Old agents/passphrases are not guaranteed available after
reboot. Exact historical routing/teardown and chronological observations are in
`git show 9f3baa7:docs/implementation-status.md`; they are not current cleanup permission.

Old React and Go/HTMX browser harnesses are retired with their frontends. Recover
historical scripts only from their matching Git revision and scope; do not rerun
them against current API-only source or relabel their results as Sodaspaces proof.
Use [installation](installation.md) for a separately authorized fresh build/target,
[native validation](native-validation.md) for product checks and the handoff for
actual performed evidence. No command here was run during documentation cleanup.
