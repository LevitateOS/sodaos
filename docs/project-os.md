# Project OS baseline

The foundation for the selected development workspace, **not a new distribution or
a design for every possible workload**. Keep the existing persistent Rocky + mise
userspace and ordinary Linux extension points. The [Sodaspaces plan](sodaspaces-plan.md)
owns implementation order; this guide consolidates the project contracts and actual
gaps. [Terminal](terminal-integration.md), [development](development-environment.md),
[services](project-services.md) and [CLI](project-clis.md) guides own their details.

The earlier “unbuilt/unvalidated” description is obsolete. There are native builds
and bounded x86_64 account, shared-tool, nested-workload and persistence results;
[the handoff](implementation-status.md) owns their exact bytes/targets. Tmux sessions,
key-free browser onboarding and their combined workflow are still unimplemented.
This baseline is documentation, not a new build, installed inventory or acceptance.

## Ownership and trust

| Owner | Responsibility |
| --- | --- |
| Appliance/operator | CoreOS/kernel, host Podman/network/storage, restricted helper and approved delivery of required platform integration. No human developer host accounts. |
| Project OS | Systemd PID 1, native accounts/groups, SSH, shared directories/tools, project-local workload engine and the selected future terminal supervision. |
| Soda backend | Stable repository/project/user associations, original membership login, browser authorization and bounded terminal lifetime. Not native passwords, copied Forgejo permissions or a general guest-management agent. |
| Developers/project administrators | Personal checkouts/preferences/credentials; native shared package/tool/service administration through the existing project-local permissions. Forgejo owns Git/collaboration authority. |

This is a **trusted-team container sharing the host kernel**, not hostile-tenant
isolation or a VM. Project root/wheel administrators can read every project home and
credential and change native state. Unix file permissions protect against ordinary
peers, not project sudo/root or appliance root. Never represent a personal credential
in a project as hidden from its administrators.

New members get Bash, a real home and `soda-project` group membership. The stable
identity marker/original login binds the account; unassociated account collisions
refuse instead of adopting a Linux user. The current account helper grants wheel
when the stable user ID matches the container's
**creation-time owner label**. Wheel grants passwordless project-local sudo and access
to the rootful nested engine. Soda's current Forgejo-backed lifecycle authorization
is separate: a repository transfer does not automatically grant/revoke Linux wheel,
remap accounts or remove already installed SSH keys. Do not claim that current web
ownership and native sudo membership are synchronized; preserve existing rights and
report that limitation rather than silently reconciling them.

Source owners: [`internal/host/daemon.go`](../internal/host/daemon.go),
[`project-account`](../project-os/rootfs/usr/libexec/soda/project-account),
[`environment_authority.go`](../internal/web/environment_authority.go) and
[`sudoers`](../project-os/rootfs/etc/sudoers.d/soda-project).

## Supported userspace and ordinary extension points

- Keep Rocky's native package/signature mechanisms, systemd, OpenSSH, Git, Bash and
  OS Python. Dependency versions belong in [the Containerfile](../project-os/Containerfile)
  and existing locks, not a second version table here. A Rocky base tag does not freeze
  rolling RPM repositories: retain the resolved digest and actual package inventory.
  System helpers use the OS interpreter/absolute paths; project-selected language
  versions must not replace it.
- The small interactive baseline includes a UTF-8 login environment, file/text/process
  tools, a terminal editor (`vi`/`vim-minimal`), pager (`less`), CA certificates and
  working terminal descriptions. The saved `dad2945` x86_64 build RPM inventory lists
  those editor/pager/CA packages, but not tmux. That is a past image inventory, not a
  current survey of retained roots. Make required tools explicit in recipe/check
  ownership rather than relying indefinitely on transitive base-image contents.
