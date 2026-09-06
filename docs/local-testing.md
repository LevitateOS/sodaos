# Local Soda dashboard and test host

## New support source versus this persistent guest

The [native support tools](native-support.md) are source-only additions, not a replacement for this existing guest or new evidence about it. `scripts/test-vm.sh`, `.artifacts/test-vm/` and the overlay's backing image are preserved. New tool fixtures require fresh paths/ports/identities and separate permission; never point their work/cache outputs at these persistent files.

## Open the dashboard

**The dashboard, Forgejo and HTTPS proxy are now running in `soda-test`.** Native Forgejo login, OAuth consent/callback, the authenticated Projects/Profile/People pages and Soda sign-out have been exercised in Chromium with certificate verification enabled.

On your laptop, keep this SSH tunnel to the builder running:

```sh
ssh -N -o ExitOnForwardFailure=yes \
  -L 24443:127.0.0.1:24443 -L 24444:127.0.0.1:24444 \
  vince@192.168.2.253
```

Then open:

- **Soda dashboard:** <https://localhost:24443/>
- **Forgejo:** <https://localhost:24444/>

Use these exact localhost origins; the ports are part of the OAuth configuration. Both ports must be forwarded because sign-in visits Forgejo and returns to Soda. You do not need a hosts-file change.

Select **Sign in with Forgejo**. The VM-local administrator is `operator`. Its generated password is stored privately on the builder at:

```text
/home/vince/Projects/sodaos/.artifacts/test-vm/forgejo-operator-password
```

View that file privately, not in shared logs/chat. This identity is distinct from the host's `root` login and from the builder's existing infrastructure Forgejo. No developer host account was created.

### Choose a repository

Projects now offers a Forgejo repository picker instead of a text field. It lists repositories owned by the signed-in user that do not already have a Soda environment (including incomplete reservations). If none are available, use **Create a repository in Forgejo**, then return and select **Refresh repositories**. Other existing Soda environments remain accessible from the Projects table. Loading failures show an error rather than pretending there are no repositories.

### Test TLS

All three browser endpoints use the VM's test certificate for `localhost`, issued by its private CA. The public CA certificate is `.artifacts/test-vm/cockpit-ca.pem`. Copy only that public file to your laptop and trust it for websites in your intended test browser/profile:

```sh
scp vince@192.168.2.253:/home/vince/Projects/sodaos/.artifacts/test-vm/cockpit-ca.pem soda-test-ca.pem
```

The CA is not publicly trusted production TLS. Browser trust on your laptop is **not** installed automatically. The automated browser test used a separate private NSS trust database; it did not disable TLS verification or change the builder's normal browser/system trust store. The dashboard/proxy use the same localhost test certificate already used by Cockpit.

## Cockpit and VM access

Cockpit remains at <https://localhost:29090/> through the existing host tunnel. From another client, forward `29090:127.0.0.1:29090` to the builder separately if needed. Username: `root`; its separate password is `.artifacts/test-vm/root-password` on the builder.

Cockpit's Accounts navigation entry is intentionally hidden; native host-account tools remain available. Existing sessions may need logout/login to discard cached manifests. Also log out and back in if an old session still reports `tailscaled.sock: connect: permission denied`: the initial PAM stack omitted Fedora's SELinux session transition, which is now corrected. SELinux remains enforcing.

From the repository on the builder:

```sh
scripts/test-vm.sh status
scripts/test-vm.sh ssh
scripts/test-vm.sh console
```

Builder-to-VM tunnels were started during setup. If either is missing, keep the corresponding command running in a builder terminal; do not start duplicates while their ports are occupied:

```sh
scripts/test-vm.sh web-tunnel  # Soda 24443 and Forgejo HTTPS 24444
scripts/test-vm.sh tunnel      # Cockpit 29090 and legacy Forgejo bootstrap 23000
```

Port 23000 remains loopback-only diagnostic/bootstrap access; use Forgejo's configured HTTPS origin for normal browser sign-in now.

For a later start, `scripts/test-vm.sh start` reuses the existing disk. When explicitly shutting down **only this guest**, use `scripts/test-vm.sh ssh 'systemctl poweroff'`. There is no automatic disk deletion or replacement.

## Current instance and bootstrap

Native execution started on 2026-09-06 on `linux-infra.dimensionlab.net` (Rocky Linux, x86_64). The new `soda-test` VM uses:

- Fedora CoreOS `44.20260817.3.2`, 4 vCPU, 8 GiB RAM, direct QEMU/KVM (not a libvirt domain).
- A persistent 64 GiB sparse overlay at `.artifacts/test-vm/disk.qcow2`, backed by `.artifacts/downloads/fedora-coreos.qcow2`. **Keep both files; do not replace the backing image.**
- QEMU user networking, with only SSH forwarded to builder loopback `127.0.0.1:22220`; browser access goes through pinned SSH tunnels.
- Project subnet configuration `10.89.0.0/24`. It is **not routed to developer clients** yet.
- Private provisioning/operator files in `.artifacts/test-vm/` (directory mode 0700), excluded from Git and container build contexts.

