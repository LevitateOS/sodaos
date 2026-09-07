# Complete Soda frontend: required integration work

## Decision and current limits

The user requires **every Forgejo-backed workflow to stay in Soda's interface**.
Native Forgejo pages, iframe/re-skinned HTML, external fallback links and silently
omitted features are not acceptable, even temporarily. This includes developer,
administrator and authentication/security workflows. Forgejo still owns identity,
password verification, MFA, permissions, repositories and business rules. No new
Soda password authority, copied permission inventory or unrestricted proxy follows
from this decision. Separate operator Cockpit and ordinary Git/SSH/package
protocols remain selected.

Removing links is not implementing their replacements. The current OAuth login,
first-password change and consent still visit Forgejo; legacy HTMX and the current
browser origin still expose it. These are **open integration/cutover blockers**,
not exceptions to the requirement. Do not disable working authentication or alter
live origins/listeners before a replacement is implemented and verified.

## Newer upstream investigation

Public metadata from
`https://codeberg.org/api/v1/repos/forgejo/forgejo/releases?limit=20` returned
**v16.0.3** as the newest stable release (published 2026-08-20), alongside the
selected v15.0.7. Also inspected the development branch at immutable commit
`bdc33af0c11568873c336137d404fc327ce0a40e`.

Research, not dependency resolution or an upgrade: public source/metadata copies
are retained under `.artifacts/research/soda-only-forgejo/`. Source was fetched
through the repository contents API with explicit tag/commit refs. No release,
image lock, appliance configuration or installed bytes changed.

| Inspected surface | v16.0.3 and development snapshot findings |
| --- | --- |
| `routers/api/v1/api.go` and `templates/swagger/v1_json.tmpl` | No blame route/schema found. Comparison remains `/repos/{owner}/{repo}/compare/{basehead}`. Diff routes cover a single commit or existing PR; `/diffpatch` is a write operation, not an aggregate read API. |
| `routers/api/v1/repo/compare.go` | Still appends `apiCommit.Files` from every commit to the result. The `files` option controls inclusion, not aggregate-patch generation. It cannot supply a net changed-file/diff view. |
| `routers/api/v1/repo/commits.go` | Download still calls `git.GetRawDiff` for a single SHA. |
| v16.0.3 `modules/git/diff.go` | `GetRawDiff` supplies an empty start commit; `GetRepoRawDiffForFile` first resolves the end as one commit. Passing a revision range to that endpoint does not turn it into a supported aggregate diff. |
| v16.0.3 `routers/web/repo/blame.go` | HTML view uses native `git.CreateBlameReader`, handles ignore-revs behavior and groups attribution. This is reusable upstream implementation, not a headless response contract. |
| v16.0.3 `routers/web/repo/compare.go` | `PrepareCompareDiff` uses `gitdiff.GetDiffFull`, before/after SHAs, native limits and explicit direct versus merge-base comparison. The engine exists; its data contract needs exposing without the HTML view. |

Provenance examples (upstream Git blob IDs, not deployment checksums): v16.0.3
API router `ec516e79f160d67c46b9ffdbbbc0e0574c3e356a`, comparison handler
`5ddf158a727b64e9afaa1a0045c3cbf31ce2b931`, schema
`d2347b6311cfe4b6aa139db7ccdf5e61fe6a3222`; development comparison handler
`43ee3286cba5db6d4b4ab98b960c1b57938639ed`.

**Conclusion:** upgrading to the newest release inspected does not provide the
two missing U09 data interfaces. No upgrade is recommended merely to solve them.
This is not a claim about all possible future releases or all Forgejo interfaces.
The v16 authentication implementation has changed (`services/auth/method/oauth2.go`
and `auth_result_oauth.go`); do not transplant the v15 route-gate explanation as
proof of v16 behavior. A wider upgrade/authentication audit remains necessary.

## Architecture revision plan

The [first-class headless integration revision plan](forgejo-architecture-revision-plan.md)
turns this research into a proposed ownership/contract/build/acceptance model,
including authentication and the existing U milestone assignments. It is not
approval to implement a particular patch or change the running appliance.