- Add the selected **tmux** package and bounded native supervision for browser
  terminals; do not swap in a new web IDE or multiplexer backend. Bash remains the
  initial supported login shell. Preserve personal dotfiles, `EDITOR`/`VISUAL` and
  native editor choices; do not force ordinary SSH logins or personal tmux servers
  into Soda's managed sessions. Other shells need their native startup configuration,
  not a promise that Bash's profile automatically configures every shell.
- Shared mise installations/shims live in `/opt/mise`, with global configuration in
  `/etc/mise/config.toml`. Project administrators install shared versions; ordinary
  members consume the same installed bytes. Shell profiles and SSH `SetEnv` provide
  the native paths for interactive and noninteractive access. The tmux login must
  preserve these effective settings and real groups, not inherit backend secrets.
- Keep normal personal Git clones, repository `mise.toml`, explicit trust decisions,
  language package managers/virtual environments and native Compose/systemd files.
  Users may install personal tools in their homes within ordinary permissions;
  separate personal mise storage requires explicit native environment configuration,
  not a Soda selector. Shared OS packages/build prerequisites need project-admin
  installation. Do not preinstall every compiler, editor, AI tool or database.

GUI/GPU/device access, kernel-dependent workloads and every third-party tool are not
implied compatibility promises. A concrete new need gets its own native compatibility
review; no privileged-parent shortcut, arbitrary host device/mount or automatic
capability expansion. This is extensibility through Linux, not a plugin/OS-profile
catalog or private-resource branching system.

## Persistent state and lifecycle

| State | Contract |
| --- | --- |
| Account databases, `/var/lib/soda/accounts/`, homes | Preserve original identities, UID/GID/groups, checkouts, dirty/untracked files, user configuration and private inputs in the existing writable root. No rename/remapping or checkout reset on Open/Start. |
| `/etc/ssh/ssh_host_*`, `/etc/ssh/authorized_keys/<login>` | Preserve host identity; managed inbound public-key files stay root-owned and change only through explicit account/key actions or deliberate native administration. Private host keys never enter browser responses/evidence. |
| `/srv/project/shared`, `~/shared` | Real shared files, root:`soda-project` setgid directory (2775) and personal symlink. Setgid inherits the group, not group-write permission for every file: use deliberate shared-file modes/umask, not globally weakened home permissions. |
| `/opt/mise`, `/etc/mise`, installed packages/configuration | Real persistent installations/configuration, not merely a shared download cache. Native administrator changes persist; Git merges do not promote them into live state. |
| `/var/lib/containers/storage` and native workload state | Persistent nested images, writable layers, named volumes and native secrets. Application-consistent backup/restart remains a separate operation, not an effect of terminal reconnect. |
| `/run`, PTYs, sockets, processes | Runtime state, not durable project data. Startup recreates needed runtime directories; no terminal-process resurrection or durable tmux transcript is selected. Do not use temporary paths as durable storage. |

- **Create** reserves/provisions/starts the shared container once; it does not join
  the creator or clone a repository. **Join** provisions the real account and records
  membership only after confirmed success. Failed/unconfirmed operations are not
  automatically replayed.
- **Open/Refresh/navigation/Hide** never create, join, start, repair or update a project.
  The tmux contract will preserve/reattach its selected authorized session; today's
  request-owned PTY still does not survive navigation/transport loss.
- **End terminal/terminal expiry** will end that managed terminal's supervised
  processes, not delete files or stop independent SSH/services/workloads. Anything
  deliberately launched outside that service follows its own native lifecycle.
- **Stop** disables host-boot start and stops the existing container, interrupting
  everyone's processes. **Start** enables/starts that same container. The host unit
  has failure-restart policy, but it never recreates the container. Project/host
  restart preserves files/identities, not running process memory. Enabled services
  follow native startup policy; existing nested workloads in the recorded lifecycle
  proof were explicitly started, not automatically resurrected. No idle project stop.