The upstream CoreOS image was checked against both compressed and uncompressed SHA-256 metadata. Provisioning used `scripts/render-provisioning.py` and strict Butane conversion, adding a test hostname and pre-generated SSH host keys to the private input. SSH pins those host keys rather than disabling checking.

Forgejo's native web installer created the VM-local `operator` administrator. A scoped native token (`read:user`, `write:user`, `write:admin`, `read:repository`) was saved privately on the VM. `/usr/local/sbin/soda-setup` created the actual OAuth application/configuration, and `/usr/local/sbin/soda-activate` enabled the services with bind IP `127.0.0.1`. Use those absolute command paths: the CoreOS root SSH PATH does not include `/usr/local/sbin`.

The dashboard runs as UID/GID 2000, without effective capabilities. Credentials are root:soda 0640 under `/etc/soda` (0750); its database directory is soda:soda 0700; the helper socket remains root:soda 0660. Caddy and the Go listener bind only guest loopback. Activation restarted only the VM's Forgejo container as required; it did not restart infrastructure services or reboot the builder/VM.

No Soda installation, physical-disk installation, host firewall/routing change, Tailnet enrollment or provider-runner registration was performed on the builder. This workspace is already prepared; `scripts/test-vm.sh` is not a general installer. A new target still needs [installation](installation.md) and separate operator provisioning. No Soda installer ISO has been built.

## Repeat the checks

Source/staging checks (Go, TypeScript, Cockpit and packaging tests; the merged tree adds checks not yet executed):

```sh
export PATH="$PWD/.artifacts/tools/node_modules/.bin:$PATH"
export GOTOOLCHAIN=go1.26.7
scripts/check-native.sh x86_64
```

Read-only native first-install checks:

```sh
scripts/test-vm.sh ssh 'SODA_NATIVE_VALIDATE=soda-test bash -s' < tests/installed/host.sh
```

The opt-in browser check creates authentication/consent/session state for the explicitly selected operator, then signs out of Soda. It does not create people, repositories, project environments, keys or workloads. This workspace's isolated browser home already trusts the test CA and Playwright's Chromium is installed:

```sh
node tests/installed/dashboard.mjs \
  https://localhost:24443 https://localhost:24444 operator \
  "$PWD/.artifacts/test-vm/forgejo-operator-password" \
  "$PWD/.artifacts/test-vm/browser-home"
```

For another builder, prepare a private isolated browser home with the selected target's CA trusted; do not bypass certificate verification or use a personal browser profile containing unrelated credentials.

## Evidence and remaining work

The initial native run used `3a12d135bc4362273c209d78a40b7458993cb373` plus then-uncommitted startup fixes, subsequently committed in `88be176`. Accounts-navigation changes are in `95a194d`; subsequent dashboard-access/test and repository-picker changes are in `c96530c`. These recorded results do not validate the combined tree with the branding, console and project-CLI follow-ups, which has not been rebuilt or retested. No artifact publication was performed. Logs are under `.artifacts/logs/`, including:

- `build-native.log`, `check-native.log`, `vm-first-boot.log`, `vm-install.log`, `vm-host-check.log`;
- `cockpit-tailnet-before.log` / `cockpit-tailnet-after.log` and `cockpit-packages-before.log` / `cockpit-packages-after.log`;
- `dashboard-activation.log`, `dashboard-services.log` and `dashboard-browser-check.log`;
- `repository-picker-tests.log`, `repository-picker-build.log` and `repository-picker-deploy.log` for the later dashboard-only picker deployment.

The native browser picker check exercised the empty state because the test operator had no repositories. Populated listings, pagination, ownership filtering and forged submissions have Go test coverage; no live repository/project fixture was created for this change.

The initial build/install exposed concrete curl-package, mise-checksum, CLI-validation, binary-mode, CoreOS copy/ownership/label and Cockpit PAM defects. Failed-attempt logs are retained. The first installation was repaired after its partial copy; **a fresh-disk run of the final installer remains unverified**.

Next, exercise the developer journey in [native validation](native-validation.md): create people through Soda, repositories through Forgejo, register public keys, create/join project environments, use shared tools and nested workloads, and validate persistence. Real client routing to project IPs must be established deliberately before claiming direct SSH works; port forwarding is not proof of it.

Still unvalidated: the full developer onboarding/create/join/SSH journey, nested Podman, shared tools, project persistence, Tailnet enrollment/routing/exit-node operations, provider-scheduled runner jobs and AArch64. There is one local Forgejo operator identity and its Soda profile, but no developer users, repositories or project environments were created during dashboard bootstrap.
