# Mandatory software freshness

**Freshness takes priority over reproducibility. AI provider CLIs MUST stay on
the latest official stable release, or the owner's explicitly selected channel.
Known stale provider CLIs must not start new work.** This rule applies to Codex,
Muse and every additional provider, across normal terminals, Factory, broker
enrollment, hosts, new and retained Projects, development and qualification.
It supersedes older plans that hold dependencies at obsolete versions.

This document owns the freshness rule; [AGENTS.md](../../AGENTS.md#mandatory-freshness)
makes it mandatory for coding agents. The
[CLI updater direction](../reference/project-os.md#automatic-cli-updates-proposed)
owns the proposed mechanism. A documented requirement is not proof of deployed
automation. Never claim that an installation is current without an actual check.

## Keep every managed input current

Apply the same priority to all Soda-managed libraries, lockfiles, build tools,
language runtimes and toolchains, OS packages, base/container images, GitHub
Actions, browser assets, SDKs and external release inputs, including Fountain
and Canlang. Follow official current stable channels unless the owner selects
another channel. Adopt major releases too; repair actual integration failures
instead of indefinitely ignoring them. Preserve unrelated project work and data.

AI CLIs need automatic release discovery and activation independently of OS
rebuilds: consume upstream release events where available, poll metadata at least
every five minutes while online, and refresh before new work when that check is
due. A detected release must be fetched and checked promptly, without a waiting
period or a paid model call. Check actual required flags and protocols; do not
substitute version-number compatibility guesses. Keep the chosen model/profile
and resource limits unchanged by a CLI update.

Other dependencies need event-driven updates where available and frequent
automatic checks otherwise. The repository Dependabot configuration requests
hourly checks for its covered ecosystems, with no release cooldown. Custom
checksum descriptors, Containerfile inputs, independently published SDKs and
installed-system updates still need their own upstream resolvers/activation
paths; Dependabot does not cover every Soda input. A bot proposal alone does not
make a dependency updated: apply it, run the affected checks and ship it through
the owning component's update path. Do not rebuild the complete OS to update an
independent CLI or rerun unrelated expensive checks.

Discovery intervals bound detection latency; they cannot guarantee simultaneous
installation with an upstream publication. Show last successful check, upstream
selection, installed/active selection and failures. Network or upstream failures
are unresolved freshness, not evidence that an old release is current. Known
stale or unverified inputs must block claims of a fresh baseline/qualification;
known stale provider binaries must block new provider work until corrected.

## Create a fresh baseline

1. Resolve official upstream release metadata immediately before selecting inputs
   for a new build, baseline or qualification. Refresh stale locks, checksums,
   source/SDK selections and base images first. Reuse cached bytes only when they
   match that current selection.
2. Verify the selected artifact's integrity and actual required behavior. If the
   current release fails, report and fix that precise incompatibility. Do not
   silently restore an older pin to obtain a green result.
3. Record the resolved release identities, digests and check time, then hold those
   exact bytes for that running attempt. This record explains what ran; it is not
   a mandate to reuse those versions in the next attempt.
4. Recheck freshness before qualification and promotion. If an upstream input has
   advanced, update the affected component and its evidence before claiming a
   current baseline. Preserve prior artifacts/results as historical evidence;
   never rewrite an immutable release or relabel older proof as current.

Historical replay may use the original inputs when explicitly selected as replay,
but it must remain labelled historical and cannot stand in for current testing.
Existing sessions retain their captured executable until completion so an update
does not corrupt active work. New sessions select the newly current release.
Retain only current, previous and still-referenced CLI bundles; reference cleanup
must preserve credentials, accounts, budgets, data and required failed evidence.

## Implementation status

The automatic installed Codex/Muse updater is implemented in source (run
UP-20261026-v1): shared retained store, upstream discovery, verified
admission, status/due-check, active references and reference-aware cleanup in
`lib/provider-tools`; broker and host staging consumers; `cmd/soda-provider-update`
operator command; installed `system/host/services/soda-provider-update.service`
and `.timer` units; project launch resolution and baseline freshness. The frozen
shared contract lives in [provider-freshness-contract.md](provider-freshness-contract.md).
Repository dependency automation is configured locally and takes effect only
after publication to the repository's default branch.

Proven locally in this run: unchanged-refresh reuse, atomic failure behavior,
process retention, credential preservation, stale/unknown launch and baseline
refusal, installed-path Project access, and live restage with legacy pins
ignored. Accepted exact host bundle lifetime (`b8e578fa`, independently
accepted): a staged execution retains its exact bundle reference from staging
through reservation and the live run, so refresh/cleanup cannot collect it;
the reference releases only after confirmed retirement and is preserved when
retirement is uncertain; retired unreferenced bundles become collectable
while the current selection stays untouched. Accepted external results:
staged bundle survives two updates and GC while held, retired bundle
reclaimed, current bundle preserved.

Reproduce from the repository root (no credentials; hermetic lifetime tests,
no model calls):

- `cargo check -p soda-host --jobs 4`
- `cargo test -p soda-host --jobs 4 --test muse_serve_oracle`
- `bash scripts/check-no-unit-tests.sh`

Observed on `b8e578fa`: check pass; oracle 27 passed, 0 failed (including
2 lifetime tests); no-unit-tests gate pass.

Remaining and unverified: installed/appliance activation (no production
install was mutated or restarted here), Factory new-selection adoption and
persisted cold-recovery integration (T06, foreign writer unreleased), nested
live restage (no podman-capable guest image), and the full
`prepare_execution` resolve leg, which needs a rootful host — production runs
`User=root` with `UsernsMode=private`, while rootless scratch reports
`UsernsMode=""` and is denied there by design. Do not present this policy
change as an update of all dependencies or VMs, and never claim an
installation is current without an actual check.