At each boot, [`project-init`](../project-os/rootfs/usr/libexec/soda/project-init)
reasserts the named managed-directory modes/ownership, prepares the bounded
network-sysctl mount and generates only missing SSH host keys before marking
`/run/soda-project-ready`. It does not rebuild accounts or reset checkouts. The marker
is initial setup completion, **not** proof of SSH reachability, account provisioning,
application health or tmux readiness. Retaining the root is not backup against lost
host storage or protection from ordinary native writes.
[`soda-project@.service`](../appliance/services/soda-project@.service) and
[`management.go`](../internal/host/management.go) own existing-container lifecycle.

## Access, credentials and connectivity

Browser terminal access uses Soda's authenticated bridge and the existing Linux
account, **not SSH or a device private key**. The intended browser-only Join must
allow zero external SSH keys without enabling password/root SSH. Today the API,
helper and account script still require a nonempty key set; changing only the UI is
insufficient. Optional device → project SSH keeps explicit public-key review/apply,
key-only OpenSSH, SSH/SCP/SFTP and independent host-key trust. No automatic profile-
key import, later synchronization or termination of unrelated authenticated SSH.
Explicit Soda logout/expiry and confirmed authority loss remain hard browser-access
boundaries, not Linux account deletion or atomic Forgejo/SSH logout. Hiding a drawer
or changing focus is not logout; the terminal guide owns finite retention policy.

