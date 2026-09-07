# First native read contracts — U09/H03/H04 review

**Draft, not implemented or advertised.** Development source candidate: the
[Forgejo source lock](../appliance/forgejo/source.lock.json). The
[single action register](forgejo-api-coverage.md) remains authoritative for
CO05/CO07 and wider code/PR coverage. This draft does not complete U09 or select
native computation limits without measurement.

## Proposed wire boundary

Use explicit additions under the existing `/api/v1/repos/{owner}/{repo}` native
repository assignment/scope/code-reader gates, not a generic internal-method
proxy. Proposed suffixes are `/soda/blame` and `/soda/compare-diff`; these routes
**do not exist**. Namespace/schema naming needs review with the native patches.
Soda calls the configured provider with the acting user's grant. Preserve native
repository/token restrictions, private visibility and revocation; no Sudo, owner
credential, temporary PR, task-token-only protocol or Soda Git clone.

| Request | Native data contract |
| --- | --- |
| `GET .../soda/blame?commit=FULL_SHA&path=PATH&bypass_ignore=false` | Echo validated immutable repository/commit/path identity; return ordered attribution spans, original commit/path/line positions, display lines and previous identities where native semantics provide them; explicit ignore-revs/faulty-ignore flags and content/limit status |
| `GET .../soda/compare-diff?base=FULL_SHA&head=FULL_SHA&mode=direct` | Echo immutable base/head identities, explicit mode and merge-base when used; net added/deleted/renamed/type/binary files and ordered hunks/old-new ranges, native limits/truncation and pagination semantics |

Specify fields before coding the DTOs:

- IDs remain lossless strings. Full SHAs must match the repository's native object
  format; no mixed mutable ref results. Resolve branch/tag selections using the
  existing acting-user ref reads before this call, then verify returned identity.
- Blame spans need destination start/count, original start/path/commit and previous
  commit/path when available. Contiguous destination lines with the same SHA are
  **not necessarily** contiguous original lines or the same original path.
  Preserve native ignore-revs fallback semantics and report a faulty ignore file,
  rather than silently ignoring the request.
- Compare files need old/new path, type, binary status, additions/deletions and
  bounded sections; sections need exact old/new starts/counts and ordered line
  kind/content. Native binary/rename/deletion status is not an invented textual
  patch. Direct mode compares base to head; merge-base mode compares the native
  merge-base to head. Never label concatenated per-commit files as net changes.
- Choose a byte-preserving representation or explicit invalid-encoding status;
  Go JSON replacement of invalid UTF-8 is not faithful source transport. Long
  lines must not be silently shortened. Untrusted content remains inert in React.
- An output limit is explicit; no partial list marked complete, guessed zero
  changes or fictional next page. If native computation returns only a limited
  view, record that limitation rather than recomputing Git in Soda.
- Same-repository comparisons are the first implementation case. Native web
  comparisons also support other repository/fork contexts; those need explicit
  base/head access and object binding before fetching/comparing in Forgejo.
  They remain required coverage, not silently excluded by these query examples.

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

Measure ordinary, long-line, deeply historical, divergent/rename-heavy, large
ignore-file and cancelled requests before selecting concrete file/work/wall-time/
output/concurrency limits. The selected source example sets
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

H03's proposed small read-only descriptor should report only implemented contract
revisions, initially `revision`, `blame` and `compare_diff`. A feature is advertised
only after its real handler and tests exist. There is **no descriptor route or
compatibility client yet**, rather than a client for an invented/unimplemented API.
Each request still authorizes natively; unknown/missing/incompatible metadata,
401/403 and network failure remain distinct. No persisted capability inventory,
indefinite cache or stock/patched runtime selector. Pre-authentication compatibility
is reviewed separately with the [H05 design](forgejo-authentication-design.md).

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
