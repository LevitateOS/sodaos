# Fast development builds — implementation plan

## Priority and approval

**F1–F3 complete within their scoped checks; active priority has returned to M4.**
The owner requested and approved this side workstream, then explicitly requested
resuming production qualification. Development success does not complete M4. Current bounded execution
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

Implemented flags; candidate-only production, default/fast media readback and a
native diskless fast-media boot are verified within the scope below:

```text
soda-build --development --target candidate [existing worker/arch/output options]
soda-build --development --target media [existing options plus media inputs]
```

- `candidate` runs the shared source-to-verified-host/application path and stops
  before media-only work. It retains all five application images, current payload
  format, shipping programs/assets, and current source/static candidate checks.
- `media` uses that same producer and the existing native packaging path for
  installer development. Defaults match production. `--media-compression fast` selects
  a distinct development host whose upstream EROFS setting uses LZMA level 1 instead
  of level 6; filesystem, fragments and 1 MiB cluster size stay unchanged.
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

**Complete — `bb17280`, focused race/vet checks and native candidate runs passed.**
The vertical change uses the existing Go owner:

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

**Complete — cold worker 8m29s; warm worker 5m30s, both exit 0.** Runs
`dev-candidate-03` and `dev-candidate-04` used the same clean `bb17280` source/controller
through the actual x86_64 CLI/worker. All five apps and all candidate checks remained;
there was no media workspace, Ignition generation, packaging, signing or VM. The
worker configuration omitted media authority entirely. The warm result meets the
historical roughly six-minute reference; it is not a timing guarantee for other hosts.

[History](implementation-history.md#fast-development-candidate-production) owns exact
receipts, cache state and setup refusals. The selected native work did not require an
ARM build, installation, M4 update test or public service.

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

**Complete — `0fd8def`; both native media runs and fast-media diskless boot passed.** Read-only
inspection of the selected Assembler image and actual Soda invocation disproved the
plan's assumption that `live-rootfs-fsoptions` is an external packaging override. Its
live-artifact stage reads that field from `usr/share/coreos-assembler/image.json`
**inside the candidate deployment**; `cosa buildextend-live` exposes no compression
argument. Changing `config/build-args.conf` does not override that read.

The owner explicitly allowed distinct **development-only candidate metadata** for
this comparison. Build default and fast media from the same committed source and
recipes, with compression metadata as the only intentional content difference; bind
each run's distinct host/payload/media identities rather than claiming equal bytes.
Production defaults and qualification are unchanged. Do not fork Assembler.

The selected fast value is `-zlzma,level=1 -Efragments -C1048576 --quiet`, replacing
only level 6 in the reviewed upstream EROFS defaults. The flag is refused outside
`--development --target media`. Native host readback binds `image-config.json` to
the staged metadata; signed packaging inventory and media results record the exact
filesystem/options. `logs/media-events.jsonl` timestamps existing upstream osmet,
rootfs and ISO messages without modifying upstream stages. These are **log-arrival
windows**, not CPU profiles: rootfs includes CPIO/hashing and osmet includes checksum
verification. Retain the original phase timers and report this measurement limit.

The bounded comparison demonstrated a useful saving; the development-only flag is
retained. Both runs used the same committed source, resolved app inputs, exported
tools, resource bounds and fresh packaging workspaces. The host image configuration
differed only in `live-rootfs-fsoptions`; each distinct candidate/media identity is
recorded separately. This is not a byte-reproducibility claim for separate builds.

| Measurement | Default (LZMA 6) | Fast (LZMA 1) |
| --- | ---: | ---: |
| Controller wall time | 21m24s | 18m35s |
| P8 packaging | 14m59s | 12m03s |
| Observed rootfs window | 414.25s | 223.19s |
| Minimal ISO bytes | 160,432,128 | 160,432,128 |
| Rootfs bytes | 1,803,103,744 | 1,854,429,184 |

Fast saved **2m49s overall / 2m56s packaging**, at **51,325,440 additional download
bytes (+2.85%)**. At 100 Mbit/s that difference alone is about 4.1s in an ideal
transfer, not a measured Internet result. Both native packaging/readback paths passed.
The fast ISO then booted disklessly under KVM to the reviewed Soda welcome in 23.22s;
its local rootfs GET transferred exactly 1,854,429,184 bytes in 5.66s. ISO/rootfs hashes
were unchanged, with no installation target, enrollment or disk installation.
The VM and loopback listener stopped. [History](implementation-history.md#fast-development-media-compression)
owns exact receipts and measurement limitations. This is not installed, update,
recovery, production-release or minimum-resource qualification.

A packaging comparison is not a resume/promote facility for a failed release run.
Use existing packaging/test entry points with fresh outputs, not another permanent
producer. Limit the initial comparison to one baseline/alternative pair and stop if
it does not provide a useful saving. More CPUs, broad parallelization, persistent
OSBuild cache integration and application feature switches are not prerequisites.

## Completion and limits

Track F1–F3 here; keep B1–B6 progress in its existing owner. F1/F2 are delivered;
F3 is also complete within its bounded development-media scope. The owner subsequently
explicitly resumed M4; its progress and grants remain in the status owner.
M4 fixtures, real custody and unrelated work stay untouched; experimental retention
is not a reason to add compatibility code. The status/grant owner records any later
approval for specific execution effects. Only the scoped receipts above support
completion; development success is not production or installed qualification.