Project → Forgejo Git and Tea/gh authentication are separate personal credentials,
not borrowed Soda grants or inbound SSH keys. The proposed per-user/project Git key
stays in the original home; it still carries that person's normal cross-repository
Forgejo permissions and is readable by project administrators. Consent, at-rest/
passphrase handling and `write:user` authorization remain decisions **before automated
Git setup**, not prerequisites for an existing-account terminal. Follow the
[credential direction](sodaspaces-plan.md#ssh-directions-and-the-proposed-git-setup).
Project Git/CLI private material must not enter images, shared files, argv, transcripts
or Soda's database. Soda's own encrypted OAuth grants remain a separate backend-only
credential; never export them into a project.

Use actual reachable endpoints and independently trusted host keys/CA roots. Trust
in the browser does not install trust inside the project; never copy appliance TLS
private keys into it or disable verification. Direct development access is
`user@project-ip`, not project DNS or an SSH gateway. Project IP can change after
Start; identity is the project/container/account and trusted host key, not the IP.
Browser HTTPS/terminal connectivity does not prove laptop access to project HTTP,
database or SSH ports. Private client routes/authorized native forwarding remain
[deployment responsibilities](installation.md#4-establish-real-project-reachability);
no automatic port proxy or public ingress follows from this baseline.

## Runtime and supervision boundaries

Keep the single [nested Podman profile](project-services.md): the outer project has
shifted user IDs, private namespaces, bounded selected capabilities and `/dev/fuse`,
with disabled container SELinux labeling and the host's default seccomp policy.
No new isolation/security guarantee follows from owning the userspace. Inside,
the root:root engine exposes only its root:wheel project socket; ordinary members
consume service endpoints, not project-root engine access. Inner `--network host`
means the project network namespace, not the appliance's.

**Three different process owners:** the host unit manages the whole container;
project systemd manages its native services; nested Podman manages workloads with
inner workload cgroups disabled. That last setting is not proof that tmux's proposed
project-systemd service can or cannot enforce its own cgroup cleanup. Verify it on
the actual profile. Do not copy the engine's `KillMode=process` or the outer unit's
`KillMode=none` into terminal supervision, or change capabilities/inner cgroups to
hide a failed test. The [terminal contract](terminal-integration.md#bounded-native-ownership)
requires an owned service/cgroup plus an independent safety lease, not just a tmux
client that leaves a detached server behind.

Current creation sets no explicit per-project CPU/memory/disk quota policy; effective
engine/host limits may still apply. Terminal/session/IO bounds are not appliance
capacity isolation. No resource scheduler, quota UI or GPU/device broker is selected.

## Deliver required additions without replacing roots

**Selected approach: narrowly scoped native maintenance of the same project**, not
an automatic image updater. A build creates an image for future project creation;
loading/retagging it or changing the helper's image default does not patch existing
containers. Existing roots can legitimately differ from each other and the newest
image. Do not introduce an image-ID-equality gate or silently adopt/normalize them.

For a required addition such as tmux, the product change must include fresh-image
packaging **and** a reviewed existing-root maintenance recipe:

1. Resolve the exact project/CID, original accounts, native creation profile,
   installed required packages/files and currently running work. Declare the required
   helper/API/project-file compatibility and exact package transaction, including
   dependencies, scriptlets, config changes and any service interruption. No blanket
   `dnf upgrade`, capability retrofit or implicit install on Open/Start.
2. Obtain the applicable target/action scope and a fresh matching backup of project
   state/package configuration and any affected Soda DB/config/key/artifacts. Include
   the necessary quiescence explicitly. An old application-only backup cannot roll
   back project homes, RPM state or workload data, and an old snapshot cannot erase
   subsequent work. This is scoped maintenance, not a new backup/recovery platform.
3. Use native verified packages and exact Soda-owned file updates. Admit new paths
   or explicitly reviewed prior Soda files; refuse unsafe ancestors, unexpected
   occupants or modified/operator files rather than merge, recursively chown homes
   or copy the whole image rootfs over a mutable project. Preserve accounts, host
   keys, developer keys/credentials, native configuration and unrelated workloads.
4. Reload/restart only the specifically authorized affected services. Prove the new
   operation against the original account and confirm existing-root/state/access
   preservation. Missing native support must report unavailable; no disposable-PTY
   fallback, auto-recreation or new privilege to claim compatibility.
5. Retain partial failures and later writes; do not retry/rollback automatically.
   If the existing creation profile cannot support the feature without replacement,
   stop for a decision rather than migrate IDs/storage or recreate it in secret.

The recipe/compatibility checks are **not implemented yet** and do not authorize
maintenance. Keep `scripts/build-native.sh`, native build metadata, staging/verifier
and [installation](installation.md) as production owners. Do not use first-install
or activation as upgrade tools. Project administrators own their ordinary native
tool/service changes; required platform additions/security maintenance need deliberate
operator/project coordination. No automatic fleet patching is currently supplied.

## Concrete gaps and next implementation

| Gap | When it matters |
| --- | --- |
| Tmux package, project-local supervision/lease and same-process reattachment | Immediate single-terminal slice; verify native cgroup ownership, login/terminfo/editor behavior and failure cleanup before calling it resumable. |
| Required-tool checks and same-root package/unit delivery | Part of that feature, not a separate OS platform. Saved RPMs are not proof every retained root has the required commands; maintenance is a gate to existing-target delivery, not to source work. |
| Zero-key real account provisioning and explicit Forgejo public-key selection | Browser-only onboarding slice; preserve current accounts/key files and password-SSH denial. Not a prerequisite for testing tmux with an existing member. |
| Personal Git credential trust/consent/passphrase choice and native endpoint trust | Resolve before automated Git setup/combined clone-edit-build-push proof; do not fabricate credentials or treat profile keys as repository-scoped. |
| Current Forgejo authority versus already issued Linux sudo/SSH rights | Explicit limitation on rename/transfer/offboarding claims; no automatic native permission reconciliation is selected. |

The existing foundation does not justify a new OS backend or universal-workstation
planning phase. Continue with one resumable terminal, then onboarding and the real
combined workflow. Extend the existing [product validation](native-validation.md)
entrypoints for fresh and maintained projects; package lists, source tests or socket
closure are not native continuity/preservation proof. Broader operator/client and
native aarch64 acceptance stay independent. [Deferred scope](deferred.md) remains
closed: no private-resource selectors, generalized recovery/deletion, release/update
platform or speculative host fallback.
