# Forgejo API coverage audit

**Mandatory closure rule:** every Forgejo workflow stays in Soda's interface,
including authentication/security and administration. Native frontend fallbacks
are rejected, not a disposition available for U17 acceptance. See the
[newer-version investigation and proposed integration work](forgejo-frontend-integration.md).
Missing APIs remain implementation requirements; no patch or upgrade is selected
by recording a gap.

**U01 in progress.** Schema presence is source evidence, not proven scope/authority or installed behavior. No new live provider operations have run. The [core plan](dashboard-implementation-plan.md) owns the implementation; [current JSON contracts](dashboard-api.md) distinguish shipped source handlers from this inventory.

## Evidence

Inspected the saved schema `.artifacts/downloads/forgejo-swagger.json`, reporting **15.0.7+gitea-1.22.0**, SHA-256 `79bf16a6a3a3df4bbe6d8607ff7bdc482f71f10c1ec3e7ae8d2c68cb819f900c`. Its declared license is MIT for interoperability. Inspected existing provider/auth/store source at baseline `b7241b2`. The original operator OAuth journey established basic sign-in only; it does not prove the new operations below.

The Swagger security definitions include header tokens, BasicAuth and Sudo mechanisms, but do **not** specify each route's OAuth scope or complete middleware authority. Do not infer admin authority from their presence. Soda must not use Sudo impersonation, borrowed cookies or operator-token fallback. The first-workflow OAuth/middleware audit below now supplies source evidence; broader endpoint-specific authorization and native verification remain pending.

## Selected-version OAuth and first-workflow source audit

Additional public source inspected at upstream tag **v15.0.7** (not live API
execution):

