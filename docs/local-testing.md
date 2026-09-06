# Local test host

## Current instance

On 2026-09-06, native execution started on `linux-infra.dimensionlab.net` (Rocky Linux, x86_64). A **new, isolated `soda-test` KVM VM** was booted and Soda's first-install components were installed there. No Soda installation, infrastructure-service restart, physical-disk installation, firewall change or Tailnet enrollment was performed on the builder.

- Fedora CoreOS `44.20260817.3.2`, 4 vCPU, 8 GiB RAM.
- Persistent 64 GiB sparse overlay: `.artifacts/test-vm/disk.qcow2`.
- Backing image: `.artifacts/downloads/fedora-coreos.qcow2`. **Keep both files**; do not replace the backing image while the overlay exists.
- QEMU user networking; only SSH is forwarded to builder loopback `127.0.0.1:22220`.
- Guest project subnet configuration: `10.89.0.0/24`. This is **not routed to developer clients** yet.
- Private operator/provisioning files are under `.artifacts/test-vm/` (directory mode 0700), excluded from Git and container build contexts. Do not publish this directory.

The upstream stable-stream metadata and both compressed/uncompressed SHA-256 checks were used for the CoreOS image. Provisioning used `scripts/render-provisioning.py` and strict Butane conversion, with test-only hostname and pre-generated SSH host-key files added to the private Butane input. SSH uses the corresponding pinned `known_hosts`; it does not disable host-key checking.

