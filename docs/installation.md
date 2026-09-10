# Native build and installation

Native execution has begun on the local x86_64 builder and an isolated CoreOS VM; see [local testing](local-testing.md) for observed results and remaining gaps. Use the actual authorized matching-native Linux builder and selected appliance target. A build or boot does not establish usable end-to-end development environments.

The [Sodaspaces plan](sodaspaces-plan.md) and production callers own application
payload/configuration/credentials/migrations/cutover. [Native support](native-support.md)
supplies artifact inspection/bundling and provisioning transport, not a second
installer or product gate. The [CoreOS installer implementation](coreos-installer.md)
now has source and focused local tests for upstream ISO customization, a Go console
and installed-host continuation instead of Anaconda. The console ships on media,
not through a required hosting URL; see the handoff for generation evidence.
Bounded diskless BIOS/UEFI boot checks passed; fresh-disk validation remains unrun.
A prepared Soda QCOW2 is a recommended future download; its producer remains
unimplemented. See
[handoff](implementation-status.md) for actual native build/check limits. The recipes
below use sealed bundles and private provisioning. The separately built installer
ISO carries an installable bundle in the replacement source recipe; it does not
contain a configured Soda appliance. No Soda host OCI or preinstalled
QCOW2 is supplied. Do not replay first-install as a service upgrade.

## Publication direction

The 10 September 2026 discussion records the following delivery direction, not
artifacts already available or permission to publish them:

| Artifact | Intended role | Current state |
| --- | --- | --- |
| SodaOS ISO | Primary download for physical USB installation and manual VM installation | Previous required-key media exists; replacement password-only/payload/setup source is under local validation, with rebuilt media and full fresh-install proof deferred |
| SodaOS QCOW2 | Recommended second download: a prepared VM disk booting into the same first-time operator setup | No preinstalled Soda product image or producer exists; the exact image-production and first-boot recipe still needs design |
| SodaOS host OCI | Optional delivery architecture for a versioned host OS; not required for the current CoreOS approach | Not produced; no bootc migration or whole-host OCI update path is selected |
| Sealed Soda payload | Matching native programs, configuration and application/project OCI archives needed to install Soda | Existing native bundle contract; replacement ISO recipe includes a matching snapshot for pre-removal copying. QCOW2 inclusion remains future work |

The target user-facing downloads are ISO and QCOW2, containing the matching Soda
payload, with release/architecture identity and verifiable checksums. The artifact
distribution/signing contract still needs definition; existing checksums are not a
Soda release signature. An ISO is bootable installation media;
QCOW2 is a virtual disk, not a complete VM definition. Generic media must contain no
operator credentials or initialized personal app state. Establish the password,
per-machine identity/host keys and remaining configuration for each new installation;
do not distribute a copy of a retained test appliance.

Payload inclusion must reuse the production build, inventory, verifier and native
installation contracts. The ISO must preserve the verified payload on the selected
destination for installed-host continuation before asking the user to remove media;
the replacement source implements that handoff for native validation. The QCOW2
producer must deliver the same release and first-boot behavior without cloning
credential-bearing fixture state. No manual builder-bundle transfer should remain
in the normal product-media journey. Including application images does not remove
the current network requirement for host RPM dependencies or provide offline
marketplace apps.

