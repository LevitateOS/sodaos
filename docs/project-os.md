# Project OS baseline

The foundation for the development workspace. The current implementation is the
persistent Rocky + mise userspace with ordinary Linux extension points. The user
has selected the bounded Rocky/Fedora and headless/KDE expansion below; GNOME is
deferred. This does not replace existing roots. The [Sodaspaces plan](sodaspaces-plan.md)
owns implementation order; this guide consolidates the project contracts and actual
gaps. [Terminal](terminal-integration.md), [development](development-environment.md),
[services](project-services.md) and [CLI](project-clis.md) guides own their details.

The earlier “unbuilt/unvalidated” description is obsolete. There are native builds
and bounded x86_64 account, shared-tool, nested-workload and persistence results;
[the handoff](implementation-status.md) owns their exact bytes/targets. Managed tmux
is implemented with bounded isolated reload/cleanup proof; broader native safety/UX,
key-free browser onboarding and the combined Git workflow remain incomplete.
This baseline is documentation, not a new build, installed inventory or acceptance.

## One foundation for every profile

This guide remains the Project OS authority. Distribution and desktop choices extend
the already planned environment; they do not create independent vanilla machines,
new account systems or a second development product. The responsibilities remain:

| Contract | Owner |
| --- | --- |
| Product order, onboarding and Git-credential decisions | [Sodaspaces plan](sodaspaces-plan.md) |
| Accounts, native permissions, shared installations, persistence and maintenance | This Project OS guide |
| Personal checkouts, editors, shared mise and ordinary developer workflows | [Development environment](development-environment.md) |
| Native workload engine, service permissions and project networking | [Project services](project-services.md) |
| Tea/gh packaging and personal provider authentication | [Project CLIs](project-clis.md) |
| Managed terminal identity, supervision, leases and cleanup | [Terminal integration](terminal-integration.md) |
| Shared page/drawer views and rendering | [Spaces design](spaces-design.md), [drawer design](spaces-drawer-design.md), [Lit sequence](lit-migration-plan.md) |
| Service catalog and issue/PR automation integration | [Services and AI plan](services-and-ai-plan.md) |

A desktop session belongs to the existing project-local account. Terminal, desktop
apps and ordinary SSH use that account's real home, checkouts, permissions and shared
tools. Desktop launch must establish the correct user session/environment, including
mise paths; opening an app must not silently create another home, clone, tool store
or credential identity. Per-user graphical sessions must not expose peers' displays
or browser credentials; project sudo/root retains the documented administrator trust.

A desktop is not a large terminal lease. It has its own desktop ID, display socket,
input lease, runtime directory and graphical process scope. Ending that scope must not
end the account's managed tmux terminals, ordinary SSH sessions, user service manager,
native services or nested workloads. Conversely, a retained terminal neither retains
nor authorizes a desktop. The first desktop slice is account-owned: one graphical
session for one `(project, original login)`. Another Linux account cannot observe it;
duplicate browser views from its creating Soda sign-in context may observe the same
display, but exactly one holds keyboard/pointer control.

Installed OS packages, native service data and shared tools belong to the persistent
project root. A KDE profile adds the graphical session and its access transport to
that foundation. Forgejo still owns identity/Git authority; Soda binds authorized
viewers to exact native targets. No blank VM or desktop image alone completes a
Project OS profile.

The existing onboarding, credential, CLI, reachability and native-proof gaps remain
work. Keep their evidence and priority; profile expansion must neither reset completed
work nor describe those gaps as solved. Read this guide and the relevant owner above
before extending images, runtime, desktop sessions or automation environments.

## Batteries included by default

Every supported profile must arrive as a complete working Project OS. Soda owns
installation and integration of the non-preference foundation; users must not
assemble missing accounts, tools, desktop plumbing or workload support themselves.
This supersedes the earlier minimal-tool interpretation. A small image is not a
goal when it leaves an ordinary supported workflow incomplete.