This workspace is already prepared. `scripts/test-vm.sh` manages that instance; it is not a general provisioning/downloader command. For a different machine or a fresh workspace, follow [installation](installation.md) and the [upstream QEMU provisioning guide](https://docs.fedoraproject.org/en-US/fedora-coreos/provisioning-qemu/), using a new disk and new private credentials.

## Open the host

From this repository on the builder:

```sh
scripts/test-vm.sh status
scripts/test-vm.sh ssh
```

An SSH tunnel was started during this session. While it is running:

- **Forgejo first-run installer:** <http://localhost:23000/>
- **Cockpit host administration:** <https://localhost:29090/>

Cockpit username: `root`. The generated password is in `.artifacts/test-vm/root-password`; view it privately on the builder, not in shared logs/chat. Its test certificate uses a private CA: `.artifacts/test-vm/cockpit-ca.pem` is the **public** CA certificate fetched over pinned SSH. Trust it only in your intended test browser/profile. This is not publicly trusted production TLS.

If your browser is on your laptop rather than the builder, run there:

```sh
ssh -N -o ExitOnForwardFailure=yes \
  -L 23000:127.0.0.1:23000 -L 29090:127.0.0.1:29090 \
  vince@192.168.2.253
```

Then open the same localhost URLs on the laptop. Do not bind these unfinished installers to `0.0.0.0`.

To restart a missing builder-to-VM tunnel, keep this command running in a builder terminal (do not run a second copy while its ports are occupied):

```sh
scripts/test-vm.sh tunnel
```

If an existing Cockpit session reports `tailscaled.sock: connect: permission denied`, **log out completely and sign back in**. The initial custom PAM stack omitted Fedora's SELinux session transition; that configuration is now corrected in source and this VM. Existing sessions retain their old context until logout. A fresh authenticated bridge has been verified to read Tailscale status and preferences with SELinux enforcing, without changing socket permissions or restarting the daemon.

For subsequent VM starts and diagnosis:

```sh
scripts/test-vm.sh start      # reuses the disk; does not recreate it
scripts/test-vm.sh console
scripts/test-vm.sh ssh 'systemctl --failed --no-pager'
```

When you explicitly want to shut down **this test guest only**:

```sh
scripts/test-vm.sh ssh 'systemctl poweroff'
```

## Next: activate the product

**The Soda dashboard/proxy are not activated yet.** A host login or an installer page is not the full product journey.

1. Choose distinct Soda and Forgejo HTTPS browser origins, browser-trusted TLS material, and how your test client will resolve/reach them. The localhost tunnel ports above are bootstrap access, not the final OAuth origins.
2. Complete Forgejo's native installer and create its administrator. This is the VM's new Forgejo, **not** the builder's existing infrastructure Forgejo.
3. Follow [operator setup](operator-setup.md): create the restricted Forgejo token, store it privately on the VM, and run `soda-setup` there with the chosen origins.
4. Follow [activation](installation.md#3-install-the-built-components) with those certificates and the guest's selected private bind IP. Add the corresponding HTTPS tunnel or private connectivity deliberately; the current helper forwards only bootstrap ports.
5. Establish real client routing to the project subnet before claiming direct project SSH works. QEMU port forwarding alone cannot prove that journey. A bridged/routed host VM network or approved Tailnet subnet route requires a separate networking decision.
6. Run the Alice/Bob, shared-tool, nested-workload and persistence journeys in [native validation](native-validation.md). Provider registrations and destructive lifecycle checks need their own approved test resources/actions.

## Observed evidence

Source revision: `3a12d135bc4362273c209d78a40b7458993cb373` **plus the uncommitted native-startup fixes in this working tree**. No commit or artifact publication was made.

Executed successfully:

- Dependency resolution with Go `1.26.7`; real `go.sum` and indirect requirements generated.
- `scripts/build-native.sh x86_64`: native commands, both Cockpit pages, Rocky project/dashboard images, verified GitHub runner download and staging.
- `scripts/check-native.sh x86_64`: Go tests, TypeScript checking, 10 Cockpit test files / 60 tests, and initially 4 packaging checks. All 5 packaging checks now pass after adding the Cockpit PAM regression.
- CoreOS first boot, native extension installation, and a reboot of only the new VM to activate the packages.
- Soda installation after correcting the install copy mechanism; active Forgejo, helper socket, Cockpit socket and Tailscale daemon. No failed systemd units observed.
- Forgejo installer HTTP 200; Cockpit login page HTTP 200 with explicit CA verification; Cockpit native root authentication HTTP 200.
- `tests/installed/host.sh`: first-install service state, root-owned native commands, helper socket permissions, dashboard directory ownership, SELinux labels and Cockpit page files.
- The dashboard image's executable starts as UID 2000 (`-h` smoke check without networking or dashboard configuration).
- Fresh authenticated Cockpit bridge: native operator SELinux context, Tailscale status `NeedsLogin`, and LocalAPI preferences HTTP 200. The exact reported permission error was reproduced before the PAM correction; logs are `cockpit-tailnet-before.log` and `cockpit-tailnet-after.log`. Native PAM account checks allow root and deny `core`; no account was created or changed.

The first executions exposed and corrected the Rocky `curl`/`curl-minimal` conflict, mise's `./`-prefixed checksum filenames, runner CLI argument-validation ordering, and dashboard executable permissions. The original recursive installation copy failed on CoreOS's read-only `/usr` and imported builder ownership/SELinux labels. The installer now extracts into the three writable prefixes without changing existing parent metadata, uses root ownership, and restores native labels on delivered paths. The partial attempt's affected ownership/labels were repaired in this new VM; **a fresh-disk rerun of the final installer remains a separate check**.

Logs are under `.artifacts/logs/`, notably `build-native.log`, `check-native.log`, `vm-first-boot.log`, `vm-install.log`, `vm-services.log` and `vm-host-check.log`. Failed-attempt logs are retained rather than represented as passes.

Read-only first-install checks can be repeated without creating projects or enrolling providers:

```sh
scripts/test-vm.sh ssh 'SODA_NATIVE_VALIDATE=soda-test bash -s' < tests/installed/host.sh
```

To repeat source checks using the workspace-local pnpm installation:

```sh
export PATH="$PWD/.artifacts/tools/node_modules/.bin:$PATH"
export GOTOOLCHAIN=go1.26.7
scripts/check-native.sh x86_64
```

Still unvalidated: dashboard/OAuth/browser journey, project creation/join/SSH, nested Podman, shared tools, project persistence, Tailnet enrollment/routing/exit-node operations, provider-scheduled runner jobs, and all AArch64 behavior. No real provider accounts, developer users or projects have been created in the VM.
