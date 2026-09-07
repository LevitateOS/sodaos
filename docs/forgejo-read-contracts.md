# First native read contracts — U09/H03/H04 review

**U01 contract candidate, not implemented or advertised.** Wire and resource
choices below now have a concrete engineering disposition; native implementation,
packaged-runtime checks and conformance remain U02/U03/U09 work. Source input: the
[Forgejo source lock](../appliance/forgejo/source.lock.json). The
[single action register](forgejo-api-coverage.md) remains authoritative for
CO05/CO07 and wider code/PR coverage. Chosen safety ceilings and the narrow
primitive experiment below are not U09 completion or measured product capacity.

## Proposed wire boundary

Use explicit additions under the existing `/api/v1/repos/{owner}/{repo}` native
repository assignment/scope/code-reader gates, not a generic internal-method
proxy. Proposed suffixes are `/soda/blame` and `/soda/compare-diff`; these routes
**do not exist**. Namespace/schema naming needs review with the native patches.
Soda calls the configured provider with the acting user's grant. Preserve native
repository/token restrictions, private visibility and revocation; no Sudo, owner
credential, temporary PR, task-token-only protocol or Soda Git clone.

The revision-1 section below is the single wire definition. Resolve mutable refs
using existing acting-user reads, then bind results to immutable native identities.
Do not reconstruct native Git semantics in Soda; render byte content inertly.

## Shared native implementation and newly found blockers

Exact v16.0.3 source paths inspected:

- `routers/web/repo/blame.go::{performBlame,fillBlameResult}`: extract non-rendering
  blame/ignore-revs result handling into a concrete native owner used by both web
  and API callers. Keep avatar/time/HTML rendering in the web handler.
- `modules/git/blame.go::BlameReader.NextPart`: `bufio.Reader.ReadLine` returns a
  prefix for long lines. The current implementation appends the first content
  fragment, then reads/discards the continuation fragments in its “munch” loop.
  It also does not retain porcelain original line numbers/filename fields.
  **Source findings, not a reproduced native test result:** reusing this output
  directly would not satisfy full long-line/original-attribution API semantics.
  Add native regression tests and a shared parser correction before the endpoint.
- `BlameReader.Close` waits on command completion before closing pipes. A bounded
  consumer stopping early must cancel/drain/close in a proven order; otherwise a
  pipe writer may block completion. `tryCreateBlameIgnoreRevsFile` copies the native
  ignore-revs blob without a separate bound. Native context/timeouts and cleanup
  need exact tests, not an HTTP response cap pretending to cancel Git.
- `routers/web/repo/compare.go::PrepareCompareDiff` and
  `services/gitdiff/gitdiff.go::{DiffOptions,GetDiffFull}`: reuse native base/head/
  merge-base calculation and limited diff production. Do not serialize the full
  web context or duplicate its permission/merge calculations in Soda.

## Resource/error/compatibility review

The initial ceilings below have a policy/primitive-feasibility disposition.
U09 still must measure ordinary, deeply historical, divergent/rename-heavy,
large-ignore-file and cancelled requests against the actual native implementation. The selected source example sets
`UI.MAX_DISPLAY_FILE_SIZE` to 8 MiB; this is a source default, not an observed live
setting or proof that blame memory is bounded by 8 MiB. Response pagination alone
cannot bound native Git computation. A full resource budget and cancellation test
are blockers for exposing these expensive operations.

Use native 401/403/404 semantics for authentication/permission/hidden targets;
400/422 for invalid request data as the reviewed native conventions require;
explicit bounded/unavailable errors for exceeded budgets; and cancellation that
actually stops native work. Preserve safe Soda error translation, not raw Git
stderr, private paths, stack traces or query credentials. These reads issue no
repository mutation or automatic alternate request after uncertainty.

The descriptor contract below includes pre-auth availability for the
[H05 design](forgejo-authentication-design.md). There is no descriptor route or
compatibility client yet; metadata never substitutes for native authorization.

## U01 wire disposition — revision 1

Implement these as explicit native additions, sharing native computation with the
web callers. This schema replaces the earlier illustrative choices; it is not a
claim that any endpoint exists.

**Common rules:** repository assignment and native account/token restrictions,
`read:repository` and code-unit reader checks run before computation/admission.
Soda supplies only the signed-in user's grant. IDs and SHAs are strings; all
counts/line positions are bounded nonnegative integers safe for JavaScript.
Unknown/duplicate query fields or unsupported modes fail with 400; no caller-
selected limits, executable, native configuration or arbitrary repository path.
Byte fields use canonical padded base64; paths decode to native relative Git paths
without NUL, absolute paths or `.`/`..` components, at most 4,096 bytes. Preserve
invalid-UTF-8 paths/content as bytes, not Go JSON replacement characters.