## Proposed direction — not yet selected or implemented

Prefer small **Forgejo-owned API additions over its existing engines**, with Soda
rendering their results. Seek upstream acceptance; if immediate delivery requires
a maintained downstream patch, obtain approval for that concrete patch/build
responsibility first. Do not create a Soda repository clone/index or parse Forgejo
HTML to avoid that decision.

### U09: two bounded read contracts

- **Blame data:** authenticated repository-code read, stable repository and full
  commit SHA, validated file path and bounded line ranges. Return attribution
  spans, original paths/line numbers, native commit identities and explicit
  ignore-revs/binary/encoding/size states. Reuse native blame behavior and expose
  real truncation; pagination must not imply unlimited native computation.
- **Aggregate comparison:** authenticated reads for both native targets, resolved
  base/head/merge-base SHAs and explicit direct versus merge-base semantics.
  Return net changed files, rename/binary status and bounded diff hunks, with
  continuation/omission semantics. Reuse Forgejo's comparison/diff engine and
  permission checks. Do not synthesize a temporary PR or concatenate commit
  patches to imitate an aggregate diff.
- Keep handlers inside Forgejo's normal API authentication/scope/repository
  middleware; narrow Soda adapters return bounded DTOs using acting-user grants.
  No operator credential, host helper or direct provider filesystem/database
  access is needed. Endpoint names/DTO details remain proposal work, not invented
  existing upstream calls.
- Tests must cover private/read-denied repositories, stale/moving refs, empty and
  renamed/deleted/binary files, malformed paths, huge history/output, cancellation,
  ignore-revs, divergent histories, limits and native Git readback. Then build
  Soda line attribution and aggregate-diff navigation entirely in React.

### Maintenance and delivery cost

A downstream patch means maintaining a Forgejo source/build input instead of
only consuming its stock image: exact upstream revision plus reviewed patch,
licenses/source availability, native x86_64/aarch64 builds, packaging identity,
upstream security-update rebases and regression tests. Coordinate that with the
core build/config owners; no independent release/update platform is proposed.
Upstreaming reduces this ongoing cost but does not guarantee acceptance or a
release date. No contribution, issue, image publication or patched build has
been performed by this investigation.

These two read contracts do **not** solve the complete frontend on their own.
Authentication/security may need substantially more upstream work; its scope and
risk must not be hidden inside a “small two-endpoint patch.”

## Complete-frontend coverage tasks

| Area | Required work / owner |
| --- | --- |
| Blame and aggregate diffs | Above contracts plus Soda views and native proof; U09/U17. |
| Login, initial password change, MFA/passkeys/recovery and consent | Audit headless authentication/challenge capabilities, keeping Forgejo as verifier and enforcing all native checks, replay binding, rate limits and CSRF/session protections. Propose a complete design before replacing current OAuth redirects; U04/U05/U16/U17. No password-store or cookie-borrowing workaround. |
| Applications/tokens, email/security and site/auth settings | Map every action to verified native authority and interface, then implement required Soda forms; U05/U16/U17. |
| Actions jobs/steps/logs/artifacts and run controls | Human-authorized endpoints and bounded data/streaming; never runner impersonation. U14/U17. |
| Hooks, reviews and releases | Finish missing secret/event edits, review details and asset/empty-field semantics instead of provider page links; U11/U12/U15/U17. |
| Boards, graphs, wiki/packages and remaining administration | Complete the inventory at action level; missing API coverage remains required integration, not a waiver. U10/U13/U15/U16/U17. |
| Navigation and deployment closure | React explicit provider-page links removed in this source follow-up. Still audit provider-returned URLs, Markdown links, legacy pages, authentication redirects and direct browser ingress. Route owned objects to Soda and deliver files through authorized protocols. Before U18/U20, prove browsers cannot reach Forgejo's frontend while API/Git/package transports and Cockpit remain correctly separated. |

The source-only unavailable messages are truthful development states, **not
implemented substitutes or acceptable release completion**. U17 cannot close an
item by recording “native fallback”; U18/U20 cannot claim a complete frontend
while a required interaction still exposes Forgejo.
