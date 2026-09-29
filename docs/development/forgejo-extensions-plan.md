# Fountain: extensible Forgejo implementation plan

This is the remaining implementation plan for the native Forgejo extension
platform and its complete Soda integration. It starts from the extension
foundation already implemented in the `forgejo-ext` repository. It is an active
transition plan, not a claim that the planned interfaces exist. Absorb its final
contracts into the owning guides and remove this document when the transition is
complete, following the [documentation authority rules](../README.md#authority-rules).

The delivery is complete when Soda runs through independently packaged native
Forgejo contributions, uses native browser authority, retains a real terminal
while navigating Forgejo, keeps its privileged backend isolated, and ships through
the existing native appliance pipeline. A notes-panel demonstration is only the
foundation for that result.

## Starting point and remaining gap

Inspected baseline: Soda `c5fda072`, Forgejo extension commit `90a2fcf97d`, based
on Forgejo v15.0.9 LTS (`19b9b9d216bbfb501c18514bd1a8c980246ca3f7`). These are
planning inputs, not instructions to downgrade a checkout or discard later work.

| Area | Already implemented | Still required |
| --- | --- | --- |
| Package runtime | Administrator-installed process packages; manifest; private Unix HTTP; restart activation; install, replace, enable, disable | Independently consumable SDK, declared host capabilities, useful diagnostics, removal, mandatory-package behavior |
| Native contributions | Global, user, repository and site-admin pages; native menus/layouts; coarse route permissions | Soda pages and assets, narrower application authorization, correct operator navigation, preferred workspace entry |
| Request authority | Native session only; native actor/repository context; credential/header stripping | Live authority handles, callback capabilities, explicit operation authorization, separate Soda service bridge |
| Persistent browser host | Native pages in a frame beside retained extension panels; history and cached-document handling | Real Lit/xterm terminal, continuous stream authorization, account-change handling, automatic entry preserving product behavior |
| Backend transport | Buffered ordinary HTTP, 30-second deadline, 8 MiB body bounds | WebSocket transport and cancellation across the actual session middleware and containers |
| Core extensibility | Package discovery and page/panel registration | A concrete native policy hook with complete caller coverage; selected observation events |
| Soda adoption | Existing OAuth adapter, shell, template customizations and service remain | Native replacement, old-path deletion, isolated backend integration, updated owning contracts |
| Delivery | Local macOS arm64 host/example builds and browser evidence | Native Linux container proof, patched appliance image, separate Soda package, installation and qualification |

The foundation has focused Go and browser checks, actual Chrome navigation and
cache-restoration checks, and package replacement evidence showing unchanged host
bytes and retained notes. It has not proved a Soda terminal, Linux container
execution, session-safe streaming, or a qualified appliance release. Existing Soda
tests exercise the current OAuth and shell implementation until ported.

## Target architecture and decisions

```mermaid
flowchart LR
    B[Browser] --> F[Forgejo native session and routes]
    F --> W[Core workspace host and extension UI]
    F --> E[Installed Soda extension process]
    E -->|private application requests and streams| S[Isolated Soda API service]
    S -->|scoped host-capability calls through extension| F
    S --> H[Existing host and identity helpers]
    S --> D[Soda private database]
```

1. **Keep a small downstream native substrate.** Core owns session admission,
   native contribution placement, a persistent workspace host, explicit host
   capabilities, and selected policy/event seams. Soda owns product behavior.
   Do not export Forgejo internals, database handles, or arbitrary service calls.
2. **Keep the Soda service isolated.** It currently runs as UID 2000 with its
   own database, configuration and host/identity helper access; Forgejo runs as
   UID 1000 with its data directory. The extension is a thin native-authority and
   UI/transport adapter. Moving all Soda backend code into Forgejo would expose
   those resources to Forgejo and every installed extension sharing its user.
3. **Use a narrow private bridge, with core-validated authority.** Give the
   extension access only to a dedicated application IPC endpoint. The Soda
   service validates a live, scoped admission through the host capability path;
   plain actor headers or socket access alone are not authentication. No broad
   `/run/soda` mount, database mount, grant key, host-helper socket, or Podman API
   is added to Forgejo. Prove the two-way connection before choosing deployment
   permissions and before porting every endpoint.
4. **Native browser authority replaces the browser OAuth adapter.** Cookies and
   raw session IDs remain inside the native host. There is no temporary new
   OAuth-backed extension implementation to remove later. Preserve the separate
   broker/account-custody and factory grant mechanisms described by
   [Credentials](../reference/credentials.md); they are not browser adapter code.
5. **Prove streaming first for memory and file sessions.** Inspect the effective
   fixture configuration first. Other providers must reject extension streams
   explicitly until their revocation semantics are proved; ordinary extension
   HTTP remains usable. This avoids promising identical provider behavior without
   evidence or building a replacement session system.
6. **Use mature upstream session/transport mechanisms where they work.** The
   pinned `go-chi/session` already has `IgnoreReleaseForWebSocket`; integrating it
   still requires real response-writer and invalidation proofs. A second browser
   credential or independent authentication database is not the solution.
7. **Add policy at the actual common native seam.** A small dependency-neutral
   callback can cover user creation/rename without moving every caller into a new
   service hierarchy. Notifications cannot veto operations. Add durable delivery
   only for a demonstrated obligation that cannot be met by reading current state.
8. **Packages remain administrator-trusted.** Process separation is a lifecycle
   boundary, not a sandbox. Same-origin extension JavaScript and same-user backend
   processes are trusted. The private bridge limits access to Soda's privileged
   service; it does not promise isolation from a malicious installed extension.

Three independently reworded Jev consultations agreed on native-boundary-first
delivery, memory/file-first streaming, and the common neutral hook. Three further
consultations agreed on preserving the separate Soda backend through a private
bridge. Requests, responses and wording/equivalence audits are retained in ignored
planning evidence. Agreement is advice, not proof; the gates below provide proof.

## Dependency order and parallel ownership

Use three implementation lanes, with one integrator reviewing their shared
contracts. Use Codex implementation agents; the explicit instruction excluding
Muse applies throughout this work. The executable task order below replaces
whole-milestone barriers: P0–P8 remain the detailed scope and acceptance contracts,
not a requirement to finish an entire milestone before starting another.

| Lane | Owned changes | Dependencies and handoffs |
| --- | --- | --- |
| A — Native authority and core hooks | Forgejo session integration, admission/capability handlers, HTTP/WS route enforcement, policy/event seams | Unblocks the authenticated bridge and terminal first; policy implementation follows the required-package runtime |
| B — SDK, host runtime and delivery | Public SDK, callback connection lifecycle, package manager, native workspace host, Linux fixture and appliance/release integration | Publishes compilable interfaces early; prepares the native browser host while C ports product APIs; owns the long-running native fixture |
| C — Soda API and Lit consumer | Thin Soda extension, existing API/auth owners, product policy, Lit entrypoints and retained terminal, old browser integration removal | Prepares mount/adapter code against the shared SDK; proves a narrow real bridge before broad API migration |
| R — Integrator | Shared contracts, file allocation, cross-repository commits, targeted review, integration receipts, owning documentation | Keeps downstream tasks ready; does not become a fourth long-running implementation lane |

Assign files before parallel edits. The integrator owns cross-repository interface
changes, documentation decisions, final acceptance and commit boundaries. Do not
have two agents independently invent bridge DTOs, authentication semantics or
release drivers. Reuse existing agents and evidence; each review answers a concrete
remaining question rather than restarting an architecture audit.

## Ordered implementation task list

All boxes below start unchecked; the preceding foundation is already complete.
`Needs` lists hard predecessors whose stated output must exist. A written interface
permits preparation, but never counts as a working capability or native proof.
Task IDs are stable references, not numeric execution order. Each task includes
its focused checks and updates to the owning interface documentation when behavior
lands. Use Git history and ignored receipts for results rather than another status
file. This list is part of the active plan and is removed with it at completion.

### First dispatch and scheduling rules

1. R completes **R00**, then immediately dispatches **A01**, **B01** and **C01**
   together. These use the completed baseline and do not need a new protocol.
2. When C01 returns the caller map, R completes **R01**. B prioritizes **B02**;
   A can finish the session proof while C prepares the exact UI/adapter inventory.
3. After B02, start **C02**; start **A02** as soon as A01 has also passed. Run them
   concurrently when both are ready. A can use a wait on SDK work for A05's bounded
   caller mapping rather than editing B's files. B03 can begin after R01 without
   waiting for live authority; B10 later integrates actual account restoration.
4. Once A02, B01 and C02 all pass, **C03** proves the real bridge while A prioritizes
   **A03** and B finishes **B03/B10**. R accepts the bridge as **R02** without rerunning
   unchanged checks. Once R02 and A03 pass, **A04** and **C04** proceed in parallel
   with B's remaining host/lifecycle work. They do not wait for A08's other reads.
5. R accepts the real terminal path as **R03**. C prioritizes **C05**, then **C06**;
   A finishes the mandatory policy and selected events while B prepares packaging.
   Coordinate C07/B08 cutover before the integrated development journey.
6. Release admission work has its own dependency chain after native development
   proof. Final x86_64 and aarch64 qualification may run in parallel on their
   respective native hosts after one shared source/artifact freeze.

These are earliest-start bands, not all-lanes-finish-together waves. Start a task
as soon as its own predecessors and files are available. If a task blocks, select
another ready task in that lane; reassign a ready task only with an explicit file
handoff and without exceeding three workers plus R. Do not manufacture additional
features to keep agents busy.

No duration estimates are assumed. Prioritize work that unlocks the next real
integration boundary, then independent long-running native probes, then other ready
tasks. Reassess this priority using observed duration/blockers; the dependency graph
establishes a safe order, not a measured globally shortest schedule.

| Lane | Preferred ready queue; skip blocked entries |
| --- | --- |
| A | A01 → A02 → A03 → A04 → A08 → A05 → A06 → A07. Pull A05 forward during an SDK/bridge wait; it unblocks B05. A08 can also fill a bridge wait and unblocks C06. |
| B | Launch B01 → prioritize B02 when R01 is ready → B03 → B10 → B05 → B06 → B07 → B04 → B08 → B09 → Q01 → Q02. Use B04/B09 during a blocked handoff; B05 unblocks mandatory policies. |
| C | C01 → C02 → C03 → C04 → C05 → C06 → C07. Pull C06 forward while waiting for streams or the browser host; do not delay ready C05 for optional UI refinement. |
| R | R00 → R01 → R02 → R03 → R04 → R05 → R06 → R07 as each gate becomes ready. Review small handoffs between gates. |

Q03/Q04 replace A/C's queues after R06. A fixture process waiting for a bounded
result is not permission to add another agent; its lane can perform disjoint ready
work against frozen source inputs, then inspect the result when it completes.

### R — Contracts, acceptance gates and integration

- [x] **R00 — Record the starting state and reserve files.** Needs: none.
  Inspect both canonical checkouts; preserve unrelated changes; record revisions,
  existing receipts, three lane assignments and shared-file ownership. Done when
  A01/B01/C01 can run without overlapping writers or mutable build inputs. Scope: P0.
- [x] **R01 — Publish the minimum shared contract.** Needs: C01.
  Fix the SDK DTO/callback signatures, live admission versus session-generation
  binding, operation/context requirements, mount contract, and policy/runtime
  handshake. Update intended architecture owners. Done when each lane knows its
  inputs/outputs and one owner holds each shared file; no native proof is claimed.
  Keep public publication/namespace commissioning outside local development. Scope: P0/P2.
- [ ] **R02 — Accept the authenticated bridge gate.** Needs: A02, B01, C03.
  Review the actual read/mutation, forgery/Origin denials and container isolation
  receipts. Done when ordinary native API migration and stream integration can use
  this demonstrated boundary; do not wait for every capability or UI page. Scope: P1c.
  The live bridge currently passes on macOS with both processes sharing one UID.
  B01 separately proves the Forgejo package runtime on native x86_64 Linux; the
  bridge still needs its Linux cross-UID, socket and SELinux isolation receipt in C03.
- [ ] **R03 — Accept the real terminal gate.** Needs: A04, C04.
  Exercise the existing terminal driver through native admission, both proxy
  directions and Soda to an actual retained shell. Done when input/output,
  revocation, disconnect and cross-request continuity meet P3/P4. This is backend
  terminal proof; persistent browser-element proof is C05.
- [ ] **R04 — Land the complete browser cutover.** Needs: A06, A07, B04, B07, B08, C07.
  Review and commit the coordinated routes/assets/configuration/policy change.
  Done when only the native integration is shipped, required-package activation
  is coherent, and retained credential/branding behaviors have focused evidence.
  Do not use a transient development seam as a shipped compatibility path. Scope: P4–P7.
- [ ] **R05 — Accept integrated native development.** Needs: R04.
  Run/reuse the affected native image, installed browser and lifecycle journeys
  against the same identified artifacts. Done when the P8 development matrix is
  satisfied, including package replacement, preserved data and actual isolation.
  Label the result development-only; this does not restore production admission.
- [ ] **R06 — Freeze qualification inputs and current guides.** Needs: R05, B09, Q02.
  Complete the P8 owner-document map and upstream maintenance procedure, commit
  coherent source in both repositories, and bind the exact native inputs. Done
  when Q03/Q04 have committed, reproducible identities and no pending source edits.
- [ ] **R07 — Close the delivery plan.** Needs: Q03, Q04.
  Check native evidence for every advertised architecture and record exact limits.
  Absorb final contracts and delete this completed plan/task list. External
  publication and production activation remain separate actions under the existing
  scope rules. If a required target is unavailable, leave its task and R07 incomplete.

### A — Native session, authority, streaming and policies

- [x] **A01 — Prove and fix native session/response behavior.** Needs: R00.
  Implement the smallest real-server memory/file echo-stream probe and bounded
  middleware/writer fixes. Done when upgrade, logout/delayed-release race,
  regeneration, expiry and provider failure meet P1a without a new session system.
  Focused TLS probes pass through the real Forgejo web middleware and native login/
  logout path, plus memory/file invalidation probes.
- [x] **A02 — Implement the minimum live admission slice.** Needs: A01, B02.
  Add request-lifetime admission/invalidation, instance-bound callback validation,
  native session-generation binding, and the current-actor read needed by C03.
  Done when the real extension process receives a host callback, forged browser
  headers are ignored, and stale session/admission attempts fail. Scope: P2.
- [x] **A03 — Complete terminal authority and permission rechecks.** Needs: A02.
  Supply current actor, stable repository identity/code permission and the live
  permission-recheck capability needed by reserve/create/attach/end. Done when
  native permission changes revoke this authority and both stream/consumer lanes
  can use the working slice. Picker, public-key and page-guard work is A08 and
  does not gate the terminal. Scope: P2/P3. The callback supplies the current actor,
  stable repository ID and current Code permission; the native integration check
  rejects a reader after the repository becomes private. Continuous stream revocation
  remains A04's responsibility.
- [ ] **A04 — Implement host-side WebSocket proxy and revocation.** Needs: R02, A03.
  Extend the native route transport with P3's Origin/header, backpressure, timeout
  and continuous authority rules. Done when the real private path carries an echo
  stream, closes both directions correctly and preserves ordinary HTTP bounds.
  Coordinate with C04; only R03 can claim the real Soda terminal works end to end.
- [x] **A05 — Map the policy seam and all native callers.** Needs: R01.
  Identify create/rename entrypoints, transactions, filesystem effects, CLI loading
  and the bounded required-policy interface. Decide the current Soda policy applies
  to personal accounts only, and record ActivityPub preflight and bootstrap ordering.
  Done when A06 can insert its veto before side effects without unbounded RPC under
  transaction locks. Scope: P6. Caller map and decision are in ignored evidence;
  three consultations are advice, not implementation proof.
- [ ] **A06 — Enforce the username policy across web and CLI.** Needs: A05, B05, C02.
  Implement native hooks and the narrowly assigned Soda policy handler using the
  common contract. Done when all supported real entrypoints enforce the policy;
  missing/crashed/timeout decisions fail closed and leave no identity/path changes.
  Keep bootstrap ordering explicit and retain existing users. Scope: P6/P7. ActivityPub
  identities are intentionally excluded because their federation names cannot satisfy
  Soda's Linux login policy. Signup, admin UI/API, self-rename and CLI checks pass;
  installer runtime wiring compiles, but full installer/browser bootstrap remains open.
- [x] **A07 — Wire only the selected observation consumers.** Needs: A02, C01.
  Implement/document the required bounded observations and snapshot refresh;
  reuse authority invalidation already in A02/A04. Done when actual consumers have
  truthful timing/coverage/missed-event behavior. Add no unused event families or
  durable bus; if no extra observation is needed, record that finding. Scope: P6.
  Source review found no additional event or snapshot hook is needed: the bridge checks
  authority per request and the actual consumers refresh through current bounded reads.
- [x] **A08 — Finish the remaining reads and contribution guards.** Needs: A03.
  Reuse the current native permission checks for repository picker/search,
  ownership and public keys; add bounded narrowing of page/navigation access.
  Done when real consumer and operator/non-operator cases pass. This unblocks C06
  independently of WebSocket completion; no arbitrary core write API. Scope: P2.
  Native integration covers owner-only search, organization ownership, public keys
  and page guards; a two-page search regression confirms no repository is skipped.

### B — SDK, native host, package runtime and delivery

- [x] **B01 — Prove the existing package in native Linux containers.** Needs: R00.
  Use the pinned baseline to identify the executable/entrypoint, build the native
  host/example and check UID/labels/socket/data/replace behavior. Done when P1b's
  receipts identify real native bytes. Do not wait for session or UI work. Existing
  Receipt `.artifacts/fountain-b01/receipt.md` records a native x86_64 rootless Podman
  run with SELinux enforcing against the pinned Alpine image: patched musl host and
  standalone package startup, UID/GID 1000 runtime/socket/data ownership, `/data`
  labeling, stop/start replacement, retained private data and unchanged host hash.
  This does not qualify aarch64 or a production candidate.
- [x] **B02 — Extract the shared SDK and working callback channel.** Needs: R01.
  Move the single current wire contract and SDK into its independent module;
  connect the instance-bound callback transport and update host/example imports.
  Done when an external consumer builds without the host checkout and A/C can
  compile against real types/channel wiring. A02 now enforces admission over the
  callback transport; a transport echo alone would not prove that authorization.
  Scope: P2.
- [x] **B03 — Prepare generic native workspace behavior.** Needs: R01.
  Implement preferred entry, exclusions, stable panel roots and ordinary
  navigation/forms/history/subpath behavior against the mount contract. Done when
  existing sample-panel checks pass without a Soda shell. This can proceed before
  live admission; account restoration is a separate hard gate in B10. The preferred
  entry is opt-out for the current navigation and eligible signed-in pages only.
  Scope: P5.
- [ ] **B04 — Add operator-visible package status and safe removal.** Needs: B01, B02.
  Implement bounded startup/exit diagnostics, status and stopped-runtime removal
  retaining private data. Done when lock/replace/remove/crash cases pass and
  diagnostics disclose no credentials. Local lifecycle tests pass; accept package
  replacement/removal after B01's Linux runtime check. Scope: P7.
- [x] **B05 — Add required-package and policy lifecycle enforcement.** Needs: A02, A05.
  Implement required IDs, offline/online CLI loading/lock behavior, crash admission
  cancellation and fail-closed hook availability. Done when A06 can rely on real
  runtime enforcement and bootstrap/activation ordering is testable. Scope: P6/P7.
- [ ] **B06 — Wire the proven image, package and isolated service.** Needs: R02, B02, B05.
  Promote the small Linux/IPC proof into the patched image and separate Soda package
  recipes and narrow units/configuration. Done when the task-owned development
  fixture starts the intended components; no full release/media orchestration yet.
  Keep recipe edits separate from C's frontend package content. Scope: P7.
- [ ] **B07 — Integrate existing candidate provenance and inventories.** Needs: R03, B06.
  Extend current staging/build/installed readers for host/package/service hashes,
  native architecture and independent package replacement. Done when the existing
  development producer records actual artifacts without a second release driver.
- [ ] **B08 — Remove obsolete staging and public adapter routing.** Needs: B10, B07, C07.
  Remove only obsolete template/bootstrap/assets/proxy/configuration entries using
  C07's replacement map. Done when packaged content includes native mounts and
  retains branding, notification/action/switcher features and credential custody.
  This is the packaging half of R04's coordinated cutover. Scope: P4/P5/P7.
- [ ] **B09 — Finish source/notices and maintenance packaging.** Needs: B02, B06.
  Verify source and license inclusion for fork/SDK/package and the reproducible
  upstream update procedure. Done when the distribution assumptions are accurate
  and R06 can bind the final artifacts; generator success alone is insufficient.
- [x] **B10 — Integrate native account restoration into the browser host.** Needs: B03, A02.
  Connect the real generation binding to mount/cache restoration and account-change
  invalidation. Done when stale private UI/access cannot resume under another
  session; use actual native session checks, not a stub. C05 waits for this gate
  before claiming the real persistent workspace. Scope: P2/P5. The generation is
  derived from the live Forgejo session and actor; browser cache/focus invalidation,
  account-change behavior and stale-request rejection have focused tests. Persistent
  terminal behavior remains C05.

### C — Soda service, API migration and Lit workspace

- [x] **C01 — Produce the exact caller and retirement map.** Needs: R00.
  Map current UI/API operations to native/Soda authority, session continuity,
  mounts/assets and retained nonbrowser credentials. Select the smallest real read
  and mutation for C03. Done when R01 can define only the needed interfaces and
  later deletion has named replacements. Scope: P0/P4/P5.
- [x] **C02 — Build the thin Soda consumer and entrypoint skeleton.** Needs: B02.
  Build the separate extension against the SDK; prepare existing Lit mount adapters
  and Soda's private HTTP/callback connection within existing Go owners. Done when
  the package compiles and can call A02 once available; no stub authority, alternate
  browser credential or speculative UI rewrite is accepted as working integration.
  The starter read/preferences proxy checks native actor and contribution, forwards
  no browser credentials, and bounds requests and responses. The live same-UID bridge
  now exercises page and panel session/preferences calls through native admission;
  final C03 acceptance still depends on R02's Linux isolation proof.
- [ ] **C03 — Prove one native read and mutation across isolated services.** Needs: A02, B01, C02.
  Exercise the real callbacks and IPC with the P1c denial/isolation cases. B lends
  the existing fixture and applies any owned unit edits; C owns the product adapter.
  Done when R02 has actual evidence, including unsafe Origin and forged context.
  The same-UID development test now covers page and panel reads plus preference
  mutation/readback; Linux process, group and SELinux separation are still unproved.
- [ ] **C04 — Port terminal control and its backend stream endpoint.** Needs: R02, A03.
  Port reserve/create/attach/end and their membership/repository/generation guards
  first. Implement the Soda side of P3 alongside A04. Done when focused terminal
  and session-continuity checks pass and R03 can exercise a real retained shell.
- [ ] **C05 — Mount and prove the persistent real terminal.** Needs: R03, B10.
  Reuse Lit/xterm in the native persistent root. Done when the same element,
  renderer, socket and target survive native navigation, with account-change,
  history/cache, form, focus and mobile behavior covered. Opening must not create,
  join or start a project. Scope: P5.
- [ ] **C06 — Finish the remaining native product APIs and pages.** Needs: R02, A08, C02.
  Port Spaces/project creation/join/settings, identity/broker continuity, access
  controls, Runners and Tailnet with their existing policies and mounts. Done when
  C01 has a real replacement for every browser caller, including non-admin operator
  access. This work can run before A04 finishes if C04 is waiting for transport;
  prioritize C05 once its predecessors pass. Scope: P4/P5.
- [ ] **C07 — Delete the replaced product auth and shell implementation.** Needs: C05, C06.
  Remove old browser OAuth/session/return, expected-user workaround, shell/frame,
  `soda-view` and injection callers with their tests; update current API/credential/
  terminal guides. Done when product source has one current path and B receives
  the exact obsolete staging/config entries. Preserve shared credential custody
  and independent native presentation; R04 waits for B08 as well. Scope: P4/P5.

### Q — Existing production-qualification dependency

This tail is required for the production-qualified appliance, not for proving the
working development integration. R tracks its known missing prerequisites early;
do not divert the critical native integration lanes into a new release framework.
Q tasks reuse lanes after their preceding implementation work is complete.

- [ ] **Q01 — Establish protected qualification observations (lane B).** Needs: R05.
  Reuse valid native install/recovery evidence and prove missing boundaries before
  controller orchestration. Select upgrade sources only when actually qualified.
  Done when existing acceptance/delivery owners can consume concrete observations
  bound to candidate/media identities. Scope: P8's release prerequisite.
- [ ] **Q02 — Wire native admission and finalization safely (lane B).** Needs: Q01.
  Implement the missing protected admission/controller path in existing owners.
  Done when failed, cancelled, incomplete and mismatched evidence cannot qualify
  or publish, fixture signing is distinguished, and the development-only guard
  can be replaced by actual admission rather than simply deleted. No public
  publication or production activation is part of this task.
- [ ] **Q03 — Qualify native Linux x86_64 (lane A).** Needs: R06.
  Produce and qualify a fresh committed candidate with native evidence for the
  complete P8 matrix and release checks. Done only for the exact qualified bytes;
  retain honest failure scope and do not resume a failed release as qualified.
- [ ] **Q04 — Qualify native Linux aarch64 (lane C).** Needs: R06.
  Run the corresponding native target independently with the same source contract.
  Can run alongside Q03 when distinct native hosts/resources are available. Done
  only with native evidence; cross-compilation and macOS arm64 do not substitute.

For either target, reuse valid native boundary receipts or run the smallest
unproved image/package/IPC check before an expensive candidate/media build. A
passing probe on the other architecture is not a reason to skip that check.

### Critical path and file handoffs

```mermaid
flowchart LR
    R00 --> A01
    R00 --> B01
    R00 --> C01
    C01 --> R01 --> B02
    A01 --> A02
    B02 --> A02
    B02 --> C02
    A02 --> C03
    B01 --> C03
    C02 --> C03
    C03 --> R02
    A02 --> A03
    R01 --> B03
    A02 --> B10
    B03 --> B10
    R02 --> A04
    A03 --> A04
    R02 --> C04
    A03 --> C04
    A04 --> R03
    C04 --> R03
    R03 --> C05
    B10 --> C05
    C05 --> CUT[Complete pages, policies and packaging cutover]
    CUT --> R05 --> Q01 --> Q02 --> R06
    R06 --> Q03
    R06 --> Q04
    Q03 --> R07
    Q04 --> R07
```

The diagram highlights the terminal path and production tail; the task `Needs`
lists are the complete dependency source, including policy/packaging joins. In
particular, C06 does **not** wait for WebSockets, A05 does **not** wait for all SDK
capabilities, B01 does **not** wait for the session probe, and B03 does **not** wait
for live admission or finished Soda pages. The terminal path waits for A03's
permission/recheck slice, not A08's other reads. B02 may proceed while a native
probe waits only if that probe uses frozen inputs; never change source underneath
a running build.

| Shared surface | Single writer and handoff |
| --- | --- |
| Public SDK, manifests, host/example dependency files | B implements R01's contract. A/C request required changes through B; no duplicate types or simultaneous module/lock edits. |
| Forgejo extension manager and callback connection lifecycle | B owns manager/bootstrap edits. A owns new authority/capability handlers. Agree the call boundary in R01; B lands thin wiring changes needed by A. |
| Forgejo extension HTTP/WS and native guards | A owns Go handlers/proxy/auth changes. B's browser host uses assigned JS/template files; R sequences shared route/navigation registration edits. |
| Username hooks and required-package runtime | A owns native operation seams and the assigned policy handler; B owns availability/CLI loading. A06 waits for B05 so enforcement cannot be bypassed. |
| Soda auth/API, adapter and browser assets | C owns these files through deletion. A's policy handler uses separately assigned files. B consumes built package output rather than editing C's source. |
| Containerfile, appliance units, proxy, staging and release inventories | B owns all changes. C supplies obsolete-entry/replacement maps; R groups C07/B08 into the final cutover. |
| Canonical architecture/product documentation and Git index | R owns shared document changes and stages/commits exact reviewed paths. Workers supply needed edits/receipts; no concurrent staging, broad commits, resets or worktrees. |
| Native fixture, caches and expensive execution | B owns setup/lifecycle under `/home`; other lanes reserve use before a check or restart. Do not restart/reconfigure it during another lane's stream/browser checks. Freeze build inputs and reuse unchanged artifacts. Q03/Q04 use distinct native target resources. |

A completed task handoff is its reviewed code, exact source/artifact identity,
focused check result, and the next consumer it unblocks. R integrates those
handoffs promptly; gates require evidence, not a new user approval or a repeated
full test run. Failures block only affected dependents. Do not start broad
packaging/qualification orchestration to fill time while a native assumption fails.

## P0 — Establish the target contracts and exact caller map

**Deliverables**

- Update the intended architecture in [Overview](../architecture/overview.md) and
  [Trust](../architecture/trust.md): replace the current explicit no-fork decision,
  native browser authority, trusted extensions, isolated Soda service, and retained
  host/identity boundaries. Update the intended [Spaces](../product/spaces.md)
  behavior for the core-owned workspace host. Reference docs change when the
  corresponding interface actually lands, not ahead of working code.
- Map each current browser request to its actor, repository, operation and Soda
  policy requirements. Start with `internal/web/api/environment_authority.go`,
  `repositories.go`, `environments_create.go`, and `internal/web/auth/forgejo_keys.go`.
  Include identity/project controls and operator operations, not only Spaces reads.
- Separate browser OAuth/session code from Forgejo credential custody, Git helper
  access, factory grants, service credentials and account provisioning. Name the
  exact obsolete callers before removing anything from the shared auth area.
- Define the initial public SDK fields and contribution authorization callback.
  JSON identifiers consumed by JavaScript use decimal strings consistently with
  Soda's existing rule; replace the unreleased numeric actor-ID shape and all its
  callers together. Do not add compatibility readers or protocol negotiation.
- Choose one source owner for each contract using [Go ownership](go.md). Keep
  `web/auth` responsible for web authority, `web/api` for product HTTP/WS, existing
  domain packages for domain behavior, and `web` for composition. Update the
  ownership guide and architecture test together only if new ownership is needed.

**Acceptance:** the caller map accounts for every route and UI mount being moved;
the SDK and product authorization rules have one owner; the intended architecture
no longer contradicts the approved extension direction. No full build is needed.

## P1 — Prove the native boundaries before orchestration

Run these small probes in parallel where independent. Store development receipts
under ignored `.artifacts/`. On a development Linux host, all task workspace,
build, cache, state and temporary paths belong under `/home`.
P1 uses the smallest vertical slice of the P0 contract; it does not wait for a
finished SDK or every callback. Promote successful probe code into P2–P4 instead
of maintaining a parallel prototype or building a general framework for the probe.

### P1a: Actual session middleware and long-lived response

- Run one echo WebSocket through the real Forgejo web middleware, native login,
  wrapper stack and selected session provider. Use a real HTTP/TLS server and
  client; a response recorder cannot establish that connection hijacking works.
- Inspect the effective session configuration. The inspected Forgejo setting
  defaults to memory; the image template sets a provider configuration path but
  does not establish the selected provider. Both pass through `VirtualSession`.
- Exercise the pinned upstream WebSocket release option and necessary response
  writer forwarding, including Forgejo's context wrapper, deferred session writer
  and compression behavior. Do not assume a setting alone makes upgrades work.
- For both memory and file: pause an authenticated request, complete logout,
  release the paused handler, and verify that neither it nor the stream can
  recreate authorization. Also prove session regeneration, expiry and provider
  failure behavior. Inspect the same native session authority, not a Soda cookie.
- A session recheck must not extend expiry accidentally. File-provider reads can
  touch timestamps; raw file release can recreate a destroyed SID. Account for
  `VirtualSession` behavior explicitly rather than assuming every missing-ID read
  creates a file. Notifications accelerate cancellation but are not durable proof.

**Exit evidence:** successful upgrade through the actual wrappers, denied stale
authority after each invalidation, and no delayed session write restoring access.
If the assumption fails, fix the bounded native seam or revise the design before
building the consumer or production pipeline around it.

### P1b: Actual Linux image and package runtime

- Inspect the pinned image's executable location, entrypoint, user, runtime
  dependencies and `/data` permissions without dumping secret-bearing inspection
  output. Build the fork and example for the native Linux development target.
- Put that binary into a task-owned derivative of the pinned image and install a
  separately built example package. Prove child execution, Unix sockets, data
  writes, package lock, stop/start and package replacement under the actual
  container/user/SELinux configuration.
- Record host and package hashes before/after replacement and retained example
  data. Reuse the resulting development image for driver changes that do not alter
  shipping bytes. Do not start with a full appliance/media build.

**Exit evidence:** the real container runs the fork and independent package, with
native architecture evidence and working persistent data permissions.

### P1c: Isolated service bridge

- Connect the extension and existing Soda service using a dedicated private IPC
  mount and the scoped host callback path. Prove one current-actor read and one
  authorized mutation before migrating more routes. Reuse mature Go HTTP/Unix
  transport; do not add a new privileged daemon or a general credential service.
- The core validates an opaque admission bound to the extension instance and
  operation. Soda receives only the required authoritative identity/context and
  checks its own policy. Reject fabricated handles, copied actor headers, wrong
  extension/operation use, expired handles and calls after the owning process ends.
- Prove actual UID/GID and label permissions in both directions. Forgejo and
  extension processes must still be unable to open Soda's database, secrets and
  host/identity helper sockets; project workloads cannot reach the bridge.
- Verify cancellation, service failure and stream disconnect propagation. No
  request may turn a transport failure into an authenticated fallback.

**Exit evidence:** one native authenticated read/mutation crosses the bridge,
forgery and unauthorized mutations fail, and the existing private resources remain
inaccessible from Forgejo. Socket access by itself is never sufficient authority.

Keep this proof on the native extension route. A separate Caddy authentication
subrequest would create another admission path and only authenticate a WebSocket
handshake. It is not an assumed shortcut around live authority. If native evidence
requires reconsidering that choice, account explicitly for original-method/Origin
checks as well as continuous revocation before changing the architecture.

## P2 — Finish the authoring SDK and live native authority

**Source owners:** Forgejo `modules/extensions`, `services/extensions`,
`routers/web/extensions`; standalone SDK module and independent example.

- Extract only the public protocol types, server entrypoint, request context and
  capability client into a standalone Go module. Keep installation, internal
  models and host lifecycle code out. Host, example and Soda consume one definition
  of the wire contract; no duplicate DTO or forwarding compatibility package.
- Keep a single current protocol for this unreleased work. Unsupported inputs fail
  clearly. Pin the SDK/toolchain and document an independently buildable package,
  manifest, private data directory, asset URLs, startup, cleanup and failure model.
- Prove a clean external consumer without checking out the Forgejo source tree.
  A task-local module proxy can prove the module boundary before external
  publication; the final public module path must use an owner-controlled namespace.
  Do not invent a published repository or make publication a prerequisite for
  local development.
- Add host-owned request/stream admission handles. Bind each to the native session
  generation, actor, extension instance, contribution, permitted action and any
  explicitly admitted stable repository ID. Make validation and registration
  atomic with invalidation so logout cannot race a newly registered live stream.
  Never expose raw native cookies or session IDs to the extension or Soda service.
- Separately expose a host-verified opaque session-generation binding for browser
  mounts and multi-request flows. It is stable within one native authenticated
  session generation, but is not an authentication credential and grants no access
  without current native authentication. Validate it against the current session
  on each step; never silently rebind an old page or in-progress operation after
  account switching. Replace existing cross-request `ContextID` and CSRF bindings
  deliberately rather than confusing them with ephemeral request admissions.
- Implement the first narrow capabilities required by actual Soda callers:
  current actor; repository identity/code permission; visible owned-repository
  search; organization ownership; actor public SSH keys. Use bounded pagination
  and explicit errors. Recheck current native authorization on each call.
- Bind callback channels to the registered extension instance. An extension cannot
  choose an arbitrary actor or turn a read capability into a write capability.
  Enforce declared capabilities in the host, not just in SDK documentation.
- Add a bounded contribution authorization callback where Soda needs narrower
  visibility/access than native scope. Core native guards run first and cannot be
  relaxed by an extension. Apply the same rule to navigation and direct routes.
  A callback failure denies the affected application contribution.

**Acceptance:** an external example builds against the SDK; a real Soda repository
picker and SSH-key read work through native authority without browser OAuth;
forged actor/repository input and lost permission fail; optional display context
cannot grant panel repository authority. No arbitrary core mutation API is added.

## P3 — Carry real terminal WebSockets safely

**Depends on:** P1a/P1c proof and P2 admission/callback contract.

- Extend the private host/extension/Soda HTTP path for WebSocket upgrades using
  upstream transport support. Preserve strict same-origin admission, native
  session requirements, endpoint authorization and subpath-safe supplied URLs.
  Strip cookies, Authorization and user-supplied authority headers; forward only
  the protocol headers needed for the handshake and terminal subprotocol.
- Use the current [Terminal](../reference/terminal.md) contract to set handshake,
  message, connection and buffering bounds. Stream with backpressure rather than
  applying the ordinary buffered 8 MiB/30-second response path. Bound startup and
  idle behavior without putting a short HTTP deadline on a live terminal.
- Keep core authority attached to the whole stream. Close both transport
  directions on logout, session invalidation/expiry, account restriction,
  repository or membership loss, extension death, Soda failure and host shutdown.
  Repository/operation rechecks use current native state plus Soda's membership
  policy, not the repository currently displayed in the frame.
- Start from the existing Soda 15-second authority heartbeat and 5-second check
  timeout where polling is needed; retain immediate invalidation signals for
  faster cancellation. Document and test the resulting bound. Never claim
  instantaneous permission revocation merely because an event exists.
- Cancellation detaches browser execution/attachment authority. It does not erase
  projects, kill retained shells, remove Linux users or discard keys. Preserve the
  distinction between hiding a terminal, disconnecting it, and explicitly ending
  a managed session.

**Acceptance:** one real tmux-backed terminal carries input/output through all
layers; permission loss and native logout close access within the documented
bound; a stale request cannot restore access; extension/backend failure leaves no
orphan proxy stream. The HTTP path retains its existing body and deadline bounds.

SSE remains a separate, explicit platform follow-up: it has no demonstrated Soda
caller in this migration. It would need incremental flushing, cancellation and
session writeback proof; removing the current content-type rejection is not an
implementation of that contract.

## P4 — Move Soda browser APIs onto the native extension boundary

**Source owners:** Soda `internal/web/auth`, `internal/web/api`, existing domain
packages, the extension executable, and narrowly scoped service configuration.

- Build the Soda extension as a separate executable/package with its own frontend
  assets. Its private adapter forwards application requests to the existing Soda
  service and exposes only the bounded native callback operations that service
  needs. Keep database and privileged integration logic in their existing owners.
- Replace browser adapter authentication at the actual API entrypoints with
  core-validated admission. Preserve per-operation Soda authorization: operator
  stable ID, repository permission, environment membership, project access and
  credential-grant authority. A signed-in actor alone is insufficient.
- Port Spaces listing/creation/joining, repository settings, terminal reserve,
  create/attach/end, project access controls, public-key reads, identity controls,
  and operator runner/Tailnet operations according to the P0 caller map.
  Retain unsafe-method origin protection through the full proxy chain.
- Use the existing environment creation flow for an already selected Forgejo
  repository. It creates Soda project state and invokes the host; it does not need
  a new core repository-create operation or automatic provisioning event.
- Preserve broker/account custody, factory grants and Git/publication credentials.
  Browser admission is not a substitute for a factory actor's bounded authority.
  Retain deliberate denial of unsupported account/operation combinations.
- Remove browser OAuth login/return, adapter cookie/session and expected-user
  workaround callers once replaced. Remove their routes, config, provisioning,
  tests and documentation together. Do not remove shared credential custody by
  name matching, and do not keep the old browser auth as a fallback.

Preserve cross-request session continuity for terminal Reserve→Create and broker
enrollment→callback using P2's native generation binding. Retain origin/CSRF
protection for mutations. Logout between steps, another tab signing in as a
different account, or a stale cached page must invalidate the old flow rather
than execute it under the new actor.

**Acceptance:** native login alone supports the complete mapped Soda browser API;
direct requests cannot bypass native or Soda policy; browser credentials never
reach Soda; legacy browser adapter entrypoints are gone; existing nonbrowser
credential and factory authorization checks continue to pass. The two multi-request
flows succeed normally and fail after logout/account switch without silently
adopting the new session or actor.

## P5 — Port the complete Soda UI and persistent workspace

**Source owners:** existing `frontend/spaces`, `frontend/runners`,
`frontend/tailnet`, related native integration modules, Forgejo workspace host,
Soda asset staging and affected presentation templates.

| Existing surface | Native destination and retained behavior |
| --- | --- |
| Spaces page (`sodaspaces-page.ts`) | Global extension page; existing project list/actions and explicit create/join behavior |
| Repository Spaces (`soda-repository-spaces.ts`) | Native repository contribution/settings; exact read/write/admin policy required by each operation |
| Workspace (`sodaspaces-workspace.ts`, `sodaspaces-terminal.ts`) | Persistent extension panel; retained Lit element, xterm renderer, socket and selected terminal target |
| User/project identity and access controls | Appropriate native user/repository contribution; preserve existing membership and grant decisions |
| Runners and Tailnet pages | Operator-authorized contribution in an accessible native scope, with operator policy enforced on navigation and API |

The factory remains its protected operator CLI; there is no factory dashboard to
port. Issues, pull requests, reviews and Actions remain native Forgejo pages.
Identity launch and connection actions already belong to Spaces and need their
API authority ported, not an invented additional dashboard.

- Reuse existing Lit components and terminal logic through small extension
  entrypoints. Do not rewrite the UI or rebuild xterm state merely to change its
  host. Bootstrap authoritative context through the protected extension API;
  browser mount context is not an authorization source.
- Preserve the operator distinction: Soda's configured operator stable ID is not
  equivalent to Forgejo site-admin. A generic admin-only page would incorrectly
  exclude a non-admin operator. Use P2's narrowing callback with a global/user
  contribution as appropriate; never broaden core site-admin access.
- Add a generic preferred workspace entry policy so the enabled Soda integration
  automatically hosts eligible native pages, preserving the product's current
  intended behavior. Retain explicit authentication, installation, external and
  credential-flow exclusions. Keep one outer host and no nested drawers/frames.
  Supply path/subpath information through the host contract. Update the canonical
  Spaces route contract to the chosen native workspace URL; no legacy shell alias
  or second route-encoding implementation is required.
- Keep the terminal element in its stable parent while native pages navigate.
  Frame navigation may update display/navigation context, but cannot change the
  attached repository, project, user or tmux session. Selecting another terminal
  is a separate explicit operation. Do not create/start/join merely by opening UI.
- Finish native link/form navigation, back/forward, dirty-form cancellation,
  redirects, mobile layout and keyboard/focus behavior. On cached-document return,
  revalidate the account before showing stale private state or resuming access.
  Reload can remount the client and reattach an authorized retained shell; it is
  not evidence that the same browser socket survived.
- Delete replaced `sodaspaces-shell.ts`, `sodaspaces-frame.ts`,
  `internal/web/api/shell.go`, `soda-view` dispatch, connection/settings/workspace
  injection and old footer/drawer bootstrap paths with their callers and tests.
  Remove only template overrides superseded by native contributions. Preserve
  canonical branding, attribution, and independent presentation customizations
  until an actual native replacement covers each one.
- Update asset builds, staging inventories and browser fixtures with the same
  change. Audit each retained template by purpose so the cutover does not silently
  drop repository actions, notifications, the repository switcher, appearance
  controls or other existing native presentation. In particular, the existing
  `custom/footer.tmpl` combines some of those features with old drawer bootstrap;
  `custom/header.tmpl` and `custom/extra_links.tmpl` also serve independent styling
  and theme behavior. Do not delete those files wholesale as shell cleanup.

**Acceptance:** the actual Soda workspace works beside representative native
pages; navigation preserves the same terminal DOM element, xterm instance, socket
and underlying shell; changing the frame repository never retargets it. Native
forms, auth exits, history, cached restoration, operator access and responsive
layout pass real browser checks. The former Soda shell is no longer shipped.

## P6 — Add concrete core policies and observation events

**Can run in parallel with P3–P5 after P2.** Source owners are the actual native
operation seams and a small dependency-neutral extension hook contract.

- First implement the concrete username policy needed for Linux project members.
  Inspect `models/user.createUser` and `services/user.renameUser`; creation does
  not funnel through a shared `services/user.CreateUser`. Cover self-signup,
  administrator UI/API, external-auth creation, import paths and participating
  CLI callers. Record the actual caller coverage rather than assuming one router
  hook reaches every writer.
- Run native validation/authorization first and policy before the first
  irreversible effect, including filesystem creation/rename as well as database
  commit. Inspect callers already inside transactions before choosing the exact
  insertion point: do not hold database locks across unbounded extension RPC.
  Use bounded timeouts, deterministic policy ordering, structured allow/deny
  errors and no recursive core mutations or secret-bearing payloads.
- The policy checks only known universally invalid/reserved Linux-member names.
  Preserve existing identities and still check actual project-local account
  collisions when joining. Do not silently rename Linux accounts or treat
  organization/nonhuman identities as interactive Linux users.
- Ensure mandatory policies cannot be bypassed by missing, disabled or crashed
  packages. Handle offline/online CLI explicitly: a shared hook loader or existing
  authenticated control path must enforce the rule, and inability to obtain a
  mandatory decision must fail closed. The current web-only manager and package
  lock do not yet solve this. Order initial bootstrap before mandatory activation;
  do not introduce an undocumented permanent bootstrap exemption.
- Add only selected observation events needed for UI refresh, authority
  invalidation or existing Soda bookkeeping. Define payload, timing and caller
  coverage for each. Existing notifier callbacks are void, have differing timing,
  and do not imply post-commit or complete delivery. Consumers refresh current
  state after missed observations; live authorization never depends on delivery.
- Do not provision/delete environments automatically on repository events, veto
  every repository operation, or introduce a global durable event bus without a
  concrete product requirement. Existing projects and shells are retained when
  attachment authority is revoked.

**Acceptance:** a forbidden new name/rename is rejected consistently through the
real supported entrypoints, including CLI behavior and an unavailable mandatory
policy; ordinary native validation still wins where appropriate. Each introduced
event has an actual consumer, documented delivery semantics and a focused check.
Denial, timeout and unavailable mandatory policy leave neither committed identity
changes nor created/renamed native paths.

## P7 — Finish lifecycle operations and appliance packaging

**Depends on:** P1 Linux/bridge evidence; complete service contract from P4.

- Add concise package startup/exit diagnostics, status/health output and useful
  operator errors without logging admission handles or credentials. On a late
  crash, cancel live admissions and expose the unavailable state. Restart-based
  activation remains sufficient; no hot reload or restart supervisor is required.
- Define required package IDs for the appliance and fail startup/activation when
  they are absent, disabled or invalid. Required policy decisions fail closed
  after a runtime crash. Do not replace the current fail-closed startup with a
  blanket fail-open package scan that silently removes Soda policies.
- Add locked package removal while the runtime is stopped. Default removal keeps
  the package's private data; any explicit purge identifies that package's exact
  data directory and respects required-package checks. Preserve replacement
  staging, file limits, symlink rejection and data retention. No broad cleanup.
- Update `appliance/forgejo.Containerfile` to ship the actual patched binary.
  The current file only overlays presentation assets on a stock image. Pin source,
  upstream base, toolchain and image identity; verify entrypoint and native runtime
  dependencies established by P1b rather than guessing a binary path.
- Produce the Soda extension package separately from the host image and retained
  Soda service image. Stage its manifest, executable and assets; install/replace
  only while Forgejo is stopped. Specify deterministic first-install/bootstrap,
  required activation, replacement and service restart ordering. A package-only
  update must not rebuild Forgejo or erase existing Soda/extension data.
- Wire the three components and narrow IPC permissions through existing appliance
  units and configuration. Keep Caddy's public routing consistent with native
  extension URLs; remove replaced browser-adapter routes when no actual caller
  remains. Do not change shared networking or trust as an incidental workaround.
- Extend existing `internal/release/build/production.go`, payload/staging inventory
  and installed checks to identify the fork and independent package. Reuse
  `tools/soda-build` and its candidate provenance. Record source revisions,
  architecture and content hashes; do not create another release orchestrator or
  pretend an unsupported production phase has been implemented.
- Include corresponding source, license notices and existing canonical assets
  for the modified Forgejo, SDK and extension dependencies. The present stock-image
  assumptions in licensing notes must be revisited. A license generator with
  ignored errors is not proof of a complete distribution.

**Acceptance:** a small native Linux development image boots the intended host,
package and isolated Soda service; mandatory policy activation is deterministic;
package replacement preserves data and host bytes; installed inventory identifies
the real artifacts; package failure is observable and cannot silently disable a
required policy. Full release qualification follows P8, not this development run.

## P8 — Qualify the integrated result and close the transition

Run the smallest affected checks as each milestone lands. Once the actual terminal,
native policies and container boundary work, run the integrated native development
journey. Production qualification has an additional existing prerequisite:
`tools/soda-build` currently rejects non-development dispatch with
`production builds removed; development only`. The final protected native
qualification/delivery path is not a command ready to run today.

Account for that release dependency explicitly, through the existing
[release workflow](release.md), [release architecture](../architecture/release.md)
and [native support tools](native-support.md):

- The release owner must connect the actual native installation, selected upgrade
  and recovery observations to protected admission of the exact candidate/media
  hashes. Extend the current `internal/acceptance` and `internal/release/deliver`
  owners; do not create a second qualification engine for extensions.
- Wire the protected controller and finalization/custody boundary only after those
  native observations work. Producer success or fixture signing is not admission.
  Failures, cancellation, missing evidence and mismatched bytes must remain
  non-qualified and unable to publish. Do not merely remove the development-only
  guard. The current payload accepts no qualified upgrade paths; advertise one
  only after actual native evidence supports it.
- Qualify a fresh committed candidate after these prerequisites, preserving the
  separation between qualification and separately authorized signing/publication.
  Completing the extension development milestones can precede this wider release
  work, but it must not be called a production-qualified Fountain appliance.

Retained failed-run artifacts may support clearly labelled development checks; a
failed release is never resumed or relabelled as qualified. Driver-only fixes do
not automatically require rebuilding binaries.

| Contract | Required evidence |
| --- | --- |
| Native auth | Native login succeeds; anonymous/token/Basic/proxy credentials and spoofed context fail; unsafe cross-origin calls fail |
| Cross-request continuity | Terminal Reserve→Create and broker enrollment→callback succeed only for their original native session generation; logout/account switch and stale tabs cannot adopt a new actor |
| Live authority | Memory/file logout, regeneration, expiry and delayed-release race fail closed; account/repository/membership changes revoke access within the stated bound |
| Soda service isolation | Native read/mutation succeeds over narrow IPC; Forgejo cannot read Soda DB/secrets/helper sockets; project workloads cannot reach the bridge |
| Native contributions | Correct page/nav visibility, direct-route denial, repository permissions, non-admin operator access and non-operator denial |
| Real workspace | Same element/xterm/socket/shell across native navigation; explicit terminal target, history/form behavior, cached-account revalidation and subpath/mobile checks |
| Terminal semantics | Reserve/create/attach remain separate; opening does not create/join/start; disconnect preserves shell; explicit End has its documented effect |
| Core hooks | Actual signup/admin/API/external-auth/import/CLI paths covered; timeout/crash/missing mandatory package cannot bypass policy or leave identity/filesystem side effects |
| SDK and lifecycle | External build without host checkout; declared capability denial; install/replace/remove/required-package behavior; retained data and unchanged host bytes on package-only update |
| Cutover | Old browser OAuth/shell/dispatch assets and staging entries removed; retained broker/factory credentials still pass their authorization checks |
| Distribution | Patched host, SDK, package, service, architecture and notices/source identified by candidate inventory; native installed journey matches those bytes |

For Soda changes, select the affected Go packages plus `go test ./internal/archcheck/`
when boundaries change, applicable TypeScript/lint checks, frontend/Forgejo suites
and existing installed journeys. For Forgejo, run affected extension/session/core
operation Go tests and frontend checks. Add real-browser/native fixtures only for
the boundaries that unit tests cannot establish. Port existing drivers before
creating another harness; avoid source-string tests that merely mirror new code.

Qualify Linux x86_64 and aarch64 independently on native targets when each is
advertised. Cross-compilation and the completed macOS arm64 development evidence
do not qualify either Linux target. Do not block useful development on the sibling
architecture, but do not declare the two-architecture delivery complete from one.

Update the owning reference guides as their interfaces land:

| Owner | Final change to absorb |
| --- | --- |
| [Forgejo](../reference/forgejo.md) | Native extension integration, remaining presentation overrides and upstream maintenance boundary |
| [API](../reference/api.md) | Actual native extension routes, authority requirements and removed browser adapter routes |
| [Credentials](../reference/credentials.md) | Removed browser OAuth session versus retained broker/factory/account custody |
| [Terminal](../reference/terminal.md) | Native stream admission, disconnect/revocation semantics and supported session providers |
| [Configuration](../reference/configuration.md) | Required package, bridge and service settings; removed adapter settings |
| [Operator setup](../guides/operator-setup.md) and [Installation](../guides/installation.md) | Bootstrap, activation, package replacement and real artifact/service ordering |
| [Release architecture](../architecture/release.md) and [Release workflow](release.md) | Fork/package provenance and existing pipeline integration |
| [Testing](testing.md), [Local testing](../guides/local-testing.md), [Native support](native-support.md) | Current fixture and qualification entrypoints |
| Forgejo `contrib/extensions/README.md` and standalone SDK documentation | Authoring, capability, policy, event and lifecycle contracts that actually exist |

Retain concise receipts in commit history and ignored artifacts, not new permanent
status documents. Remove this plan after its requirements and remaining explicit
limitations have been absorbed into their owners.

## Extension platform follow-ups with separate entry criteria

These are accounted for, but do not block the current Soda integration without an
actual caller or explicit added requirement.

| Follow-up | Implementation trigger and bounded path |
| --- | --- |
| SSE | A caller requires event streaming: add explicit stream declaration, incremental flush/cancellation and actual session-release proof |
| Additional session providers | A deployment needs one: inspect its existing read/release/destroy semantics, prove the same race/revocation contract, then enable streams for it |
| Native core mutations | A real extension needs a specific write: wrap the existing native service with explicit actor/action authorization, bounded DTO and normal audit behavior; no arbitrary internal API |
| Additional policy hooks | A product needs a concrete veto: identify the common seam and all callers, timing and failure behavior before exposing it |
| Durable events | A concrete committed operation must reach a consumer despite restart: implement the smallest transactionally coupled delivery mechanism for that obligation, not a universal event subsystem |
| More presentation extension points | An existing retained template customization needs native replacement: add the smallest reusable slot/contribution and remove that override with its callers |

No marketplace, remote package downloader, untrusted-code sandbox, arbitrary Go
interception, hot reload, general compatibility layer or whole-Forgejo UI rewrite
is part of this delivery. Those are separate products or requirements, not hidden
prerequisites for an independently extensible host.

## Maintenance, execution boundaries and definition of done

The fork needs an explicit upstream maintenance owner and reproducible update
procedure: keep the extension patch attributable against the pinned LTS, assess
actual upstream security/maintenance releases, refresh source/image pins together,
and repeat the affected session, capability, native-route and package contracts.
Do not mix an incidental upstream major upgrade into this transition. Record
unsupported providers/features in the current authoring guide rather than silently
falling back to another authentication scheme.

Development implementation, bounded provider calls, focused tests and task-owned
fixtures follow the existing task authorization. External publication, production
appliance activation, changes to unrelated/shared trust or networking, and material
new cost remain distinct scope boundaries unless explicitly authorized. Prepare a
concrete reviewed artifact before requesting any such final action. Public module
namespace selection can remain pending without preventing local implementation.

The delivery is finished only when all of the following are true:

- An independently built Soda package supplies the mapped native pages and
  persistent workspace without rebuilding the Forgejo host.
- Native login is sufficient; operation permissions are current, enforced at the
  core and Soda boundaries, and no browser adapter credential remains.
- A real persistent terminal survives native navigation and loses attachment
  authority correctly, while retained shells and projects survive disconnects.
- The isolated Soda service retains exclusive access to its database, secrets and
  privileged helpers; the new bridge has actual native container evidence.
- Required policies cover their real native and CLI callers and fail closed;
  introduced observations have honest, tested delivery semantics.
- The standalone SDK, diagnostics, package lifecycle and separately staged package
  work outside the Forgejo source tree and in the native appliance image.
- Replaced code, configuration, templates, assets and tests are removed together;
  surviving product behavior and credential custody remain covered.
- The existing pipeline identifies and qualifies the actual native artifacts for
  each advertised architecture, with source/notices and upstream maintenance
  accounted for. Development success is not presented as release qualification.
- Owning guides describe the final implementation, explicit optional work is
  separated, and this completed transition plan is absorbed and removed.
