# Local Soda dashboard and test host

## New support source versus this persistent guest

The [native support tools](native-support.md) are source-only additions, not a replacement for this existing guest or new evidence about it. `scripts/test-vm.sh`, `.artifacts/test-vm/` and the overlay's backing image are preserved. New tool fixtures require fresh paths/ports/identities and separate permission; never point their work/cache outputs at these persistent files.

## Current React preview candidate

The existing guest runs matching dashboard/helper/default new-project image
`8b823db`, with React preview at <https://localhost:24443/app/>. Default routes
remain HTMX; this is not U18 cutover. Full native build/check and populated-v3
backup/startup rehearsals passed before rollout. Earlier schema-1→3/key-failure
and rollback-copy rehearsals remain historical evidence, not a lossless live
rollback now. See [current evidence](implementation-status.md#u08-accepted--bounded-native-x86_64-first-product-proof).

The installed React operator OAuth/native-read/navigation/logout check passed
with certificate verification enabled. Older native Soda consent was explicitly
replaced through the operator's own Applications UI; no bootstrap-token fallback
was used. The React browser harness has now been removed from current source;
its scripts and instructions remain in Git at `88fc21f`, with their original private
evidence. Do not run a retired journey against a new candidate or infer Sodaspaces
acceptance from its results. Native tab/drawer browser coverage still needs writing.

`SODA_RECONSENT_APPLICATION='SodaOS dashboard'` is a separate explicit mutation:
it revokes only this user's uniquely named native Soda grant before reconsenting.
Do not enable it as an automatic retry or for unrelated applications/users.
The same users, `u08-alice-8417` and `u08-bob-8417`, now use four retained
environments: the original two and Alice's approved `u08-completion-952f3b3`
and `u08-completion-c96c108` repository/environments. Explicit joins and native collaboration remain separate.
Original inputs are `.artifacts/test-vm/u08-8417a90/`; completion inputs, verified
addresses and state comparisons are in `.artifacts/test-vm/u08-completion-952f3b3/`
and `.artifacts/test-vm/u08-completion-c96c108/`.
Do not infer addresses from old examples: stop/start and reboot changed them.

From infra, direct SSH/SCP/SFTP, personal Git/shared tools and ordinary bridge
HTTP/PostgreSQL passed, as did corrected project stop/start and `soda-test` reboot
preservation. Both new personal Git keys were unlocked after reboot; older Git
agents died and their original passphrases were not retained. Runtime routes and
browser/Cockpit/Git transports were restored. This does not route a laptop or
prove automatic workload startup. The further approved `c96c108` fixture now
passes exact-image fresh creation and different-UID/default-user/PTY exec, while
Bob remains denied engine administration. **U08 is accepted for bounded native
x86_64 first-product proof** after merged `8b823db` build/rollout and affected
checks, reusing earlier lifecycle evidence for unchanged mechanisms. No further
reboot occurred. Full P11/U20 operator proof remains pending: the corrected
operator test exposes the preexisting missing console welcome hook. Another fixture or capability change needs new scoped approval. Never replace roots or restore an
outdated DB to repair these limits.

`SODA_U08_FIXTURES_DIR` enables real fixture creation through the core-owned
`tests/installed/developer-first-workflow.mjs`. It requires a private fixture
manifest with two specifically named users, separate initial/final password files
and development public keys. Its explicitly selected
`SODA_U08_RESUME_FIXTURES=1` reuses retained observed identities and verifies state;
it does not retry incomplete reservations or reset anything.
`SODA_U08_CONNECTIONS_DIR` enables read-only authenticated connection checks and
independent operator verification of **public** host keys. Neither mode proves
client reachability; private keys remain on the client.

## Latest Git/shared-tool/workload results

Personal native Git clone/commit/push/readback and shared root-owned Node/files
now pass from infra and the retained project users. Real HTTP source-mount edits
and committed PostgreSQL TCP data also pass, but only with nested **project-network
mode**. Default Compose bridge startup failed for missing NET_ADMIN; its source
correction has not been validated on a fresh project. Alice's existing project
received native API service/socket fixes, not a replacement image/rootfs.
See [exact evidence, credentials and remaining limits](implementation-status.md#personal-git-shared-tools-and-nested-workload-evidence).

Git transport adds two private forwards for Forgejo's actual loopback clone URL;
it does not change that URL or use shared Git authentication. Temporary Git agents
and active workloads/probes/volumes remain. Do not assume those agents survive a
reboot or destroy failed Compose resources. Lifecycle persistence remains pending.

## Approved project routing from infra

The private Layer-3 SSH tunnel is now approved and running. Infra has a runtime
route to `10.89.0.0/24` through `tun8417`, restricted by dedicated firewall rules;
no LAN/Tailnet change was made. `tests/installed/developer-access.py` passed direct
SSH/PTY/SCP/SFTP for Alice in her project and Bob in both projects, including sudo
boundaries and denial of Alice's authentication to the unjoined second project.
See [current routing evidence and exact teardown](implementation-status.md#approved-private-routing-and-direct-developer-access).

This routes **infra**, not your laptop. Browser tunnels remain separate. Keep the
run-owned transport and probe data for subsequent runtime/persistence checks;
no project or VM lifecycle test has been performed. Earlier direct-access-pending
statements in the historical observations below are superseded by this result.

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

Still unvalidated: the full developer onboarding/create/join/SSH journey, nested Podman, shared tools, project persistence, Tailnet enrollment/routing/exit-node operations, provider-scheduled runner jobs and AArch64. The original bootstrap created only the operator. The later React fixtures described above now add real developer users, private repositories, environments and memberships; preserve them.