| Supplied and wired by Soda | Remains a user or repository choice |
| --- | --- |
| Native account/home/group setup, locale, shell startup, permissions, CA trust and persistent directories | Personal dotfiles, preferred shell, locale and appearance overrides |
| Working terminal, tmux, basic editor/pager, file search, archives, transfer and process/network diagnostics | Preferred editor/IDE, extensions, keybindings and terminal theme |
| Git, packaged forge CLIs, shared mise, standard native build/debug tools and development prerequisites | Repository language/runtime versions, dependency graph, build/test commands and trust decisions |
| Native workload engine, Compose support, service supervision, storage and usable access integration | Which databases/services to run, their versions, configuration and data |
| On KDE: functioning user desktop/session, display/input transport, fonts, clipboard integration, file manager, basic graphical editor, browser and credential-storage integration | Preferred applications, browser profile, extensions, desktop customization and personal accounts |
| Required dependencies and integration for every offered optional tool/app | Whether to install/use that tool, AI provider/model, subscription, credentials and prompts |

Ship useful defaults even where users may later choose alternatives. A working
basic editor/browser is part of a usable environment; choosing a different one
must not require undoing Soda's account or tool integration. Clipboard access is
explicit and directional, not automatic copying of the user's local clipboard.
Private identity, credentials and repository trust are never preconfigured with
someone else's account to create the appearance of readiness.

Missing standard build prerequisites are a product packaging gap, not a task for
each project administrator. The image recipes must enumerate the selected native
build/debug/development packages for each distro and verify representative build,
link and diagnostic operations as the ordinary project user. Keep version pins and
resolved package inventories in their existing source/build owners. A tool-manager
binary alone is not proof that the development foundation is complete.