### Blame

`GET R/soda/blame?commit=SHA&path_b64=BASE64&start=1&limit=200&bypass_ignore=false`
(`R` is the assigned `/api/v1/repos/{owner}/{repo}` route.) `commit` is mandatory
and full native-format SHA; path is mandatory. `start` is one-based; `limit` is
1–200 with default 200; `bypass_ignore` is optional, default false. Native revision
resolution/authorization must bind that exact object, not substitute a moving ref.

Response object (all fields required unless explicitly nullable):

- `revision: 1`, `repository_id`, `commit_sha`, `blob_sha`, `path_b64`;
- `total_lines`, `start`, `next_start` (integer or null; null only at actual EOF);
- `ignore_revs`: `{requested: boolean, applied: boolean, faulty: boolean}`;
- `lines`: ordered objects `{number, content_b64, has_newline}`;
- `spans`: ordered objects `{final_start, original_start, line_count, commit_sha,
  path_b64, previous, ignored, unblamable}`, where `previous` is null or
  `{commit_sha, path_b64}` and the last two fields are booleans from native porcelain.

Spans cover the returned lines exactly without gaps/overlap; split at changes in
original path/position or native flags, not just SHA. Empty files have zero lines,
empty arrays and null `next_start`. A start beyond a nonempty file's EOF is 422.
The lines come from the immutable blob bytes (including CR and final-newline state),
not a lossy porcelain display parser. Native commit detail remains the existing
commit endpoint; do not copy provider user lookup/permission rules into this DTO.
Binary content produces typed `binary_file` (422), with the existing Soda raw
content/download flow available, not an invented attribution or native-page link.

### Net comparison

`GET R/soda/compare-diff?base=SHA&head=SHA&mode=direct|merge-base`
requires all three fields. Response: `revision: 1`, `repository_id`, `base_sha`,
`head_sha`, `merge_base_sha` (null for direct mode), `from_sha`, `mode`,
`complete: true`, `files`. `from_sha` is base for direct mode and the native
merge-base otherwise. No common merge-base is `no_merge_base` (422), never an
implicit switch to direct comparison. Equal SHAs yield a verified empty result.

Each file has `old_path_b64`/`new_path_b64` (null for an absent side), `status`
(`added`, `deleted`, `modified`, `renamed`, `copied`, `type_changed`), `binary`,
`additions`, `deletions`, and `hunks`. Hunks have `old_start`, `old_count`,
`new_start`, `new_count`, `heading_b64`, and `lines`. Lines have `kind`
(`context`, `add`, `delete`), `old_line`/`new_line` (null on absent side),
`content_b64` and `has_newline`. Preserve native old/new positions and newline
markers; metadata-only/binary files have empty hunks. Unknown native statuses
are a contract failure, not silently converted to `modified`.

This first complete-result contract has **no fabricated pagination or partial
success**: if native computation is incomplete or any budget is exceeded, return
a typed limit error, not `complete: true`. Broader cross-repository comparisons
still require a reviewed extension with both resources authorized; their absence
keeps the wider U09 workflow incomplete rather than granting cross-fork access.

### Limits, admission and cancellation

Selected conservative initial ceilings, not throughput guarantees:

| Boundary | Limit / reason |
| --- | --- |
| Blame source blob | min(native display setting, 8 MiB); respect the reviewed native default and stricter operator configuration |
| Blame page | 200 lines; output and original-attribution work remain bounded even for very long lines |
| Ignore-revs blob | 64 KiB; validate/limit before copying, with native faulty-file behavior explicitly reported |
| Comparison | 1,000 files / 20,000 diff lines and native stricter diff settings; no success for a natively limited result |
| Complete encoded response | 12 MiB; accommodates base64 of an 8 MiB long line plus bounded metadata, otherwise 413 |
| Native operation | 10-second total wall budget: at most eight seconds across resolution/blobs/ignore fallback/diff, reserving two seconds for termination; not a fresh budget per subprocess |
| Native expensive read admission | One in flight per Forgejo process across these endpoints; no unbounded waiting queue; 429 + `Retry-After: 1` after authorization |
| Git worker | 256 MiB address-space and 8-second CPU ceilings, imposed before execution; memory ceiling is address space, not a claimed RSS measurement |
| Native Git stdout | 32 MiB total across the operation, enforced while streaming, including blob/porcelain/diff metadata before parsing; cancel on excess |
| Native parsed data | Bounded streaming/collections and final encoding; reject before growing beyond the response/line/file/path ceilings; keep at most 4 KiB stderr internally and never return it |