OCI means image packaging, not inherently a whole-host updater. Application OCI
images remain ordinary components of the Soda payload and may be distributed through
that payload without a separately operated registry. A specially built host OCI is
an optional different delivery choice. See [host and application update ownership](os-product-strategy.md#update-ownership).

The immediate [installer correction](coreos-installer-plan.md) removes the public-key
prompt from USB/VM disk installation and uses the native root password. Publishing a
usable image also requires the complete first-boot, access and application setup
journey. Historical tests used upstream CoreOS QCOW2 plus private Ignition and SSH
installation; they are runtime evidence, not proof of a public Soda QCOW2 or manual
ISO installation. See [recorded VM setup](local-testing.md) and the [handoff](implementation-status.md).

Keep the following commands as **component/fixture recipes**, distinct from the
manual media journey. Source documentation does not authorize builds, new
fixtures, destination disk writes, uploads, retained-target changes or publication.

## 1. Prepare the native builder

Use x86_64 first when access exists; repeat independently on aarch64 later. Install Go 1.26.7, Bun 1.4.2, Python >=3.12, GNU make and native Podman through the builder's normal mechanisms. Do not cross-compile/emulate and report native evidence. The root `package.json` pins Bun; run `bun install --frozen-lockfile` at the repository root. See [TypeScript development](typescript.md) for local commands and compiler boundaries.

Soda is a Go API/OAuth command with no embedded or external standalone UI. Both original Go/HTMX and React frontends are removed from source; the read-only Sodaspaces hook/drawer passed its isolated local browser journey, not an appliance install. The standalone React directory, lock, build and external asset payload are removed; Cockpit retains its frontend/build inside the root Bun workspace and shared lock. New bundles reject retired SPA payloads. Old installed bundles/evidence retain their original verifier and source revision. Dependency/build/deployment actions still require their applicable scope.

Real Go dependency metadata was resolved during the first native x86_64 build and is now in `go.mod`/`go.sum`. For intentional dependency changes, run `go mod tidy` and review the resulting metadata. The root Bun lock preserves the existing Cockpit dependency versions and records shared tooling dependencies. Then invoke:

```sh
scripts/build-native.sh x86_64
```

Use a clean exact-revision checkout with fresh `.artifacts/native/x86_64` output; another attempt requires a fresh checkout, not global artifact removal. The build serializes that checkout, forces matching-native Go/local Podman, builds application commands and separate support tools, builds the Go API/OAuth dashboard command in the same command loop, bundles the existing Cockpit payload, builds Tea, and builds the Rocky project/dashboard images once from those outputs. There is no standalone Soda frontend build or asset placeholder; Cockpit's frontend build remains. It resolves the unchanged Forgejo/Caddy references for the selected platform, saves all four archives explicitly as OCI, records public native dependency/package/CLI metadata, stages the core configuration, inspects ELF/OCI identity and seals the payload. Read-only, network-disabled image-inspection containers are part of this build recipe, not project lifecycles. No publication, install, VM or product test follows automatically.

Run `scripts/check-native.sh x86_64` separately. Export the verified allowlist with the built `tools/soda-artifacts bundle` command as shown in [support recipes](native-support.md#build-and-artifact-contract); do not transfer the entire build tree.

## 2. Provision the upstream host

Candidate: Fedora CoreOS stable 44.20260817.3.2. Use its upstream installer/image matching the target architecture and native install instructions. Do not add a separate Soda distribution/release pipeline.

For the on-media console recipe, see [CoreOS installation media](coreos-installer.md).
It reuses the bootstrap/bundle/setup contracts below; it is not an offline appliance
image or permission to reinstall an existing host.

Produce private provisioning input using operator-selected files:

```sh
umask 077
scripts/render-provisioning.py --operator-key-file /secure/operator.pub \
  --root-password-hash-file /secure/root.hash --out /secure/soda.bu
/path/to/soda-artifacts convert-butane --arch x86_64 \
  --source /secure/soda.bu --out /secure/soda.ign
```

Use a real mode-0700 parent and new absolute output paths; private inputs/outputs are restricted regular files. The public bootstrap is `appliance/provisioning/base.json`. For fresh QEMU fixtures, the [support guide](native-support.md#fresh-vm-contract) adds a unique hostname and a pre-pinned per-instance SSH host key, verified against one complete `fw_cfg` Ignition input; do not assume automatic disk/fragment merging.

The root password hash is a native crypt(3) hash supplied by the operator, not plaintext or a Forgejo password. Treat both generated provisioning files as secrets. The input configures only operator host access and a one-time native extension install; it never creates developer host accounts.

Use the generated Ignition file with the upstream `coreos-installer` on the **explicitly authorized installation disk/VM**. Disk selection and permission to erase/install are not supplied by this document. The first boot's `soda-extensions.service` requests Cockpit, Tailscale, native Forgejo runner and their host dependencies through rpm-ostree. Inspect its journal and `rpm-ostree status`, then reboot explicitly to activate the extensions. The service does not automatically reboot or implement a Soda updater.

The Tailscale repository is the upstream Fedora stable repository. Native package closure, especially Cockpit/Forgejo runner/.NET dependencies, must be observed and corrected against the chosen deployment; inherited package source evidence is not installed proof.

## 3. Install the built components

Transfer only the sealed matching deployment bundle through the pinned `soda-acceptance transfer` path or another explicitly approved trusted transfer. Establish its `SHA256SUMS` identity over that trusted channel before executing any bundled program. The inventory identifies each delivered file; it is not a signed release. No publication is needed. Choose a **non-overlapping private IPv4 subnet** for project IPs, for example `10.89.0.0/24` only if suitable for that deployment.

```sh
sudo /path/to/bundle/x86_64/install-native.sh /path/to/bundle/x86_64 10.89.0.0/24
```

This is an explicit first-install operation, not an updater. It stages native files, establishes the dedicated service identity and non-conflicting Podman subordinate range, loads the four verified image archives and restores the existing core service references, and starts loopback-only Forgejo/Cockpit plus the root:soda helper socket and native Tailscale daemon. It does not enroll in a Tailnet, create provider runners or activate public browser endpoints. It refuses blind reinstall over existing Soda configuration or a retained `/etc/soda/install-started` marker. Bundle/platform/package/subnet/identity preflight precedes payload writes; partial installation still requires an operator recovery decision, not deleting the marker and retrying as if clean.

Follow [operator setup](operator-setup.md). The native Forgejo installer is initially accessible only over an operator SSH tunnel to port 3000. `soda-setup` uses the resulting operator API token to create its actual OAuth application/configuration. Current source uses one Forgejo/Sodaspaces browser origin with Soda routes under `/-/soda/`; `public_url` and `--public-url` are removed. The isolated browser/proxy proof does not establish deployment to an appliance.

For a private IP browser origin, the replacement source supports Caddy's native
local issuer without an owned domain:

```sh
sudo /usr/local/sbin/soda-activate --bind-ip PRIVATE_APPLIANCE_IP --local-tls
```

The configured HTTPS origin must use that same private IP. Explicitly copy and
verify the appliance's public CA certificate over trusted SSH, then trust it on
intended clients; see the [guided setup](coreos-installer.md#configure-private-browser-access).
Automatic trust installation is disabled. A completed command is not a browser
login or reachability test. Existing certificate-based deployment remains available:

```sh
sudo /usr/local/sbin/soda-activate --bind-ip PRIVATE_APPLIANCE_IP \
  --certificate /secure/browser-cert.pem --private-key /secure/browser-key.pem
```

Activation applies file ownership for the unprivileged dashboard, retains operator-only native access, binds Caddy and Forgejo Git SSH to the selected private IP, and starts the actual services. If Tailnet Git access is intended, enroll through operator Cockpit before activation and select that Tailnet private IP; later advertisement refresh refuses to substitute a Tailnet address while Git SSH only binds a LAN IP. Configured browser origins must resolve through the deployment's normal browser/network setup; this is unrelated to project SSH, which uses project IPs directly. Native Forgejo Git SSH uses port 2222; project SSH uses each project IP's port 22.

First activation also derives the local [Soda avatar provider](avatars.md) from
the configured Forgejo origin. Enable provider avatars and disable federation
through Forgejo's native administration settings. Saved database-backed settings
and explicit offline mode are not silently overridden by activation.

### Sodaspaces customization delivery

The earlier merge's partial-payload hold is replaced in source by
`internal/nativebuild/forgejo-payload.json`: an exact source/destination inventory
shared by staging and the embedded Go verifier. It includes all 229 selected template
overrides, shared presentation assets/fonts/notices, the mounted Sodaspaces content/
terminal and five locked renderer/CSS/MIT-notice files, beneath
`/var/lib/soda/forgejo/gitea/`. See the [mounting contract](terminal-integration.md).
Adding an arbitrary template is still refused; expand the reviewed inventory explicitly.
Clean native build/check/export passed at `dad2945`, including public-mode
normalization for private checkouts. The preceding `2aa4960` application/helper
passed bounded installed integration on the isolated fixture; see the handoff for
its exact bytes and recorded legacy public-mode differences. The newer whole
bundle is not an installed-appliance or retained-rollout result.

The build fetches locked terminal distributions into `terminal-assets` and verifies
Forgejo 15.0.7's complete English catalog via `appliance/forgejo/locale.lock.json` before
adding the Soda-only namespace into `forgejo-locales/locale_en-US.ini`. It never installs
a partial replacement catalog or downloads at runtime. Stage checks locked renderer
bytes again; bundle verification requires exact files, modes and LICENSE/NOTICE.
Files are 0644, new readable directories 0755. The installer applies UID/GID 1000 only
to the admitted new files/directories, not recursively to a mutable Forgejo tree.
First-install preflight refuses occupied hook/asset destinations, including
symlinks, before host writes. Resolve conflicts explicitly, never merge or overwrite
operator hooks automatically. This is not an upgrade interface. Actual CustomPath,
labels, reload requirements and browser behavior passed bounded native first-delivery
proof at `bdbce8e`; retained-target cutover and whole-appliance acceptance remain separate.

### Existing-state dashboard migration

The current source dashboard requires `grant_key_file` and schema v9, retaining the
session-grant encryption introduced in v3. Do not run first-install or OAuth bootstrap again on an existing target.
Follow the [controlled credential migration and rollback procedure](dashboard-credentials.md),
including a consistent SQLite backup, matching config/key/artifact set and
separately approved rehearsal/deployment. A missing or wrong key fails closed;
a prior binary is not assumed compatible with the new schema.

### Retained Sodaspaces cutover

This is the bounded affected-component procedure for the retained `soda-test`,
**not a general upgrade tool or permission to replay maintenance**. Fresh delivery,
copied private v3 → v5 / paired rollback and the separately approved retained cutover
passed; see the [handoff](implementation-status.md#approved-retained-cutover). Any new
maintenance still needs its own exact scope and current backup.

1. Approve the exact candidate/configuration and short Soda/Forgejo/proxy interruption.
   Record installed image IDs and effective units, not assumed `:dev` tags: the retained
   dashboard Quadlet is image-pinned. Inventory the four roots, their containers/images/
   running state and original membership logins. Stop if identities or customizations
   differ; do not force an old fixture inventory onto later writes.
2. Quiesce Soda writes and stop only the old Soda service for a **new** consistent
   SQLite backup and matching config/key/credential/artifact/unit/proxy/custom-file set.
   The rehearsal backup is not current rollback data. Retain prior callbacks through
   the actual owner's supported interface, file owners/modes/labels and the prior image.
   Do not freeze, snapshot-restore or restart project workloads to manufacture equality.
3. Verify the delivered bundle and stage the image by its actual config digest, plus
   matching `soda-dashboard` and strict-config `soda-runners` binaries. The unchanged
   helper, project image/default, project services, Cockpit and runner services are not
   upgrade targets. Update the effective dashboard Quadlet's single image pin and
   reload systemd; do not rely on loading an image to change a pinned unit.
4. Preserve all credentials, native Forgejo origin/client, service UID, socket and
   stored identities. Remove only legacy `public_url` from the prepared Soda config.
   Deliver the namespaced Caddy recipe and remove only `SODA_ORIGIN` from `proxy.env`,
   keeping `FORGEJO_ORIGIN`, private bind and TLS. Do not run first-install/setup/
   activation recipes or generate a new grant key/OAuth application.
5. Before any custom-file write, inspect the exact four hooks/assets and all ancestors,
   including CustomPath, types, ownership and labels. Refuse unexpected occupants;
   preserve operator changes. Install only the approved header/footer/CSS/JS, with
   0644 files and readable new directories; no recursive mutable-tree chown. Retain
   native theme/cache settings. Apply the reviewed query-free native logging settings
   without replacing unrelated Forgejo configuration; general upstream logs remain.
6. As the application's actual owner, use native Applications settings to update only
   the intended callback to `FORGEJO_ORIGIN/-/soda/oauth/callback`, preserving unrelated
   registered callbacks, name, confidential-client setting and secret. Retained app 4
   currently has only the old Soda callback. **No API PATCH or secret regeneration.**
   Verify the owner-visible client/callback metadata, and retain the exact prior list
   for a reviewed rollback decision. Do not edit Forgejo's DB.
7. Restart only the affected Forgejo/proxy/Soda services once their paired inputs are
   ready. The correct key is checked before the backend migrates v3 → v5. Before new
   browser login, verify integrity/FKs and preservation of original profile/key/
   project/membership/session/grant columns and ciphertext. Run native runner `list`
   with `{}` input, without registration/jobs or restarting runner services.
8. Verify the running backend/image, custom bytes and native query-free logging.
   Verify the unchanged Forgejo origin and native routes/protocols, the new scoped
   namespace/cookies and removed separate Soda listener. Users explicitly sign in
   again; old cookies/pending OAuth are not imported or replayed. Use the declared
   private-repository read-only journey for the retained private repository, not a
   visibility change, with real OAuth/identity/blur/BFCache/native-form observations.
9. Compare all four root/container identities and original membership logins. Observe
   own existing connection/host-key values and actual own-key SSH from a recorded
   client path; do not add keys, join/create/start environments or change routing to
   conceal a failed observation. Record the precise client, payloads and outcomes.

On failure, retain candidate DB/WAL, credentials, partial delivery and all later writes.
Do not run an old binary on v5, lower a schema marker or automatically overwrite data
with a rehearsal snapshot. Rollback needs an explicit review of the matching prior
DB/config/key/image/unit/callback/custom-file set **and** subsequent writes. Stop for
that decision rather than inventing repair/reconciliation or deleting project roots.

## 4. Establish real project reachability

The implemented profile is a native routed Podman bridge (`soda0`) on the appliance. Host-to-project access is through that bridge; developer clients need a route for the chosen project subnet via the appliance. Set that route on the deployment's LAN router, or use a native Tailscale subnet route with the required Tailnet administrator approval. Respect existing firewall policy and authorize only the intended private ingress/forwarding. No project DNS, SSH gateway or extra identity authority is required.

Host Tailnet enrollment by itself does not route the project subnet. Port-forward-only access to a builder VM is not proof that real developer clients can reach project IPs. Verify the actual routing/firewall setup in the product journey; do not call a Podman-only address usable because it appears on the dashboard.

## 5. Operator services and state

Cockpit initially binds loopback 9090; use an operator SSH tunnel unless private native access is deliberately configured. Its PAM policy permits only root. Tailnet/Runners remain operator pages. The dashboard runs as native service UID/GID 2000 with no host capabilities and reaches only the restricted project helper socket; secret files are root:soda 0640, the database directory soda-owned 0700. The helper is root and exposes only fixed project operations over that Unix socket, never a public control listener.

Project environments are created once and started/stopped as existing containers. Do not run `podman system prune`, delete project containers, use `--rm`, or replace their writable roots as ordinary management. That root stores account records, installed packages/tools and service data. Backups/recovery and automatic image replacement remain deferred.

Required new native support, such as tmux, follows the [Project OS same-root
maintenance contract](project-os.md#deliver-required-additions-without-replacing-roots).
New image builds/defaults do not update retained roots. The handoff records one
exact isolated-root tmux transaction with native signature/dependency checks and
bounded browser proof; the user waived backups only for that target. Other
package/file/service transactions still need their own compatibility review and
applicable target/action scope; first-install/activation are not those recipes.

Run the later [native validation guide](native-validation.md) only with explicit targets and permissions. Installation/activation command success is not validation. Keep source, build and installed evidence separate.
