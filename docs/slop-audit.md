# Slop audit — Layer 1 (static) checklist

Static Layer 1 run against the canonical `~/Projects/sodaos` checkout, verified
indicator by indicator. This document owns the finding list and fix order.
Methodology reference lives outside this repo
(`~/Projects/slop-audit`: `spec/`, `spec/dimensions/00-quick-reference.md`,
`tools/l1_analyzer/README.md`).

- Commit: `59dbc38`, branch `main`, clean tree at time of run.
- Tool: `slop-audit-l1 1.0.0+gec358a56`, static only.
- Command: `uv run slop-audit-l1 /Users/vince/Projects/sodaos --no-exec`
  (from `~/Projects/slop-audit/tools/l1_analyzer`).
- Raw JSON: `/tmp/sodaos-l1.json` (re-runnable; not committed).
- Verdict: **Grade F** — 24 promiscuous decision-driving state cells;
  96% of decided state finitely testable (514/538); silence 34.8% (below the
  50% no-grade floor). **Mixed** pattern: 6 of 18 measured Slops, of which two
  collapsed on verification (effective: 4 real + 1 placement artifact).
- Runtime half (L1.19 coverage share, L1.20 determinism) not run; it executes
  this repo's own suite including long native build/media/VM tests.

## Fix checklist — cheap and selected

- [x] Pre-commit hooks (L1.9 was Slop: no `.pre-commit-config.yaml`, no
  `.husky`). `.githooks/pre-commit` now gates staged changes only: `git diff
  --check`, credential shapes (`tskey-*`, private-key blocks, `AKIA*`), <!-- slop-audit-allow: documents the hook's scanned credential shapes -->
  `gofumpt`, complexity below 10 on staged production Go
  (`scripts/check-complexity.sh`), `go vet` on staged packages. Clone setup:
  `git config core.hooksPath .githooks` (local, not committed).
- [x] `tskey-` fixtures kept by design (decision A1): the prefix is the <!-- slop-audit-allow: documents why production sites keep the Tailscale-shaped prefix -->
  production validation shape (`credentialPattern`, `authKeyPattern`), so
  dummies must wear it to exercise the real path — renaming them would untest
  the validator. Instead the hook allowlists: `slop-audit-allow` markers on
  the 4 production sites, path exclusion for `*_test.go` / `*.test.ts` /
  `tests/`. Private-key blocks and `AKIA*` are never allowlisted. All four
  hook paths probed (block unmarked, silent marked, AKIA always blocked,
  tskey-in-tests silent). L1.14's 9 hits were all read and are false <!-- slop-audit-allow: documents the hook probe covering tskey-shaped values in tests -->
  positives; zero confirmed credentials.

## Fix checklist — needs localization before code changes

- [ ] Localize L1.13 fuzzy duplication (15.57%, Slop; per-file clusters not yet
  extracted from the JSON — no file list exists there). Fix only the clusters
  that are real shared logic; leave fixture resemblance alone.
- [ ] Review `any` / `interface{}` density (L1.15 Slop, 8.53/KLOC, ~195 `any`
  lines + 14 `interface{}`; diffuse, no hotspot). Judge per use; generic
  envelopes and helpers may be legitimate. No blanket rewrite.

## Known F drivers — deferred by design, not queued work

- [ ] Bound the 24 promiscuous decision-driving cells only when the APIs
  stabilize (trust stores, peer tables, image payloads:
  `internal/releasedelivery/model.go`, `internal/releasedelivery/native.go:102`,
  `internal/appliancerelease/payload.go:18,57`,
  `internal/host/terminal.go:107,329`, `internal/web/provider.go:30`,
  `internal/web/terminal.go:57,63`, `internal/web/terminal_sessions.go:141,146`,
  `internal/web/management.go:93`, `internal/tailnet/management_validation.go:28`;
  5 of the 24 are test doubles). Bounding key sets is API-stabilization work;
  doing it while the product is unreleased and moving is the wrong order.

## Deliberately not fixed (indicator-chasing, not engineering)

