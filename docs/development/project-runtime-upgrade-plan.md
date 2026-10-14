# Project runtime upgrade implementation plan

Repair terminal support in retained Projects and make release activation preserve
compatibility between the host and those Projects. A successful upgrade must leave
existing Projects usable, with their accounts, files and container identities intact.
Users receive one release and one operator workflow; they do not assemble helpers.

This plan remains open for coordinated activation and release qualification; it is
not a supported upgrade contract. The implemented repair and readiness contracts
are owned by [Project OS](../reference/project-os.md#terminal-support),
[Terminal](../reference/terminal.md), [API](../reference/api.md) and
[Release](../architecture/release.md#host-update-ownership).

Implementation boundary:

- U01 demonstrated native locked staging, interrupted restart, explicit activation
  and rollback on a disposable FCOS fixture with Zincati active. Locked staging
  disappeared after interrupted restart, requiring actual native-state inspection.
  This proves the explicit primitive, not admission integration with ordinary
  native activation, rollback or Zincati. Supported upgrades remain refused.
- U02 binds one compiled helper to all three uses and verifies assembled/archive
  bytes in the build and delivery paths. Focused mismatched-copy regressions pass;
  a fresh complete candidate has not been built or release-qualified here.
- U03/U06 implement bounded readiness and typed errors through Rust, Go and the
  browser, with focused path and authentication/no-replay regressions.
- U04 implements explicit running-root repair, shared admission, atomic replacement
  and interrupted-record reconciliation. The admitted transition requires empty
  managed state on Rocky 10.2. Stopped or nonempty/uncertain roots refuse explicitly.
- U05 remains unimplemented pending proven activation/admission integration and a
  transition that admits all enrolled roots. No post-boot guard substitutes for it.
- U07 demonstrated old-binary repair and interruption recovery with native fixtures,
  and repaired all three protected development-VM roots with preservation checks.
  Native login, one terminal creation, a harmless command, same-shell reconnect
  and End of only that test-owned terminal passed. This is not upgrade qualification.
- U08 integrates artifact metadata, consumers, source CI coverage and owning
  documentation. Upgrade declarations, the complete release-pair journey and public
  upgrade claims remain withheld. No publication, push or merge is part of this work.

## Problem and current ownership

The observed VM has a newer host terminal helper and an older helper inside a
retained Project. The native reservation path requires their SHA-256 hashes to be
identical. The Project rejects the mismatch before creating a reservation or shell.
The browser then replaces the service failure with generic sign-in advice even
though the native session and repository permissions are valid.

The source baseline for this plan is Soda `455a2f16`. Current ownership is:

| Responsibility | Existing owner and behavior |
| --- | --- |
| Host helper production | [build_compile.rs](../../lib/soda-release-image/src/build_compile.rs) compiles the host's `project-terminal`. |
| Project helper production | [production_assets.rs](../../lib/soda-release-build/src/production_assets.rs) stages the Project helper using the same [compiler recipe](../../lib/soda-release-build/src/production_compile.rs). Separate calls exist; same-release build divergence has not been demonstrated. |
| New Project image | [daemon/config.rs](../../lib/host/src/daemon/config.rs) selects the installed release's exact `project-os` image. Creation uses local pinned content. |
| Existing Projects | [image import](../../cmd/soda-image-import/src/import.rs) imports images without replacing workloads. The [Project unit](../../system/host/services/soda-project@.service) starts the original container. Neither updates its helper. |
| Compatibility refusal | [native.rs](../../lib/host/src/terminal/native.rs) supplies the host helper hash; [term_create.rs](../../cmd/soda-project-terminal/src/term_create.rs) verifies the Project helper before reservation. |
| Error propagation | The Project agent, [Go terminal client](../../internal/host/terminal_client.go), [terminal API](../../internal/web/api/extension_terminal.go) and [browser attachment](../../frontend/spaces/sodaspaces-terminal-attachment.ts) successively lose the specific failure. |
| Explicit maintenance | [soda-project-maintain](../../cmd/soda-project-maintain/README.md) owns the moved Muse operation and explicit terminal-support repair in the same retained root. |
| Host deployment | [Release architecture](../architecture/release.md#host-update-ownership) retains native FCOS ownership. Current payload readers reject qualified upgrade paths; `soda-activate` is initial browser/Git setup, not a host upgrader. |

The missing contract is coordinated maintenance of persistent Project support
during release activation. Importing a new Project image cannot satisfy it.

## Proposed decisions

1. Retain exact helper matching for this correction. A protocol negotiation layer,
   permissive fallback or multiple permanent helper versions would expand the
   problem without proving old state is safe for new code.
2. Compile the shared helper once and stage the same bytes for the host, new Project
   image and explicit maintenance input. Record its identity with the existing
   release payload; avoid an independent version database.
3. Update the same retained Project root. Keep the original container identity,
   creation profile, distribution, accounts, SSH keys, checkouts, volumes, installed
   user tools and unrelated workloads. Do not rewrite creation labels to pretend
   the Project was recreated from the new image.
4. Use an explicit operator maintenance workflow. Terminal Open, refresh, navigation,
   Project Start and image import do not install support. Normal users see a useful
   state; they are not asked to download or install anything.
5. Coordinate changes during a bounded maintenance window. Stop new managed work
   admission, wait for affected terminals and factory/identity operations to finish,
   then switch the helpers and host. Never silently end someone else's work.
6. Include running and stopped Projects in admission. Unknown, stopped-but-unreadable
   or incompatible Projects cannot be skipped and reported as a successful upgrade.
7. Keep GitHub Releases as the delivery source and native FCOS as the host deployment
   owner. Add local admission and maintenance integration, not a download service,
   registry requirement, background fleet updater or replacement OS updater.

The existing [same-root maintenance contract](../reference/project-os.md#same-root-maintenance)
supports explicit tool maintenance. General container replacement and fleet updating
remain excluded. The implementation must update the owning contracts when the
specific coordinated operation is ready to become supported behavior.

## Target operator and user experience

An operator selects a release. A read-only preflight reports which Projects match,
which need support maintenance, which have active managed work and which cannot be
inspected. It names the actual blocker and the required next action.

Once affected work is idle, the operator initiates one maintenance operation. The
operation stages its supplied content, preserves the previous helper, coordinates
the native host activation, verifies every affected Project and resumes admission.
It reports an interrupted or blocked operation explicitly. Reopening the operator
view reads that operation's state; it does not blindly replay writes.

For an ordinary Project member, a mismatch reads: **“Terminal support needs an
update. An appliance administrator must update this Project.”** Only a genuinely
expired native login asks the person to sign in. Project runtime state and terminal
support readiness remain separate: a running Project can need terminal maintenance.

## Ordered implementation tasks

### U01 Define the maintenance boundary and prove native activation control

Owner: release and host integration. Dependencies: none.

Define the exact source and target release pair, support files, runtime-state
assumptions, admission lock and recovery rules. For this incident the known differing
file is `project-terminal`; inspect its bundled terminal, key and identity-broker
callers before deciding whether a transition requires companion files or units.
Do not turn it into a general Project package update.

Use a small disposable native experiment to establish where the current FCOS
deployment can be staged, checked, activated and rolled back. Identify the existing
caller or unit that can enforce admission before the active host changes. A service
that only fails after reboot is not a pre-activation safeguard. Do not attach this
to initial `soda-activate`, disable Zincati as a shortcut, or claim to gate ordinary
upstream updates without proving that integration.

Completion: one concrete transition sequence and a demonstrated activation boundary.
If native integration cannot yet enforce it, retain the unsupported-upgrade refusal;
U02–U04 and U06 can still deliver explicit repair and accurate errors. Do not write
extensive upgrade orchestration before this fact is established.

### U02 Produce and bind one support artifact

Owner: Rust release build/image code. Dependencies: U01's file scope.

Remove the duplicate helper compilation between
`lib/soda-release-image/src/build_compile.rs` and
`lib/soda-release-build/src/production_assets.rs`. Reuse the existing staged-copy
pattern in `payload_stage.rs`; do not refactor unrelated tool compilation.

Bind the helper's digest, destination and applicable Project image to the existing
candidate metadata. Change the canonical release wire owner and its actual consumers
together if a field is necessary. Verify the bytes in the assembled host and Project
image, not just two input filenames. Include the maintenance input in the same
release payload, with no separate end-user download.

Completion: one candidate has identical helper bytes in all three uses, and its
selected new-Project image is the one checked. One focused staging regression
detects an intentionally mismatched copy.

### U03 Expose read-only support readiness and precise failure codes

Owner: Rust host, retained Go API and browser DTO owners. Dependencies: U01.

Add a bounded host observation of the actual helper in an exact Project container.
Reuse native container inspection and file-admission rules. A legacy helper need
not implement a new command to be inspected; image labels and self-reported version
strings are insufficient. Include stopped roots through a proven read-only native
path without starting them. If that path is unavailable, return unknown explicitly.

Report a small current contract: ready, support mismatch, inspection unavailable,
or maintenance in progress. Expected/observed digests belong in operator diagnostics;
members need the state and useful action. Separate this from native authentication,
repository permission, membership and whether the container is running.

Recheck compatibility at terminal reservation so a stale inventory cannot admit
work. Keep the existing guest verification. Handle observed mismatch before invoking
an old helper that only reports `launch_failed`. Preserve bounded typed reasons
through `lib/host/src/terminal/`, `internal/host/terminal*.go`,
`internal/web/api/extension_terminal*.go` and Spaces inventory. Never return raw
stderr, request headers, credentials or arbitrary exception text to the browser.

Completion: a mismatched legacy Project produces a specific support error through
the real admission path, with no reservation or shell creation. Permission failures
and unavailable observations remain distinct.

### U04 Implement explicit maintenance of retained Project support

Owner: Rust operator maintenance. Dependencies: U01, U02, U03.

Use the concrete file admission, Project identity checks and staged replacement
patterns already in `cmd/soda-muse-maintain`. The proposed shared owner is
`cmd/soda-project-maintain`, with explicit Muse and terminal-support operations.
Move existing Muse behavior and all real build/unit/documentation callers together;
remove the superseded command rather than add a forwarding compatibility wrapper.
Confirm this placement in the package ownership plan before the move. Keep terminal
repair separate from Muse's package installation and launch-interface effects.

The terminal operation accepts an exact Project and admitted release content. It
rechecks the container identity, support bytes and active managed work while holding
the same admission exclusion used by terminal creation and factory launches. No
check-then-write race with new work is acceptable. Browser disconnection alone does
not mean a terminal or identity-bound process has finished.

Before replacement, establish that the target helper runs on the retained Project's
original OS and required libraries/tmux, and can read its existing managed state.
Equal hashes alone do not establish those properties. Refuse unsupported transitions;
do not silently upgrade the distribution or invent generic state migrations.

Preserve the old file and ownership/mode, stage the verified target on the same
filesystem, replace it atomically and read back the result. A matching target is
a no-op. Reinspect actual state after interruption before resuming. Multi-file
transitions keep admission closed until every declared file is confirmed; individual
renames must not be described as an atomic bundle transaction.

Handle stopped Projects without starting user workloads. Prove the native mechanism
for safe writes to their retained root, or block that operation with a concrete
reason. Do not bypass preservation by recreating the container or changing images.

Completion: explicit maintenance aligns support while retaining the exact Project
root and protected state. Active or uncertain managed work blocks maintenance;
the tool never performs implicit End, takeover, reservation replay or Project restart.

### U05 Coordinate release activation and recovery

Owner: native host lifecycle integration, using U04. Dependencies: U01–U04.

Inventory all enrolled Projects and acquire the admission exclusion before final
preflight. Account for Project creation/deletion and state changes between initial
preview and activation. Keep the immutable target release and exact affected
container identities fixed for the operation.

The normal sequence is:

1. Stage and verify the complete candidate, maintenance inputs and previous support
   needed for rollback. Read-only preflight can run before the maintenance window.
2. Enter the window, block new affected work, and confirm existing affected work is
   idle. If work is active, leave it running and report the blocker.
3. Apply the admitted support transition and native host deployment under the proven
   U01 sequence. During any mismatch interval, affected managed operations remain
   unavailable; ordinary users cannot encounter an apparently usable terminal action.
4. Read back the active host identity, actual Project helper identities and selected
   image for future Projects. Reopen admission only when the required set matches.
5. On failure, either restore the previous admitted pair or leave admission closed
   with precise recoverable state. Never claim success merely because services start.

Keep a minimal durable operation record: source/target release, affected container
IDs, old/new digests, saved old file locations and completed mutation boundaries.
Tie it to native deployment state; do not create a second release-discovery ledger.
On reboot, reconcile files and deployment state before allowing managed work.

Host rollback must include Project support rollback. Restoring only the OS recreates
the mismatch. Verify that the previous helper can read retained terminal/broker state;
after new work has produced incompatible state, downgrade cannot be automatic. Keep
admission closed and explain the required recovery instead of deleting state or
restoring an old database over later writes. Retain recovery bytes only as long as
the admitted transition requires them.

Completion: mismatch/unknown observation prevents activation, concurrent new work
cannot slip through, and an interrupted transition never reopens incompatible work.

### U06 Make the terminal UI explain the actual outcome

Owner: TypeScript Spaces and retained Go API. Dependencies: U03's error contract.

Use `SodaRequestError.status` and its bounded code instead of the blanket catch
message in `sodaspaces-terminal-attachment.ts`. Preserve the existing locator,
terminal owner and screen on failure; do not retry creation as error recovery.

| Confirmed condition | User action |
| --- | --- |
| Native login expired | Sign in through Fountain. |
| Native session generation changed | Refresh the current native page. |
| Membership or repository permission missing | Explain the missing access; do not recommend signing in again. |
| Project support mismatch | Explain that administrator maintenance is required. |
| Maintenance in progress | Show that the Project's managed access is temporarily unavailable. |
| Helper/transport unavailable | Offer retry or operator inspection without claiming the login expired. |
| Unknown result after a mutation | Preserve uncertainty and the original locator; inspect rather than replay. |

Show support readiness in the existing Project settings and terminal state, without
adding a new dashboard or nested navigation. The normal terminal view stays compact.

Completion: the incident's 503 no longer produces authentication advice; real session
expiry still uses Fountain's normal login flow. There is no alternate login mechanism.

### U07 Qualify the native transition and repair the current VM

Owner: integration/operator. Dependencies: U02–U06 for complete upgrade evidence;
the already-upgraded VM repair can use U02–U04 and U06 after their native proof.

Use one disposable native fixture carrying the actual older Project helper and
retained state, not a fabricated image that already has the new helper. Prove the
transition before applying maintenance to the protected VM. Use supported terminal
actions for the test's own session; never end a user's session to make a test pass.

For the current VM, select the installed host as the repair target, verify its
release provenance and maintenance input, identify all affected Projects and run
the same explicit maintenance operation. An unqualified development repair is
labelled as such; it cannot retroactively qualify v0.1.1 upgrades.

Verify one real terminal creation, a harmless command and reconnect, using the
existing trusted native-browser driver. Compare protected file/account/container
facts before and after. End only the terminal created by this check. Preserve all
other processes and the owner's working data.

Completion: the VM's actual terminal works with its native Fountain session and
retained Project, without a login workaround or manual component assembly.

### U08 Integrate the supported transition into delivery and documentation

Owner: release delivery and owning guides. Dependencies: U05–U07.

Only admit a source-to-target upgrade pair after the native transition and recovery
evidence exists. Update the coordinated release-payload readers/producers which
currently reject nonempty upgrade declarations; do not merely remove their refusal.
Fresh-install qualification and upgrade qualification remain separate claims.

Keep hosted source CI on GitHub Free, explicit release actions and the existing local
native builder. Add the focused contract checks to applicable source CI. Run the
native upgrade journey once for the release pair on the local builder. Stage and
promote through the existing GitHub Releases process; no new server or automatic
release on every push to main.

Update the owning release, Project OS, terminal, API and operator guides with the
implemented procedure, effects, recovery and supported transitions. Coordinate the
maintenance owner with the existing Canlang Project-tooling plan without changing
that plan's product scope. Absorb this completed plan into those owners and remove
the temporary plan when implementation is complete.

Completion: published upgrade claims identify the qualified transition, operators
have one concrete workflow, and release notes distinguish upgrades from first install.

## Parallel work and dependencies

| Lane | Tasks and integration point |
| --- | --- |
| Release artifacts | U02 after U01, then delivery work in U08 after native qualification. |
| Host and maintenance | U03, U04, U05 in that order; owns the operation boundary and package move. |
| Error presentation | U06 once U03's DTO/code contract is fixed; can proceed alongside U04/U05. |
| Native integration | Small U01 feasibility experiment first, then U07 against the integrated bytes. |

One implementer owns shared release metadata and one owns shared terminal DTO/wire
changes. Other lanes consume those definitions rather than creating parallel ones.
Use the canonical checkout and preserve concurrent documentation work.

## Minimum sufficient verification

| Check | Evidence required |
| --- | --- |
| Artifact assembly | One focused check detects mismatched host/Project helper bytes and proves the common staged artifact feeds both. |
| Admission and maintenance | Focused existing-owner tests cover matching, mismatch and unreadable support; active-work refusal; changed container identity; interrupted replacement/recovery; stopped-Project behavior. Test the actual effect boundary, not every helper function. |
| Error propagation | One end-to-end component path carries support mismatch to the UI; keep separate existing coverage for genuine authentication failure and no creation replay. |
| Native retained Project | One old-to-new transition using actual native binaries, an existing Project with sentinel data, and a stopped Project. Exercise one injected interruption at the helper/host boundary and verified recovery, then complete the transition. |
| Installed use | One normal native login, terminal creation, harmless command and reconnect on the repaired target. Establish preservation and future-Project image/helper alignment in the same journey where practical. |

Run targeted format/type/lint checks for changed code. Do not run full production
builds to investigate unresolved native assumptions. Reuse a valid built candidate
and existing browser/VM drivers. Repeat a check only after its failure or a relevant
change; stop after the stated evidence exists. Existing first-install release checks
remain the baseline and are not recursively repeated as proof of this new contract.

The work is complete when existing and new Projects have matching admitted support,
activation and rollback preserve that condition, protected Project state survives,
and terminal failures tell the user the actual problem.