The Rocky recipe now explicitly adds GCC/C++, native C/C++ headers, Make,
CMake/Ninja, pkg-config, binutils, GDB/strace, OpenSSL/zlib development packages,
and archive/transfer/network diagnostics. Language versions still belong to mise.
`tests/installed/project-foundation.sh` owns the ordinary-user C/C++ link/check/debug
probe, with a fresh private HOME and retained test-only artifacts; see the
[native journey](native-validation.md#explicitly-permitted-product-journey).
Packaging/parser/probe-guard tests have local coverage. The expanded RPM installation
and real compiler/debugger probe have **not run in a newly built native image**;
this is not yet batteries-included acceptance. Existing roots receive no automatic
package addition or maintenance from this recipe change.

Repository-specific runtimes and dependencies still follow native mise/package
workflows and authorized trust/install decisions. When a user selects a supported
optional app, Soda must supply its declared dependencies and working launch path;
"install these prerequisite packages yourself" is not the completed integration.
Personal provider sign-in and consent remain explicit. An unsupported app must be
reported honestly rather than offered as a working preset.

Fresh-image acceptance starts with an ordinary new member and no operator repair:
join/open a terminal, manipulate files, use Git, build/debug a representative native
sample, use shared tools and a native service; KDE also opens, edits and browses
the same project through the real desktop. Cover required setup/error paths and
persistence. Manual package fixes invalidate that candidate's batteries-included
claim until incorporated into packaging and rechecked. This is a requirement for
the existing product tests, not another provisioning framework or readiness daemon.
The current image is not declared complete merely by adding this policy.

## Selected environment profiles

The user selected Linux for desktops and these creation choices on 2026-09-10:

| Profile | Distribution | Interface | Status |
| --- | --- | --- | --- |
| Rocky headless | Rocky Linux | Terminal | Existing implementation; preserve as the default |
| Rocky KDE | Rocky Linux | Terminal and KDE Plasma desktop | Selected; not implemented |
| Rocky GNOME | Rocky Linux | Terminal and GNOME desktop | Deferred |
| Fedora Server | Fedora Linux | Terminal / headless | Selected; not implemented |
| Fedora KDE | Fedora Linux | Terminal and KDE Plasma desktop | Selected; recommended first desktop implementation |
| Fedora GNOME | Fedora Linux | Terminal and GNOME desktop | Deferred |

**Mise is selected for every supported Project OS profile and the separate Rocky
CI job image.** Development and CI use the repository's same tool declarations and
lockfile when provided, with explicit native trust/install steps. Project tool
installations remain in the persistent project; CI uses its own installations and
cache lifecycle. Reusing configuration does not mount developer homes, credentials
or shared project tool stores into jobs. Native compilers, headers and system
libraries remain image responsibilities. The [runner guide](runners-port.md) owns
the CI image and cache design; only the current Rocky project image already ships
mise. This decision does not imply that the additional profiles or CI image exist.

The creation UI uses one **Project OS** dropdown with the four selected combinations,
as specified in [repository settings](sodaspaces-plan.md#settings-pages-and-os-selection).
Distribution/interface remain profile properties rather than independently saved
UI settings. GNOME stays out of executable choices until its
implementation is selected and ready. Existing environments show their original
profile; this is not a live distro/desktop switcher. Resolve a bounded profile ID
server-side to installed, architecture-compatible artifacts. Do not accept arbitrary
image references, package lists or native runtime flags from the browser.

Profiles describe initial userspace and interface, not six independent backends.
Preserve the shared Soda contracts for accounts, home/shared files, mise, Git, SSH,
terminals and Start/Stop persistence. Keep common authored files with their existing
owner and isolate real distribution/package differences in the image recipes.
Container init and native guest boot are distinct; do not blindly copy
container-only units, seccomp/capability policy or network assumptions into a VM.

The current helper retains one configured image and directly creates a persistent
Podman container. New source resolves its labelled native Rocky headless identity,
persists immutable creation metadata and uses the exact image ID with no pull.
The shared settings/drawer selector accepts only that supported installed profile;
legacy omission of `profile_id` selects it too. There is no multi-image profile map,
Fedora recipe or desktop/VM backend today. Preserve that Rocky mechanism
for existing projects. First investigate KDE session/display integration against the
existing persistent Project OS boundary. Selecting Linux/KDE does not select a VM.
If a concrete native requirement cannot fit that boundary, document the blocker and
the effects on accounts, storage, workloads and access before a runtime decision.
Do not implement a parallel VM backend or silently split a project into a headless
container and disconnected desktop. A profile is not itself a runtime selector.
Fedora Server names the requested headless experience: document the actual Fedora
image/edition used during packaging rather than representing a generic Fedora OCI
image as an installed upstream Server edition.

The current build/install path also has one project-image slot:
[`scripts/build-native.sh`](../scripts/build-native.sh) builds one `project-os`
artifact and requires its base reference to match the dashboard base;
[`scripts/install-native.sh`](../scripts/install-native.sh) loads one
`localhost/soda-project-os:dev` tag; and the host helper configuration contains one
image reference. A fixed profile map therefore requires deliberate build metadata,
staging, install and helper changes. Decouple project profile bases from the appliance
application base rather than multiplying the whole appliance build per distribution.
Keep the existing single image path until a complete replacement caller exists.

The new schema-v8/API/helper source now persists immutable creation metadata
(ordinary local checks, not newly installed proof). Each new environment must persist: the bounded profile
ID, distribution ID and version, interface/session type, native architecture, exact
OCI image ID/digest and Soda recipe revision. Put the same trusted profile ID on the
native project object so database and helper observations can be compared. Mutable
tags and the current configured default are not environment identity. For legacy
roots that lack this metadata, report a legacy/unknown profile plus separately
observed `/etc/os-release` and image facts; do not backfill a precise profile from
today's default or change the root to make the label true.

The shared settings/drawer **Inspect current OS** control now implements that
separate observation through the protected `/api/environments/{id}/os` route and
fixed native `/os` helper operation. It reads a bounded, regular `/etc/os-release`
(standard symlinks allowed), validates only ID/version/display-name, and never
shell-sources it; Python runs in isolated mode without site initialization or bytecode writes.
Other fields are not returned. The original container image is
read from actual inspection, not the configured default; neither fact describes
an immutable inventory of subsequently installed packages. Stopped roots stay
stopped and have no OS-release read. Malformed/missing/special/oversized files are
unavailable, not repaired. Reads never assign profiles or change readiness; the
session is rechecked before returning observations. Local parser, Go and synthetic
browser checks passed; installed helper/root observations remain unrun.

**Use Fedora KDE as the first desktop compatibility target** within the shared
foundation work, not as a replacement for outstanding Project OS work. OpenAI currently
supports Fedora 43/44 in its Linux preview, while Rocky is outside its supported
distro list. An RPM package alone does not prove Rocky compatibility. Both
[Fedora KDE](https://www.fedoraproject.org/kde/download/) and
[Rocky KDE](https://docs.rockylinux.org/teams/rel_eng/image/#about-live-images) have
upstream x86_64/aarch64 media; availability is not installed Soda evidence. Exact
version/digest/package baselines belong in reviewed recipes/locks when implemented.

Desktop availability and individual AI capabilities must be reported separately.
The [OpenAI Linux preview](https://learn.chatgpt.com/docs/linux/linux-app) currently
lacks built-in computer use. The
[Claude Linux desktop beta](https://code.claude.com/docs/en/desktop-linux) supports
Ubuntu/Debian, not Fedora/RHEL, and also lacks computer use. These are current app
compatibility gaps, not reasons to change the user's selected Linux distributions
or quietly substitute an unofficial package. Recheck before each app integration;
do not promise Claude GUI parity on the selected profiles yet.

Fedora KDE has shipped only the Plasma Wayland session since Fedora 40; X11 apps run
through Xwayland, but a Plasma X11 session is not a supported fallback. KDE's August
2026 remote-desktop status says its single-user headless work is still in progress
and a multi-user remote server is unlikely for Plasma 6.8. That upstream limitation
does not prove a private virtual KWin session cannot work in the project container;
it makes per-account headless startup the first native investigation. Do not choose
TigerVNC/Xvnc or xrdp by relying on a Plasma X11 session Fedora does not ship.
Sources: [Fedora's Plasma 6 change](https://fedoraproject.org/wiki/Changes/KDE_Plasma_6),
[KDE remote-desktop status](https://planet.kde.org/david-edmundson-2026-08-25-whats-happening-in-kde-remote-desktop-improved-unattended-mode-and-more/).

After a private Plasma Wayland session works, investigate KRFB with a private RFB
endpoint and a bounded WebSocket/noVNC adapter as the first browser-transport
candidate. It matches the existing-session model, the browser surface and separate
view/control modes. It is not selected production source: verify exact Fedora/Rocky
packages, headless operation, resize, clipboard, software rendering and cleanup.
The RFB listener must remain private. Any RFB credential is ephemeral runtime state,
confined to the account-owned graphical scope and revoked at End; never deliver it in
a browser URL/JavaScript payload or logs/evidence. The project account and project/
appliance root retain their documented native visibility. Reject the candidate if a
secure session-scoped native setup is impossible, not merely because RFB itself uses
a password. KRDP/RDP remains a comparison, but an RDP-to-browser gateway does not
itself create the needed headless session. [KDE's KRFB guide](https://docs.kde.org/stable_kf6/en/krfb/krfb/krfb.pdf)
describes its existing-desktop and view/control behavior.

Deliver each new profile through creation, native provisioning, account/access
wiring, packaging/staging and the real Spaces/drawer experience before enabling
its choice. Prove persistence and authorization per profile and claimed native
architecture. Profile changes apply to new environments; existing CIDs, roots,
accounts, keys and installed tools remain untouched. No automatic conversion,
OS upgrade, image replacement or project migration is selected.

## Desktop session and access boundary

Project accounts remain password-locked. Forgejo/Soda authorizes desktop access;
Project OS does not add a native password authority, enable password SSH or expose an
SDDM/xrdp greeter. After an explicit **Start desktop**, the restricted host helper
must resolve the trusted project, original identity marker and Linux login and invoke
one fixed native start operation. **Open desktop** attaches only to that exact existing
desktop ID. Opening a page, selecting a tab, Refresh or returning from navigation must
not create a replacement session.

Starting the graphical process with `podman exec --user` alone is insufficient. The
launcher must establish a login-quality session for the real account: correct UID,
GID and supplementary groups, `HOME`, shared mise environment, a private
`XDG_RUNTIME_DIR`, a registered logind/PAM session or a documented equivalent, user
service manager, D-Bus session, Wayland display, portals and the PipeWire services the
selected transport needs. Review the actual distribution PAM and user-session stack;
do not copy the host Cockpit PAM policy, accept a browser-supplied Linux password,
expose a general session launcher or pass Soda OAuth/session material into the desktop.

The initial lifecycle is finite and separate from managed-terminal implementation:

- One desktop ID belongs to one project, original login and creating Soda sign-in
  context. Other sign-in contexts, including another login by the same person, cannot
  silently adopt, replace or renew it. Duplicate views in the creating context may
  observe it; one explicit input lease controls keyboard/pointer and transfers without
  concurrent injection or implicit eviction.
- Hide, page/drawer navigation and unexpected loss of the last **visible authorized**
  view start a 30-minute detached deadline for the exact desktop. A successful visible
  reattachment from the creating context, after fresh matching user, membership and
  repository authority checks, clears that grace deadline. Output, retries and hidden
  views neither clear nor renew it. No attachment extends the hard cap or creating
  context's authority.
- The desktop has a 12-hour hard cap and never outlives the configured expiry of its
  creating Soda sign-in context. A short independent native safety lease is renewed
  only while those bounds and authority remain valid. Lease loss, explicit Soda logout,
  context rotation/expiry or confirmed authority loss closes every display attachment
  and ends the owned graphical scope. Cleanup failure remains unavailable/uncertain;
  it must not free the ID or start a replacement.
- **End desktop** terminates only the owned graphical session/cgroup and its
  desktop-launched descendants. Include GUI apps activated through D-Bus/systemd,
  which may not remain ordinary child processes: prove their exact session-owned
  units/scopes and cleanup without killing shared user infrastructure. It does not
  kill by UID or stop the user's service
  manager, personal tmux, ordinary SSH, project services or nested workloads.
  **Stop project** retains its existing whole-project meaning and interrupts the
  desktop with everything else.

Show the live retention/expiry deadline and the effect on unsaved GUI work. A native
Plasma logout ends this desktop and is observed as ended; it does not log out the Soda
web account or launch a replacement. Expired/ended IDs show their reason and require
an explicit new Start after confirmed cleanup.

Desktop Lock must have a defined, usable unlock path. The accounts have no reusable
Linux password, so a stock locker cannot be assumed to work and automatic login is
not an answer. Before enabling KDE, select and test a bounded unlock design tied to
the same authorized account/context, without turning Soda/Forgejo credentials into a
general PAM password or allowing another sign-in context to unlock the display. If
that cannot be done with the selected session stack, stop for a product decision;
do not silently disable or leave a trapping Lock action. Project suspend remains the
existing explicit whole-project Stop, not a desktop power-menu side effect.

The first desktop surface has one resizable virtual display with software rendering,
keyboard, pointer and explicit directional clipboard actions. It does not promise
audio, microphone/camera, USB/device passthrough, GPU acceleration, printing, multiple
monitors, cross-user screen sharing or drag-and-drop file transfer. Existing SSH/SCP/
SFTP remains the file-transfer path. This keeps Fedora/Rocky and x86_64/aarch64
acceptance about the same Project OS rather than a workstation-peripheral platform.

Browser cookies, app profiles, provider sign-ins, wallets/keyrings and clipboard data
belong to the original user. Store them only in account-owned persistent paths with
private modes; never in `/srv/project/shared`, image layers, Soda's database, transport
URLs, argv, logs, screenshots or retained evidence. The account currently has no login
password, so do not assume PAM will unlock KWallet or another Secret Service. Select
and test an explicit per-user unlock/sign-in behavior before advertising credential
storage as integrated. Project root/wheel and appliance root retain their documented
ability to inspect project state; the UI must not imply stronger secrecy.

Codex/ChatGPT and Claude availability are application claims layered on this desktop.
As of 2026-09-10, OpenAI's Linux desktop preview supports Fedora 43/44 on x64 and
ARM64 but explicitly lacks Computer Use, and Claude's Linux beta supports Ubuntu/
Debian rather than Fedora/RHEL and lacks Computer Use. Desktop success therefore
does not imply either vendor app is available: advertise and validate OpenAI's app
only on its supported Fedora/version/architecture combinations, and do not offer the
Claude desktop app on the four selected profiles today. Codex CLI and Claude Code CLI
are separate terminal integrations with their own package/runtime checks. Report
missing native automation features unavailable instead of adding screen-scraping
automation or changing the Project OS runtime. Claude Cowork's optional QEMU/KVM guest
is an application feature and does not select a VM for Soda. Sources:
[OpenAI Linux desktop](https://learn.chatgpt.com/docs/linux/linux-app),
[OpenAI Computer Use](https://learn.chatgpt.com/docs/computer-use),
[Claude Linux desktop](https://code.claude.com/docs/en/desktop-linux),
[Claude Code installation](https://code.claude.com/docs/en/installation).

## Ownership and trust

| Owner | Responsibility |
| --- | --- |
| Appliance/operator | CoreOS/kernel, host Podman/network/storage, restricted helper and approved delivery of required platform integration. No human developer host accounts. |
| Project OS | Systemd PID 1, native accounts/groups, SSH, shared directories/tools, project-local workload engine and managed-terminal supervision. |
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
- The currently packaged interactive subset includes a UTF-8 login environment, file/text/process
  tools, a terminal editor (`vi`/`vim-minimal`), pager (`less`), CA certificates and
  working terminal descriptions. The saved `dad2945` x86_64 build RPM inventory lists
  those editor/pager/CA packages, but not tmux. That is a past image inventory, not a
  current survey of retained roots or fulfillment of the batteries-included contract.
  Make required tools explicit in recipe/check
  ownership rather than relying indefinitely on transitive base-image contents.
- Keep the implemented **tmux** package and bounded native supervision for browser
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
  not a Soda selector. Soda supplies the standard OS/build prerequisites described
  above. Project administrators install additional project-specific packages through
  normal native permissions; toolchain versions, preferred editors/AI tools and
  application services remain deliberate choices with complete dependency handling.

GUI/GPU/device access, kernel-dependent workloads and every third-party tool are not
implied compatibility promises. A concrete new need gets its own native compatibility
review; no privileged-parent shortcut, arbitrary host device/mount or automatic
capability expansion. The selected creation profiles above reopen only those
specific userspaces/interfaces; arbitrary profiles and private-resource branching
remain outside scope.

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
  Managed tmux preserves/reattaches the selected authorized session within its
  deadlines. Lit rendering and future layouts cannot replace that native owner.
- **End terminal/terminal expiry** ends that managed terminal's supervised
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
account, **not SSH or a device private key**. The new browser-only Join source allows zero external SSH keys through the API,
helper and account script without enabling password/root SSH. The script creates
a real password-locked account/home, identity marker, shared link and empty managed
key file. Repeated provisioning refuses changed native keys rather than truncating
them; use explicit key maintenance. Local script tests mock only root/account commands,
not file effects. Fresh native account/access proof and same-root delivery remain
pending. Optional device → project SSH keeps explicit public-key review/apply,
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

An exact isolated-root recipe was executed for `22d8591` delivery on
`soda-native-spaces-658f2af`, after the user explicitly waived backups for that
target. It admitted the original CIDs/files, verified the native tmux RPM signature
and dependency transaction, and updated only tmux and the two Soda-owned files.
See the [handoff](implementation-status.md) for retained recipes, failed attempts
and bounded browser continuity/cleanup evidence. This exception is not a general
backup waiver, fleet-maintenance tool or permission for another root/target.

The later approved b8af68c delivery maintained all four original `soda-test` roots
with fresh private backups and copied-state rehearsals: one signed tmux RPM, the
exact program and one directory line in each Git-proven init. Other init behavior,
capabilities, accounts, keys, homes/shared files and original agent socket inodes
were preserved. Do not assume raw Podman exports have container-local ownership:
selected 5.8.4 emitted host-mapped IDs. That run instead used native GNU tar inside
the original namespace, without numeric translation or original-root replacement.
Virtual kernel trees and runtime/socket/process state are not restorable workload
snapshots. See the handoff for the failed attempts, exact backups and observed scope;
none permits automatic rollback over later writes or borrowing agent credentials.

Keep `scripts/build-native.sh`, native build metadata, staging/verifier
and [installation](installation.md) as production owners. Do not use first-install
or activation as upgrade tools. Project administrators own their ordinary native
tool/service changes; required platform additions/security maintenance need deliberate
operator/project coordination. No automatic fleet patching is currently supplied.

## Concrete gaps and next implementation

| Gap | When it matters |
| --- | --- |
| Tmux source candidate native proof | Package recipe, private guard/service, lease, attach-only API and drawer restore are authored; verify actual native cgroups, login/terminfo/editor behavior and failure cleanup before acceptance. |
| Required-tool checks and same-root package/unit delivery | Part of that feature, not a separate OS platform. Saved RPMs are not proof every retained root has the required commands; maintenance is a gate to existing-target delivery, not to source work. |
| Browser-only onboarding native proof | Zero-key account provisioning and explicit own-Forgejo key review/select/Save now have local source/script/browser coverage. Fresh native access and paired same-root delivery remain pending; existing accounts/key files and password-SSH denial are preserved. |
| Personal Git credential trust/consent/passphrase choice and native endpoint trust | Resolve before automated Git setup/combined clone-edit-build-push proof; do not fabricate credentials or treat profile keys as repository-scoped. |
| Current Forgejo authority versus already issued Linux sudo/SSH rights | Explicit limitation on rename/transfer/offboarding claims; no automatic native permission reconciliation is selected. |
| Multiple complete Project OS profiles and native metadata proof | Rocky headless installed-image preflight, immutable metadata, exact-image creation and shared settings/drawer selection now have local source coverage. Build/install still has one image slot; native delivery, independent bases, new profiles and separate legacy OS observations remain pending. Never backfill/convert legacy roots from the default image. |
| KDE login-quality session, Lock and credential store | Current password-locked accounts have shell/SSH setup only. Prove the selected user-session/unlock design and private per-user Secret Service before advertising a desktop profile. |
| Fedora/Rocky headless Wayland and browser transport | Investigate virtual KWin in the existing container, then the bounded KRFB/private-RFB/WebSocket/noVNC candidate. Upstream headless limitations guide the probe but do not select a VM or prove this candidate fails. |

The selected profiles extend this foundation through concrete implementation work;
they do not require a universal-workstation planning phase. The
[Lit workspace sequence](lit-migration-plan.md) preserves this
implemented terminal mechanism while adding real multi-session UI/backend support.
Onboarding and the combined Git workflow keep their separate scope. Extend the existing [product validation](native-validation.md)
entrypoints for fresh and maintained projects; package lists, source tests or socket
closure are not native continuity/preservation proof. Broader operator/client and
native aarch64 acceptance stay independent. [Deferred scope](deferred.md) remains
closed: no private-resource selectors, generalized recovery/deletion, release/update
platform or speculative host fallback.
