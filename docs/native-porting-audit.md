# Native porting remaining-work audit

## Scope and verdict

Audited source baseline: **`58ddc0d0707b9b2c72f97363e8daa1b53853f341`**. The working tree was clean at audit start. This is a source/documentation audit of the [native support plan](native-porting-plan.md), its tools, callers, focused tests and recorded execution history. No builds, tests, dependency resolution, SSH/VM/provider operations, installation or artifact verification were executed for this audit. Historical results below are **recorded evidence**, not newly reproduced observations; private credentials and raw private evidence were not opened.

**Verdict: substantial active implementation exists, but “source complete; only validation remains” is too strong.** There are concrete source defects, missing negative-path tests, incomplete shared-payload/evidence contracts, and substantial native proof still to obtain. P04 has real x86_64 build/seal evidence now; repeating the old claim that every support component is unbuilt/untested is also wrong.

The [dashboard implementation plan](dashboard-implementation-plan.md#coordination-with-native-support-porting) remains authoritative. This audit does not create another product acceptance gate, require optional media, or transfer U08/U20 work back into P07/P08. Source findings below are reasoned from the cited implementation; their regression tests and fixes have **not** been executed or implemented by this audit.

## Source remediation follow-up

Source pass based on `15e49b1`; **unbuilt and unexecuted**, not a replacement for
this audit's historical evidence. Detailed handoff:
[implementation status](implementation-status.md#native-support-audit-remediation--source-only).
The findings/line references below describe the audited baseline, not an assertion
that those exact defects still exist in the edited source.

| Finding | Implemented follow-up | Still to establish |
| --- | --- | --- |
| A01 | Explicit current public `/etc` export paths; actual credential/unknown-path rejection tests | Fresh full staging regression; filenames cannot prove arbitrary unknown secrets absent from otherwise permitted content |
| A02 | Linux non-reaping leader wait, group termination before reap, bounded escalation; resistant-descendant cases | Execute lifecycle/cancellation tests; owned execution on non-Linux now refuses rather than offering weaker cleanup |
| A03 | Separate structured serialization, escaped-secret handling, quote-safe URL matching and numeric-identity tests | Run failure/redaction cases; unknown credentials remain outside exact-secret guarantees |
| A04 | Exclusive pending record followed by final-name publication only after retention checks; failed-launch cleanup facts | Execute finalization and native partial-launch cases; no power-loss durability or job-recovery claim |
| A05 | Directory-relative evidence/report/tree/copy operations, exact decoded-record digest, copy/stream hash checks and destination-parent preflight | Execute changed-parent/special-file/native extraction cases; preserve supported CoreOS writable-prefix mapping |
| A06 | One core `internal/frontend` validator consumed by startup and export; real dashboard dependency inputs and build-identity checks | Rebuild fresh matching payload; rerun core web/staging/aggregate checks; old bundles retain their old verifier |
| A07 | OCI schema/media/local-descriptor/rootfs checks and real tiny tar-layer fixture | Native import/decompression proof for compressed layers and remaining malformed-input coverage |
| A08 | Transfer manifest digest retained; handoff shows invocation/exit/cleanup/artifacts and labels requested identity honestly | Actual native end-to-end transfer and candidate-bound handoff; generic exec is not independently identity-certified |
| A09 | Missing SSH detected before VM state; qemu-img version captured; bounded signature/decompression tools and stricter URLs | Actual trusted signer/keyring/tool/firmware selection and fresh boot/restart evidence |
| A10 | Route/container-network collisions, writable destination ancestry and booted deployment preflight | Behavioral failure fixtures and clean first installation on an explicitly approved new target |
| A11 | Configured credential/TLS modes, native socket versus DNAT boundaries, enforcing browser check, installed-identity fixtures | Current native checks and separately approved operator/provider journeys |
| A12 | Current guide/handoff distinguishes remediation, historical evidence and unexecuted new source | P12/P13 exact-candidate evidence reports after execution |

New coverage also includes a synthetic exact-source remote dispatcher and
trusted-CA download server. Neither is installed/native evidence. Additional
external-tool/VM/installer failure coverage and all relevant native exits remain
open; **the plan is not marked complete**. Newer core evidence already establishes
personal Git/shared files/tools and project-network workload subsets; follow the
[leading core snapshot](dashboard-implementation-plan.md#current-execution-snapshot--u08-completion-run), not section 5's historical pending list.

## 1. Credit what already exists

### Implemented source

- Outside `tools/soda-acceptance` and `tools/soda-artifacts`, with `internal/acceptance` and `internal/nativebuild`; infrastructure commands remain outside appliance `cmd/`.
- Literal/pinned SSH and stdin transport, private evidence, exact-source remote phase dispatcher, QMP/KVM fixture ownership and retained disks/NVRAM.
- Native full-build integration, separate core dashboard payload build, OCI identity inspection, allowlisted export, verified transfer, CoreOS retrieval/signature checks, private Butane conversion and first-install preflight.
- Host/order/PAM/TLS and retained operator observation entrypoints, plus a manually selected provider-job fixture outside automatic CI.
- P12/P13 report interfaces, notices and operational recipes. These are not completed native milestone exits.

### Recorded execution that must not be erased by stale headers

See [implementation status](implementation-status.md#native-first-developer-fixtures-and-complete-core-build), [preview rollout](implementation-status.md#native-react-preview-migration-and-operator-browser-proof), and [routing evidence](implementation-status.md#approved-private-routing-and-direct-developer-access).

| Evidence | Credit and limit |
| --- | --- |
| Full x86_64 build/stage/seal at `8417a90` | Real P04 progress, including support-tool compilation and four-image packaging. Packaging defects were corrected in source. Not a fresh P05 installation or a successful run of every support CLI action. |
| Cockpit type checks/60 tests, dashboard type checks/21 tests, 12 build-fixture tests, 9 staging tests | Recorded successful components. Existing support Go tests also participate in the full Go suite. They cover only the authored cases. |
| Aggregate `check-native.sh` at `8417a90` | **Stopped**, due to two fixture-directory permission assumptions. `eca7673` corrected them; the full pinned Go suite then passed with Go 1.26.7, CGO disabled and umask 022. No successful rerun of the complete aggregate entrypoint is recorded. Do not combine different revisions into an aggregate PASS. |
| Installed dashboard `35df189`; helper/new-project image `8417a90` | Deliberate mixed-component deployment, not an installation of the whole `8417a90` bundle. Preserve those actual identities in reports. |
| Historical CoreOS installation, native PAM/SELinux repair, root Tailnet reads and Accounts-menu adjustment | Useful substrate/operator evidence for those earlier bytes. The repaired persistent guest is not clean-install proof for the current installer or fresh UEFI fixture helper. |
| Actual developer onboarding, project provisioning and routed SSH/PTY/SCP/SFTP | Core U08 evidence. Infra is the approved routed client; no automatic laptop route, Tailnet route or P03 fixture proof follows. |

The retained worktrees have been removed, but their build artifacts were moved intact under `.artifacts/retained-worktree-builds/`. Their `retention.json` files map historical locations to retained paths. A successor must use that mapping, not assume logs' original paths still exist or rewrite old evidence. No native P03/P05/P06/P11 completion, P12 consolidated exact-candidate handoff, or native aarch64 result is established by these records.

## 2. Prioritized source findings

### A01 — High: the bundle filter admits current Soda credentials

**Owner:** P04 with U03/U04's public/private payload contract.

[`allowedPayload`](../internal/nativebuild/bundle.go#L31) broadly permits `rootfs/etc/`. Its substring/extension exclusions do **not** reject:

- `rootfs/etc/soda/oauth-secret`
- `rootfs/etc/soda/admin-token`
- `rootfs/etc/soda/grant-key`

These are the actual secret filenames created by [`soda-setup`](../cmd/soda-setup/main.go#L72), not hypothetical names. If one enters the stage before sealing, `tree` hashes it, the inventory records it and `Bundle` copies it. The current clean staging recipe does not intentionally copy these files; **this finding is not evidence that a retained bundle leaked credentials**. It is a missing fail-closed boundary in the advertised public-bundle contract.

**TODO:** consume the core's public stage contract, reject its real credential/runtime paths, and avoid reliance on a few filename suffixes alone. Add contamination tests for all three names, personalized configuration and private parent directories. Preserve legitimate vendor files such as the already-fixed npm `installed-deep.js`; do not reintroduce broad false-positive substring filtering.

### A02 — High: cancellation does not finish owned process-group cleanup

**Owner:** P02/P03.

[`Execute`](../internal/acceptance/command.go#L60) sends SIGTERM to the group, but relies on `exec.Cmd.WaitDelay` afterward. Go's timeout kill targets the child process, **not the whole process group**. A descendant that ignores TERM can survive. [`Process.Stop`](../internal/acceptance/process.go#L52) also returns when the direct child's `done` channel closes, without establishing that descendants have exited; the TERM-completion branch discards the child's result.

The existing tests exercise cancellation **before start** and an already-exited successful child. They do not cover this case. Remote `timeout --kill-after` is a separate server-side bound and does not fix local child ownership.

**TODO:** bounded group-level escalation and truthful cleanup outcomes, without stale-PID adoption or killing unrelated groups. Add a live TERM-resistant descendant fixture, leader-first exit, cancellation during execution, forced termination, output-pipe inheritance, and repeated cleanup cases. Preserve command/evidence errors independently. No actual processes were launched to reproduce this finding in the audit.

### A03 — High: raw log redaction is also used to serialize machine-readable state

**Owner:** P02/P12.

[`redactingWriter.flush`](../internal/acceptance/evidence.go#L201) rewrites HTTP(S) URLs in all bytes, including JSON written by `e.Write`. [`RedactString`](../internal/acceptance/evidence.go#L144) is applied before JSON encoding in some CLI fields and then the writer applies redaction again. The regex is not JSON-aware: a URL ending in a JSON-escaped quote/backslash can be changed into bytes that break JSON quoting. Exact raw-secret matching also does not recognize the JSON-escaped spelling of a registered secret containing a quote, backslash or control character. Tests cover only a plain-text password and ordinary absolute redirect URL.

**TODO:** keep safe structured values and JSON encoding separate from raw-stream sanitization; preserve parseable observations and exact public hashes. Cover escaped secrets, escaped/relative redirect representations, overlapping secrets, URL punctuation, split writes and final buffered fragments. Do not claim protection against arbitrary unknown secrets. This needs synthetic fixtures, not real credentials.

### A04 — High: final evidence failure can leave a success-shaped observation

**Owner:** P02/P03/P12.

[`tools/soda-acceptance`](../tools/soda-acceptance/main.go#L253) chooses `Outcome`/`Evidence` and hashes retained files **before** writing `observation.json` and performing its last leak scan. A final write/close/scan error affects the CLI return value but is not reflected in an already-written success record. `Handoff` has no independent failed-finalization receipt and accepts a structurally valid completed record whose listed files match.

VM startup also returns only an error after internally attempting cleanup: the CLI can retain `Execution: not-started` and a generic cleanup description even when QEMU actually launched and then failed. Cleanup errors after a successful boot are folded into execution rather than recorded as their own error result.

**TODO:** finalize evidence before publishing a success observation; retain a fail-closed result when finalization fails. Return enough bounded startup/cleanup facts to distinguish not-started, failed execution, failed retention and failed cleanup. Add disk/write/close failure injection, final-scan failure and partial-VM-launch report tests. No durable job/recovery framework is needed.

### A05 — Medium: filesystem confinement is incomplete outside the writer

**Owner:** P02/P04/P05.

`Evidence.Writer` uses an `os.Root`, but [`CheckSecrets`](../internal/acceptance/evidence.go#L90), [`Hashes` and parts of `Handoff`](../internal/acceptance/report.go#L37) return to pathname walking/opening/hashing. `Handoff` performs a confined `Lstat` and then hashes via an ordinary joined path; pathname replacement between these operations escapes that guarantee. [`bundle.tree`](../internal/nativebuild/bundle.go#L62) starts at `tools/soda-artifacts` without walking/validating the `tools` parent, so a symlinked parent can supply an outside verifier to seal/local export. The transfer path is stricter because it subsequently opens through `os.Root`.

The full-build output-parent guard also calls `mkdir(parents=True)` before rejecting a symlinked `.artifacts` ancestor. Installer extraction validates source links, not every existing destination ancestor; supported CoreOS `/usr/local -> /var/usrlocal` must remain distinguishable from an unexpected link in a writable target subtree.

**TODO:** carry directory-relative confinement through scans/hashes/copies and validate parents before creating outputs. Exercise renamed/swapped parents, `tools` parent links, existing destination links, traversal and special-file inputs. Preserve the legitimate CoreOS mapping. This is a source-level confinement gap, not a claim that a hostile process modified current artifacts.

### A06 — Medium: P04's public input inventory and completeness checks lag U02

**Owner:** U02's payload definition; P04 consumes it.

[`native-build-info.py`](../scripts/native-build-info.py#L29) copies Cockpit's package/lock inputs but omits `dashboard/package.json` and `dashboard/pnpm-lock.yaml`; the outer allowlist has no corresponding dashboard-input entries. The actual staged React files are included and hashed, but the advertised dependency-input record is incomplete.

[`tree`'s required-file list](../internal/nativebuild/bundle.go#L108) requires the Cockpit indexes, not the core dashboard index/license/Vite manifest. The bundle fixture can pass without any React payload. `inputs/native-build.json` is hashed but not decoded or cross-checked against the requested revision/platform/image identities. [`check-native.sh`](../scripts/check-native.sh) does run the core staging tests afterward, but direct `seal`/`bundle` does not establish those content checks.

**TODO:** add the real dashboard dependency inputs and coordinate one core-owned completeness check with the seal/export boundary. Reuse the existing [core staging assertions](../tests/packaging/test_staging.py#L26) or an explicit core validation interface; do not author a second frontend builder/manifest policy. Test missing assets/licenses/manifest, contradictory build metadata and stale payloads without manufacturing replacement assets.

### A07 — Medium: OCI inspection proves selected identities, not full format validity

**Owner:** P04.

[`InspectOCI`](../internal/nativebuild/oci.go) verifies tar entry types, blob hashes/sizes, config OS/architecture and selected labels. It does not validate index/manifest `schemaVersion`, descriptor media types or config rootfs/diff-ID consistency. Directory entries are skipped before duplicate accounting. The synthetic OCI fixture explicitly contains a non-filesystem “layer” and lacks several real image fields.

**TODO:** define and enforce the selected OCI subset, reject inconsistent descriptors/structure, and add valid small OCI fixture inputs plus wrong-schema/type/index/duplicate/truncated/oversized metadata cases. If layer-content validation remains outside this inspector, document that limit and retain authorized native import evidence rather than advertising full validity. Keep identities/checksums distinct from signatures and reproducibility claims.

### A08 — Medium: generic execution/transfer records do not guarantee artifact binding

**Owner:** P02/P05/P12.

`--artifact-file` is optional. Generic `exec` records a **requested** revision/target; it does not establish either from the arbitrary invoked command. [`TransferBundle`](../internal/acceptance/transfer.go#L19) computes and verifies a manifest hash internally, but the CLI's transfer observation retains only source/destination path arguments unless the caller separately supplies artifact references. [`Handoff`](../internal/acceptance/report.go#L64) verifies the observation and listed evidence files, not `Artifacts` identities/content. Its rendered summary omits invocation, exit code, cleanup and artifact fields, leaving them only in the cited JSON; blocked versus not-reached is not represented distinctly.

**TODO:** automatically retain the selected bundle identity for transfer/build/export observations; distinguish caller-declared source/target from verified native facts for generic checks. Require relevant public artifact references where the observation is presented as candidate-bound. Show cleanup/exit/artifact status in the human handoff and preserve blocked/not-reached/failure distinctions. Do not invent signatures or certify product acceptance. Retain the ability to report an honest partial result.

### A09 — Medium: fresh-VM preflight and native-input closure need completion

**Owner:** P01/P03/P05.

[`VMConfig.preflight`](../internal/acceptance/vm.go#L39) checks QEMU, qemu-img, KVM, private input/trust and port availability, but does not check that the `ssh` executable exists. A missing SSH client is discovered only after creating the VM and retrying readiness. QEMU `--version` and firmware hashes are retained; qemu-img/Butane/GPG/xz versions and a selected, exercised per-platform firmware/input combination are not a complete recorded input set. Tool subprocesses in CoreOS retrieval do not all have their own bounded deadlines/output caps.

**TODO:** complete missing-tool-before-write checks and useful failure diagnostics, record the actual selected tool/firmware inputs, and bound external-tool phases. Confirm genuine native architecture/KVM rather than treating compiled `runtime.GOARCH` alone as hardware evidence. Exercise successful boot, occupied port/path, wrong/mutable base, invalid trust, launch failure, mid-boot cancellation, retained NVRAM and normal shutdown/restart. No emulation fallback or adoption of `soda-test`.

### A10 — Medium: installer preflight is narrower than its operational claims

**Owner:** P05 around core installation/configuration contracts.

[`configure_network check`](../scripts/install-native.sh#L41) checks RFC1918 syntax and subordinate-ID collisions. It does not inspect actual host routes/interfaces or a conflicting pre-existing `soda-projects`/`soda0` network. Package presence is checked, but activation of the requested rpm-ostree deployment still needs an explicit observed check. Existing-container/state guards are useful and must remain.

**TODO:** agree the exact read-only network/deployment preflight with the core owner, or clearly expose these as operator-supplied prerequisites rather than checked facts. Add failure-path coverage that proves no install marker/payload/state mutation happens on rejected inputs. Then perform a separately authorized clean first install, verify writable-prefix modes/owners/labels and four-image load/tag restoration, and observe partial-install refusal. Do not test this by reinstalling the persistent guest or clearing its marker.

### A11 — Medium: P06/P11 checks cover a subset of their exits

**Owner:** P06/P11; core owns actual service/config semantics.

- [`host.sh`](../tests/installed/host.sh) asserts pre-activation Forgejo mappings but mainly prints `ss -lnt` after activation. It does not assert every intended listener/bind boundary. It checks `dashboard.json`, not all configured OAuth/admin/grant-key/TLS file permissions.
- [`VerifyInstalled`](../internal/nativebuild/installed.go) compares selected immutable files/modes/links and service image IDs; it intentionally does not inspect mutable config or existing projects. It requires a first-install revision marker and is not a verifier for the present mixed-component rollout. Do not relax it or relabel the rollout as a whole-bundle installation to make it pass.
- `service-ordering.sh` observes generated units and failed units; `cockpit-account.py` observes an account gate, not a password/session journey.
- `operator.mjs` observes root login, native read paths, menu hiding/origin preservation and logout. Enrollment, approvals, exit-node/LAN changes, provider runner lifecycle/jobs, interactive console and visual branding remain separate. A browser SELinux-domain check should be paired with actual enforcing-state evidence.

**TODO:** finish the assertions actually promised by P06 using core-provided configuration, with focused denied/missing/failing-native-command tests. Execute each P11 native mutation/review under its own grant and record cleanup. A new catch-all operator scenario engine is neither necessary nor in scope.

### A12 — Medium: published completion language and evidence handoff need repair

**Owner:** P01/P12/P13 documentation.

The plan/support/validation headers still said that all checks were unexecuted, while the handoff records later P04 and Go/build/packaging passes. Some guides also still described gh image installation or routed developer SSH as pending despite newer evidence. Historical source-only entries are valid **at their recorded point**, not as the current summary. Conversely, current source defects/test gaps prevent retaining an unconditional “active source complete” claim.

**TODO:** maintain one current snapshot linked to exact evidence and this audit; preserve chronological failure/repair entries. Identify the next exact candidate and distinguish its whole bundle from component rollouts. Locate retained evidence using the worktree-retention mapping. Produce a useful scoped P12 report only after identifying what actually ran and what remains absent; do not fabricate observations to fill the report's missing scopes.

## 3. Milestone-by-milestone remaining work

| Milestone | Current state | Still TODO / exit evidence |
| --- | --- | --- |
| **P01 — contracts/provenance** | Ownership, interfaces, notices and default media meanings documented; not a native-action milestone. | Resolve A01/A05/A06 shared-contract gaps; audit dependency/notice closure and actual tool/firmware/signer inputs. Keep all action-specific permissions explicit. Source refinements must follow the core plan. |
| **P02 — transport/evidence/remote** | Implemented; existing focused Go/Python cases have recorded suite execution. | A02–A05/A08 fixes and regression cases; actual pinned SSH success and precise host-key/auth/transport failures; cancellation under load; end-to-end fresh prepare → separately requested build/check/bundle, bad/stale receipts, wrong revision/target and no automatic later phase. Current direct U08 client proof did not exercise this remote dispatcher. |
| **P03 — native VM fixture** | QMP, KVM-only args, trust, new overlay/NVRAM and cleanup source exist. | A02/A04/A05/A09; meaningful lifecycle tests beyond string checks and pre-cancelled contexts. Select fresh paths/identity/firmware/port; verify minimal guest boot, restart of same files and bounded shutdown/cleanup. **No recorded native run of this helper.** |
| **P04 — artifacts** | Real x86_64 build/stage/seal at `8417a90`; focused checks passed in the recorded revisions. | A01/A05–A08; one clean exact-candidate full build + aggregate check after fixes; exported-bundle verification/transfer/import evidence and public input/notice closure. Do not count aggregate components from different revisions as one successful run. |
| **P05 — input/install** | CoreOS lock, signer/hash retrieval, private provisioning/conversion, transfer and first-install source exist. | A01/A04/A05/A08–A10; downloader/conversion/transfer/install negative tests; independently selected trusted signer/keyring; actual checksum+signature+decompression path; strict per-platform Ignition/firmware behavior; extensions and approved activation reboot; clean first install with no repair edits. Existing dashboard migration/helper rollout is not this exit. |
| **P06 — substrate** | Checks authored; earlier persistent-guest substrate/PAM observations available. | A11; run current host/order/PAM/TLS checks against a fresh exact installed candidate, before and after separately approved core activation; record deployment/packages, generated units, identities/caps, ownership/labels, listeners and failed units with SELinux enforcing. No new product OAuth/People flows under P06. |
| **P07 — developer/workloads** | **Moved to U08/U20.** | No independent P implementation or completion task. Supply support only when useful; see core remainder below. |
| **P08 — persistence** | **Moved to U08/U20.** | No independent P suite. P03 may provide an authorized appliance restart; core defines assertions about existing project state. |
| **P09 — ISO** | **Unselected; not implemented.** | First decide public/private media, network needs and exact disk selection. Only if selected: verified CoreOS ISO inputs, supported customize wrapper, payload/secret checks, wrong-input/overwrite tests, fresh exact-disk install and boot without media; core acceptance if claiming product support. No Anaconda/bootc port or default erasure. |
| **P10 — QCOW2 delivery** | **Unselected; not implemented.** P03 overlay support/P05 pristine cache do not implement it. | First decide companion kit versus single preinstalled image. If selected: standalone/non-running disk checks, optional compression/equality, secret exclusion and independently provisioned instance identities; subsequent boot retention and selected resize/capacity checks. No export/sanitization of `soda-test`; application separation remains core-owned. |
| **P11 — retained integrations** | Production pages/backing logic/tests retained; native read and build/version subsets exist. | Current root Cockpit/PAM/SELinux/Accounts journey; Tailnet enrollment/approval/exit-node/LAN behavior and exact cleanup; listener-bound advertisement and intended-client key probe; both providers' real runner registration/lifecycle/trusted job/removal/leftovers; interactive console/quiet transfer; native visual branding and opt-in renderer. See detailed list below. |
| **P12 — x86_64 handoff** | Report source exists; substantial reusable but mixed-candidate evidence is documented. | A04/A08/A12; freeze actual selected scope/bytes, produce an accurate report of performed versus failed/blocked/not-reached checks and cleanup. Native completion still needs the selected support observations. Can report partial results now without waiting for U20; cannot call missing native exits complete. |
| **P13 — aarch64** | Architecture branches and upstream input row exist; **no matching-native build/install/operator evidence recorded**. | Independently name ARM Linux/KVM hardware, tools, firmware, signer and targets; repeat selected P02–P06/P11 work and issue its own handoff. Verify provider/tool/package closure on ARM. No x86 barrier, laptop-architecture substitute or emulation claim. |

### P11 native checklist (not satisfied by version strings)

- **Cockpit:** trusted-CA root password/session/logout, correct native SELinux domain with enforcing policy; a real non-root account denial; Accounts navigation hidden without replacing authorization.
- **Tailnet:** present native state/peers truthfully; separately authorized sign-in, approval, exit-node selection/advertisement and LAN preference. Preserve existing daemon state. Subnet-route testing needs its own exact routing grant and does not follow from enrollment.
- **Forgejo advertisement:** test refusal when actual private Git mapping cannot serve the selected Tailnet address; success only with a serving mapping; preserve browser/OAuth origins. Probe from the intended client with a trusted key. Git authentication/permissions remain core-owned.
- **Runners:** separately approved Forgejo and GitHub registration, actual unprivileged native service identity/capacity, start/stop/restart and a trusted provider-scheduled job. Retain provider URL/attempt/result, distinguish listener activity from execution, and record explicitly approved removal plus provider leftovers. Reuse existing runner APIs/UI and the opt-in fixture, not a new CI scheduler.
- **Console:** actual root interactive login and accurate configured/observed origins; quiet noninteractive SSH/SCP/SFTP. The current shell-hook probe is only a subset.
- **Branding:** canonical native assets, supported narrow/wide/theme/focus checks, authorized component-sheet capture and renderer tag. Source/staging presence is not a visual pass. Remove only the exact run-owned review sheet when permitted.
- **Tea/gh:** credit successful x86 build/delivery/version observations; verify ARM delivery independently. Personal credentials/API use, separation of each user's configuration, and real project Git remain U08/U20—not new P fixtures.

## 4. Missing authored coverage versus tests merely awaiting rerun

The inspected outside packages contain **28 Go test functions** and `test_native_support.py` contains **8 Python test methods**. This is an inventory of source cases, not newly measured coverage or test execution. Existing production/Cockpit/packaging suites add useful coverage, but do not supply the missing cases below.

| Area | Already authored | Missing or materially shallow |
| --- | --- | --- |
| Command/evidence | Literal argv, stdin, expected exit 23, pre-cancel, private/exclusive paths, split plain-secret/URL writes, output-size bound, error identity | Cancellation after launch, resistant descendants, output/close failures, escaped/structured secrets, valid final JSON and finalization failure; exact denial versus failed SSH transport end-to-end. |
| SSH/remote | Option-string inspection, request revision/target mismatch, unknown phase rejection | Actual trusted/untrusted/changed-host-key SSH, key/password auth failure distinction, phase success ordering, occupied/failed/stale receipts, source changes during phase, native target mismatch and safe timeout behavior. |
| VM/QMP | Negotiation/native error/malformed responses; pre-cancel; invalid architecture before disk; argv inspection; synthetic trust; already-exited child cleanup | Occupied port/path, missing SSH, wrong/read-write base, partial launch, live unexpected exit/cancellation, NVRAM/disk identity across restart and exact cleanup. String presence is not lifecycle proof. |
| OCI/bundle | Wrong platform/revision, escape/misnamed format, one changed blob, export/no overwrite/private umask, two denied file names, missing Cockpit page, signer-status parsing | A01 credentials, real core React closure, input-metadata consistency, malformed schemas/descriptors, duplicates/symlinks/special entries, truncation/limits, changed source during copy and directory-ancestor confinement. |
| CoreOS/Butane | Small `VALIDSIG` text cases; source private-mode/plaintext/profile/host-key-argv tests | No direct `ReadCoreOS`/`FetchCoreOS` end-to-end fixture tests; bad/missing signature/checksum, unsafe redirect, interrupted/bounded decompression, stale cache, output collisions, strict Butane failure/secret diagnostics and tool deadlines. |
| Transfer/install | Installer source-substring/order checks | No direct `TransferBundle`/streaming tests; changed bytes/verifier, destination collisions, interrupted transfer, paths/modes/links. No behavioral mocked installer preflight tests across wrong OS/arch/package/identity/network/partial state; no `VerifyInstalled` tests. |
| Host/operator/HTTPS | Production backing tests, one HTTPS-origin parser test; installed check scripts | Actual TLS wrong-CA/hostname/redirect/failure cases, failed native inspection, correct phase/bind/secret-mode assertions, operator browser failures and provider lifecycle evidence/cleanup. |
| CLI/report | One failed/missing-scope handoff test including architecture/file tampering | No `tools/*` command tests; CLI finalization/cancellation/exit fields, malformed records, unknown/mixed states, artifact binding, cleanup visibility and moved evidence paths. |

Add focused doubles/fixtures at these existing boundaries, not an exhaustive new recovery framework or a copied product suite. Executing existing tests again does not author these missing tests; writing them does not establish native exits.

## 5. Core-owned work still pending, not native-port omissions

The current [core snapshot](dashboard-implementation-plan.md#current-execution-snapshot--u08-completion-run) is the authority. Do **not** start a P07/P08 implementation for these:

- U08: personal Git clone/commit/push through the routed client; truly shared files and one shared mise installation executed by both users; remaining host-engine/cross-project isolation; nested web/database workload, bind mounts and durable data.
- U08/U20: separately authorized existing-project stop/start and guest reboot; compare existing container identity, users/homes/SSH host keys, dirty work, shared installs/files, configuration and service data. The temporary route is not reboot-persistent.
- U03/U04/U18/U20: complete populated-state migration/credential/security/cutover coverage. The current preview is not default SPA cutover.
- U14/U16: provider Actions/admin UI. P11 supplies local runner observations, not those frontend/adaptor features.
- U20: final product acceptance on selected actual targets/architectures. P12 neither duplicates nor replaces it.

Keep Alice/Bob, their repositories/keys/memberships/project roots, the tunnel and retained evidence intact. The old pre-migration backup predates later writes and is not a lossless rollback. No fallback runtime, unrestricted host socket, privileged parent, VM workspace replacement, deletion/recovery subsystem or private-resource branching is authorized by an unresolved native test.

## 6. Recommended completion order and permission boundary

1. **Source first:** fix A01–A05 and author their regression tests; reconcile A06–A11 against current core-owned contracts. Complete P01 input/provenance notes. Do not change production policy to make a support probe pass.
2. **Named native build/check grant:** select one exact revision/fresh checkout, build and run the complete aggregate check successfully; export and verify that same bundle. Preserve the `8417a90` stage and earlier failed logs; use the retention mapping rather than deleting artifacts.
3. **Named new-fixture/input grant:** independently select trusted signer/keyring, approved download tools, firmware, identity/ports/private inputs; exercise fetch/conversion and minimal P03 boot/shutdown. Request restart permission separately as applicable.
4. **Named fresh-install/reboot grant:** deliver the verified bundle, request/activate extensions and perform a genuinely clean first install. Use core-owned setup/key/TLS/activation steps only when their actions are approved. Run P06 facts and rejection cases; no recovery edits counted as clean proof.
5. **Per-integration P11 grants:** trusted browser profile/CA, exact Tailnet/provider resources, job/repository/labels, actions and cleanup limits. Record denied/unavailable integrations honestly; do not broaden network exposure or schedule untrusted jobs.
6. **P12 handoff:** cite actual hashes, revisions, target/client topology, invocations, outcomes and leftovers. A useful partial handoff is allowed; missing scopes remain missing and do not certify a deployment.
7. **P13 independently:** use approved native aarch64 hardware and its own inputs; do not delay useful x86/core work while waiting.
8. **Only if selected:** P09/P10 delivery decisions precede their implementation and disk/media permissions. Public cloud/bare-metal/offline/signing/publication claims remain outside the selected support scope.

This audit is not any of those execution grants. Immediate work may remain source-only while the existing core proceeds through its authorized tools and retained fixtures.
