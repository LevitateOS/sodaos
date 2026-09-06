# Forgejo API coverage audit

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

Connected **unbuilt/untested** source now covers per-session encrypted grants,
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
| Account security, auth sources/site configuration, boards, advanced graphs and missing Actions interactions | Complete custom coverage not established | Native supported mechanism/reauthentication required | U16/U17; native fallback while researched, not silently completed parity |

Confirmed option distinctions: `CreateRepoOption` includes name/description/private/initialization/default branch/template/trust settings; `CreateUserOption` includes username/email/password/native onboarding options; `EditUserOption` includes sensitive admin/active/prohibit_login/permission fields; `CreateKeyOption` contains key/title/read_only. Do not reuse an administrator-edit payload for ordinary profile updates, or impose Linux join naming restrictions on upstream account creation.

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
source work, **not** a claimed upstream gap. All new tests remain unexecuted.

## Authority register

- Forgejo: identity/password/MFA, account fields, repositories, organizations/teams, collaboration, Git keys, CI and upstream administration.
- Soda session: stable provider identity link plus separate Soda-only preferences/development keys/environment associations. Existing session rows carry no user API grant.
- Configured Soda operator: narrow extension/operator association, not arbitrary Forgejo admin or host root.
- Human repository owner: initial environment administration, not host privileges; explicit join remains required.
- Project membership: join-time Linux/public-key access, separate from native Git permission; no implied later offboarding/key synchronization.
- Host root/Cockpit/local runners: native operator scope. Outside tools do not acquire authority by calling the core API.

## Dependency research and unresolved work

Public npm package metadata inspected for `react-router-dom` **7.18.3**: MIT, Node >=20, React/ReactDOM >=18; selected in the new dashboard manifest against the existing Node 24/React 18 baseline. This is metadata compatibility, not a resolved lockfile or tested bundle.

Selected `react-markdown` **10.1.0** and `remark-gfm` **4.0.1** from public npm metadata (both MIT, compatible unified 11 dependencies; react-markdown requires React/types >=18). Inspected react-markdown's matching-tag README for `skipHtml`, default URL sanitation and component overrides. Actual safe GFM/README callers and tests now exist; dependency resolution/transitive license review/build remain unexecuted. Matching Forgejo `repo/file.go` source confirms the raw-file endpoint/ref query and separates it from LFS-redirecting media; the download adapter does not follow the latter.

`@vitejs/plugin-react` latest metadata reports **6.1.1** with Vite 8/OXC-related peers. It was **not** added blindly: the initial preview uses Vite+'s existing JSX compilation path, as Cockpit does, without claiming Fast Refresh. New plugin/Markdown/diff/OAuth-library selections remain owning-milestone work. Inspected the installed Vite+ 0.3.0 exports to confirm `vite-plus/client` and `vite-plus/test` declaration paths.

Next audit work: exact upstream scope/middleware/refresh and admin consent behavior; bounded endpoint-specific DTO/error/pagination contracts; verify required package peers/transitive licenses and generate a real dashboard lockfile only in the authorized dependency phase. The API families above must not be replaced by an unrestricted proxy to avoid that work.
