# Forgejo API coverage audit

**U01 in progress.** Schema presence is source evidence, not proven scope/authority or installed behavior. No new live provider operations have run. The [core plan](dashboard-implementation-plan.md) owns the implementation; [current JSON contracts](dashboard-api.md) distinguish shipped source handlers from this inventory.

## Evidence

Inspected the saved schema `.artifacts/downloads/forgejo-swagger.json`, reporting **15.0.7+gitea-1.22.0**, SHA-256 `79bf16a6a3a3df4bbe6d8607ff7bdc482f71f10c1ec3e7ae8d2c68cb819f900c`. Its declared license is MIT for interoperability. Inspected existing provider/auth/store source at baseline `b7241b2`. The original operator OAuth journey established basic sign-in only; it does not prove the new operations below.

The Swagger security definitions include header tokens, BasicAuth and Sudo mechanisms, but do **not** specify each route's OAuth scope or complete middleware authority. Do not infer admin authority from their presence. Soda must not use Sudo impersonation, borrowed cookies or operator-token fallback. Exact selected-version middleware, OAuth grant/refresh semantics and scopes still require upstream-source audit before U04/broader endpoints.

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

## Authority register

- Forgejo: identity/password/MFA, account fields, repositories, organizations/teams, collaboration, Git keys, CI and upstream administration.
- Soda session: stable provider identity link plus separate Soda-only preferences/development keys/environment associations. Existing session rows carry no user API grant.
- Configured Soda operator: narrow extension/operator association, not arbitrary Forgejo admin or host root.
- Human repository owner: initial environment administration, not host privileges; explicit join remains required.
- Project membership: join-time Linux/public-key access, separate from native Git permission; no implied later offboarding/key synchronization.
- Host root/Cockpit/local runners: native operator scope. Outside tools do not acquire authority by calling the core API.

## Dependency research and unresolved work

Public npm package metadata inspected for `react-router-dom` **7.18.3**: MIT, Node >=20, React/ReactDOM >=18; selected in the new dashboard manifest against the existing Node 24/React 18 baseline. This is metadata compatibility, not a resolved lockfile or tested bundle.

`@vitejs/plugin-react` latest metadata reports **6.1.1** with Vite 8/OXC-related peers. It was **not** added blindly: the initial preview uses Vite+'s existing JSX compilation path, as Cockpit does, without claiming Fast Refresh. New plugin/Markdown/diff/OAuth-library selections remain owning-milestone work. Inspected the installed Vite+ 0.3.0 exports to confirm `vite-plus/client` and `vite-plus/test` declaration paths.

Next audit work: exact upstream scope/middleware/refresh and admin consent behavior; bounded endpoint-specific DTO/error/pagination contracts; verify required package peers/transitive licenses and generate a real dashboard lockfile only in the authorized dependency phase. The API families above must not be replaced by an unrestricted proxy to avoid that work.
