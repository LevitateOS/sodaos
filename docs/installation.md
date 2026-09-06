# Native build and installation

Native execution has begun on the local x86_64 builder and an isolated CoreOS VM; see [local testing](local-testing.md) for observed results and remaining gaps. Use the actual authorized matching-native Linux builder and selected appliance target. A build or boot does not establish usable end-to-end development environments.

The [leading core plan](dashboard-implementation-plan.md#coordination-with-native-support-porting) owns application payload/assets, configuration, credentials, migrations and cutover. The subordinate [native support plan](native-porting-plan.md) supplies artifact inspection/bundling and provisioning transport around those contracts. Its ISO/QCOW2 wrappers are conditional proposals, not implemented or required for core delivery. The [support source and recipes](native-support.md) have now participated in the sealed x86_64 `8417a90` build; their remaining native/architecture limits are recorded in the [implementation status](implementation-status.md). The commands below retain the existing installation path with sealed bundles and private provisioning; do not assume a Soda host OCI, installer ISO or preinstalled QCOW2 is available, or rerun first-install as a dashboard migration.

## 1. Prepare the native builder

Use x86_64 first when access exists; repeat independently on aarch64 later. Install Go 1.26.7, Node 24.20.0, pnpm 11.25.0, Python >=3.12, GNU make and native Podman through the builder's normal mechanisms. Do not cross-compile/emulate and report native evidence.

The React preview has a real resolved `dashboard/pnpm-lock.yaml`, exercised by the native build. Both build entrypoints require it; do not fabricate metadata or silently reuse Cockpit's lockfile. See [dashboard build/migration notes](../dashboard/README.md). Dependency resolution, compilation and deployment still need their applicable authorization.

Real Go dependency metadata was resolved during the first native x86_64 build and is now in `go.mod`/`go.sum`. For intentional dependency changes, run `go mod tidy` and review the resulting metadata. Cockpit's predecessor dependency lockfile is retained. Then invoke:

```sh
scripts/build-native.sh x86_64
```

Use a clean exact-revision checkout with fresh `.artifacts/native/x86_64` output; another attempt requires a fresh checkout, not global artifact removal. The build serializes that checkout, forces matching-native Go/local Podman, builds application commands and separate support tools, invokes the core-owned `scripts/build-dashboard.sh ARCH --payload-only` for React assets and the dashboard binary, bundles the existing Cockpit payload, builds Tea, and builds the Rocky project/dashboard images once from those outputs. React assets are staged with their Go binary; there is no second frontend build or missing-asset substitute. It resolves the unchanged Forgejo/Caddy references for the selected platform, saves all four archives explicitly as OCI, records public native dependency/package/CLI metadata, stages the core configuration, inspects ELF/OCI identity and seals the payload. Read-only, network-disabled image-inspection containers are part of this build recipe, not project lifecycles. No publication, install, VM or product test follows automatically.

Run `scripts/check-native.sh x86_64` separately. Export the verified allowlist with the built `tools/soda-artifacts bundle` command as shown in [support recipes](native-support.md#build-and-artifact-contract); do not transfer the entire build tree.

## 2. Provision the upstream host

Candidate: Fedora CoreOS stable 44.20260817.3.2. Use its upstream installer/image matching the target architecture and native install instructions. Do not add a separate Soda distribution/release pipeline.

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

Follow [operator setup](operator-setup.md). The native Forgejo installer is initially accessible only over an operator SSH tunnel to port 3000. `soda-setup` uses the resulting operator API token to create its actual OAuth application/configuration. Supply valid, browser-trusted TLS material covering the distinct Soda and Forgejo origins, then:

```sh
sudo /usr/local/sbin/soda-activate --bind-ip PRIVATE_APPLIANCE_IP \
  --certificate /secure/browser-cert.pem --private-key /secure/browser-key.pem
```

Activation applies file ownership for the unprivileged dashboard, retains operator-only native access, binds Caddy and Forgejo Git SSH to the selected private IP, and starts the actual services. If Tailnet Git access is intended, enroll through operator Cockpit before activation and select that Tailnet private IP; later advertisement refresh refuses to substitute a Tailnet address while Git SSH only binds a LAN IP. Configured browser origins must resolve through the deployment's normal browser/network setup; this is unrelated to project SSH, which uses project IPs directly. Native Forgejo Git SSH uses port 2222; project SSH uses each project IP's port 22.

### Existing-state dashboard migration

The current dashboard requires `grant_key_file` and schema-v3 encrypted session
grants. Do not run first-install or OAuth bootstrap again on an existing target.
Follow the [controlled credential migration and rollback procedure](dashboard-credentials.md),
including a consistent SQLite backup, matching config/key/artifact set and
separately approved rehearsal/deployment. A missing or wrong key fails closed;
a prior binary is not assumed compatible with the new schema.

## 4. Establish real project reachability

The implemented profile is a native routed Podman bridge (`soda0`) on the appliance. Host-to-project access is through that bridge; developer clients need a route for the chosen project subnet via the appliance. Set that route on the deployment's LAN router, or use a native Tailscale subnet route with the required Tailnet administrator approval. Respect existing firewall policy and authorize only the intended private ingress/forwarding. No project DNS, SSH gateway or extra identity authority is required.

Host Tailnet enrollment by itself does not route the project subnet. Port-forward-only access to a builder VM is not proof that real developer clients can reach project IPs. Verify the actual routing/firewall setup in core U08 (the current owner of the historical M16 proof); do not call a Podman-only address usable because it appears on the dashboard.

## 5. Operator services and state

Cockpit initially binds loopback 9090; use an operator SSH tunnel unless private native access is deliberately configured. Its PAM policy permits only root. Tailnet/Runners remain operator pages. The dashboard runs as native service UID/GID 2000 with no host capabilities and reaches only the restricted project helper socket; secret files are root:soda 0640, the database directory soda-owned 0700. The helper is root and exposes only fixed project operations over that Unix socket, never a public control listener.

Project environments are created once and started/stopped as existing containers. Do not run `podman system prune`, delete project containers, use `--rm`, or replace their writable roots as ordinary management. That root stores account records, installed packages/tools and service data. Backups/recovery and automatic image replacement remain deferred.

Run the later [native validation guide](native-validation.md) only with explicit targets and permissions. Installation/activation command success is not validation. Keep source, build and installed evidence separate.
