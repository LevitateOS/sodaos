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
  --check`, credential shapes (`tskey-*`, private-key blocks, `AKIA*`),
  `gofmt -l`, complexity below 10 on staged production Go
  (`scripts/check-complexity.sh`), `go vet` on staged packages. Clone setup:
  `git config core.hooksPath .githooks` (local, not committed).
- [ ] Rename `tskey-`-prefixed dummy values to a shape no scanner treats as a
  Tailscale key: `internal/tailnet/management_validation.go:18`,
  `internal/tailnet/policy_test.go:23,68`. L1.14's 9 hits were all read and are
  false positives (`synthetic-*` fixtures, the Tailscale `TokenURL` constant in
  `internal/tailnet/enrollment.go:93`); zero confirmed credentials.

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

Hook checks, in order: `git diff --check`, credential shapes, `gofmt -l`,
complexity below 10 on staged production Go, gofumpt, staticcheck
(GOOS=linux), `go vet`. Whole-repo scripts below are red-but-ratcheting: the
hook blocks new violations in staged files; legacy backlogs do not block
unrelated commits. Mechanical-only commits that restage legacy-violating files
(e.g. the gofumpt reformat) go through with `--no-verify` and a note.

- [ ] Complexity: `scripts/check-complexity.sh` via pinned `go tool gocyclo`
  (v0.6.0). Whole repo: 223 prod violations (88 at 20+). Top:
  `tools/soda-acceptance/main.go:35` (86), `internal/web/terminal.go:16` (86),
  `internal/hostimage/assemble.go:144` (72).
- [x] gofumpt: `scripts/check-gofumpt.sh` via pinned `go tool gofumpt`
  (v0.9.1), zero tolerance. First measurement undercounted (24) through a
  `tee | head` SIGPIPE truncation; true backlog is ~185 files. 22 files
  reformatted (format-only, linux build passes); no mass reformat of the rest
  without an explicit decision — the hook ratchets staged files instead.
- [ ] staticcheck: `scripts/check-staticcheck.sh` (pinned v0.8.1, must analyze
  GOOS=linux or installer files drop out; the script builds the tool for the
  host first — passing GOOS=linux to `go tool` builds an unexecutable binary).
  Whole repo: 48 findings, gate exits 1.

## staticcheck backlog (48) — fix order

- [ ] Bulk mechanical (one commit): 21 × ST1005 capitalized error strings,
  12 × ST1013 numeric HTTP codes (use `http.Status*` constants).
- [ ] Judged separately: 8 × SA1019 deprecated APIs (`runtime.GOROOT`,
  `tar.TypeRegA`, own `AdminTokenFile` markers in tests).
- [ ] Removal candidates (each matches an L1.12 unreferenced def — two
  independent instruments agree): `companionName`
  (`internal/host/tailnet_companion.go:32`), `writeNewJSON`
  (`internal/nativequalification/inputs.go:117`), `buildCapture`
  (`tools/soda-host-image/legacy.go:11`).
- [ ] Dead store (real, benign): `install_linux.go:236` (`err = errRestart`
  is clobbered by line 197's `:=` before any read; retry works via loop
  fall-through, the flag misleads). One-line removal.
- [ ] Not a bug (staticcheck false positive, close-read): the `break` at
  `console_linux.go:248` exits the switch onto line 259's loop break, and the
  `openEditor` flag correctly skips the next ask (lines 194–195, 202) into the
  `case "edit"` nmtui path. Needs a `//lint:ignore SA4011,S1023` with this
  reason when the gate approaches green, not a restructure.
- [ ] Test-file nits (lowest priority): S1007 regexp raw string
  (`terminal_native_test.go:32`), ST1013 in `client_test.go:25`.

## Open verification items

- [ ] Grep git history for credential shapes (L1.14 covered the working tree
  only: git-tracked files, 44 unreadable/oversized excluded).
- [ ] Decide the runtime run: full exec (long native tests) or scoped to fast
  packages for L1.19/L1.20.
- [ ] Optional Layer 2 full inspection on the weakest dimensions (4.10 CI/CD
  Absent at kill-check level; 4.11 Partial; 4.16 weak; 4.15 strong).
