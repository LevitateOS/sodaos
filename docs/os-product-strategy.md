# CoreOS product strategy and feature priorities

SodaOS should pursue the user-selected direction of a custom Linux product based
on Fedora CoreOS to deliver a dependable shared development machine. The operator should get a
supported host configuration, usable private access, controlled resource use and
a clear path through maintenance and machine failure. Developers should keep
ordinary Linux accounts, Git, mise, SSH and native workloads inside lasting projects.

This records the user's requested product direction and recommended investments.
It is a strategy document, not implementation acceptance or a new execution grant.
The [Sodaspaces plan](sodaspaces-plan.md) still owns immediate coding order; the
[deferred guide](deferred.md) still distinguishes proposed work from selected features.
The host strategy here is separate from the [Project OS foundation](project-os.md),
currently implemented on Rocky and selected for Fedora/headless/KDE variants.
Those profiles retain the same account, tools, services, persistence and maintenance
contracts. KDE is a project desktop, not a desktop installed on CoreOS. Profile
selection does not select a VM platform, host device broker, new OS updater or any
of this strategy's still-proposed capacity/recovery features. The leading plan owns
their integration order alongside existing onboarding and credential work.

## The three recommendations

- **Most valuable proposed host capability:** enforced resource protection for shared
  development and CI, with room left for appliance services. Backups with tested
  restoration are the next major investment in protecting the team's work.
- **Easiest useful first addition:** a read-only machine-health and first-boot report
  through the existing operator console/CLI. Budget roughly 3–5 engineer-days for
  a reviewable source candidate, followed by bounded native validation.
- **Strongest long-term reason to ship an OS:** a supported boot-to-workspace
  deployment, including first-boot storage/security policy, known host components,
  controlled host maintenance and a tested way back to a compatible bootable system.
  That requires ownership of the host lifecycle and is a larger investment.

Finish the selected developer workflow while developing these priorities. Resumable
terminals, real browser-only joining and an agreed Git credential flow remain
valuable even though they can also be delivered through services. An OS strategy
does not make an incomplete development experience acceptable.

## What can actually be impossible as a service?

