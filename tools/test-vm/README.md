# soda-test-vm

Start and access the prepared local x86_64 development VM in
`.artifacts/test-vm`. This tool is for operators and developers using a retained
test fixture; it uses the existing disk and provisioning rather than preparing
a new VM.

## Prerequisites

Run from the repository root. Starting requires native x86_64 Linux, read/write
access to `/dev/kvm`, and QEMU at `/usr/libexec/qemu-kvm`. Set `QEMU` to another
absolute executable path if needed. The guest receives 4 vCPUs and 8 GiB RAM.

Prepare these files under `.artifacts/test-vm/` before starting:

- `disk.qcow2`: the fixture disk, which the guest can modify.
- `soda.ign`: the fixture's Ignition provisioning document.
- `operator`: the private SSH authentication key.
- `known_hosts`: the trusted guest SSH host-key entry.

Keep private inputs restricted. SSH uses strict host-key checking and connects
as root through `127.0.0.1:22220`; it requires the system `ssh` client.

## Use the fixture

```sh
cargo run -p soda-test-vm -- status
cargo run -p soda-test-vm -- start
cargo run -p soda-test-vm -- ssh
cargo run -p soda-test-vm -- ssh hostname
```

With no action, the command defaults to `status`. A stopped VM returns exit
`1`; an already running VM is left running by `start`. Starting creates a
lock, pidfile and console log beside the fixture inputs.

| Action | Result |
| --- | --- |
| `tunnel` | Keep SSH forwards open for Forgejo at `http://localhost:23000` and Cockpit at `https://localhost:29090`. |
| `web-tunnel` | Keep the fixture's private HTTPS endpoint forwarded at `https://localhost:24444`. |
| `console` | Follow the last 80 lines and subsequent output of `console.log` using `tail`. |

The services must already be configured inside the guest. Trust the fixture CA
for HTTPS. Stop a tunnel or log tail with Ctrl-C; the VM keeps running.
For an intentional guest shutdown, use `cargo run -p soda-test-vm -- ssh poweroff`.
There is no `stop` action or disk-deletion command.

See [local testing](../../docs/guides/local-testing.md) for fixture access and
[native support](../../docs/development/native-support.md#fresh-vm-contract)
for preparing a separate fresh VM with bounded acceptance evidence.
