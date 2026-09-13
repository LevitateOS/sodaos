# Local test host and retained access

This guide owns recorded access paths for the fresh Tailnet VM and `soda-test`,
and the retained instance's wrapper effects.
Installed versions, active grants and preservation results have one home in the
[current handoff](implementation-status.md); historical access receipts are in
[implementation history](implementation-history.md). None of the addresses below
is a fresh liveness observation.

## Spaces frontend development on this computer

Run `bun run dev:spaces` from the canonical checkout, then open
**http://127.0.0.1:24455/**. This starts a Bun development server on this computer's
loopback interface. It needs the locked workspace dependencies and Python 3 for
the existing terminal asset preparation, but no VM, SSH tunnel, Forgejo account,
appliance deployment or project provisioning. Stop it with Ctrl-C. To choose a
different port, run `bun run dev:spaces --port 24456`.

The server serves the real emitted Spaces components and canonical Soda styles.
Its HTTP and WebSocket handlers reuse the component tests' synthetic model.
The labelled development toolbar can reset to Welcome, Project created / Join,
First terminal, working tabs, two panes, long names, stopped/unavailable projects
and expired access. The normal Create → Join → New terminal flow also works.
Responses can be slow, unavailable, expired, or reject Create/Join; themes can be
switched without changing product code. Reconnect is simulated locally.

Frontend TypeScript, CSS and fixture-shell changes rebuild and reload the browser
automatically. Project state and terminal transcripts stay in server memory across
reloads; layout stays in tab session storage. Restarting the server clears its data.
Reset scenario replaces the current browser session's mock state. Separate browser
contexts have separate fixtures; tabs sharing the cookie share mock projects.
Backend/scenario-model changes require restarting the command.

The terminal renderer and controls are real; its shell is simulated. Try `help`,
`pwd`, `ls`, `git status`, `bun test`, `echo hello` or `clear`. These commands never
execute on this computer. The surrounding navigation is a labelled preview shell;
native Forgejo pages, real OAuth, project runtimes and shell identity still require
their separately authorized integration checks. Fixture routes never proxy to the VM.

Focused verification: `bun test --timeout 30000 tests/frontend/spaces-preview.test.ts`
after asset preparation. This exercises the local HTTP/socket journey, transcript
restoration, scenario resets, responsive layouts, expired-access recovery and
request boundaries. Screenshot evidence uses the existing
[component capture helper](screenshot-capture.md), not native-page proof.

## Fresh Tailnet VM access

The separately created **`soda-native-tailnet-bb3a13c`** has its own ports and
credentials. Do not use `scripts/test-vm.sh` for this instance. Its installed source,
correction, current hold deadline and permissions belong in the
[current handoff](implementation-status.md#fresh-tailnet-access-fixture).

From the laptop, keep this tunnel running while using the browser:

```sh
ssh -N -o ExitOnForwardFailure=yes \
  -L 24454:127.0.0.1:24454 \
  -L 29094:127.0.0.1:29094 vince@192.168.2.253
```

| Service | Origin | Username | Private password file on builder |
| --- | --- | --- | --- |
| Dashboard / Forgejo | `https://localhost:24454/` | `operator` | `.artifacts/tailnet-vm-bb3a13c/forgejo-operator-password` |
| Cockpit | `https://localhost:29094/` | `root` | `.artifacts/tailnet-vm-bb3a13c/root-password` |

Paths are relative to `~/Projects/sodaos`. Open password files privately; never
print them in tool output/chat/logs. Dashboard authentication uses Forgejo, not a
third password. From the native profile menu, choose **Site administration**, then
**Soda → Runners / Tailnet**. Direct entry bookmarks are
`https://localhost:24454/-/soda/settings/runners` and
`https://localhost:24454/-/soda/settings/tailnet`; they do not require visiting
Spaces first. See the [handoff](implementation-status.md#fresh-tailnet-access-fixture)
for installed state.

The public CA is `.artifacts/tailnet-vm-bb3a13c/tls/ca.pem`, SHA-256
`97b20d81c0678708c198547937ba48d999618c7580511350f6afd39c3a108064`.
Retrieve it over trusted SSH and explicitly trust only this public certificate in
the selected isolated test browser/profile before login. No laptop/global trust
was installed; the CA's private key is not a client input. Keep the exact localhost
origins for TLS/OAuth. Management SSH is pinned builder loopback port `22234`;
`23034` is the loopback-only native Forgejo bootstrap/diagnostic forward, not the
configured dashboard origin. Native root and Forgejo/OAuth/browser access passed
in the [fresh VM receipt](implementation-history.md#fresh-tailnet-vm-installation-and-access-smoke).

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
| Native Forgejo + Sodaspaces | `https://localhost:24444/` | Forgejo `operator`; `.artifacts/test-vm/forgejo-operator-password`, not host root |
| Cockpit | `https://localhost:29090/` | Native `root`; `.artifacts/test-vm/root-password` |

For laptop browser access through the recorded builder, an explicitly selected
SSH transport is:

```sh
ssh -N -o ExitOnForwardFailure=yes \
  -L 24444:127.0.0.1:24444 \
  -L 29090:127.0.0.1:29090 vince@192.168.2.253
```

Do not open duplicate occupied tunnels. Keep the exact configured localhost
origin: current OAuth stays on Forgejo at 24444 under `/-/soda/`. The old 24443
Soda listener is removed; its preserved forwarding process is not a serving UI. This is not
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
| `scripts/test-vm.sh web-tunnel` | Current Forgejo/Sodaspaces 24444 forwarding; does not open retired 24443 |
| `scripts/test-vm.sh tunnel` | Cockpit 29090 / legacy loopback bootstrap 23000 forwarding |
| `scripts/test-vm.sh start` | Starts the existing guest/disk; requires applicable scope |

Port 23000 is diagnostic/bootstrap access, not the configured normal Forgejo origin.
Use the handoff's current grants and [execution policy](../AGENTS.md#permissions-and-preservation)
before a wrapper's effects; an available command does not authorize them.

## Developer routing and historical checks

Project subnet `10.89.0.0/24` was routed **from infra** through the private Layer-3
SSH tunnel `tun8417` and exact run-owned firewall rules, not through a LAN/Tailnet
change. Direct SSH/PTY/SCP/SFTP, native Git/shared tools, HTTP/SQL and scoped lifecycle
results are in the handoff. This does not route the laptop or establish automatic
route/workload restart. Project addresses changed across lifecycle operations; old
examples, agents and recorded bindings are not automatic liveness evidence. Do not
attribute later access checks to this older route: the current handoff identifies
the actual client transport used for each delivered scope.

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