- [ ] No CI pipelines (L1.10 Slop, confirmed: no `.github/`). Adding pipelines
  for the band would be checkbox shaping; add CI only if wanted for its own
  sake under applicable approval (automatic CI needs explicit approval).
- [ ] Containerization (L1.11 reports absent — placement artifact). Six
  `*.Containerfile` exist under `appliance/` and `project-os/`; the indicator
  only checks the root. Layer 2 read: present but minimal (no HEALTHCHECK, one
  `USER` line).
- [ ] Absolute paths (additive, 38 flagged — ~35 are Printf verbs `q:\n%s` in
  Forgejo fixture tests; the two production hits are portable procfs
  (`internal/host/tailnet_companion.go:107`,
  `internal/host/tailnet_files.go:265`), not machine-specific. Real count: 0.
- [ ] Deletion ratios L1.5 (31.0%) / L1.6 (5.4%) Not Healthy — expected
  pre-release accumulation; the methodology's own report carries the same
  calibration warning for canon-like repos. Ignore until release.

## Healthy — hold the line

- [ ] L1.1 doc-only 43.2%, L1.2 code-only 9.2%, L1.3 mixed 41.8%, L1.4 doc
  lines 29.9%, L1.7 delete-heavy 24.1%, L1.8 test-to-prod 1.15, L1.12
  unreachable 0.02% (3 unreferenced defs), L1.16 trailing whitespace 0.0%,
  L1.17 god files 0.24% (1/412, unlocalized), L1.18 mutable-state 27.6%
  (Not Healthy, consistent with the F).
- [ ] Thread surface: 0 overrides; 1 production review site
  (`internal/web/terminal.go:187`, `err` shared write in goroutine) + 1 test
  review site + 4 low-precision candidates.
- [ ] L1.19 enumerated 7,384 decision points across 346 files; exercised share
  unmeasured (needs exec).

## Go quality gates (`.githooks/pre-commit`, staged scope only)

Hook checks, in order: `git diff --check`, credential shapes,
complexity below 10 on staged production Go, gofumpt, staticcheck
(GOOS=linux), errcheck (GOOS=linux), `go vet` (GOOS=linux). Whole-repo scripts below are red-but-ratcheting: the
hook blocks new violations in staged files; legacy backlogs do not block
unrelated commits. Mechanical-only commits that restage legacy-violating files
(e.g. the gofumpt reformat) go through with `--no-verify` and a note.

- [x] Complexity: `scripts/check-complexity.sh` via pinned `go tool gocyclo`
  (v0.6.0). Shipping Go is `internal/`, `cmd/`, `appliance/`, `project-os/`
  (not `tools/`, `scripts/`, `tests/`, or `*_test.go`). TypeScript cyclo is
  browser payload only (`frontend/`, `assets/branding/`).
- [x] gofumpt: `scripts/check-gofumpt.sh` via pinned `go tool gofumpt`
  (v0.9.1), zero tolerance. First measurement undercounted (24) through a
  `tee | head` SIGPIPE truncation; true backlog is ~185 files. 22 files
  reformatted (format-only, linux build passes); no mass reformat of the rest
  without an explicit decision — the hook ratchets staged files instead.
  Pre-commit no longer runs a separate `gofmt` pass; gofumpt is a strict
  superset.
- [ ] errcheck: `scripts/check-errcheck.sh` via pinned `github.com/kisielk/errcheck`
  (v1.9.0), linux analysis like staticcheck. Staged-package ratchet; do not
  mass-fix the tree.
- [ ] staticcheck: `scripts/check-staticcheck.sh` (pinned v0.8.1, must analyze
  GOOS=linux or installer files drop out; the script builds the tool for the
  host first — passing GOOS=linux to `go tool` builds an unexecutable binary).
  Whole repo: 48 findings, gate exits 1.

TypeScript format/lint/complexity (oxfmt, oxlint, cyclomatic below 10 on
production TS) are owned by [the TypeScript guide](typescript.md#formatting-linting-and-complexity),
not this Go checklist. Python format/lint/complexity (Ruff) are owned by
[the Python tooling guide](python.md). The cyclomatic threshold matches production
Go; the hook is staged-only for the same ratchet reason. TypeScript first
measurement: 129 files fail formatting, 92 production functions at cyclomatic 10+,
3 oxlint correctness findings. No mass TypeScript or Python rewrite.

## staticcheck backlog (48) — fix order

- [x] Bulk mechanical: 21 × ST1005 lowercased, 12 × ST1013 `http.Status*`
  constants (verified no test asserts the old strings; the one match,
  `command_test.go:54`, reads the untouched CLI guidance in `command.go:23`).
- [x] SA1019 production set fixed: `tar.TypeRegA` → `TypeReg` (identical
  value, no behavior change); 6 × `runtime.GOROOT()` replaced with a
  run-time `exec.LookPath("go")` resolver in `hostimage/build.go` and
  `nativebuild/progress.go` (GOTOOLCHAIN pin retained, so the version
  guarantee survives; PATH decides the installation). The pinned-compiler
  test now asserts LookPath resolution. Whole-repo staticcheck is exactly
  the 2 intentional own-`AdminTokenFile` markers in tests — left standing.
- [x] Removals (each matched an L1.12 unreferenced def — two instruments
  agreed; grep confirmed single occurrence): `companionName`, `writeNewJSON`
  (+ its orphaned `encoding/json` import), `buildCapture` alias.
- [x] Dead store: `install_linux.go:236` replaced with a comment (retry works
  via outer-loop fall-through).
- [x] `console_linux.go:248`: `//lint:ignore SA4011,S1023` with the close-read
  reason (false positive, not a bug).
- [x] Test-file nits: S1007 raw regexp, ST1013 in `client_test.go:25`.

## Complexity top-10 triage (workflow audit, 10 auditors + synthesis)

Verdict: **0 REMOVE, 8 RESTRUCTURE, 2 KEEP**. No separable obsolete or
callerless capability found — the complexity is load-bearing, so there is no
deletion program here, only a restructure program. Savings below are auditor
estimates from branch counts, not gocyclo reruns on split code.

- [x] (`6be9fe1`) `tools/soda-acceptance/main.go:35` run (86 → 7): 5 per-action executors
  (exec/native, probe-ssh, transfer, vm) + shared validate/setup/finalize.
- [x] (`936bc7e`) `internal/hostimage/assemble.go:144` assembleMedia (72 → 9): 9 phase
  extracts (authority, candidate, fast-compression gate, P7 sign+inventory,
  P8 cosa build, meta verify, installer pin, readback, seal).
- [x] (`8008eb7`) `internal/nativequalification/state.go:50` GuestState (70 → 7): guards /
  bootstrap / config-client / repo-setup / mutate / observation. Note: host-side
  P9 SSH driver does not exist yet (B4 unstarted).
- [x] (`ada6d06`) `internal/installer/install_linux.go:357` collectDiskInstallChoices
  (63 → 5): per-step extracts (network/disk/hostname/password/subnet+review) +
  one shared navigation-input helper.
- [x] (`a1946de`) `internal/hostimage/build.go:102` build (63 → 5): admitBuildInputs /
  freezeBaseImageConfig / compileShippingTools; build() stays as sequencer.
- [x] (`f218c4a`) `internal/installer/enrollment_linux.go:154` armEnrollment (59 → 5):
  guardExistingEnrollmentState / publishEnrollmentState /
  waitEnrollmentResult + named enrollmentSession cleanup type.
- [x] (`d79a0ea`) `internal/releasedelivery/publish.go:96` Publish (57 → 9): guard-ledger /
  observe-only / signed-admission / immutable-commit / promotion / finalize.
- [x] (`4c296be`) `internal/tailnet/management.go:325` HostAction (50 → 9): per-action
  executors + verifiers; dispatcher at 9.
- [x] Initial exemptions: `apiTerminal` (86) and `StartTailnet` (49) were
  initially kept in Batch 1, then completely eliminated with zero exemptions in Batch 4 below.

## Complexity top-10 batch 2 (restructure program)

All 10 functions restructured with structural moves only; zero gaming, all helpers and entrypoints strictly below 10 ($\le 9$):

- [x] (`56cfed8`) `tools/soda-release/main.go:39` run (40 → 2): per-operation executor table + argument parser.
- [x] (`8db45ca`) `internal/nativebuild/oci.go:19` inspectOCIImage (36 → 5): manifest verification, index architecture matching, config extraction.
- [x] (`d4b2e45`) `internal/nativebuild/installed.go:42` verifyInstalled (38 → 4): input validation, bundle inventory, binary checks, system requirements.
- [x] (`c910117`) `internal/installer/enrollment_keys.go:132` appendEnrollmentKeyWithWriter (48 → 8): key generation, file read/stat, formatting, atomic file writing.
- [x] (`f0e7991`) `internal/host/tailnet.go:21` tailnetHandler (40 → 7): modular HTTP action dispatchers and request parser.
- [x] (`ee83ed4`) `internal/host/daemon.go:19` ServeHTTP (38 → 8): request routing, modular route dispatchers, health checks.
- [x] (`222ddc5`) `internal/web/login_cancel.go:14` cancelLogin (36 → 6): preflight validation, token resolution, and cancellation execution helpers.
- [x] (`be0b144`) `internal/web/repositories.go:37` apiRepositories (36 → 9): query validation, repository choices resolution, session verifier.
- [x] (`31bd722`) `internal/web/environments_api.go:111` apiCreateEnvironment (36 → 8): input validation, prechecks, reconfirmation, provisioning, tailnet helpers.
- [x] (`65e3542`) `internal/web/spaces.go:33` apiSpaces (40 → 7): domain-aligned helpers for authority resolution, native inspection, terminal inventory, tailnet check, session verification.

## Complexity top-10 batch 3 (restructure program)

All 10 functions restructured with structural moves only; zero gaming, all helpers and entrypoints strictly below 10 ($\le 9$):

- [x] (`d97a53e`) `internal/installer/payload_linux.go:232` copyInstalledPayloadWith (36 → 6): modular stat, tar entry validation, whiteout, payload copy, and error reporting helpers.
- [x] (`e01c2aa`) `internal/host/terminal.go:126` (TerminalFrame).outputValid (36 → 7): split into modular UTF-8 decode, control byte filter, escape sequence parser, and frame content validation helpers.
- [x] (`38cf95c`) `internal/installer/payload_linux.go:441` recheckInstalledRootFromJSON (35 → 3): unmarshaling, root path validation, manifest record comparison, and payload inventory verification helpers.
- [x] (`b2106ee`) `internal/host/terminal.go:374` (*Daemon).terminalHandler (35 → 7): admission, handshake, launch, and bidirectional I/O pump helpers.
- [x] (`b8785aa`) `internal/host/management.go:91` (*Daemon).lifecycle (35 → 9): modular systemd unit inspection, transition dispatch, and outcome verification helpers.
- [x] (`4fa2f2c`) `internal/acceptance/vm.go:124` LaunchVM (35 → 9): modular base configuration, fixture setup, format resolution, and workspace preparation helpers.
- [x] (`28e7ca9`) `internal/web/auth.go:136` (*Server).callback (34 → 4): validation, token exchange, session completion, and redirect dispatch helpers.
- [x] (`f5eadee`) `internal/host/tailnet_runtime.go:37` processRunIdentity (34 → 7): modular stat, ID map resolution, namespace admission, and boot ID helpers.
- [x] (`0a0513d`) `internal/nativebuild/production.go:327` (Production).exportImages (33 → 7): modular Rocky base resolution, app images, Forgejo image, proxy image, and tailnet image export helpers.
- [x] (`5f46f60`) `internal/tailnet/policy.go:246` (*policyStore).update (32 → 7): modular request validation, credential check, lock/load, rotate/save mutation, default/disable toggling, and atomic publication helpers.

## Complexity top-10 batch 4 (zero exemptions)

All 10 functions restructured with structural moves only; zero gaming, all helpers and entrypoints strictly below 10 ($\le 9$). Eliminated all legacy exemptions:

- [x] (`dbd57cb`) `internal/web/terminal.go:16` (*Server).apiTerminal (86 → 8): monolithic WebSocket handler split into 23 focused helpers covering headers, auth, peer lookup, workspace admission, PTY launch, pump, and cleanup; previous exemption eliminated.
- [x] (`a7d3700`) `internal/host/tailnet_companion.go:250` (*Daemon).StartTailnet (49 → 8): linear pipeline decomposed into policy verification, admission, runtime state, container launch, enrollment, and finalize helpers; previous exemption eliminated.
- [x] (`bed7134`) `internal/installer/payload_linux.go:121` installedRootFromJSON (32 → 3): unmarshaling, disk identity validation, partition discovery, and payload root verification helpers.
- [x] (`d282dd4`) `internal/nativebuild/oci.go:37` InspectOCI (31 → 3): archive opening, entry name validation, directory validation, regular entry check, blob reading, tar header processing, entry inventory, and index inspection helpers.
- [x] (`5721ec9`) `internal/installer/setup_linux.go:82` configurePrivateInstall (31 → 8): preexisting install check, address querying, address prompting, token verification/prompting, IP assignment recheck, and activation execution helpers.
- [x] (`8c3330a`) `internal/installer/enrollment_linux.go:468` ServeEnrollment (31 → 3): environment validation, broker listener setup, connection acceptance, deadline configuration, key reading, status reporting, key commit, and serving loop helpers.
- [x] (`7af7575`) `internal/tailnet/policy.go:416` (*policyStore).project (30 → 6): request validation, policy lock/load, binding mismatch check, binding validation, project mutation, initial state resolution, and view projection helpers.
- [x] (`1187b70`) `internal/hostimage/prepare.go:104` Prepare (30 → 6): preparedWriter state encapsulating file copying, base files creation, rootfs file map, symlink/extras staging, rootfs staging, build record emission, and base inputs loading.
- [x] (`73e73e6`) `internal/tailnet/management.go:235` (*Management).observe (29 → 8): native status fetching, native prefs fetching, backend state validation, host preferences population, self peer application, peer list sorting, and host revision digest helpers.
- [x] (`d5af8ee`) `internal/releasedelivery/model.go:217` AdmitChannel (29 → 7): release reference validation, channel releases validation, channel identity validation, timing/freshness check, progression verification, and highwater state advancement helpers.

## Complexity top-10 batch 5 (zero exemptions)

All 10 functions restructured with structural moves only; zero gaming, all helpers and entrypoints strictly below 10 ($\le 8$):

- [x] (`198c2e7`) `internal/nativebuild/oci_layout.go:21` InspectOCILayout (28 → 7): input validation, layout root opener, blob loader, descriptor inspector, image set inspector, and file tally helpers.
- [x] (`1a2906c`) `internal/acceptance/disk_vm_linux.go:19` LaunchDiskVM (28 → 6): config validator, paths/disk checks, input evidence recorder, ISO handler, and QEMU argument builder helpers.
- [x] (`46bb8a5`) `internal/tailnet/project_status.go:8` ProjectStatus (27 → 6): status response parser, preferences validator, binding/self matchers, and peer resolution helpers.
- [x] (`c501b5d`) `internal/tailnet/management_types.go:85` (HostRequest).Validate (27 → 6): extra-field check, signin validator, confirmed action validator, exit-node validator, and advertise action validator helpers.
- [x] (`c789fc9`) `internal/tailnet/management_types.go:136` (EnrollmentRequest).Validate (27 → 4): tags validator, policy validator, mutation validator, payload check, and toggle validator helpers.
- [x] (`e7453f2`) `internal/releasedelivery/fetch.go:102` verifyReleases (27 → 6): architectures resolver, image metadata/manifest verifiers, copy validator, release images verifier, identity tracker, and per-architecture release verifier helpers.
- [x] (`46986ae`) `internal/hostimage/complete.go:145` Complete (27 → 7): payload validator, quadlet generator/writer, branding/MOTD checker, factory defaults writer, systemd service/dropin configurator, and release metadata writer/normalizer helpers.
- [x] (`3ca434b`) `internal/acceptance/vm.go:41` (VMConfig).preflight (27 → 5): config identity validator, paths validator, SSH/trust validator, input files validator, and host toolchain/KVM checker helpers.
- [x] (`a10fa38`) `internal/web/management.go:39` (*Server).apiLifecycle (26 → 7): environment preflight checker, request decoder, operator authorization, session validator, stop terminal manager, and mutation handler helpers.
- [x] (`4fde77f`) `internal/nativebuild/bundle.go:127` tree (26 → 4): bundle root opener, entry file resolver, tree entry processor, payload subtree walker, and required/Sodaspaces payload verifier helpers.

## Complexity top-10 batch 6 (zero exemptions)

All 10 functions restructured with structural moves only; zero gaming, all helpers and entrypoints strictly below 10 ($\le 9$):

- [x] (`d8d1ba6`) `tools/soda-build/main.go:38` run (26 → 6): flag parser, worker admission, signal watcher, env sanitizer, progress, source binding, and parent dispatch helpers.
- [x] (`4aceca7`) `internal/host/terminal.go:88` (TerminalRequest).valid (26 → 4): actor, window, scope, list, sized/idle action, and action dispatcher helpers.
- [x] (`7e7d988`) `internal/web/management.go:177` (*Server).apiAccessKeys (25 → 8): environment, member authorization, confirmation, mutation, and fingerprint helpers.
- [x] (`7ecb495`) `internal/tailnet/management_validation.go:107` (ProjectView).Validate (25 → 4): availability, identity, persistence, and runtime/connected-state validators.
- [x] (`ced4539`) `internal/releasedelivery/model.go:137` (Release).Validate (25 → 8): identity, notes/evidence size, payload decode, provenance, and evidence-map validators.
- [x] (`18e8055`) `internal/nativebuild/bundle.go:56` allowedPayload (25 → 4): Forgejo, retired presentation, private rootfs, rootfs tree, and bundle manifest matchers.
- [x] (`b3efe81`) `internal/nativebuild/production.go:222` (*Production).ResolveInputs (25 → 7): recipe refs, locked Tailnet base, image pull, and provenance-record helpers.
- [x] (`995f2ee`) `internal/hostimage/assemble.go:55` prepareAssembler (25 → 5): media-tools lock, config fetch, build-args pin, layer verify, and wrapper helpers.
- [x] (`91550a8`) `internal/hostimage/build_payload.go:16` completeCandidate (25 → 7): package admission, command linking, Forgejo staging, inspect, and payload seal helpers.
- [x] (`501a9d8`) `internal/web/auth.go:77` (*Server).login (24 → 6): query parse, destination, OAuth context, cookie, begin-attempt, and redirect helpers.

## Complexity gate (package-grouped finish)

Later leftovers were split in larger package commits rather than one function per
commit. `scripts/check-complexity.sh` passed on the full production tree.

- [x] (`cec843a`) installer disk/host inspect/account and release admit helpers
- [x] (`8de50c4`) web HTTP handlers
- [x] (`7b179d4`) acceptance evidence, VM, and probe helpers
- [x] (`eb35ba8`) remaining installer, host, and hostimage leftovers
- [x] (`6e20148`) nativebuild, delivery, tailnet, Forgejo, store, runners, tools

## Open verification items

- [ ] Grep git history for credential shapes (L1.14 covered the working tree
  only: git-tracked files, 44 unreadable/oversized excluded).
- [ ] Decide the runtime run: full exec (long native tests) or scoped to fast
  packages for L1.19/L1.20.
- [ ] Optional Layer 2 full inspection on the weakest dimensions (4.10 CI/CD
  Absent at kill-check level; 4.11 Partial; 4.16 weak; 4.15 strong).
