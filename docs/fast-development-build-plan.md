# Fast development builds — implementation plan

## Priority and approval

**Active priority: shorten the development feedback loop. M4 is paused, not complete.**
The owner explicitly requested this side workstream before further qualification
work and has now requested implementation of this plan. Current bounded execution
grants and retained state remain in
[implementation status](implementation-status.md#current-permissions).

This guide owns the development-build targets and their implementation order.
The [release contract](release-engineering-plan.md#single-run-release-build-contract)
still owns production qualification. This is a separate workstream, **not another
producer, branch/worktree or replacement B1–B6 roadmap**.

## Why this comes first

The recorded warm B3 run took **21m34s** before its media-observer failure:

| Work | Time |
| --- | ---: |
| P8 native media assembly | 15m15s |
| P3 programs/assets/tests | 4m27s |
| Other phases | 1m52s |

Source: `.artifacts/releases/b3-media-33ea3f5-20260914T135430Z/logs/timing.log`.
The native live-artifact stage inside P8 took **601.78 seconds**; its individual
packing/compression costs were not separately timed. Cached program compilation
was about two seconds and application image production about 22 seconds.

**First payoff: do not spend fifteen minutes producing installation media when the
change needs only host/application artifacts.** Approximately six minutes is the
historical warm candidate reference, not a guarantee for the new isolated worker or
cold caches. Do not run another full media build merely to reproduce these timings.

## Selected first interface

Implemented flags (native timing validation pending):

```text
soda-build --development --target candidate [existing worker/arch/output options]
soda-build --development --target media [existing options plus media inputs]
```

- `candidate` runs the shared source-to-verified-host/application path and stops
  before media-only work. It retains all five application images, current payload
  format, shipping programs/assets, and current source/static candidate checks.
- `media` uses that same producer and the existing native packaging path for
  installer development. Initially its compression/settings match production.
- Neither target installs, runs qualification VMs, publishes or admits final release
  signing. Candidate production can still run its existing native inspection
  containers; it must not launch the media helper VM.
- Existing source/UI commands (`go test`, Bun typecheck and prepared test suites)
  remain the quickest leaf checks. Do not add a second `check` orchestrator now.
- No compile-time feature matrix, component omission, generic `--skip-tests` or
  arbitrary resume flag in the first increment. Disabling Tailnet/Runners is not
  the demonstrated fifteen-minute saving. Existing production-supported inactive
  service configurations remain available without recompiling features out.

An explicit development target returns **0 when that requested target succeeds**,
not an error merely because production qualification was intentionally not requested.
Failures/cancellation remain nonzero. Extend the existing run result to record
purpose, requested/completed target, artifact identities and checks actually run;
use an unambiguous `development-only; not release-qualified` summary. A candidate
result has no media path. Do not create a second artifact format or stamp different
feature content into the host just to label development output.

Require both `--development` and an explicit supported target; reject target flags
on a production request, media-only options on `candidate`, and development requests
for publication/final signing. Default production behavior remains fail-closed: no
development target silently shortens it, and its incomplete B4/B5 result remains incomplete. Development receipts
cannot stand in for protected release evidence. Final production artifacts still
must be qualified and delivered unchanged; no post-test production rebuild.

## Implementation order

### F1 — Deliver candidate-only development production

**Implemented; focused Go checks passed.** The vertical change uses the existing Go owner:

1. Add explicit purpose/target parsing and validation in `tools/soda-build` and
   `internal/hostimage.Request`; reject conflicting or unknown combinations early.
2. Make `internal/hostimage.Build` stop at the verified candidate boundary. Gate
   media-only prerequisites too: rootfs URL, fixture authority, Assembler/tooling
   admission/preparation, live Ignition and P7/P8. Keep any input or program actually
   needed by the host candidate; do not replace it with a placeholder.
3. Carry target and scoped results through `tools/soda-build/worker_linux.go`.
   Candidate mode must neither require nor mount media signing inputs, and parent
   result validation must not require `media.json`. Keep the existing isolated
   build identity and protected custody boundary; no direct privileged fallback.
4. Reuse current manifests, recipes, cache locations, source freezing, progress and
   cancellation. Keep fresh outputs under the existing admitted output parent;
   do not introduce another output hierarchy or cache manager for this feature.
5. Update CLI help and [native support](native-support.md#local-host-content-image-candidate)
   with the actual implemented commands and result meanings.

**Exit:** focused existing Go tests demonstrate candidate completion without media
inputs/calls, failed checks preventing completion, correct worker/result handling,
and unchanged production refusal/completion semantics. Use current test drivers,
not a new scenario framework. Clean committed source/controller identity remains
required for artifact builds; dirty-tree packaging is not part of this increment.

### F2 — Prove the time saving and make it usable

**In progress; native run pending.** Under the recorded exact build/helper scope, run one
necessary warm x86_64 development-candidate build through the real CLI and worker.
Do not require an unrelated ARM build, native install, M4 update test or public service.

Retain the existing timing/result logs and report wall time, cache state, emitted
host/all-five image identities and checks. Confirm normal target success, no media
artifact/packaging invocation and no signing-custody requirement. Use existing
cancellation coverage plus a bounded native check if the dispatch changes require it.
Preserve failed receipts and fix the demonstrated path, not a larger imagined system.

**Exit:** a working documented command and an honest measured candidate-only receipt.
The primary acceptance is removal of unnecessary media work, not an invented speed
promise. If it remains materially slower than the roughly six-minute warm reference,
identify the measured cause before adding optimizations. F1/F2 do not depend on F3.

### F3 — Optional faster installer-development media

**Pending F2; selected for the bounded installer-iteration comparison.**
Add narrow timing around the existing native packing, EROFS and ISO stages. Benchmark
one supported lower-compression setting against the current LZMA level 6 using the
same admitted candidate content and unchanged CPU/memory bounds. Prefer upstream
`live-rootfs-fsoptions`; no Assembler fork or custom disk/ISO implementation.

If worthwhile, expose a development-only `--media-compression fast` setting while
keeping the production default unchanged. Record the exact setting and resulting
media hashes. Measure packaging time **and** ISO/rootfs size, download cost and
relevant native readback/boot behavior. Faster packaging can mean a larger download;
compression can also change installation-media identity and cannot reuse another
media image's qualification receipt.

A packaging comparison is not a resume/promote facility for a failed release run.
Use existing packaging/test entry points with fresh outputs, not another permanent
producer. Limit the initial comparison to one baseline/alternative pair and stop if
it does not provide a useful saving. More CPUs, broad parallelization, persistent
OSBuild cache integration and application feature switches are not prerequisites.

## Completion and limits

Track F1–F3 here; keep B1–B6 progress in its existing owner. Deliver F1/F2 before
expanding this side path. Do not silently resume M4 while this is the active priority.
M4 fixtures, real custody and unrelated work stay untouched; experimental retention
is not a reason to add compatibility code. The status/grant owner records any later
approval for specific execution effects. Source implementation is now underway;
completion claims require the scoped receipts above.
