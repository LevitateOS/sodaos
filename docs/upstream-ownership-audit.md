# Upstream ownership audit

## Maintenance-commitment audit — 13 September 2026

**Current review:** `e9cec3b`. Review began at `4b56271` plus the existing
working-tree edits; those edits were committed as `e9cec3b` during the review. The
commit delta was checked: it contains the already-reviewed Spaces error feedback
and documentation, not a change to the audited build/runtime mechanisms.
This is a repository-wide architecture and caller audit, not a
claim that every line or runtime path was tested. No implementation, deployment,
provider operation, fixture cleanup or upstream source build was performed.

The trigger was the assistant's proposal to build modified Forgejo for username
blocking, described as a “narrowly scoped source change.” The
[architecture record](architecture.md#username-blocking-proposal-and-understated-build-ownership)
records that failure. This review asks where the same imbalance already exists:
a small outcome causes Soda to own upstream build behavior, file formats, native UI,
release coordination or a second source of truth indefinitely.

**The pattern is present, but it is not equally present everywhere.** The strongest
examples are copied native templates for CSS classes, compiling an unmodified
optional CLI, boot-file byte editing for installer branding, a second version/fetch contract
for the native locale behind eleven labels, and duplicated application build orchestration. Some costly
features were explicitly selected; their cost must be exposed, not retroactively
called unauthorized. Tests, manifests and earlier assistant-written guides establish
what depends on a mechanism, not independent proof that the mechanism earns its cost.

The [refactoring plan](refactoring-plan.md#maintenance-commitment-audit-follow-up)
remains the sole implementation-decision owner. Findings below are recommendations,
not accepted deletions or permission to change retained state.

### Largest commitments and clearest reductions

| ID | Current commitment | Smaller direction | Consequence / confidence |
| --- | --- | --- | --- |
| MC1 | 244 native Forgejo templates are shadowed by local copies; some copies exist only to add styling classes. | Retire proven class-only overrides and style stable native selectors through existing hooks. | Concrete equivalent-style examples verified; not all redesign overrides can be removed without changing the design. |
| MC2 | Resolved in source: both paths fetch upstream Tea binaries. | Official Linux amd64/arm64 Tea 0.16.0 binaries replace source compilation. | Source fetcher, compiler wrapper and source lock removed; native image execution remains unverified for this replacement. |
| MC3 | Installer branding edits boot configuration bytes and preserves upstream embedding offsets. | Keep native boot-menu branding and brand the actual Soda wizard. | Gives up a cosmetic boot-menu label; does not by itself eliminate full-payload remastering. |
| MC4 | Writable installer and immutable host-image candidate separately orchestrate the same five application roles. | Share genuinely common producers and packaging inputs, then retire the superseded delivery path when a replacement is qualified. | Paths currently produce different layouts/images. Neither wholesale archive reuse nor an immediate bootc cutover is proven. |
| MC5 | Eleven Soda labels trigger a full native locale copy with its own separate version/URL lock. | Keep the required merge, but derive the native catalog from the already selected pinned Forgejo image. | Removes the second native-version authority; hard-coded English would change the selected translation contract. |
| MC6 | Join forces Forgejo usernames into Linux's namespace even though membership already stores a separate login. | Investigate an internal stable-ID-derived login for new memberships, using the existing mapping. | Fix belongs in Soda; displayed SSH/shell login changes and native checks are needed. The proposed Forgejo fork never shipped. |

#### MC1 — presentation changes create upstream template ownership

The inventory has **254 local templates: 244 shadow upstream files and 10 are
local additions**. The shadowed native surface corresponds to 16,598 upstream lines.
All 244 differ, but 47 have at most five changed lines; nine have at most two changed lines after excluding provenance comments; some
add a marker rather than replacing a class. These numbers locate the review
surface; they do not prove all copies are unnecessary.

A decisive example is [repo/view_list.tmpl](../appliance/forgejo/templates/repo/view_list.tmpl):
its only behavior-relevant difference is adding `soda-code-files` to the table.
The native table already has `id="repo-files-table"`.
[repository-code.css](../assets/branding/forgejo/repository-code.css), lines 104–119,
can target that native ID instead. The existing custom header hook already loads
Soda CSS. Copying the file makes Soda responsible for following native loading,
link, permission and markup changes merely to attach a class. The native
`.switch.issue-list-navbar` selector offers another concrete class-only candidate.

**Recommendation:** start with exact before/after comparisons of the small leaf
overrides. Keep the requested style, handlers and form behavior. Validate native
rendering, HTMX updates, relevant permissions, both themes and responsive screenshots
before removing each override. Do not delete all 244 copies or replace the redesign
with stock appearance under the guise of equivalent cleanup. Broader reductions
require explicit design tradeoffs.

#### MC2 — a developer CLI creates a second upstream compilation obligation

At the audited revision, both build paths downloaded Tea source, invoked its Makefile with controlled Go flags, staged the binary and normalized its version output. [The CLI guide](project-clis.md#source-and-packaging) records actual maintenance failures involving exported `GOFLAGS` and ANSI version output. There was no Soda patch to Tea; the source build was inherited rather than required by the product.

**Remediation authorized and implemented on 13 September 2026:** both callers now use [fetch-tea.py](../scripts/fetch-tea.py) with official Linux amd64/arm64 release binaries. The source fetcher, Make/Go wrapper, source lock and obsolete tests were removed. Tea 0.16.0 replaces 0.15.1 in accordance with the owner's latest-version preference. The [binary manifest](../project-os/locks/tea-binary.toml) records upstream URLs/checksums; the unchanged license remains staged. No new signing service or build framework was added. See the [CLI guide](project-clis.md#source-and-packaging) for packaging and validation status.

#### MC3 — cosmetic boot labels acquire boot-format compatibility work

[build-installer.py](../scripts/build-installer.py), lines 322–369, replaces exact
Fedora CoreOS menu text and GRUB classes with equal-length Soda strings so native
embedding offsets remain intact. The surrounding remaster path separately handles ISO extents,
BIOS boot-info and upstream file-layout expectations for the selected on-media payload. Every upstream layout or label
change can invalidate the branding operation even when the native installer works.

**Recommendation:** retain native boot-menu entries and brand Soda's own console
instead if this cosmetic difference does not earn the maintenance burden. The
keyboard/disk/password wizard remains. This removes the branding branch, not all
ISO verification. The same-media console and verified bundle are explicitly selected in the
[installer plan](coreos-installer-plan.md); their payload-copy/continuation and
remastering work is not an unrequested feature. Installation still needs RPM
network access ([installer guide](coreos-installer.md#the-console-ships-on-the-iso)).
An online verified release payload would change that explicit same-media
requirement and its availability/trust properties. It is not selected or recommended
as an equivalent refactor. Do not introduce a download service or new release trust
system merely to simplify a branding feature. Fresh BIOS/UEFI media proof is needed
for any boot-path change.

#### MC4 — transitional build paths are becoming duplicated release ownership

[build-native.sh](../scripts/build-native.sh), lines 45–112, builds commands/assets,
Tea and image archives for the writable bundle. [complete.go](../tools/soda-host-image/complete.go),
lines 44–73 and 145–289, independently prepares content and builds/exports the host
candidate's corresponding application roles. [installlayout](../internal/installlayout)
also carries `/usr/local` versus vendor `/usr` path variants.

The new Forgejo image embeds presentation; the old path uses stock Forgejo plus a
separate staged tree. The same role names do **not** imply interchangeable archives.
The bootable candidate is not yet an installable, upgrade-qualified replacement.
The owner selected a release train; temporary coexistence is therefore understandable.
The recurring cost is maintaining parallel commands, images, versions, metadata and
layout wiring without a concrete retirement boundary.

**Recommendation:** extract only genuinely shared production steps/inputs into the
existing build owner, preserve explicit packaging variants and define when the old
producer can be retired. Do not create a generic build framework, silently choose
bootc, or erase existing fixtures. Validate content identity per variant, then fresh
installation before changing the supported entrypoint. Zero customer installations
remove an assumed customer migration programme; they do not remove retained data.

#### MC5 — locale additions create a duplicate upstream-version authority

[forgejo-locales.py](../scripts/forgejo-locales.py), lines 1–55, locks/fetches a full
native catalog and appends [eleven Soda labels](../appliance/forgejo/i18n/en-US.ini).
The result is staged as a native locale override. Exact Forgejo 15.0.7 source confirms
custom files shadow the whole catalog rather than merging those eleven keys.
Consequently the merger is necessary **if the selected native translation mechanism
is kept**. Its presence alone is not evidence of overengineering.

The avoidable cost is the separate [locale URL/version lock](../appliance/forgejo/locale.lock.json)
next to the selected Forgejo image: each upgrade must reconcile both sources, and
building also depends on the raw catalog endpoint. **Recommendation:** extract the
catalog from the exact already-selected Forgejo image and feed the existing small
merger; remove the unused `profile_portrait` key. Verify the extraction method,
selected image identity, unchanged native catalog bytes and all used Soda labels.
This preserves translation ownership and avoids a new localization framework.

An initial recommendation to use literal English was narrowed after independent
counter-review: it would give up a documented translation contract. If the owner
chooses that tradeoff, native keys/literals can remove more of the locale surface,
but it is not an equivalent cleanup merely because current Soda labels are English.

#### MC6 — a local identity constraint escalated into an upstream-build proposal

[Join](../internal/web/environments_api.go), line 315, copies `access.actor.Login`
into the native account request. [Membership storage](../internal/store/store.go),
lines 208–215, already records `(project, Forgejo user ID) → Linux login` independently.
Existing membership is reused before a new account is provisioned. The native helper
correctly refuses an existing unassociated system account; taking over `operator`
is not a simplification.

The preceding native journey demonstrated the collision. No new mutation was
performed during this audit. A new stable, bounded Soda login derived from the
Forgejo numeric identity could use the existing association and preserve all old
memberships. This needs first-Join, repeated-Join, reserved-name, Forgejo-rename,
existing-membership, terminal and SSH checks. It changes visible native login names.
**Recommendation:** resolve that Soda policy explicitly; do not build modified
Forgejo to enforce an accidental naming equality. Neither the mapping change nor
username blocking is implemented by this audit.

### Additional commitments and their actual tradeoffs

| Area and source evidence | Assessment and smaller direction | What must remain / what is not proved |
| --- | --- | --- |
| [custom/header.tmpl](../appliance/forgejo/templates/custom/header.tmpl), lines 2–57: 53 stylesheet links and 28 manual version strings; [redesign status](forgejo-redesign-status.md#second-pass-refinement) records a missing payload stylesheet | Confirmed avoidable delivery coordination. One ordered build-generated CSS entry and version can preserve modular authoring and the current cascade. | Check URL/font rewriting, load order, themes, native subpaths and caching. No need for a new asset manager. |
| [presentation inventory](../tests/forgejo/presentation/inventory.json) and [test](../tests/forgejo/presentation/inventory.test.ts), lines 14–27: per-template callers, roles, review hashes and states | Confirmed supporting bookkeeping, not native runtime proof. Generate derived closure/callers; keep focused behavior/permission/visual tests and retire metadata whose only consumer checks that metadata. | Exact delivery allowlists and real native parity checks remain useful. Inventory size alone does not justify deleting tests. |
| [notification preview](../assets/branding/forgejo/notification-preview.ts) and [repository switcher](../assets/branding/forgejo/repository-switcher.ts), with native fragment/header overrides | Optional UI convenience creates continuing coupling to HTMX fragments and `/repo/search`. Native notifications and repository/dashboard navigation avoid the added controllers. | Removing previews/dropdowns changes convenience, not a behavior-preserving refactor. Keep unless the owner selects that tradeoff. |
| [avatar renderer](../internal/avatar/avatar.go), [HTTP adapter](../internal/web/avatars.go), proxy/activation settings and DiceBear dependency | Custom robot avatars are a shipped optional commitment. Built-in Forgejo avatars avoid a backend endpoint, renderer dependency, provider setting and snapshot/catalog work. | Custom robots were a documented design choice; no fork or replacement renderer was found. Dropping that identity treatment requires a feature decision. |
| [runtime recipe validation](../internal/host/tailnet_companion.go), lines 71–94, and [terminal service checks](../internal/host/project_terminal.py), lines 513–534 | Existing R6 remains: exact CreateCommand arrays, empty drop-in paths and textual unit properties couple ownership to incidental representation. Narrowing to effective protections could reduce upgrade fragility. | Conditional: retain exact project/account/CID identity, capabilities, mounts, namespaces, cgroup cleanup and transport isolation. A property-to-protection comparison and refusal tests are needed before relaxing checks. |
| [Tailnet host management](../internal/tailnet/management.go), lines 122–451, plus frontend models | Significant custom UI/LocalAPI maintenance. Tailscale has a [native device web interface](https://tailscale.com/docs/features/client/device-web-interface); a connected-host handoff is a candidate. | Not a drop-in: user-owned client/check-mode auth, tagged-device grants, HTTPS reachability and offline recovery can change the operator journey. Keep Soda enrollment/recovery and project policy until an actual replacement is proved. No new proxy or auth bridge selected. |
| [runner unit](../appliance/services/soda-runner@.service) and [Launch](../internal/runners/launch.go) | Smaller confirmed duplication: another executable repeats the unit's user directory/HOME then execs a fixed Forgejo runner. Direct `ExecStart` could retire the wrapper. | It would remove the descriptor precheck on boot/manual unit start. Preserve hardening and management validation; test startup failure and stop/restart. Do not add a new generic launcher. |
| [OAuth migrations](../internal/store/migrations.go), lines 21–57, [return structure](../internal/store/store.go), lines 249–255, and [return routing](../internal/web/auth.go), lines 236–255 | Smaller confirmed coupling: navigation destinations cause schema fields/CHECK changes. At a necessary schema update, one bounded destination discriminator plus existing actor/repository IDs could replace boolean combinations. | Keep PKCE, original actor, fixed URL construction and logout-winning cancellation. No arbitrary `return_to`, mass database rewrite or new routing framework. |
| [acceptance tool](../tools/soda-acceptance/main.go), lines 35–98, and [support package](../internal/acceptance) | Development-only orchestration has grown a compiled P/U workstream registry alongside QMP/process/SSH/evidence code. Separate plan-number metadata from executable transport contracts; keep scenarios in existing tests. | Do not replace this with libvirt or another daemon casually. Cancellation, descendant cleanup, pinned trust and secret-safe logs have real callers. A complete simpler replacement was not established. |
| [host package locking](../internal/hostimage/complete.go), lines 19–44, and [RPM inventory](../appliance/locks/host-packages-x86_64.json) | Large release-review surface: 17 requested packages, 170 install entries, 625 resulting inventory entries. It detects NEVRA drift but explicitly does not preserve RPM bytes. | Not proven gratuitous: qualification was selected. Do not promise reproducibility or add a package mirror as a 'small fix'. Decide whether generated inventory evidence is enough or snapshot retention is worth a separate commitment. |
| [Lit tool workspace](../tools/lit-check/package.json) and [resolver adapter](../scripts/check-lit.ts) | Moderate development cost from TS7 plus a classic TS5 analyzer. This is an adapter around upstream compilers, **not a custom compiler**. | Selected diagnostics and TS7 explain it. Revisit at the next toolchain choice; no safe equally capable replacement was proved. Do not drop checks or change the pinned compiler merely to make a line-count reduction. |

### Earlier simplification work that is still unfinished

The current [R3 reconciliation](refactoring-plan.md#removal-reconciliation-follow-up)
is corroborated: [StartTailnet](../internal/host/tailnet_companion.go), lines 291–305,
creates a new incarnation after stopping the old one; `stopTailnetRun`, lines 436–470,
never removes the stopped container, and [runFiles.prepare](../internal/host/tailnet_files.go)
creates per-incarnation directories without retirement. `/run` disappears on reboot;
Podman stopped containers/writable roots do not. This leaves ongoing operator cleanup.

This is **incomplete lifecycle follow-through**, not proof that the isolated
companion itself is unnecessary. Retire exact newly owned completed resources under
the existing lock after required shutdown/ownership checks; do not add a fleet
collector or run pruning on preserved fixtures. Cleanup-failure policy must be
explicit and must not introduce another permanent retry fence by accident. Source
command tests and authorized native lifecycle proof remain outstanding.

Old findings were not blindly counted again: the removed browser lease/heartbeat
supervisor, obsolete layout migrations, retired GitHub runner and unused bootstrap
secret retention are not reported as still-present giant mechanisms. Small remaining
duplicate dispatch/validation sites stay in the existing R4/R5/R7 records.

### Coverage, method and limits

The tracked inventory contains **1,113 files and 18 internal packages**. Root and
three independent domain reviewers inspected the package/entrypoint inventory and
followed affected build/runtime/frontend callers and tests. A separate counter-review
checked whether proposed simplifications preserve the requested outcomes. It caused
the locale recommendation to narrow, reaffirmed the selected same-media installer
requirement, and rejected describing the Lit adapter as a custom compiler. The
consolidated findings above supersede broader suggestions in individual notes.

| Repository area | Review coverage and disposition |
| --- | --- |
| `internal/web`, `store`, `forgejo`, `config`, `avatar`, `strictjson` and dashboard/setup commands | Account, OAuth/session/grant, repository/terminal, state/migration and request boundaries traced. MC6 and the navigation/optional-avatar costs; no independent authentication replacement proposed. |
| `frontend/{spaces,runners,tailnet}`, native browser hooks, local preview | Entry/disposal, state, layouts, transport, recovery and fixture owners examined. Multiple terminals and real xterm are user requirements; one disposable v3 layout and local mock development are justified. |
| `internal/host`, `tailnet`, `runners`, `linuxhost`, `filelock`, runtime CLIs and `project-os/rootfs` | Native account, terminal, service, project network and runner callers reviewed. Exact-recipe fragility, runner wrapper and completed-resource retirement noted. Fixed privileged operations and original identity checks remain necessary. |
| `internal/nativebuild`, `appliancerelease`, `hostimage`, `installlayout`, `installer`, `projectos`, appliance/tools/build scripts and locks | Build/stage/install/release/media graph traced. MC2–MC4 and package-lock tradeoff. CoreOS, Podman, standard Go compilation and digest-bound artifacts serve selected product requirements. |
| `internal/acceptance`, support commands, test-VM and installed test drivers | Transport/evidence/scenario ownership reviewed. Compiled plan numbering is unnecessary coupling; no claim a 63-line VM script replaces the stronger harness's security and cancellation behavior. |
| Forgejo templates/locales, branding/static assets, payload manifests and presentation/build tests | Exact 15.0.7 source comparison for copied templates, locale shadowing, native notification/search callers. MC1/MC5, CSS coordination and optional UI costs. Attribution/assets preserved. |
| Docs, root manifests, automation inventory and future feature plans | No tracked CI workflow declarations were found; current vs historical status and selected vs proposed scope examined. Iframe workspace, marketplace/AI/desktop and remaining release milestones are not misreported as shipped code. Product scope recorded only in docs is not independent proof of consent. |

Exact upstream Forgejo source was read from the retained 15.0.7 tree; Tea's official
release files and Tailscale's native UI documentation were checked. Findings are
source/caller evidence and explicitly bounded alternative research. No full source
suite, installer boot, browser comparison run, native runtime or destructive test
was performed for this audit. Future validation above is not a list of checks that
already passed. Detailed reviewer notes and inventory are retained in
`.artifacts/maintenance-commitment-audit/`; this document is the consolidated record.

## Historical audit — 10 September 2026

Reviewed on 10 September 2026 against installer candidate `8320f9c`.
**Source ownership review complete; findings are not implemented fixes.** This audit covers every `internal/` package present at that revision
and follows the actual application, native, frontend and build callers. It asks
whether Soda needs each responsibility and whether a mature upstream mechanism
can own more of it. It is separate from installer acceptance, a penetration test,
or permission for a broad rewrite/deployment.

**Document ownership:** this is a revision-bound record of findings, upstream
contracts and research evidence—not a backlog or current progress report. Statements
about code below describe the reviewed revisions, not necessarily today's source.
The [refactoring plan](refactoring-plan.md#7-audit-remediation-implementation-plan)
exclusively owns remediation steps, decisions, priorities and current source status;
the [handoff](development-handoff.md) owns execution and delivery evidence.
Do not maintain parallel completion flags or implementation checklists here.

The later [three-pass overengineering review](overengineering-review.md) records
findings at `7594458`, the assistant's reversals and the investigation of dependants.
It is a separate revision-bound supplement, not a rewrite of the findings below.

The [architecture](architecture.md) and [upstream-first instructions](../AGENTS.md#working-style)
remain the product boundary. A small adapter is justified when it binds native
operations to Soda's operator, project, actor or artifact contract. Reimplementing
the underlying forge, package manager, authentication protocol, container engine
or service supervisor is not.

The result supports keeping the architecture. It does **not** establish that Soda
has recreated those upstream systems wholesale. The clearest findings are unused
credential retention, inconsistent cancellation checks, file-update correctness,
missing schema checks and a provider service-entrypoint mismatch. Smaller reuse
candidates exist, but replacing working boundaries requires behavioral equivalence.

## Coverage and selected baselines

The source inventory contains **16 internal packages**. Package/file count is
coverage bookkeeping, not a design score. Reviews trace effectful callers and
existing tests rather than treating directory boundaries as separate authorities.
Production source versions are selected by the existing manifests and recipes:

- [Go modules](../go.mod): Go 1.26.7, coder/websocket 1.8.15,
  x/crypto 0.55.0, x/sys 0.47.0, modernc SQLite 1.58.0,
  DiceBear Go 10.7.0 and schema 1.5.1.
- [Frontend workspace](../package.json): Bun 1.4.2, TypeScript 7.0.2,
  Lit 3.3.3 and xterm 6.0.0. The former Cockpit React/PatternFly workspace is
  [retired in the stock-only source candidate](cockpit-port.md). [Lit analysis](../tools/lit-check/package.json)
  separately pins lit-analyzer 2.0.3 and its classic TypeScript 5.9.3 runtime.
- [Forgejo service](../appliance/services/forgejo.container): stock Forgejo
  15.0.7. [Caddy service](../appliance/services/soda-proxy.container): 2.10.2.
  [Project OS](../project-os/Containerfile): Rocky 10.2 userspace; native RPMs
  resolve during an authorized build and actual versions belong in build metadata.
- [CoreOS ISO](../appliance/locks/coreos-iso.json): 44.20260817.3.2.
  [Installer builder](../scripts/build-installer.py) selects CoreOS Installer
  0.26.0; selected FCOS metadata identifies systemd 259.8 and
  OpenSSH 10.2p1. Those base versions are not proof of downstream configuration,
  SELinux behavior or any retained appliance's installed package state.
- Historical GitHub runner baseline: 2.337.0, since removed at the user’s request.
  Forgejo Runner/Tailscale/Podman are native package inputs, not newly pinned by
  this report. Relevant upstream APIs and observed package metadata are discussed
  with their callers below.

| Internal package | Actual callers and upstream owner | Soda's remaining responsibility / verdict |
| --- | --- | --- |
| `acceptance` | [Support command](../tools/soda-acceptance/main.go), SSH/QEMU/QMP, existing build/check scripts | Exact requested phase/target, owned process cleanup, private evidence and transport. Keep bounded support; no independent product test scenarios or release scheduler. |
| `avatar` | [Web avatar adapter](../internal/web/avatars.go), dashboard startup and [preview](../tools/soda-avatars/main.go); DiceBear renderer | Original robot artwork, stable public seed and version/size contract. Keep; it calls the upstream renderer. |
| `config` | [Dashboard](../cmd/soda-dashboard/main.go), [setup](../cmd/soda-setup/main.go), runners; Go URL/JSON/filesystem APIs | Soda origins, socket/credential paths and operator binding. Keep validation; audit unused bootstrap-token retention below. |
| `filelock` | [Runner state admission](../internal/runners/native.go); kernel flock | Keep cancellable waiting around advisory locking. The kernel owns the lock; a Go-context wait is Soda's small addition. |
| `forgejo` | Web/setup/Tailnet commands; native Forgejo HTTP/configuration | Keep bounded acting-user APIs, actual consent inspection and sanitized transport. Forgejo owns passwords, repository permissions and Git; Soda owns its OAuth return/context binding. |
| `host` | [Host daemon](../cmd/soda-host/main.go), project image and web helper client; Podman, OpenSSH, Linux accounts, tmux/systemd | Keep fixed privileged operations and exact actor/project/account binding. Key-file concurrency, queued cancellation and capture bounds are findings below. |
| `installer` | [Media command](../appliance/installer/main.go), [builder](../scripts/build-installer.py); CoreOS Installer, NetworkManager, OpenSSL, Ignition, systemd/OpenSSH, Caddy | Text choices, irreversible-effect boundary, verified Soda payload and explicit first-use integration. Keep the adapter; native validation remains deferred. Custom TCP connection supervision was removed in this candidate. |
| `linuxhost` | Runner CLI/helper; NSS and pkexec | Keep the independently enforced root-only operator policy. Moving this small single-feature adapter into runners is optional cleanup, not an upstream replacement. |
| `nativebuild` | [Artifact command](../tools/soda-artifacts/main.go), build/stage/installer; Go archive/ELF/hash/filesystem, gpgv/xz, Podman | Bind source/platform and fixed public payload without runtime secrets. Keep these policies; compare the narrow OCI parser with upstream image readers before extending it. |
| `process` | [Tailnet status reader](../internal/tailnet/tailnet.go); Go os/exec | Only production consumer uses Output; unused command/trace surface and capture bounds are review findings. |
| `projectos` | Store/web/helper image inspection; Podman/OCI image identity | Keep immutable creation-profile binding. OCI owns image identity; Soda owns the selected profile/interface and its association with the created project. |
| `runners` | Native CLI/helper, Cockpit and global operator settings; provider runners and systemd | Keep local account/capacity/service integration, fixed operations and private registration input. Forgejo owns scheduling, workflow execution and cache; GitHub runner support has been removed. |
| `store` | Web/setup; SQLite and Go AEAD | Keep Soda-only associations, original account memberships, cancellation transactions and encrypted grants. Required schema verification is Soda-owned; random-nonce packing is a reuse candidate. |
| `strictjson` | Web/helper/runner requests and profiles; Go encoding/json | Bounded body, valid UTF-8, one object, top-level duplicate-name rejection and destination-struct unknown-field rejection. Keep on the selected Go baseline; not a general recursive schema engine. |
| `tailnet` | [Tailnet CLI](../cmd/soda-tailnet/command.go), [Forgejo advertisement](../cmd/soda-forgejo-tailnet/main.go), Cockpit; native tailscale CLI/LocalAPI | Keep native identity/address projection and separate Git advertisement. Review unstable LocalAPI dependence and native device UI overlap; preserve operator access and current preferences. |
| `web` | Dashboard command, native Forgejo hooks and Spaces frontend; Go HTTP/templates, Forgejo OAuth/API, coder/websocket | Keep Soda actor/context/access admission and exact terminal leases/receipts. Post-provider session admission is Soda-owned, distinct from native Forgejo and Lit/xterm responsibilities. |

## Findings

These sections establish what was observed and why the ownership boundary matters.
Their heading order is not an execution order. The plan's
[current status and order](refactoring-plan.md#current-status-and-order) is the only
remediation queue; it distinguishes completed work from outstanding findings.

### Remove unused bootstrap credential retention

[Setup](../cmd/soda-setup/main.go) copies the operator's bootstrap token into an
`admin-token` file; [configuration](../internal/config/config.go) requires its path.
[Activation](../appliance/bin/soda-activate) grants the dashboard service read
access to that file. A repository-wide production-consumer search found no runtime
credential reader: provider operations use acting-user encrypted grants. Activation
adjusts the unused file's permissions. This is **confirmed unnecessary retained
authority**, not evidence that the dashboard is making privileged provider calls.

The [setup guide](operator-setup.md) also requested admin/repository token
scopes although the inspected client calls only `/user` and `/user/applications/oauth2`.
The selected [Forgejo API routes](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/api.go)
place those operations under user scope. Operator eligibility is separate from a
need to call an administrator API. New-setup changes, compatibility and existing-file
handling belong to [phase 1](refactoring-plan.md#phase-1--retire-the-unused-bootstrap-token).

### Recheck the original session before native mutation

[Join](../internal/web/environments_api.go), [Lifecycle and key Apply](../internal/web/management.go)
capture a session, wait for provider identity/repository authorization and then
dispatch the native action without checking whether that same login context still
exists. Logout can complete during the provider wait. An ordinary HTTP context is
not cancelled by deleting its session. This is a **source-confirmed admission gap**;
the interleaving was not executed in this audit.

Create and [runner authorization](../internal/web/runners.go) already supplied a
comparison pattern. A fresh session check is an admission boundary, not atomic
cancellation or rollback of a dispatched native operation. The concrete handlers,
ordering and regression cases belong to
[phase 2](refactoring-plan.md#phase-2--finish-session-checks-at-mutation-admission).

### Complete current database validation

[Migration v8](../internal/store/migrations.go) introduces `creation_profile`,
`repository_settings_return` and the `immutable_creation_profile` trigger. At the
original audit revision the startup checks omitted all three. No damage to a
retained database was observed or alleged. [SQLite's integrity and foreign-key
checks](https://www.sqlite.org/pragma.html) validate storage and declared constraints;
they cannot infer Soda's intended schema. This warrants a required-schema check,
not an ORM or repair engine. Implementation and source status belong to
[slice B](refactoring-plan.md#b-complete-schema-v8-validation--independent-small-fix).

### Correct project-key concurrency before claiming compare-and-swap

[Project key maintenance](../internal/host/project_keys.py) reads and hashes the
current managed key file, writes a temporary replacement, rechecks the current
pathname and then calls `os.replace`. A native administrator can change the file
between that last check and replacement; Soda can overwrite the later change.
Directory `flock` coordinates cooperating Soda operations, not arbitrary native
editors. The [kernel contract](https://man7.org/linux/man-pages/man2/flock.2.html)
is advisory locking. This is a **source-confirmed correctness defect**, not a
runtime reproduction in this audit.

Native OpenSSH files remain the selected boundary. Existing-inode append can support
enrollment but does not implement whole-set revocation/replacement. An advisory lock
cannot provide compare-and-swap against noncooperating root editors. The installer
enrollment append correction does not fix this separate project path.

The subsequent planning review found a related durability issue: the staged key
file uses buffered writes and calls `fsync` before an explicit flush. The writer
contract decision, durability correction and concurrency tests belong exclusively to
[phase 3](refactoring-plan.md#phase-3--managed-key-concurrency-contract).

### Make helper admission cancellable and bound command capture

At the audit revision, [buffered host dispatch](../internal/host/daemon.go) held one
mutex across native operations, including creation's readiness wait. A cancelled
waiter could not leave that mutex wait. This is distinct from a need for parallel
project operations.

The executor also collected unbounded output before callers enforced response limits.
A post-capture limit does not bound allocation. [Go os/exec](https://pkg.go.dev/os/exec)
remains the process owner; bounded capture is an adapter responsibility, not a reason
for a process framework. Separate admission/capture status and implementation belong
to [phase 5](refactoring-plan.md#phase-5--cancellation-and-capture-bounds).

### Preserve native project and terminal ownership

The [existing-project service](../appliance/services/soda-project@.service) starts
and stops the already-created persistent Podman container. Podman's
[existing-container service example](https://docs.podman.io/en/latest/markdown/podman-generate-systemd.1.html)
supports that shape. Although [Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
is already used for appliance services and supports templates, changing retained
projects requires proof that their original container IDs, writable roots and
start/stop behavior survive. A declarative rewrite is not automatically safer.

[Project account provisioning](../project-os/rootfs/usr/libexec/soda/project-account)
uses native account tools and adds only Soda's identity association, shared path
and key installation. Persistent people/accounts are not a `DynamicUser` use case.
[Terminal integration](../internal/host/project_terminal.py) uses tmux for sessions
and systemd for lifetime; Soda binds exact accounts, PTYs and cleanup receipts.
The [nested Podman socket](../project-os/rootfs/etc/systemd/system/soda-podman.socket)
already delegates activation to systemd and the API/runtime to Podman. Preserve
these boundaries; an unrestricted host engine socket would remove necessary policy.

[OS observation](../internal/host/project_os.py) reads one bounded regular file from
the existing project. Python's [standard convenience reader](https://docs.python.org/3.10/library/platform.html#platform.freedesktop_os_release)
adds fallback behavior and does not itself establish Soda's file-size/type policy.
Keep the bounded observation unless a concrete native-parser replacement preserves
those constraints. It must neither start a stopped project nor infer installed
packages from a distro label.

### Delegate runner execution through the provider's service contract

The earlier GitHub service-entrypoint finding is historical: the user subsequently
selected removing GitHub runner support. Its original evidence is preserved in Git
at `216db47`; the correction is retired in [phase 4](refactoring-plan.md#phase-4--github-runner-service-compatibility).

The remaining [Forgejo runner](../internal/runners/) delegates accounts to Linux,
locking to flock, services to systemd and jobs/registration authority to Forgejo.
The root CLI and configured web operator gates remain distinct. Forgejo connection
tokens use a private file; no registration PTY is needed. Local removal does not
remove Forgejo's registration or history, and partial local outcomes remain explicit.

Current [Forgejo configuration](../internal/runners/native_create.go) selects host
labels, disables container-engine access and disables cache. The selected Rocky
ephemeral job image and native cache integration are **unfinished feature work**,
not existing isolation or a reason to build a Soda scheduler/cache server.
[Forgejo's guide](https://forgejo.org/docs/latest/admin/actions/) assigns jobs to its
runner, logs/artifacts to Forgejo and cache storage to the runner. Validate the actual
installed runner version and its configuration before implementing those choices;
the layered package recipe does not pin the same version as the Forgejo server.

Cockpit and the [native operator page](../frontend/runners/soda-runners-page.ts)
reuse the runner backend and [shared response decoder](../frontend/runners/soda-runner-response.ts).
Retain Cockpit's backing logic/tests until the selected replacement meets its whole
journey. Two presentation entrypoints during that migration are not evidence of two
CI schedulers. Keep provider registration, permissions, results and workflow UI upstream.

### Review Tailnet's unstable API seam and native UI overlap

The following findings describe the historical custom Cockpit caller; it is now
retired in source. [The Tailnet plan](tailnet-integration-plan.md) owns the protected
Go/Lit replacement and pending native acceptance.

The former `cockpit/src/tailscale/native.ts` adapter used native CLI
status/up/set and accesses LocalAPI preferences and interactive login through the
privileged Unix socket. Upstream's [LocalAPI source](https://github.com/tailscale/tailscale/blob/main/ipn/localapi/localapi.go)
explicitly treats the v0 routes as internal and not necessarily stable. Exact
installed Tailscale version is unresolved by the layered package recipe. Record and
validate that baseline; do not describe the endpoints as a stable public API.

Tailscale's [device web interface](https://tailscale.com/docs/features/client/device-web-interface)
already offers exit-node selection/advertisement and device settings. This is a
concrete reuse candidate before expanding Soda's Tailnet forms. It has its own
reachability, device identity and management-access requirements, so it is not a
drop-in replacement for root-operator Cockpit or a first-enrollment solution without
further validation. Preserve current controls/tests until an equivalent journey
works. Never reset unrelated preferences merely to simplify reauthentication.

The former authentication stream (`cockpit/src/tailscale/stream.ts`) framed successive
JSON objects and delegates parsing to `JSON.parse`; it is not another JSON engine.
At the review revision its accumulated partial object had no size bound. This is
an adapter resource-bound gap, not a need for a stream framework. The correction
and test cases belong to [phase 5c](refactoring-plan.md#phase-5--cancellation-and-capture-bounds).

The former Tailnet page observer (`cockpit/src/tailscale/store.ts`) could call
[Forgejo address refresh](../cmd/soda-forgejo-tailnet/main.go), which edits the native
Git SSH advertisement and may restart Forgejo. Preserve and expose that effect when
changing the UI; the complete caller chain is not read-only status. It keeps browser
and OAuth origins separate and refuses an unpublished Git listener. Tailnet presence
does not prove approved subnet routing or client reachability.

### Simplify local boundaries only where it removes real machinery

[Key maintenance](../internal/host/management.go) preloads the whole terminal module
to reuse `account_for`. This is already one validator, not duplicate identity policy.
An extraction affects both immutable installed and embedded paths; loading fewer
function definitions alone does not establish a net benefit. The decision belongs
to [slice E](refactoring-plan.md#e-native-and-terminal-coupling--separate-conditional-slices).

`internal/process` has one production consumer and unused Run/trace machinery.
A Tailnet-local output seam can remove that surface. `internal/linuxhost` is already
a small legitimate root/NSS adapter; co-locating it with its runner owner is optional.
Retain tested cancellation/root checks. These are conditional local simplifications,
not evidence of copied upstream subsystems or prerequisites for finishing features.

### Keep artifact policy; review general OCI parsing before expanding it

[InspectOCI](../internal/nativebuild/oci.go) implements a bounded single-platform
archive reader: it rejects unsafe/duplicate paths, hashes every blob entry and
validates referenced descriptor sizes/digests, resolves the local manifest/config
and binds Soda's architecture/revision
labels. It does not unpack or execute layers. The [build](../scripts/build-native.sh)
and [installation](../scripts/install-native.sh) already delegate image construction,
export and import to Podman using actual image identities.

OCI is upstream's format, and [Skopeo exposes manifest/config inspection](https://github.com/podman-container-tools/skopeo/blob/main/docs/skopeo-inspect.1.md).
Compare a pinned upstream reader or verified-copy operation before adding format
support. A successful metadata inspection alone is not an equivalent replacement
for this whole-archive verification. Keep Soda's private-payload exclusion, identity
binding and no-import preflight; use malformed/tampered fixtures to establish
equivalence. This is a **reuse candidate**, not evidence that deleting the current
validator is safe.

The [OCI v1.1.1 layout specification](https://github.com/opencontainers/image-spec/blob/v1.1.1/image-layout.md)
permits content beyond Soda's sealed, local, single-image subset. Document that
subset rather than presenting its refusals as universal OCI requirements.

### Keep the existing test runners; remove repeated preparation

At the original audit revision the [root test scripts](../package.json) invoked
`build:forgejo` through multiple aggregate branches, and existing Go HTML/browser
consumers were not all wired into ordinary test commands. These were invocation
and preparation gaps, not missing test engines. The [native checker](../scripts/check-native.sh)
and [remote support executor](../internal/acceptance/remote_executor.py) already
shared the production build/check owners. The source work and status belong to
[slice A](refactoring-plan.md#a-reliable-single-preparation-source-checks--first).

A subsequent review reproduced a macOS path-comparison failure in
[the source-command fixture](../tests/build/test_source_checks.py): the unresolved
fixture path differed from the child's physical cwd. Its bounded correction belongs
to [phase 0](refactoring-plan.md#phase-0--reliable-baseline), not a change to native gates.

### Preserve native web authority and narrow frontend cleanup

Selected [Forgejo OAuth source](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/auth/oauth.go)
omits scopes from the token response, can reuse an existing grant and exposes actual
consent through introspection. Soda's bounded grant inspection, refresh serialization,
transaction-bound callback/logout cancellation and encrypted actor/session binding
therefore remain necessary. A general OAuth client cannot replace those policies.
Similarly, Go's cross-origin middleware does not supply Soda's required exact
origin, CSRF token, actor context or WebSocket first-frame authorization. Keep those
checks rather than treating them as duplicate authentication.

SQLite owns persistence; [Go's existing random-nonce GCM API](https://github.com/golang/go/blob/go1.26.7/src/crypto/cipher/gcm.go)
could replace the manual nonce-prefix mechanics in [grant encryption](../internal/store/grants.go).
This is a small **reuse candidate**, not a cryptographic defect. Require old/new
ciphertext compatibility, wrong-key and actor/session-binding fixtures; retain the
current key and associated-data contract without rewriting stored grants blindly.

[Lit controllers](https://lit.dev/docs/composition/controllers/) already own reactive
lifecycle composition. [Task](https://lit.dev/docs/data/task/) is an optional read-only
request candidate; cancellation/latest-result handling does not undo mutations or
resolve uncertain outcomes. Xterm/FitAddon own terminal rendering/fitting. Its
[AttachAddon](https://github.com/xtermjs/xterm.js/blob/6.0.0/addons/addon-attach/src/AttachAddon.ts)
is raw transport and does not implement Soda's authenticated control frames, exact
IDs or cleanup receipts. Preserve the existing bounded transport and flat terminal
host identity; no new terminal manager or frontend state framework is warranted.

One concrete removal candidate is [terminal standalone mode](../frontend/spaces/sodaspaces-terminal.ts).
Every inspected production caller supplied a workspace locator; the no-locator
caller was an older fixture. That makes the alternate mode a reuse/removal candidate,
not permission to erase [legacy locator import](../frontend/spaces/sodaspaces-workspace.ts)
or unknown/pending sessions. Evaluation and test preservation belong to
[phase 7](refactoring-plan.md#phase-7--optional-cleanup-after-correctness).

All **253 Forgejo template overrides** were inventoried and compared for identical
copies against the retained native export; none was byte-identical. Representative
login, settings, notification and shared navigation callers preserve native handlers
and permission authority. This is ownership coverage, not independent proof of every
field/gate in all 253 templates. The [official Forgejo customization mechanism](https://forgejo.org/docs/v15.0/admin/advanced/customization/)
explicitly lacks backward-compatibility support. Keep version-specific template and
route checks; documented customization is not a stable API, and it does not require
or authorize a downstream executable fork.

### Keep bounded JSON policy on the selected toolchain

[strictjson](../internal/strictjson/decode.go) uses Go's parser, adding request
constraints that legacy `encoding/json` does not supply by default. Its duplicate
check applies to top-level field names; unknown-field refusal applies while decoding
the destination struct shape, including nested structs.
It should not claim recursive duplicate rejection or general schema validation.

Current [Go JSON documentation](https://pkg.go.dev/encoding/json) describes v2 in
Go 1.27. Inspection of the selected Go 1.26.7 source in the offline Linux checker
found v2 still behind `GOEXPERIMENT=jsonv2`, outside that version's compatibility
promise. Do not incidentally change the toolchain or enable an experiment for this
audit. Reconsider parsing/duplicate handling during an intentional baseline migration,
preserving body limits, object-only input, explicit unknown-member refusal and current
protocol behavior. Native v2 does not itself remove those application policies.

### Keep upstream-rendered artwork and native support protocols

[The avatar adapter](../internal/avatar/avatar.go) calls the installed DiceBear
10.7.0 `NewStyle`/`NewAvatar`/`SVG` implementation. Soda supplies original artwork
and deterministic seed/version policy. It is not a second avatar renderer.

The [QMP client](../internal/acceptance/qmp.go) implements capability negotiation,
request ID correlation and event skipping against the
[upstream protocol](https://www.qemu.org/docs/master/interop/qmp-spec.html).
QEMU owns virtualization. The support process wrapper owns only its launched child
groups and evidence; it does not replace the appliance's systemd supervisor.
Replacing these bounded clients with a broad orchestration framework has no
demonstrated benefit in this review.

The [Lit analysis adapter](../tools/lit-check/README.md) likewise uses the actual
upstream analyzer. Its separate classic compiler addresses documented package
resolution failures under the product's TypeScript 7 workspace. Keep it until
supported upstream versions remove that concrete mismatch; do not patch dependency
source or recreate the analyzer.

## Evidence and limits

The working reports and file inventory are retained under ignored
`.artifacts/upstream-audit-20260910/`; they are supporting evidence, not a build
dependency. Selected source, actual callers, manifests and primary upstream
contracts establish these recommendations. Current upstream pages identify
capabilities; they do not silently select a new dependency or establish installed
compatibility. Failed page retrievals were not treated as missing features.

No product refactor, native/provider action or deployment is performed by this
audit. Installer source checks and their actual results remain in the separate
[handoff](development-handoff.md). The x86 media build and full install remain
deferred. Findings outside the installer need their own bounded implementation,
regression checks and any required paired native delivery before being called fixed.

The interrupted runner review did not leave a completed artifact; the root reviewer
finished its source/caller review directly. No audit subagents remained active at
that continuation. Native/build/web reports and an independent build-support review
were retained; the additional planned independent web recheck was interrupted and
is not claimed complete. Root directly corroborated the credential consumers,
mutation paths and schema checks. Coverage includes all internal packages and
caller families, not exhaustive security proof or line-by-line parity of every
template, native dependency and UI branch.

### Earlier upstream-review provenance

The initial maintainability review used `c20abc3` and retained public responses,
source hashes and failed retrievals in `.artifacts/refactoring-upstream-review-GVDpXf/`.
It ran no product builds/tests, dependency installation, downloaded code or native
operations. Fresh Forgejo 15.0.7 template lookup and CoreOS Installer 0.26.0 CLI
source matched retained bytes; native navbar/key and tmux 3.2a findings used retained
selected source. Lit findings used locked reactive-element 2.1.2. The reviewed
systemd v259/OpenSSH 10.2p1 documentation aligned with selected CoreOS metadata,
not a survey of retained project packages or proof of downstream configuration.
An initial Bun documentation URL returned 404; the official run page succeeded.

Its corrections still matter: Lit Task supports manual execution, but cancellation
cannot undo mutations or resolve uncertain outcomes; systemd's watchdog does not
supply Soda access authority; tmux attachment is not web authorization. The installer
already had `executeDisk` and used CoreOS Installer/NetworkManager rather than needing
a new execution engine. These are ownership findings, not new implementation tasks.
The original comparison and full upstream reference list remain in Git at
`22f5c20:docs/refactoring-plan.md` (sections 1 and 6). This preserves historical
research without maintaining a second live responsibility matrix or task list.