An ordinary application cannot independently control the kernel it boots under,
the initramfs, disk provisioning before the real root mounts, bootloader trust or
selection of a previous OS deployment. Those are the strongest OS responsibilities.
For example, [Ignition](https://coreos.github.io/ignition/) performs first-boot
configuration from the initramfs, including storage, files and systemd units.

However, **no useful feature provides an absolute guarantee that Soda could never
be implemented using privileged services**. A root service can configure native
resource controls, networking and backup tools, and can prepare changes for the
next boot. If it takes over boot/provisioning/update policy, it is managing the
host as an appliance. Its executable being called a service does not change that
responsibility. An installer could also replace the host underneath it.

The meaningful distinction is the promised boundary:

| Promise | Host responsibility |
| --- | --- |
| A working website or terminal UI | Application services; weak justification for a distro |
| Resource limits, private routing and backups on supported Linux | Privileged integration; a service package remains credible |
| A specified machine configuration from first boot through maintenance and recovery | OS/appliance ownership; cannot promise this while treating the underlying OS as arbitrary and independently managed |
| A Soda-produced boot artifact with a verified boot/trust policy | Strongest distribution responsibility; substantial delivery and security obligations |

Aim to make installing the complete SodaOS system substantially easier to operate
than assembling its parts. Do not create proprietary formats, artificial kernel
dependencies or portability barriers merely to prevent a service edition.

## Use the CoreOS foundation deliberately

The current delivery provisions upstream Fedora CoreOS, layers native packages and
installs Soda's units and container payloads. A customized installer ISO now exists
with bounded diskless BIOS/UEFI boot evidence, but complete fresh-disk installation
is still unvalidated. There is no preinstalled Soda QCOW2, host OCI image or signed
Soda OS release. The [installation guide](installation.md#publication-direction)
owns the ISO-first/QCOW2-next delivery direction and matching-payload inclusion;
the [installer plan](coreos-installer-plan.md) owns the selected password-only manual
flow. The [native support guide](native-support.md) owns evidence transport, not
another installer. Product positioning does not change execution evidence.

Use upstream boot, kernel, package and service mechanisms. The recommended host
customization consists of reviewed provisioning, supported package/runtime versions,
security and resource policy, data placement and native service integration.
This does not require maintaining a custom kernel or forking Forgejo.

[rpm-ostree](https://coreos.github.io/rpm-ostree/administrator-handbook/) already
provides staged OS deployments and selection of a previous deployment. Its writable
data under /var is shared across deployments. Soda should build on that separation,
while explicitly handling the compatibility of its applications and persistent data.
A previous host deployment is not a snapshot of later database or project writes.

Product media still requires the remaining delivery work, including
authenticity, signer/key custody where applicable, supported architectures, licensing,
security updates and failure recovery. This proposal does not select bootc, a new
image builder, an independent release service or the predecessor's reserved Updates
platform. The current upstream-based installation path remains in effect.

## Update ownership

The 10 September 2026 discussion distinguishes the operating system, its deployment
tools and the application marketplace. Fedora CoreOS is the host OS.
[rpm-ostree](https://coreos.github.io/rpm-ostree/administrator-handbook/) prepares a
new bootable deployment while the running system remains on its current version;
reboot activates the new deployment and a previous one remains available for OS
rollback. Layered package requests are carried across upgrades, subject to package
resolution and compatibility. OS rollback does not rewind shared application data.

[Zincati](https://coreos.github.io/zincati/) is Fedora CoreOS's automatic-update
agent. It follows the upstream update graph, asks rpm-ostree to stage the selected
OS update and coordinates finalization/reboot. Its default strategy is immediate;
native [maintenance windows](https://coreos.github.io/zincati/usage/updates-strategy/)
can control reboot timing. This describes upstream behavior, not a fresh observation
of the configuration on any retained Soda target. No new reboot schedule or update
policy was applied in this discussion.

| Scope | Owner and remaining Soda responsibility |
| --- | --- |
| CoreOS host and layered system packages | Reuse native rpm-ostree/Zincati; Soda must account for supported package compatibility and an explicit host reboot policy |
| Soda's own programs, app containers, configuration and schemas | Coordinated Soda release delivery remains to design; a CoreOS update does not deliver all Soda changes |
| Marketplace apps such as Vaultwarden and Homepage | The [Services plan](services-and-ai-plan.md#install-retry-and-persistent-lifecycle) owns reviewed recipes and installed versions; app upgrades are separate from host updates and catalog refreshes |

After a host reboot, native systemd/Quadlet services start the instances configured
for boot. This is startup of installed versions, not an app-image upgrade. App
database migration, compatibility and backups cannot be delegated to Zincati, and
rolling back a container image alone does not restore its changed database.

**A Soda host OCI is optional.** OCI is a packaging/distribution format, QCOW2 a
virtual-disk format, and ISO installation media. A bootable host OCI can carry OS
content, unlike an ordinary application image. [bootc](https://bootc.dev/bootc/)
specializes in installing/updating such OS images; it is not the only way to use
OCI on CoreOS. [rpm-ostree also supports OCI-based OS transport and upgrades](https://coreos.github.io/rpm-ostree/container/).
Do not turn “Soda has no host OCI update path” into “CoreOS cannot use OCI,” or make
a bootc migration a prerequisite for the manual installer or Services marketplace.
The exact selected OS/version, trust and native behavior would need review before
changing transport. Existing application OCI archives remain independent of that
optional host-image decision.

These upstream references were consulted for the discussion; they do not add
native validation results, a custom updater, a central marketplace service or the
predecessor's separately reserved Updates platform to the implementation scope.

## Priority and effort comparison

These are planning estimates from source inspection, not measured completion times
or delivery promises. Confidence is low until native feasibility is checked.
Assume one engineer familiar with this repository, working
native x86_64 infrastructure, and reuse of existing interfaces. A week means five
engineer-days. The native column is **total effort including the source candidate**,
focused tests, packaging and one bounded isolated x86_64 journey.

The ranges exclude waiting for access, retained-appliance rollout, independent
aarch64/hardware acceptance and unrelated existing-work completion. Re-estimate
after the named feasibility checks. These rows overlap and should not be added as
a calendar schedule. New operator web pages and unresolved page-shell work are
excluded from the small console/configuration slices.

| Candidate and recommended priority | Dependence on owning the host | Source candidate | With bounded native proof |
| --- | --- | --- | --- |
| First: console/CLI health and first-boot guidance | Medium; fastest way to make the supported host understandable | 3–5 days | 5–10 days |
| Next: limits for supported native runner process trees | High operational value; possible through privileged services | 3–5 days | 1–2 weeks |
| Next: CPU/memory/process limits for new projects | High; requires correct outer-container control | 1–2 weeks | 2–4 weeks |
| Alongside current access work: private connection diagnosis | High integration value; existing host and client authorities | 4–7 days | 1–3 weeks |
| Small maintenance slice: native reboot-window policy and status | High host ownership; upstream scheduling already exists | 2–4 days | 5–10 days |
| Major follow-up: consistent off-machine backup and cold restore | High user value; storage/application coordination | 2–4 weeks | 4–8 weeks |
| Longer-term: specified first-boot data layout and recovery contract | Very high provisioning/storage ownership | 1–2 week layout/feasibility study | Implementation estimate follows disk, consistency and recovery choices |
| Strategic investigation: Soda boot artifact and verified boot policy | Strongest OS dependence; not a quick feature | 1–2 week feasibility study | Full implementation not credibly estimable before artifact, trust and hardware choices |

The quick reboot-window row does not include automatic updates of Soda applications,
job draining, schema migration, unattended rollback or arbitrary project updates.
Likewise, a runner process limit is not automatically a limit on every possible
container job, and a health report does not provide resource isolation.

The low-confidence backup range assumes one destination, permitted offline
quiescence and a fresh restore target with matching architecture/runtime. Spend
the first few days settling consistency, storage export/import and key recovery;
revise the estimate before implementation. It excludes live backup, arbitrary
databases, cross-platform migration and per-project self-service restoration.

## 1. First boot and machine health: the fastest useful feature

**User outcome:** the operator can tell whether this machine is ready, what is
missing and which action will resolve it without diagnosing the entire stack.

Extend the existing [console welcome](console-welcome.md) and product-owned
inspection paths. Initially provide a bounded read-only console/CLI report:

- Observed booted CoreOS deployment, pending host change and required native packages.
- Host security/runtime prerequisites, service/helper availability and current failures.
- Filesystem free space, memory headroom and configured resource policy when it exists.
- Actual private browser/Git listeners, project bridge and forwarding state.
- Separate results for host configuration, project state and intended-client access;
  unobserved client connectivity is unknown, not healthy.
- Concrete next actions with the correct operator/project/provider authority.

Reuse [console source](../appliance/bin/soda-console-welcome),
[installer preflight](../scripts/install-native.sh), existing host/runner readers
and the current native validation journeys. Keep one product-owned representation
where there is a real shared caller; do not copy tests into an outside support gate
or build a metrics platform. Reports must remain bounded and exclude credentials.

**Done means:** a clean isolated installation gives useful operator guidance;
missing configuration, a stopped dependency and an unavailable route are accurately
distinguished. Noninteractive SSH/SCP/SFTP stays quiet. Reading the report changes
no services, projects, network policy or provider state. The existing console hook's
delivery gap must be proved closed on the candidate actually installed.

This is the fastest visible improvement. Its value is reducing operator work;
a status screen alone is not the defining reason for a distribution.

## 2. Resource protection: the strongest near-term differentiator

**User outcome:** an expensive build has a known budget, and other projects and
appliance access remain usable under the tested load profile.

Start with CPU, memory and process-count limits. Reserve explicit headroom for
the host, Forgejo and Soda instead of assigning every available resource to projects
and runners. Show configured limits alongside observed usage. Disk accounting and
low-space warnings can accompany this; enforceable per-project disk quotas are a
separate storage design. Limits reduce interference, not every form of contention.

Deliver two bounded slices:

1. The [native runner unit](../appliance/services/soda-runner@.service) and its
   supported in-unit job processes. Use native systemd controls and a parent budget
   where verified. Inspect actual job process placement; work created through an
   external engine or a separate scope may not belong to the listener's cgroup.
   Provider registration, scheduling and results stay provider-owned.
2. New projects through the [existing creation path](../internal/host/daemon.go).
   Native Podman resource options are a candidate, subject to selected-version
   inspection. Establish actual container/cgroup placement before choosing controls.

The [outer project unit](../appliance/services/soda-project@.service) attaches to a
container created earlier. Putting limits on that unit may constrain only its
launcher. The [Linux cgroup model](https://docs.kernel.org/admin-guide/cgroup-v2.html)
operates on actual membership and hierarchy; inspect the payload and descendants.
Inner workload cgroups being disabled does not prove that the outer project is
limited, or that an effective outer limit is impossible.

**Done means:** native CPU/memory/process pressure reaches the expected boundary,
including nested workloads, while another project and appliance access meet declared
test criteria. Limit-triggered termination is visible and honestly reported. Limits
persist across the tested lifecycle. Existing roots are not recreated or silently
retrofitted; later application needs preserved-state maintenance and its own proof.
For runners, exercise a real trusted job for each supported provider, including
background descendants and any enabled engine mode. Listener accounting alone is
not a job-enforcement result; remote-engine work is outside the local host budget.

An optional low-space creation/start guard is narrower than a quota. Memory can grow
after admission; races across separate runner/project actions matter. Do not promise
capacity guarantees from a free-memory check or introduce a general scheduler.

## 3. Private connectivity that reaches the actual developer

**User outcome:** ordinary SSH, editor access and project service ports work from
the selected computer, with useful diagnosis when they do not.

Build on existing IP-based project networking. Check subnet overlap, local listeners,
bridge/forwarding state and configured route status; distinguish these from an actual
client probe. Provide an explicit way to check the intended client without collecting
its private SSH key. An authenticated browser terminal is not proof of direct SSH.

[Tailscale subnet routing](https://tailscale.com/kb/1104/enable-ip-forwarding)
involves forwarding, route advertisement/approval and access rules. Keep Tailnet in
Cockpit and preserve its administrator authority. Any bounded configuration action
must show its actual route/firewall effects; diagnosis never auto-enrolls or repairs.

**Done means:** a real intended client reaches an authorized project's SSH and test
service; deliberately missing routing is diagnosed honestly. Host-only observations
never receive the client-reachable label. Existing private listeners and ordinary
SSH remain intact. No project DNS, custom gateway or automatic public ingress.

Real client reachability is an existing installation/validation obligation. The
productized diagnostic is a proposed host slice building on that work; it need
not wait for a future boot artifact.

## 4. Predictable maintenance using the native OS mechanisms

**User outcome:** the operator knows which projects will be interrupted, when the
machine may reboot and which checks establish that it returned correctly.

The smallest candidate exposes current update/reboot state and an explicit native
maintenance-window policy. First inspect the exact target's Zincati availability,
enabled state and effective configuration; current Soda provisioning does not
establish those observations. The short estimate assumes the native facility is
available on the selected release. [Zincati](https://coreos.github.io/zincati/usage/updates-strategy/)
already supports reboot windows. Reuse that capability, verify the effective policy
and make interruption visible; selecting a window is not proof that sessions or
provider jobs are idle. Do not indefinitely hold security updates for active shells.

A later work-aware flow can add advance notification, supported admission/drain
controls and post-boot checks. Inspect provider-native capabilities before promising
to drain CI. Do not infer job completion from the listener service's state.

**Done means for the small slice:** the native policy defers/allows a fixture reboot
as configured, the operator sees the effective timing, and same-container identity,
data preservation and declared service startup policy pass after reboot. Runtime
process memory and tmux sessions are not promised to survive a host restart.

Host, Soda/Forgejo application, and mutable Project OS maintenance have distinct
contracts. Existing projects continue to use [bounded same-root delivery](project-os.md#deliver-required-additions-without-replacing-roots).
Broad application orchestration/rollback remains a later scope decision coordinated
with the separately reserved Updates work, not imported from the predecessor.

## 5. Complete backup and tested recovery

**User outcome:** losing the host does not require rebuilding the team's development
state from Git repositories and memory.

Start with one explicit, consistent backup and a cold restore onto a fresh isolated
machine. Include Soda configuration/database and necessary secrets, Forgejo through
supported native backup procedures, and the actual project state listed in the
[Project OS contract](project-os.md#persistent-state-and-lifecycle): homes, accounts,
SSH identities, shared installations, writable roots and nested workload data.

Quiesce declared services for the initial consistency model. A running database
directory copy is not an application-consistent backup. Encrypt before off-machine
storage with a separate recoverable key; verify integrity and practice restoration.
Restoring container storage must account for native storage metadata, ownership/ID
mapping, labels and compatible runtime versions, not merely copy visible home files.

**Done means:** restore into an isolated target; original users can reach preserved
work and a real test database contains the declared committed records. Check native
Git, project identities and files against the backup point. Keep the live source
unchanged. Restored credentials/access may be stale: review them before reconnecting
the restored machine to providers or clients, and never silently reactivate revoked
access. State the recovery point and recovery time actually observed.

This is a major new scope, not a weekend archive script. Scheduling, retention,
per-project self-service restore, online snapshots and availability come after the
bounded backup/restore contract. A snapshot on the same failed disk is not an
off-machine backup; boot fallback does not recover that disk's data.

## 6. First-boot storage policy and a supported bootable system

**User outcome:** installation produces the intended host layout, and the operator
can maintain the system without accidentally replacing the team's persistent data.

This is the strongest direction for the custom distro over time. Specify a reviewed
first-boot layout separating replaceable host components from development data,
with explicit disk selection and mount ordering. Evaluate encryption, capacity
isolation and recovery-key handling against a concrete deployment need. Fedora
already documents [custom storage layouts](https://github.com/coreos/fedora-coreos-docs/blob/main/modules/ROOT/pages/storage.adoc).
Do not select a new filesystem, storage pool manager or disk migration merely to
make Soda look more like an OS.

Begin with the current upstream image plus one reviewed provisioning path. Keep
source recipes and actual artifact/configuration identity connected to their existing
production owners. A later Soda-produced boot artifact becomes useful when it
materially reduces installation variance, required host-package incompatibility or
offline provisioning work. Choose its mechanism after investigating those needs.

**Done means:** fresh installation on the explicitly selected disk reproduces the
declared layout and host policy; wrong/occupied targets refuse safely. A deliberately
tested compatible boot fallback preserves project data and has a working operator
access path even if the Soda web service is unavailable. No unattended fallback is
claimed without its own native failure test and application compatibility policy.

Verified boot and encrypted disks can further strengthen the machine-level security
story, but need an explicit threat model, key lifecycle and hardware/platform proof.
A signed download is not a verified boot chain. Encryption at rest does not hide
unlocked project credentials from project/appliance administrators. A read-only
host filesystem does not make writable application data immutable.

## Recommended implementation sequence

1. Continue the selected resumable-terminal/onboarding/Git work under the leading
   plan. Deliver the existing console hook and real operator/client journeys.
2. Select the bounded read-only health report as the first additional host slice.
   It can run independently of unresolved web-page composition.
3. Once resource scope is selected, add verified native runner limits, then project limits after actual cgroup
   placement proof. Start with operator configuration and observable enforcement;
   do not gate it on a new dashboard or an elaborate quota UI.
4. Productize private-access diagnosis alongside the existing networking work.
   Add the small native reboot-window/status slice when maintenance scope is chosen.
5. Reopen the bounded backup/cold-restore scope as the first substantial storage
   investment, with its consistency, target and credential-recovery decisions.
6. Use that operational experience to settle the longer-term first-boot storage,
   boot-artifact and compatible fallback design. Secure boot/encryption and broader
   update orchestration need their own concrete design and target scope.

Keep candidate features in this document until selected into the leading plan with
their source owners, failure behavior and acceptance journey. Do not turn this table
into a second release gate or use it to postpone already selected product work.

## Features that do not earn priority merely by looking like an OS

Extra branding, another dashboard, an app store, a browser IDE, more preinstalled
tools and GPU support without a concrete workload do not establish the host promise
above. A custom kernel, package manager, container orchestrator or cluster scheduler
would add major maintenance obligations. Keep ordinary native interfaces and the
current trusted-team isolation claim; no privileged-parent or VM fallback follows.

The product should earn a dedicated machine through useful, tested behavior:
predictable installation, usable access, bounded resource consumption, preserved
development state and understandable maintenance. Those benefits justify supporting
the complete CoreOS-based system even while its components remain ordinary services.