- [`routers/web/auth/oauth.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/auth/oauth.go), SHA-256 `d094082c8699f88ffb949b4b72c8be5fcfc5b3c53a76d772bb42f4c3a4cf699e`: code/refresh exchange, bearer token type, seconds-based expiry, client-secret checks, optional refresh-counter invalidation, omitted response scope, and confidential-client reuse of existing consent.
- [`routers/web/web.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/web/web.go), SHA-256 `d4eb17ebf75822a26d958845058e8f4cfd6e74754cf17a5815768112cc9e357e`: supported `/login/oauth/introspect` POST registration. Introspection uses client Basic authentication and returns actual scope, active status, subject and audience; Soda verifies these rather than decoding an unverified JWT.
- [`routers/api/v1/api.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/api.go), SHA-256 `f748544314a2329168071e6538113c74ae4b3eb0db1bce59d816993619dc74f1`: method-derived read/write scope levels, user/settings/key, repository/content and administrator route middleware. Native roles are checked separately from OAuth scopes.
- [`routers/api/v1/user/repo.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/routers/api/v1/user/repo.go) and [`models/repo/repo_list.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/models/repo/repo_list.go): `ListMyRepos` sets actor/owner/private visibility and SearchRepository's default collaboration condition includes accessible collaborator repositories. This differs from the old owner-only picker. Link/total headers drive the new typed pagination metadata without assuming a page-size cap or following provider URLs.

The implementation retains the small existing standard-library OAuth transport:
its current concrete requirements are explicit code/refresh forms and the native
introspection extension, so no additional OAuth dependency was selected. This is
not a claim of general OAuth/OIDC compatibility. Native settings and browser
consent/refresh behavior remain unverified.

Connected source, now locally Go/UI/type/build tested, covers per-session encrypted grants,
actual-scope introspection, refresh/logout handling, native profile/settings/Git
keys/People creation, repository discovery/create/content and Soda environment
create/join/inspection/public-host-key connection views. See the actual
[API contracts](dashboard-api.md) and [credential migration](dashboard-credentials.md).
The inventory below is the original family allocation, not completion evidence;
all later collaboration/admin families still require their detailed action audit.

## Feature map

Paths below are upstream paths under `/api/v1`, **not** registered Soda proxy routes. Methods were inspected in the saved schema. Every row still requires typed request/result/error/pagination review and native authority/scope verification before implementation; no row is marked runtime-verified.

| Feature | Selected schema surface | Acting authority to verify | Core owner / current disposition |
| --- | --- | --- | --- |
| Current identity | `GET /user` (`userGetCurrent`) | Acting user's grant | U04; existing legacy identification, grant retention pending |
| Account settings | `GET/PATCH /user/settings` | Acting user | U05/U16; native fields include full_name, description, location, pronouns, theme; not a Soda mirror |
| Git public keys | `GET/POST /user/keys`; key-specific native removal | Acting user | U05; separate from implemented Soda development-key API |
| User repository inventory/create | `GET/POST /user/repos` | Acting user; visibility/collaborators to verify | U06; never call it with operator authority for a user's inventory |
| Repository detail/settings | `GET/PATCH /repos/{owner}/{repo}` | Native repository permission | U06/U12 |
| File read/create/update/delete | `GET/POST/PUT/DELETE /repos/{owner}/{repo}/contents/{filepath}` | Native ref/write/protection rules | U06/U09; GET ref query; writes have distinct option DTOs |
| History/refs/compare/import/fork | Native repository commit/branch/tag/compare and migration/fork families | Native read/write/import rights | U09; sub-action payload and permission audit pending |
| Issues/comments/assets | `/repos/{owner}/{repo}/issues`, `/{index}`, `/{index}/comments`, `/{index}/assets` | Native issue/read/write rights | U10; no local issue records |
| Labels/milestones | Native repository labels/milestones families | Native repository/issue rights | U10 |
| Pulls/reviews/merge | `/repos/{owner}/{repo}/pulls`, `/{index}/reviews`, `/{index}/merge` | Native review/check/protection/merge rules | U11; no Go merge engine |
| Repository access/hooks/deploy keys/protection | Native repository scoped families | Native administrator/owner permission | U12 |
| Organizations/teams | `/orgs/{org}/teams`, `/teams/{id}`, member/repository subresources | Native organization/team roles | U12; does not approve org-owned environments |
| Search/notifications/activity | `/repos/issues/search`, `/notifications`, `/notifications/threads/{id}` and native search/feed families | Acting user's visibility | U13; not a local search index |
| Actions runs/tasks | `GET /repos/{owner}/{repo}/actions/runs`, `/{run_id}`, `/actions/tasks` | Actual human-readable native authority | U14; logs/artifacts/cancel/rerun/dispatch details not established by this listing |
| Actions secrets/variables | Repository/organization Actions secrets and variables families | Native scoped automation permission | U14; never retain a second CI secret store |
| Releases/assets | `/repos/{owner}/{repo}/releases`, `/{id}`, `/{id}/assets` | Native repository/release permission | U15 |
| Wiki | `/repos/{owner}/{repo}/wiki/new`, `/wiki/page/{pageName}`, `/wiki/pages`, `/wiki/revisions/{pageName}` | Native wiki permission | U15; audit revision/conflict semantics |
| Packages | `/packages/{owner}`, `/{type}/{name}/{version}`, version files | Native package visibility/ownership | U15; publishing remains native protocol unless mapped explicitly |
| Administrator People/create/edit | `GET/POST /admin/users`, `PATCH /admin/users/{username}` | Acting Forgejo administrator, not copied Soda operator role | U05/U16; existing local People is not general upstream inventory |
| Instance organizations/unadopted repos | `/admin/orgs`, `/admin/unadopted` and native admin repository operations | Actual Forgejo administrator | U16; destructive/adoption operations require explicit scope |
| Instance hooks | `/admin/hooks`, `/admin/hooks/{id}` | Actual Forgejo administrator | U16 |
| Provider runners/jobs | `/admin/actions/runners`, runner ID/jobs/registration-token subresources | Actual Forgejo administrator | U16; local services stay P11/Cockpit |
| Native quotas/maintenance | `/admin/quota/groups`, `/admin/quota/rules`, `/admin/cron` | Actual Forgejo administrator | U16; not Soda environment quotas or permission to run live maintenance |
| Account security, auth sources/site configuration, boards, advanced graphs and missing Actions interactions | Complete custom coverage not established | Native supported mechanism/reauthentication required | U16/U17; required Soda integration, no native frontend fallback |

Confirmed option distinctions: `CreateRepoOption` includes name/description/private/initialization/default branch/template/trust settings; `CreateUserOption` includes username/email/password/native onboarding options; `EditUserOption` includes sensitive admin/active/prohibit_login/permission fields; `CreateKeyOption` contains key/title/read_only. Do not reuse an administrator-edit payload for ordinary profile updates, or impose Linux join naming restrictions on upstream account creation.

## U09 completion audit and source corrections

Inspected upstream **v15.0.7** `routers/api/v1/repo/{compare,commits,fork,migrate}.go`,
`routers/web/repo/blame.go`, `modules/git/{repo_compare,diff}.go` and
`services/repository/fork.go`, plus the pinned API/web/OAuth middleware already
retained. Public source copies are under
`.artifacts/research/u09-forgejo-15.0.7/`; `compare.go` was retrieved through the
supported repository-contents API at `ref=v15.0.7` after its raw URL returned 404.
This is upstream research, not a native operation or test pass.

| Operation | Verified contract / correction |
| --- | --- |
| History | Native `/commits` accepts `sha`, optional `path`, page/limit and exposes Link/total metadata. Existing history adapter uses that metadata, not a guessed next page. |
| Resolve selected ref | Native `/git/commits/{sha}` explicitly accepts a Git ref or SHA and supports disabling stat/verification/files. New comparison resolves both refs there, validates full returned SHAs and compares those immutable identities. Slash-ref encoding still needs installed verification. |
| Compare | Native `/compare/{basehead}` resolves refs through `parseCompareInfo`, including native code/pull-read permission checks. `GetCompareInfo` returns its commit list without pagination; REST `total_commits` is its length. `files` is concatenated per-commit affected-file lists, **not an aggregate net diff**. Soda preserves duplicates, labels this honestly and rejects inconsistent/null bounded results. There is no invented native page parameter. |
| Commit diff | `/git/commits/{sha}.diff` passes a single commit to `GetRawDiff`; `GetRepoRawDiffForFile` first resolves that commit and ordinarily diffs its first parent. A `base...head` string is not a supported substitute for a net aggregate patch. Keep 1 MiB/inert UTF-8 limits. |
| Branch/tag reads and writes | Separate GET read scope from POST write scope; upstream still owns visibility, protection and ref syntax. Empty/malformed native creation results cannot be confirmed success. |
| Fork | Native POST `/forks` returns 202 after synchronous `ForkRepositoryAndUpdates`/bare clone. This is not a task API. Personal ownership is implicit when organization is omitted. Soda verifies positive ID, acting owner, valid identity and native fork/non-mirror flags before its 201. Failed/ambiguous responses are never replayed. |
| Import | Native POST `/repos/migrate` parses remote credentials and calls its migration allowlist before creating the destination; quota, migration settings and native ownership remain enforced. Migration runs synchronously (HammerContext); success returns a repository after completion. Native deferred error handling attempts deletion of its own failed destination—Soda does not take over cleanup or assume a timeout canceled work. UI clears credential fields, retains non-secret source/name and never retries automatically. |

**U17-BLAME / U17-COMPARE-DIFF — backend integration required:** the inspected API router
has no blame data endpoint, and its comparison endpoint has no aggregate patch.
Blame is native HTML at `/blame/commit/{sha}/{path}`; aggregate comparison is a
native web view. OAuth2.Verify admits API/specific raw/archive/attachment paths,
not these ordinary web views in the inspected v15 implementation. The user has
rejected native frontend fallbacks; the links added in `1d74a08` are removed.
No scraping, cookie borrowing or Soda-side Git implementation was added. Newer
v16.0.3 and a pinned development snapshot still lack these two data contracts;
see the linked investigation. Required Soda views remain incomplete. Propose
concrete Forgejo API additions and their maintenance cost before implementing a
backend patch; do not ask again to waive coverage through a native link.

Focused Go/DOM tests are authored for pinned/invalid/denied comparisons, read-only
ref grants, copy identity validation, sanitized import failure, Soda-only navigation,
file/ref drafts and stale route responses. Local Go/TypeScript checks, 31 dashboard
tests and dashboard builds passed in the Soda-only follow-up (see handoff).
Installed U09 proof remains pending. Existing U08 native acceptance is unchanged.

## U09/U10 source follow-up

The same pinned router and schema now back explicit history/ref/file-write/fork/
basic-import and issue/comment/label/milestone/reaction/subscription/issue-asset
adapters. `routers/api/v1/repo/file.go` confirms stale file SHA/commit mismatches
produce 409 and protected writes produce 403; no Soda-side Git engine or unsafe
retry is added. The initial diff view is bounded native unified text, not an
inline-review position model. Blame and full import/diff coverage remain pending.

Router lines 1087–1199 put issues/labels/milestones under the distinct `issue`
scope with native repository/author/writer gates. Inspected matching
`routers/api/v1/repo/issue_subscription.go`: `/subscriptions/check` returns typed
WatchInfo (including subscribed/ignored), not a guessed 404-as-unsubscribed
contract. The fixed self-subscription adapter derives the username from the
session. Native issue upload uses multipart `attachment`; downloads currently
leave for the configured native `/attachments/{uuid}` route, confirmed in
`routers/web/web.go`, with no forwarded Soda grant or borrowed cookie.

The schema supplies `/issue_templates` and structured field metadata. Its faithful
custom rendering/validation and comment-asset/reaction detail remain unfinished
source work, **not** a claimed upstream gap. Local tests have since passed; installed provider journeys remain unexecuted.

U11 follow-up inspected pinned `routers/api/v1/repo/pull_review.go`,
`modules/structs/pull_review.go` and `routers/api/v1/repo/pull.go` against Swagger.
Create review maps `new_position`/`old_position` to positive/negative file line
numbers, not unified-diff offsets, and accepts explicit `commit_id`. Native review
submission associates pending comments and is not assumed atomic across comment
creation and submission. Initial custom inline input supports only the new side;
old-side mapping, existing threads and team requests remain source work. Native
merge passes `head_commit_id` through to merge execution and returns conflict for
stale head; force/auto-merge/delete-branch remain fixed false in Soda. This is
source research and authored tests, not a verified native review/merge journey.

U12 hook audit inspected pinned `routers/api/v1/utils/hook.go`,
`services/webhook/general.go`, `services/webhook/deliver.go`,
`modules/setting/webhook.go` and `modules/webhook/type.go`. Hook conversion returns
a decrypted `authorization_header` and a potentially credential-bearing URL:
Soda's explicit DTO omits both. Native PATCH unconditionally sets authorization,
events and filter; it neither assigns `config.secret` nor updates package/action
flags (unlike creation). The bounded adapter preserves omitted values and rejects
these unsupported edits. Exact native UI dependency/acceptance remains an U17
item; no fake rotation control is exposed. Empty native allowed-host configuration
selects external hosts at delivery, with TLS verification on by default; inspected
appliance source does not override these settings. No hook target was contacted
and no live hook was created, activated or tested.

Repository protection follow-up inspected pinned `routers/api/v1/repo/branch.go`
and `tag.go`: native pointer/slice PATCH fields retain omitted values; nested push
allowlist flags are only applied under explicit enabled parents. Organization/team
follow-up inspected `routers/api/v1/org/org.go`, `org/team.go`, `models/unit/unit.go`
and the router at 1223–1310. Organization scope is separate from repository scope;
profile PATCH unconditionally updates several strings, while team all-repository
changes require explicit permission and native administrator teams have fixed
admin units. Explicit adapters preserve omitted fields, reject otherwise-ignored
inputs and delegate all actual membership/repository authority. No native
organization/team or protection operation has been executed by this work.

## U14 Actions capability and gap audit

Pinned `routers/api/v1/repo/action.go` verifies repository run ownership on detail,
Actions reader/writer gates in `api.go`, explicit dispatch ref/string inputs and
native `return_run_info`. Run/task lists have body totals but no Link headers;
Soda reads `/settings/api`'s public cap for exact requested pagination. Native
`modules/actions/workflows.go` chooses the first existing standard directory and
recursively lists YAML files; the adapter retains that priority and delegates
recursive tree paging without parsing or scheduling workflows.

Repository/organization secret/variable routes remain native-admin gated and
separately scoped. Secrets return name/creation metadata only. Native variable
creation uses POST, replacement PUT. Connected source and focused local tests
cover actor/route binding, large IDs, native cap, denial without retry, secret
redaction and explicit replacement fields. No native mutation was executed.

**U17-ACTIONS-WEB — backend integration required:** `routers/web/web.go` registers run
jobs, logs, artifacts, cancel and rerun under native web routes. Inspected
[`services/auth/method/oauth2.go`](https://codeberg.org/forgejo/forgejo/src/tag/v15.0.7/services/auth/method/oauth2.go):
`OAuth2.Verify` accepts only API, attachment, OAuth userinfo/introspection,
raw/attach and archive paths; ordinary Actions web routes are excluded. Therefore
these web handlers cannot be called with the retained human OAuth grant. No
runner-token protocol, native password or borrowed session is substituted.
The former native run-page fallback is rejected and removed from React source.
Required work: verify or provide human-scoped supported job/step/log/artifact
and run-control interfaces with native repository gates, then full Soda views.
Any newer-version candidate needs its own authorization audit.
No upstream issue has been filed or patch selected. U17/U18 requires integration
and verified Soda-only workflows, not native-page access or scope omission.

## Authority register

- Forgejo: identity/password/MFA, account fields, repositories, organizations/teams, collaboration, Git keys, CI and upstream administration.
- Soda session: stable provider identity link plus separate Soda-only preferences/development keys/environment associations. Schema-v3 grants are encrypted and bound to individual sessions; legacy sessions require reauthentication.
- Configured Soda operator: narrow extension/operator association, not arbitrary Forgejo admin or host root.
- Human repository owner: initial environment administration, not host privileges; explicit join remains required.
- Project membership: join-time Linux/public-key access, separate from native Git permission; no implied later offboarding/key synchronization.
- Host root/Cockpit/local runners: native operator scope. Outside tools do not acquire authority by calling the core API.

## Dependency research and unresolved work

Public npm package metadata inspected for `react-router-dom` **7.18.3**: MIT, Node >=20, React/ReactDOM >=18; selected in the new dashboard manifest against the existing Node 24/React 18 baseline. Subsequent authorized dependency resolution produced the real dashboard lockfile and local type/UI/build checks passed; installed compatibility remains pending.

Selected `react-markdown` **10.1.0** and `remark-gfm` **4.0.1** from public npm metadata (both MIT, compatible unified 11 dependencies; react-markdown requires React/types >=18). Inspected react-markdown's matching-tag README for `skipHtml`, default URL sanitation and component overrides. Actual safe GFM/README callers and tests now exist; dependency resolution and local builds have since passed; transitive license completeness and installed verification remain pending. Matching Forgejo `repo/file.go` source confirms the raw-file endpoint/ref query and separates it from LFS-redirecting media; the download adapter does not follow the latter.

`@vitejs/plugin-react` latest metadata reports **6.1.1** with Vite 8/OXC-related peers. It was **not** added blindly: the initial preview uses Vite+'s existing JSX compilation path, as Cockpit does, without claiming Fast Refresh. New plugin/Markdown/diff/OAuth-library selections remain owning-milestone work. Inspected the installed Vite+ 0.3.0 exports to confirm `vite-plus/client` and `vite-plus/test` declaration paths.

Next audit work: exact upstream scope/middleware/refresh and admin consent behavior; bounded endpoint-specific DTO/error/pagination contracts; verify required package peers/transitive licenses and review the resolved dashboard lockfile and retain exact executed evidence. The API families above must not be replaced by an unrestricted proxy to avoid that work.