**Selected Linux candidate:** inside Forgejo's native Git command owner, a fixed
launcher sets resource limits and `exec`s the existing typed Git executable/argv.
No user data is interpolated into shell text; no command/config proxy is exposed.
The local experiment used `/bin/sh`'s `ulimit -v`/`-t` and `exec "$@"`. Keep native
argument protection and Git configuration; if packed-object mmap exceeds the
ceiling, fail explicitly, not lift the limit or change hosts. U02 must verify the
packaged Alpine shell/native Git on both architectures before advertising this
contract. No unsupported-platform fallback or new privileged helper is selected.

Request cancellation must reach **all** native child work and release pipes/temp
files before the admission slot is released. Keep the slot until workers finish,
even if the browser disconnects. A hard deadline includes bounded termination;
U09 must demonstrate forced termination for a worker ignoring graceful cancellation,
not merely cancel an HTTP context. Ordinary native web operations keep their own
policy while using the corrected shared parser/computation.

**Executed narrow evidence:** an isolated run-owned local Git repository with 24
synthetic commits on x86_64/Git 2.52.0 passed long-line (65,536-byte content), literal
shell-metacharacter path and net-diff requests under the proposed worker limits.
An oversized Python allocation failed with `MemoryError` under the same launcher;
A follow-up closed the original probe's CPU/worker-identity gaps: a one-second CPU
ceiling killed a spinning worker, and an explicit ready/PID check proved the exec'd
sleeping worker received SIGTERM. Evidence/scripts/outputs remain under
`.artifacts/research/u01-d5b5065/limit-fixture/`, `worker-controls/`,
`limit-probe.log` and `worker-controls.log`. This establishes host primitive
feasibility only—not Forgejo parser correctness, busy admission, worst-case
performance, packaged Alpine behavior or aarch64 proof.
The numeric values are chosen safety ceilings; raise them only through explicit
budget review, not by treating this small sample as capacity measurement.

### Errors and minimal compatibility

Extension errors use `{code, message}` with fixed public codes/messages and no Git
stderr/private paths: native 401/403/404 retain their authority/visibility semantics;
400 `invalid_request`; 422 `binary_file`, `invalid_range` or `no_merge_base`;
413 `limit_exceeded`; 429 `read_capacity_busy`; 503 `read_unavailable` for worker/
timeout/internal failure. No response body claims successful partial results.
Soda maps these without retrying an uncertain operation or replacing credentials.

Select `GET /api/forgejo/v1/soda` for a small pre-auth-readable descriptor from the
configured provider only: `{revision: 1, blame: 0|1, compare_diff: 0|1, auth: 0|1}`.
Each feature stays zero until its actual implementation/tests exist; zero means
not implemented, not permission denied. No roles, subjects, secret/config values,
commands, provider-selected URLs or detailed build inventory. Maximum 4 KiB,
two-second fetch timeout, at most 60 seconds of in-process successful caching;
no persistent inventory or stale-positive fallback after a failed refresh. Missing
404, unknown revision, 401/403 and network/invalid response are distinct states.
An operation's missing/incompatible response invalidates its positive compatibility
state immediately. Unrelated stock/environment operations remain available.

Descriptor reads use no user/operator token and the existing configured transport;
never weaken TLS or derive browser/RP-ID origins from it. An advertised feature
still needs native authorization on every request. A future image build must
identify its actual revisions and run the native negative tests; no descriptor or
compatibility client is added before its producer exists. Existing OAuth remains
unchanged until the reviewed replacement is proven.

## Required implementation evidence

Native tests: private/read/write/restricted-token/revoked/unit-disabled targets,
long lines/original attribution/rename/ignore-revs/binary/invalid encoding, pinned
ref movement, direct versus merge-base/net diff, large inputs and real cancellation/
pipe/temporary-file behavior. Compare results with ordinary native Git in approved
isolated fixtures. No installed provider mutations have been run in this batch.

Adapter/browser tests: exact configured path/token and DTO identities, malformed/
null/truncated/oversized results, no privilege fallback or secret leakage, inert
rendering and accurate limited states, file→blame→commit and base/head navigation,
route/logout/account races. Existing U09 writes/copy/draft guards remain intact.
Only after these contracts/budgets/native fixes are reviewed should H03/H04 code
be added, built and exercised. The source preparer is not that API implementation.
