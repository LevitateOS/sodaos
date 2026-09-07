# Forgejo workflow and interface audit

**H01 source audit, Soda baseline `0f43b9f`.** This is the single action register
for the [headless implementation plan](forgejo-headless-implementation-plan.md),
owned by the existing U01–U20 milestones. It supersedes the earlier family-only
inventory in this file. It is **not** installed validation, acceptance of U09/U17,
or approval of a Forgejo patch, version upgrade or authentication protocol.

Every Forgejo-backed developer, administrator, authentication and security
workflow must stay in Soda. Missing interfaces are integration work, not permission
to link to Forgejo pages, embed HTML screens, scrape, borrow cookies, impersonate
users, use a site token or maintain another forge/password/permission database.
Ordinary Git, SSH, package protocols and operator Cockpit remain distinct selected
boundaries. Unavailable placeholders do not complete a workflow.

## Findings and decisions needed

1. **This is substantially larger than two missing Git endpoints.** Login and
   security, boards, review state, advanced activity/search, portions of Actions,
   wiki history, package settings and site administration need native interfaces
   or extensions to existing ones. Many native handlers combine decisions with
   rendering; exposing their entire context as JSON would leak HTML, secrets or
   authority assumptions.
2. **Do not duplicate the substantial usable API.** Repository/file/ref writes,
   issue metadata/assets/timers/dependencies, reviews and merge commands,
   organizations/teams/protection, releases, package inventory and quotas already
   have interfaces. Some missing controls are only Soda work. Advanced protection
   and team-policy fields even have existing Soda adapters.
3. **Authentication is the first architectural gate.** An OAuth access token cannot
   complete the native first-password-change or mandatory-MFA-enrollment gates.
   Native API middleware explicitly refuses those accounts. Basic authentication
   is not a general replacement: it is disabled for users with WebAuthn keys and
   otherwise checks native password/TOTP policy. Account security cannot be solved
   with a password-to-token shortcut.
4. **16.0.3 changes the Actions assessment.** Its human repository APIs add jobs,
   logs, artifacts and cancellation. They do not add blame, net comparison diff,
   a complete headless login, rerun, workflow enable/disable or full step/attempt
   presentation. Consider an appropriate supported 16-series baseline before
   deciding to backport all those Actions interfaces to 15. No upgrade is selected.
5. **Permissions need more precision than “admin API.”** Repository Actions
   configuration/runner routes in 15.0.7 use `reqOwner(TypeActions)`, while their
   native settings pages use repository-admin gates. That is an authority parity
   gap for admin collaborators, not permission for Soda to elevate them. Repository deletion uses the *owner's user/organization scope*, not
   repository scope. PAT listing and PAT creation/deletion have different auth
   requirements. Package scope is not in Soda's current default consent.
6. **Current Soda-only navigation is not closed.** Issue attachments still expose
   `native_url`; account onboarding still directs people to a native password
   change; OAuth, legacy HTMX, content links and direct Forgejo ingress remain.
   The TSX `forgejo_url` source guard does not detect all of these.
7. **Source configuration is not live configuration.** The appliance pins 15.0.7
   but does not select every registration, mail, federation, quota or authentication
   option. Conditional rows below remain required when enabled; disabling features
   is not an approved way to reduce the requested frontend.
8. Before H02/H03/H05 implementation, review a concrete baseline, native contract
   scope, authentication sequence/threat model, update ownership and lifecycle
   decisions for linked identities/resources. This audit does not establish that
   one small patch or a version bump completes headless feasibility.

## Evidence and audit boundary

Full public source archives, tag references, schema operation indexes and research
responses are retained in `.artifacts/research/h01-0f43b9f/`. They were fetched for
source inspection only; no upstream code, build or test was executed.

| Inspected source | Exact commit | Retrieved archive SHA-256 |
| --- | --- | --- |
| [v15.0.7](https://codeberg.org/forgejo/forgejo/src/commit/d4de9eb2a87c26b402fdd0259e079957f8cd2b4b) — audit baseline | `d4de9eb2a87c26b402fdd0259e079957f8cd2b4b` | `10af63276050502e24c7abceb66ce1f4e63329ccaa40135b6bfe266b2387e6d6` |
| [v16.0.3](https://codeberg.org/forgejo/forgejo/src/commit/eccddb2d17c93b42b2c8995725e03e549ac9ec0c) — API delta and targeted handler/auth comparison | `eccddb2d17c93b42b2c8995725e03e549ac9ec0c` | `abcc5811fece4dcbd8088e7b015533e887c8a5e66ad126eac8d6e468f6831693` |

These are research provenance, **not** deployment checksums or a new dependency
lock. Installed Forgejo remains the previously recorded 15.0.7 image; its bytes
were not re-observed. The saved installed Swagger reports
`15.0.7+gitea-1.22.0`, SHA-256
`79bf16a6a3a3df4bbe6d8607ff7bdc482f71f10c1ec3e7ae8d2c68cb819f900c`.
The complete v15 archive's API router, web router and OAuth handler hashes match
the earlier inspected files (`f7485443…`, `d4eb17eb…`, `d094082c…`).

The schema indexes contain **314 paths / 491 operations** in v15 and **326 paths /
506 operations** in v16. These counts are mechanical discovery evidence, not a
count of frontend workflows or proven capabilities. The pass used both the full
API and web router, handler/DTO/native-service traces, alternate mounted protocols,
configuration source and Soda callers. Absence from Swagger alone is not a gap.
V16 is a targeted comparison, not a complete second-version qualification.

### Source notation

All paths below are relative to the pinned upstream tree, unless prefixed `Soda`:

- **A** = `routers/api/v1/`; **W** = `routers/web/`; **D** = `modules/structs/`.
- **R** = `/api/v1/repos/{owner}/{repo}`; **I** = `R/issues/{index}`;
  **P** = `R/pulls/{index}`. Other API paths are under `/api/v1` unless stated.
- Every A route also inherits `routers/api/shared/middleware.go`,
  `A/api.go` and its `A/permissions/` gates. Every W route inherits `W/web.go`
  and its web session, request-protection and context-assignment gates.
- `services/`, `models/`, `modules/git/` and `modules/indexer/` are **Forgejo's**
  implementation owners. Naming them is an extraction/reuse reference, never
  permission for Soda to access their storage or run their internal commands.

### Classification and source status

| Code | Meaning |
| --- | --- |
| **1** | Existing native API is sufficient for the stated action. Ordinary typed Soda clients/screens may still be missing; no native addition established. |
| **2** | An existing interface needs a bounded Soda adapter: protocol/byte handling, native option/result translation or a multi-step composition. Not a new Forgejo business rule. |
| **3** | Native interface/semantics/authority is missing for the stated outcome. A reviewed Forgejo addition or extension is required; not necessarily a new route. |
| **4** | The required Soda backend fields already exist; only the specified custom UI control is missing. This narrower claim needs a concrete Soda caller. |

`S` means connected Soda **source subset**, `M` means missing Soda coverage,
`D` means a product/security/lifecycle decision remains. None means runtime PASS.
Rows group only closely related commands with the same evidence; split read versus
mutation or available versus missing semantics where the boundary differs.
A conditional native configuration gate is not a fifth “waived” classification.

### Common authority and test requirements

Scopes below are native categories: `user`, `repository`, `issue`, `organization`,
`notification`, `package`, `admin`, `activitypub`. `A/api.go::tokenRequiresScopes`
selects read for GET/HEAD and write for other methods. Scope is necessary, not a
native role. Native repository/unit/author/team checks and token restrictions
still apply. Route nesting can require more than one category. Keep the exact
selected router authoritative rather than copying a role table into Soda.

`routers/api/shared/middleware.go::verifyAuthWithOptions` refuses inactive,
prohibited, must-change-password and required-but-missing-MFA accounts. This is
not a missing-scope response. `reqOwner`, `reqAdmin`, `reqSiteAdmin`, organization
ownership and the Soda operator are not synonyms. Native site-admin exceptions
in those predicates stay native; Soda must neither invent nor extend them.

Every row requires **N** native contract/permission tests, **A** Soda adapter tests
and **B** browser/DOM tests before acceptance. The per-section tests below add
specific cases to this common matrix:

- N: anonymous/private denial, least read versus write scope, wrong user/resource,
  disabled unit/configuration, revoked/expired/inactive actor, native role changes,
  normal result and native validation/conflict. Read success must not imply write
  authority. Cross-resource IDs must fail before disclosure or mutation.
- A: typed bounded complete responses; invalid/null/trailing/oversize data; large
  IDs as strings; real pagination; sanitized errors; no browser-selected origin,
  Sudo, operator-token fallback, storage access or uncertain write replay; CSRF,
  actor/session binding, cancellation and declared old/new contract behavior.
- B: the entire workflow and object navigation stay in Soda; loading/empty/denied/
  expired/partial/error states; forms preserve same-target drafts but not secrets
  or other users' drafts; stale route/account responses cannot win; keyboard,
  focus and accessible validation. Direct-ingress/network assertions supplement
  DOM checks, not just string scanning.

## Action register

### Authentication, consent and account security

**Owners U04/U05/U16; H05 starts before broad UI expansion.** Native owners are
`services/auth/method/`, `models/auth/`, `services/user/` and
`modules/auth/webauthn/`. Tests additionally cover login CSRF/session fixation,
challenge binding/replay/expiry, attempt limits, enumeration, required enrollment,
secret redaction, credential origin and revocation boundaries.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| AU01 | Enter sign-in; show allowed methods and native failures; verify local credentials | W `auth/auth.go::SignInPost` → `UserSignIn` and `handleSignInFull`; `/user/login` is web/session-oriented, not a headless login protocol. | 3 / D; current OAuth redirects expose Forgejo |
| AU02 | Complete first-login/forced password change before an ordinary session/grant | W `auth/password.go::MustChangePasswordPost`; `/user/change_password` and settings variant. Ordinary API middleware refuses `MustChangePassword`; admin PATCH clearing it is not an acting-user replacement. | 3 / M |
| AU03 | TOTP second factor; choose recovery/scratch alternative | W `auth/2fa.go`, `/user/two_factor` and `/scratch`; pre-authentication session state and native verification. Password API authentication is not this browser challenge sequence. | 3 / M |
| AU04 | WebAuthn assertion and successful native sign-in | W `auth/webauthn.go::WebAuthnLoginAssertion{,Post}` returns JSON but requires `twofaUid`/`webauthnAssertion` in a native session, verifies credential/counter and returns a redirect. No reusable human bearer/pre-auth challenge contract. | 3 / D |
| AU05 | Forgotten-password request, recovery-code validation, new password and required MFA | W `auth/password.go::{ForgotPasswdPost,ResetPasswdPost,commonResetPassword}`; native email/token/TOTP/account-link/session decisions must remain shared. | 3 / M; mail-dependent |
| AU06 | Register when permitted, activation/resend and verify email links | W `auth/auth.go::{SignUpPost,Activate,ActivatePost,ActivateEmail}`; registration/captcha/email/manual-confirmation settings. Admin user creation does not implement self-registration or activation. | 3 / M; conditional, not waived |
| AU07 | External OAuth/OpenID sign-in, callback and account linking | W `auth/{oauth,openid,linkaccount}.go` and `/user/oauth2/{provider}`; native source configuration/linking/MFA gates. An external identity provider may itself require browser interaction. | 3 / D; resolve origin/redirect requirement explicitly |
| AU08 | Display requested scopes, approve/deny consent, repeat or expand consent | W `auth/oauth.go::{AuthorizeOAuth,GrantApplicationOAuth}` at `/login/oauth/{authorize,grant}`; native grant/client/redirect/PKCE decisions mixed with login/HTML. Confidential-client reuse of prior consent is not a re-consent API. | 3 / M |
| AU09 | Exchange code; introspect actual subject/audience/scopes; refresh grant | Supported `/login/oauth/{access_token,introspect}`, W `auth/oauth.go`; confidential-client authentication, code/refresh types and optional refresh-counter invalidation. | 2 / S; encrypted per-session grant/serialized refresh exist |
| AU10 | End Soda session; define native session/grant logout separately | Soda `internal/web/auth.go` and encrypted grant deletion handle Soda logout. W `auth/auth.go::{HandleSignOut,SignOut}` clears native browser state; no general OAuth end-session/revocation API found in selected router. | 2 for Soda logout, 3 for native logout/grant closure / S+D |
| AU11 | View and revoke applications authorized by the user | W `user/setting/applications.go` loads grants; `oauth2_common.go::RevokeGrant` uses `models/auth.RevokeOAuth2Grant`. `/user/applications/oauth2` manages **owned clients**, not this authorized-grant list. | 3 / M |
| AU12 | List/create/read/edit/delete personally owned OAuth clients | `GET/POST /user/applications/oauth2`, `GET/PATCH/DELETE /{id}`; user scope and acting owner; A `user/app.go`, native OAuth app model. Redirect URI/confidential-client fields and one-time create secret require redacted DTOs. | 2 / M |
| AU13 | Rotate an existing OAuth client secret or revoke its grants without deleting the client | W `user/setting/oauth2_common.go::{RegenerateSecret,RevokeGrant}`; not provided by owned-client CRUD above. Delete/recreate changes identity and is not rotation. | 3 / M |
| AU14 | List personal API-token metadata/scopes/repository restrictions | `GET /users/{username}/tokens`; user scope + `reqSelfOrAdmin`, A `user/app.go::ListAccessTokens`. **Listing is not behind `reqBasicOrRevProxyAuth`.** | 1 / M |
| AU15 | Create/delete/regenerate a personal API token, showing secret once | POST/DELETE token routes add `reqBasicOrRevProxyAuth` + `reqToken`. Native Basic rejects WebAuthn-enrolled users and verifies TOTP otherwise. W `user/setting/access_token.go` supports regeneration, not equivalent to arbitrary create/delete. | 3 / D for complete grant-based/security-key workflow; existing restricted APIs are not absent |
| AU16 | Change current password with native old-password/security checks | W `user/setting/account.go::AccountPost` → native user update, password validation and session handling. No self password PATCH; admin edit has different authority. | 3 / M |
| AU17 | List/add/delete own email addresses | `GET/POST/DELETE /user/emails`; A `user/email.go` → `services/user`, user scope. API add does not perform W `EmailPost`'s confirmation-mail sequence. | 1 for CRUD / M; verification is AU18 |
| AU18 | Make verified email primary; resend/complete verification; email notification preferences | W `user/setting/account.go::EmailPost`, W `auth/auth.go::ActivateEmail`, native mail/user services. These actions are not represented by email CRUD. | 3 / M |
| AU19 | Read security/enrollment status; enroll/re-enroll/disable TOTP; regenerate scratch codes | W `user/setting/security/{security,2fa}.go`; native password/challenge and required-MFA policy. Do not issue ordinary grants before enrollment completes or expose stored seeds. | 3 / M |
| AU20 | Register/list/delete WebAuthn security keys | W `user/setting/security/webauthn.go`, `/user/settings/security/webauthn/*`; JSON registration is session-bound, not a complete token API. | 3 / D |
| AU21 | List/unlink external logins; add/delete/change visibility of OpenID identity | W `user/setting/security/{security,openid}.go`; native account-link ownership and external verification. Not API user profile fields. | 3 / M; conditional source configuration |
| AU22 | Delete own Forgejo account with native confirmation/dependency checks | W `user/setting/account.go::DeleteAccount` → native deletion service. Site-admin DELETE is not self-service authority. Soda associations/project accounts need a separate lifecycle decision; no automatic deprovisioning. | 3 / D |

**Authentication-specific findings:** `modules/auth/webauthn/webauthn.go::Init`
sets RP ID from `setting.Domain` and allowed origin from `setting.AppURL`. Changing
only Soda's UI cannot make an assertion for a different origin valid. The inspected
login is an existing-user two-factor flow, not proof of discoverable/passwordless
passkey login. Preserve enrolled credentials; review origin/RP-ID compatibility
and any explicitly required passwordless addition rather than promise migration.
Mail-generated recovery/activation links also use native origins; route rewriting
must not accidentally change Git/package advertisement. No source-only audit can
prove an unselected new challenge protocol secure. H05 must specify the native
attempt/rate/captcha policy, session/grant issuance and invalidation and credential
handling; existing protections cannot simply be removed to gain JSON access.

### Identity, preferences, profiles and discovery

**Owners U05/U06/U13.** Native user/privacy models remain authoritative. Tests add
hidden/private/limited profiles, owner versus viewer fields, key separation,
pagination, duplicate keys and identity changes with retained Soda associations.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| AC01 | Read acting identity and ordinary profile/preferences; edit supported fields | `GET /user`, `GET/PATCH /user/settings`; A `user/{user,settings}.go`, D `user.go::UserSettingsOptions`: full name, website, description, location, pronouns, language, theme, diff style, hints, hide-email/pronouns/activity. User scope and self. | 1 / S; current form/DTO is a subset |
| AC02 | Set self username/profile visibility; hide timeline comment categories | W `user/setting/profile.go::{ProfilePost,UpdateUserHiddenComments}`. These are not `UserSettingsOptions` fields. Admin rename/visibility is a different action; username changes also need association policy. | 3 / D for rename, M for missing native preferences |
| AC03 | Upload/reset native user avatar; display avatars safely | `POST/DELETE /user/avatar`, A `user/avatar.go`; user scope, base64 image validation. Public avatar byte routes exist separately. | 2 / M; bounded image adapter, not page navigation |
| AC04 | List/create/read/delete personal Git SSH keys | `/user/keys` and `/{id}`, A `user/key.go` → native key service; user scope, actual owner. Never install these automatically as development-access keys. | 1 / S |
| AC05 | List/create/delete GPG keys; get challenge and verify signed proof | `/user/gpg_keys`, `/{id}`, `/user/gpg_key_token`, `/user/gpg_key_verify`; A `user/gpg_key.go`, native verification. Do not request a private key. | 2 / M |
| AC06 | Public profile, repositories, followers/following, stars, watched repositories; follow/unfollow | `/users/{name}` and native subresources; `/user/following/{name}` PUT/DELETE; A `user/{user,repo,follower,star,watch}.go`. Native privacy; repository inventory also needs repository scope. | 1 / S for profile/repositories, M for social controls |
| AC07 | Block/unblock/list blocked users | `/user/{list_blocked,block/{name},unblock/{name}}`; A `user/user.go::{ListBlockedUsers,BlockUser,UnblockUser}`; user scope/self. Native blocking affects collaboration, not Linux membership. | 1 / M |
| AC08 | User contribution heatmap and activity | `/users/{name}/{heatmap,activities/feeds}`; A `user/user.go::{GetUserHeatmapData,ListUserActivityFeeds}`, enabled heatmap/privacy/visible repository rules. | 1 / S activity, M heatmap |
| AC09 | Discover/create repositories, initialization choices and generate from template | `/user/repos`, `/repos/search`, `POST R/generate`, `/gitignore/templates`, `/licenses`; A `user/repo.go`, `repo/repo.go::{Create,Generate}`, native repository generation/quota service. Explicit destination/owner choices, no environment creation. | 2 / S basic discovery/create, M full template/options |
| AC10 | Native about/settings/help and safe native-tool guidance | `/version`, `/settings/{ui,api,attachment,repository}`, signing-key routes; misc/settings handlers. Public reported settings are not full site configuration or proof of extension compatibility. | 1 / S partial; U02/U17 metadata also required |
| AC11 | Prove ownership of a Git SSH signing key with a native challenge/signature | W `user/setting/keys.go::KeysPost` (`verify_ssh`) → `models/asymkey.VerifySSHKey`, native time-window verification token and user-feature gate. SSH key CRUD has no matching proof command. Ordinary SSH access does not automatically complete this signing-key verification workflow. | 3 / M; public signatures only, no private key |
| AC12 | Select native avatar source/Gravatar email where enabled | W `user/setting/profile.go::UpdateAvatarSetting`; API avatar upload/delete is not source-selection/Gravatar-address configuration. Preserve disabled/federated-avatar policy. | 3 / M |
| AC13 | Own storage/quota usage and attachment/package/artifact breakdown | `/user/quota` and its resource subpaths; A `user/quota.go`. W `shared/storage_overview.go` also refuses when quotas are disabled, so this is not a reason to add unconditional usage or enable quotas. | 1 / M |

### Repository code, history, writes and copy workflows

**Owners U06/U09.** Tests add branch names containing slashes, ref movement,
SHA-1/SHA-256, empty/binary/invalid-encoding/large files, symlinks/submodules/LFS,
renames/deletes, protected branches, cancellation, native Git verification and
ambiguous write/copy outcomes. Native Git is an ordinary protocol, not an excuse
to omit an explicitly required supported browser action.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| CO01 | Repository overview, permissions, refs, README/tree/path/file | `GET R`, `/contents/{path}?ref=`, `/git/trees/{sha}`, `/git/blobs/{sha}`; A `repo/{repo,file,tree,blob}.go`. Any-repo-reader for metadata, code read for contents, repository scope and native repository assignment. | 2 / S; full path/type/limit matrix unfinished |
| CO02 | Raw file, archive/bundle download; inspect LFS data rather than mistake pointer for content | `R/raw/*`, `/archive/*`, `/{tarball,zipball,bundle}/*`; LFS batch/object routes under `/{owner}/{repo}.git/info/lfs`. A `repo/{file,download}.go`, W `repo/githttp.go`, `services/lfs/`. `/media/*` can redirect to LFS. | 2 / S raw subset; no arbitrary redirect or credential forwarding |
| CO03 | File/path and ref commit history with native paging | `R/commits?sha=&path=&page=&limit=`; A `repo/commits.go::GetAllCommits`; code-read gate, native Git history, Link/total metadata. | 1 / S |
| CO04 | Commit metadata, verification, changed files, single-commit patch/diff | `R/git/commits/{sha}` and `.{diff,patch}`; A `repo/commits.go`, `modules/git.GetRawDiff`. API accepts ref/SHA; Soda validates pinned returned identity. Single-commit diff is not a range API. | 2 / S; verification/full viewing matrix unfinished |
| CO05 | Blame attribution spans/lines/original paths; ignore-revs handling | W `repo/blame.go::{RefBlame,performBlame}` → `git.CreateBlameReader`; no blame API in either inspected release. Code read, native file/work limits. Ordinary web route is not OAuth-accessible. | 3 / M; U17-BLAME retained |
| CO06 | Compare commits with resolved immutable base/head | `GET R/compare/{basehead}`; A `repo/compare.go`, native compare resolution and Git. Returns whole commit list, `total_commits` = its length; `files` concatenates per-commit entries. | 2 / S; duplicates retained, not a net file list |
| CO07 | Net comparison files/hunks, rename/binary/deletion and direct versus merge-base view | W `repo/compare.go::PrepareCompareDiff` → `gitdiff.GetDiffFull`, `modules/git/repo_compare.go`. No aggregate patch API; single-commit diff, write-only `/diffpatch` and an existing PR are not substitutes. | 3 / M; U17-COMPARE-DIFF retained |
| CO08 | List/read/create branches and tags | `R/branches`, `/tags`, individual ref routes; A `repo/{branch,tag}.go`. Code read versus writer/token/non-archived/quota checks; native protection. | 1 / S |
| CO09 | Rename/delete/restore branches and delete tags | PATCH/DELETE branch and DELETE tag routes cover rename/deletion. A branch list hardcodes `IsDeletedBranch=false`; W `repo/branch.go::RestoreBranchPost` resolves a repository-bound deleted-branch record before native push/update. No API deleted-history/restore workflow; creating from a caller-guessed SHA does not supply that record. Preserve protected/default-branch rules. | 1 rename/delete + 3 deleted-history/restore / M |
| CO10 | Create/edit/delete/upload/rename files, select target branch and commit metadata | POST/PUT/DELETE `R/contents/{path}`, A `repo/file.go` → `services/repository/files/`; `reqRepoBranchWriter` includes native maintainer-write-to-branch behavior, non-archived/quota/protection gates. SHA/last-commit checks return conflict. | 2 / S basic text/edit/upload; complete metadata/rename/delete UI unfinished |
| CO11 | Multi-file commit or apply an explicitly supplied patch | `POST R/contents`, `POST R/diffpatch`; D file options, same branch/quota/protection owner. Input/output bounds and one native transaction, not one independent commit per UI file. | 2 / M |
| CO12 | Fork to an authorized personal/organization destination and inspect result | `GET/POST R/forks`, A `repo/fork.go` → `services/repository.ForkRepositoryAndUpdates`. Disabled-forks and native destination/quota rules. 202 follows synchronous clone, not task creation. | 2 / S personal fork, M remaining choices |
| CO13 | Import/migrate supported source data, credentials/options and destination | `POST /repos/migrate`, A `repo/migrate.go` → native migration allowlist/quota/ownership and `services/migrations/`. Synchronous; error cleanup is native; timeout is not proof of cancellation. | 2 / S basic HTTPS import, M source-specific choices/data |
| CO14 | Observe native web migration task progress/retry where offered | W `user/task.go::TaskStatus`, W `repo/migrate.go` and settings migration actions; web task IDs/session and service-owned task state. Synchronous REST import does not expose this progress contract. | 3 / M; no Soda durable job/cleanup engine |
| CO15 | Inspect ahead/behind and synchronize existing fork branch | GET/POST `R/sync_fork` and `/{branch}`; A `repo/sync_fork.go` → native fork sync; read versus writer/non-archived. Not a new Soda merge policy. | 1 / M |
| CO16 | Read/set/remove commit notes | GET/POST/DELETE `R/git/notes/{sha}`; A `repo/notes.go`, native Git notes, repository/code writer for writes. | 1 / M |
| CO17 | Search code across allowed repositories/ref/path; search commits | W `repo/search.go::Search`, W `repo/commit.go::SearchCommits`, explore code handler → native indexer/Git grep. No equivalent search API found; fetching all files/history is not a native search adapter. | 3 / M; global indexer configuration applies |
| CO18 | Cherry-pick/revert through the native browser editor | W `repo/cherry_pick.go` and editor routes → native repository files service. A submitted `/diffpatch` does not supply the native operation's commit/parent/conflict semantics by itself. | 3 / M; preserve native conflict/branch rules |

Existing U09 corrections remain valid: compare resolves selected refs via
`/git/commits/{escapedRef}?stat=false&verification=false&files=false`, resolves equal
refs once, validates complete SHA/list/total results and compares immutable SHAs.
A fork result must identify a personal non-mirror fork; an import must identify
the expected non-mirror destination. No malformed/timeout result is replayed.
Soda bounds currently include 256 KiB text, 8 MiB downloads, 1 MiB diffs and 32 KiB
writes. These are current subset contracts, not blanket parity with every native
upload or file size. U09/U15 must make limits honest and usable, not redirect the
user to a Forgejo frontend.

### Issues and conversation

**Owner U10.** A `repo/issue*.go`, `label.go`, `milestone.go` delegate to native
issues/services/models. Issue scope is separate from repository scope; local
issues versus pull units, author/writer/admin, blocking and attachment/time settings
are checked natively. Tests add author versus writer, locked discussion, required
form fields, private dependencies, timer ownership and attachment byte safety.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| IS01 | List/filter/search/paginate, read/create/edit/close/reopen issues | `R/issues`, `I`, `/repos/issues/search`; A `repo/issue.go`; native issue/indexer service, actor/metadata checks and quota. | 1 / S |
| IS02 | Choose Markdown/YAML issue template; contact/blank-issue configuration; validate form | `R/issue_templates`, `/issue_config`, `/issue_config/validate`; A `repo/repo.go::{GetIssueTemplates,GetIssueConfig,ValidateIssueConfig}`. Structured fields exist; frontend rendering/required validation is missing, not a missing template API. Native issue creation still owns permission/metadata. | 2 / M |
| IS03 | Read conversation plus native timeline; create/edit/delete comments | `I/comments`, `I/timeline`, `R/issues/comments/{id}`; A `repo/issue_comment.go::{ListIssueComments,ListIssueCommentsAndTimeline}`. Author/writer checks and correct issue/comment identity. | 1 / S comments, M full timeline |
| IS04 | Assign/unassign people, ref, labels, milestone and due date | Issue PATCH, `I/labels`, `I/deadline`, `R/assignees`; A issue/label/milestone handlers. Native IDs and writer rules; clear versus omitted fields must be explicit. | 2 / S subset |
| IS05 | List/create/edit/delete labels and milestones; list associated issues/PRs | `R/labels`, `/milestones`, individual IDs; native writer checks and issue query filters. Organization labels remain native, not duplicated. | 1 / S subset |
| IS06 | Add/remove/list issue and comment reactions | `I/reactions`, `R/issues/comments/{id}/reactions`; native supported emoji and self reaction identity. | 1 / S issue, M comment UI |
| IS07 | Read subscribed/ignored state, watch/unwatch and subscribers | `I/subscriptions`, `/check`, `/{user}`; A `repo/issue_subscription.go`. `/check` returns `WatchInfo`, not guessed 404 state. Soda self operations derive actor. | 2 / S self subset |
| IS08 | List/upload/rename/delete issue and comment assets | `I/assets`, `R/issues/comments/{id}/assets`, individual IDs; multipart native upload, attachment/author/write/quota checks. | 2 / S issue upload/list, M remaining controls |
| IS09 | Download issue/comment attachment in Soda without exposing HTML/auth redirects | W `/attachments/{uuid}`, `repo/attachment.go::GetAttachment`; bearer is accepted on this mixed-content route, with native target permission. Metadata API is not the bytes. | 2 / M; current `native_url` escape remains |
| IS10 | Start/stop/delete own timer; list/add/delete tracked time and totals | `I/stopwatch/{start,stop,delete}`, `I/times`, `/user/{times,stopwatches}`; A `repo/{issue_stopwatch,issue_tracked_time}.go`. Native time-tracker enablement and record-owner rules. | 1 / M |
| IS11 | List/add/remove blockers and dependencies | `I/{dependencies,blocks}`; A `repo/issue_dependency.go`, native cross-issue visibility/cycle/permission rules. | 1 / M |
| IS12 | Pin/unpin/reorder and list pinned issues/PRs | `I/pin`, `/pin/{position}`, `R/{issues,pulls}/pinned`, `/new_pin_allowed`; A `repo/issue_pin.go`. Admin-gated mutations and native pin capacity/order. | 1 / M |
| IS13 | Lock/unlock a discussion with a native reason | W `repo/issue_lock.go::{LockIssue,UnlockIssue}` → native issue lock model. Read DTO `is_locked` is not a mutation API. | 3 / M |
| IS14 | Inspect edited issue/comment content history, revisions and soft-delete permitted history | W `repo/issue_content_history.go::{GetContentHistoryList,GetContentHistoryDetail,SoftDeleteContentHistory}`. Native author/writer/history/issue binding; body diff rendering mixed with data. | 3 / M |
| IS15 | Delete an issue, including explicit multi-selection | `DELETE I`, A `repo/issue.go`, native administrator/issue deletion service. Batch can use explicit independent deletes with per-item outcomes, never claim atomic bulk deletion. | 2 / M; no environment deletion |

### Pull requests and review/merge

**Owner U11; CO07 shared with U09, CI actions with U14.** P routes inherit
repository scope, `mustAllowPulls`, code-reader and native per-operation checks;
conversation uses issue scope. Tests add fork/base/head authorization, old/new
line identities, pending/other-user reviews, team review requests, moved refs,
merge protection/check failures and actual target Git ref verification.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| PR01 | List/filter/read/create/edit/close/reopen PR; retarget; maintainer edits | `R/pulls`, `P`, base/head lookup route; A `repo/pull.go`, D `pull.go`; native permission/service decisions. | 2 / S subset |
| PR02 | PR templates and draft/ready presentation | Native contents support template retrieval. Native title PATCH and read `draft` exist, but W `repo/pull.go` (`PrepareViewPullInfo`) also supplies configured `WorkInProgressPrefixes` to the toggle; no equivalent public settings field was found. Do not hardcode `WIP:`/`Draft:` or invent separate Soda PR state. | 2 templates/title edit + 3 configured draft-toggle metadata / M |
| PR03 | Commits/files/unified diff/patch, checks/statuses | `P/commits`, `/files`, `P.diff`, `P.patch`, `R/commits/{ref}/{status,statuses}`; native pull/Git/status handlers. Different from arbitrary range comparison. | 2 / S; full binary/range/check display unfinished |
| PR04 | List/read reviews and inline comments; create pending/comment/approve/request-changes review; submit/delete pending review | `P/reviews`, `/{id}`, `/comments`; A `repo/pull_review.go` → native review services. Explicit commit plus old/new line positions, not unified-diff offsets; submit is not assumed atomic with earlier comment writes. | 2 / S new-side/basic submission, M full review lifecycle |
| PR05 | Comment in existing code conversation; inspect outdated/resolved context and attachments | A `repo/pull_review.go::CreatePullReviewComment` calls `services/pull.CreateCodeCommentKnownReviewID`, which finds the first comment by review/line/path and preserves its commit/patch/invalidated state for replies. `prepareSingleReview` hides another user's pending review. Comment asset APIs cover uploads separately. Preserve that tuple rather than invent a parent-thread model. | 2 / M; native reply interface exists, existing/outdated/asset cases still need tests |
| PR06 | Resolve/unresolve code conversation | W `repo/pull_review.go::UpdateResolveConversation` → `CanMarkConversation`/`MarkConversation`; comment/repository/PR binding. No corresponding API command. | 3 / M |
| PR07 | Mark files viewed/unviewed and retain changed-since-viewed state | W `repo/pull_review.go::UpdateViewedFiles` → `models/pull.UpdateReviewState`, user/PR/head SHA. Not browser-local completion flags or a Soda review-state table. | 3 / M |
| PR08 | Request/remove individual/team reviews; dismiss/undismiss approvals | `P/requested_reviewers`, `/reviews/{id}/{dismissals,undismissals}`; D `PullReviewRequestOptions` includes team reviewers; native permission/dismissal rules. | 1 / S individual requests, M teams/dismissal controls |
| PR09 | Update/rebase PR branch; perform native merge/squash/rebase/manual choices and optional branch deletion | `POST P/update`, `POST P/merge`; A `repo/pull.go`, native pull/merge services, actual writer/protection/allowed-style checks. `head_commit_id` is a native stale-head precondition; do not force or bypass native checks. | 2 / S conservative merge subset, M remaining permitted choices |
| PR10 | Schedule/cancel auto-merge and show actual pending state | POST/DELETE `P/merge`, native auto-merge service. Command exists; API PR DTO lacks the web merge panel's full scheduled/eligibility detail. | 2 commands + 3 missing state / M |
| PR11 | Explain effective merge blockers/options before submission | W `repo/pull.go::{PrepareViewPullInfo,PrepareViewPullInfoActions}` → native review/protection/check logic. API `mergeable` is not a full “all requirements pass” decision. Individual statuses/reviews can be displayed but must not be reinterpreted as Soda merge authority. | 3 / M full native decision DTO |
| PR12 | Browse selected commit ranges/expanded context in PR diff | Existing-PR diff/file endpoints cover default range; W `repo/pull.go::ViewPullFilesForRange` and CO07 supply native range semantics. Blob reads can supply bounded unchanged text; need stable native identities. | 2 unchanged text + 3 arbitrary range / M |

### Issue projects and boards — not Soda environments

**Owner U10, coordinated with U12 for organization/user scope.** W
`repo/projects.go`, `org/projects.go`, `shared/project/column.go` → `models/project/`.
Both repo-unit gates and owner visibility/`canWriteProjects` are required. Cross-repo
cards must not leak invisible issues. `services/f3/driver/projects.go` is a
migration driver, not a human CRUD API. Neither release has an equivalent REST
board family. Tests cover scope/owner, hidden cards, column ownership, ordering,
default-column changes and concurrent moves.

| ID | Required action / data and commands | Native interface / missing boundary | Class / Soda |
| --- | --- | --- | --- |
| BD01 | List/filter/read repo, organization and personal issue projects and cards | W project `Projects`/`ViewProject`; native project and issue visibility/counts before HTML. | 3 / M |
| BD02 | Create/edit/open/close/delete a native issue project | W `NewProjectPost`, `EditProjectPost`, `ChangeProjectStatus`, `DeleteProject`; native owner/repo association and writer checks. | 3 / M; deletion here is not environment deletion |
| BD03 | Add/edit/delete/reorder/default columns | W column handlers and `shared/project.MoveColumns` → native column ownership/order methods. | 3 / M |
| BD04 | Add/remove/move/reorder cards and associate an issue with a project | W `UpdateIssueProject`, `MoveIssues` and organization equivalents; native issue/project/repository membership checks. | 3 / M |

### Repository administration, hooks and organizations

**Owner U12; sensitive lifecycle actions also U07/U16/U17.** Tests add owner versus
admin collaborator versus reader, organization units, last-owner safeguards,
private team visibility, preservation of omitted PATCH fields, secret/event
rotation and SSRF/delivery safety. No hook target may be contacted by an audit.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| RS01 | Read/edit repository name/description/website/visibility/default branch and supported feature/merge settings | `GET/PATCH R`; A `repo/repo.go`, D `EditRepoOption`, native repository services. `reqAdmin`, actual unit/setting behavior. Some enabled-unit configuration is separate in RS03. | 2 / S subset; rename lifecycle D |
| RS02 | Topics, repository avatar and stars/watchers/subscription | `R/topics`, `/avatar`, `/stargazers`, `/subscribers`, `/subscription`, `/user/starred/{owner}/{repo}`; native repo/user handlers and settings. | 2 / M beyond existing metadata |
| RS03 | Configure repository units and native merge/issue/wiki settings | D `repo.go::EditRepoOption` includes external tracker/wiki, globally editable wiki, wiki branch, project/release/package/Actions toggles and merge policies. W `repo/setting/setting.go::UnitsPost` additionally exposes code-unit enablement and close-issues-via-any-branch, absent from the edit DTO. | 2 supported fields + 3 missing settings / M |
| RS04 | List/check/add/update/remove collaborators and team-repository assignments | `R/collaborators`, `/{name}/permission`, `R/teams`; A `repo/{collaborators,teams}.go`. Reader versus administrator mutation gates; native effective permissions remain upstream. | 1 / S subset |
| RS05 | Branch/tag protection read/create/edit/delete | `R/{branch_protections,tag_protections}` and identifiers; A `repo/{branch,tag}.go`, D protection options. `reqAdmin`, non-archived gates for creation/edits; nested allowlists only with enabled parents. | 2 / S |
| RS06 | Advanced branch allowlist/file-pattern/protection form fields already exposed by Soda | Soda `internal/web/protections_api.go`, `internal/forgejo/protections.go`; `dashboard/src/protections.tsx` currently omits advanced fields. Preserve native parent/child option semantics. | 4 / M controls, not a new native API |
| RS07 | List/create/delete deploy keys with read/write permission | `R/keys`, `/{id}`; A `repo/key.go`, native key service, repository administration. Distinct from personal Git and Soda development keys. | 1 / S subset |
| RS08 | List/read/create/edit/delete native Git hooks if enabled | `R/hooks/git`, `/{id}`; A `repo/git_hook.go`, `reqAdmin` + `reqGitHook`, global `DISABLE_GIT_HOOKS` and native account allowance. This grants native code execution, not a harmless hook form. | 1 / D; do not enable or elevate it |
| RS09 | Inspect/configure/sync pull/push mirrors and convert mirror | `R/mirror-sync`, `/push_mirrors`, `/push_mirrors-sync`, `/convert`, repository PATCH; A `repo/mirror.go`/repository handlers. Native migration host/credential/quota/non-archived and owner/admin gates. | 2 / M; credentials are transient/redacted |
| RS10 | Rename/transfer/accept/reject/archive/delete repository | Repository PATCH, `POST R/transfer{,/accept,/reject}`, `DELETE R`; A repository/transfer handlers. DELETE is `tokenRequiresRepoOwnerScope` + `reqOwner`, user or organization write scope, not repository-write alone. | 2 / D linked-environment/ownership policy, no implicit Linux mutation |
| RS11 | LFS lock/list/unlock/download | Existing Git LFS protocol; W routes and `services/lfs/`, native object/ref/lock authority. Protocol adapters, not a new Soda LFS store. | 2 / M |
| RS12 | Administer native LFS inventory, pointer association and deletion | W `repo/setting/lfs.go::{LFSFiles,LFSPointerFiles,LFSAutoAssociate,LFSDelete}`; native repo admin. Standard LFS transfer/locking is not complete administrative object inventory/association. | 3 / M |
| RS13 | Detach fork; initiating owner cancels pending transfer | W `repo/setting/setting.go::SettingsPost` cases `convert_fork` and `cancel_transfer`, native repository/transfer services. REST `/convert` is mirror-only; `/transfer/reject` checks the recipient via `CanUserAcceptTransfer`, not the initiating owner's cancellation. | 3 / D; native mutation and linked-association decision |
| RS14 | Edit native signing trust; site-admin repository health-check/index maintenance | Same `SettingsPost`, cases `signing`, `admin`, `admin_index`; native trust model, `IsFsckEnabled` and stats/code/issues indexers. Creation's `trust_model` does not make it an edit field; index/health actions explicitly require native site admin. | 3 / M |
| RS15 | Configure repositories followed through federation when enabled | `SettingsPost` case `federation` and native federation services. Protocol inbox/outbox is not this repository settings command; keep native network/authority checks. | 3 / M; conditional |
| RS16 | List/adopt/delete own unadopted repositories when permitted | W `user/setting/{profile,adopt}.go::{Repos,AdoptOrDeleteRepository}`, native adoption/deletion settings and fixed user-owned path. `/admin/unadopted` has different authority. No Soda directory scanner, local-import privilege or arbitrary path input. | 3 / M; conditional, distinct from AD07 |
| HK01 | List/read/create/delete repository/personal/organization/instance hooks | `R/hooks`, `/user/hooks`, `/orgs/{org}/hooks`, `/admin/hooks`; A `utils/hook.go`, native webhook model/service. Respective admin/self/org-owner/site-admin gates and webhooks setting. Native conversion exposes decrypted authorization and potentially secret URLs; Soda DTO must not. | 2 / S repo/org subset |
| HK02 | Edit supported hook URL/type-specific config, active/filter/auth-header/common events | PATCH same resources; A `utils/hook.go::editHook`. Omitted authorization/events/filter are not preserved natively; a typed adapter must preserve intended values without exposing them. | 2 / S bounded subset |
| HK03 | Rotate signing secret, edit package/Actions event flags and create/edit full handler-specific fields | Same PATCH exists but does not assign signing secret/package/Actions flags; unchanged in inspected v16. REST creation also handles only selected type-specific metadata. W `repo/setting/webhook.go::{WebhookCreate,WebhookUpdate,ParseHookEvent}` uses registered native handlers and complete fields, including HTTP method. | 3 / M for unsupported fields; U17 semantic gap, not transport absence |
| HK04 | Explicit test delivery | `POST R/hooks/{id}/tests`; A `repo/hook.go::TestHook`, native refs/events/allowed-host/TLS rules. Other owner scopes need their native test contract, not a repository token substitution. | 2 repo test / M; 3 if required outside supplied scope |
| HK05 | List/read delivery attempts and response/error; replay an exact attempt | W `repo/setting/webhook.go::{WebhookEdit,WebhookReplay}` → native hook-task/delivery service. No human REST delivery history/replay family. Request bodies/headers can contain secrets and need explicit output policy. | 3 / M |
| HK06 | Manage instance default versus system hook kind | `/admin/hooks` list/create is system-only; individual GET/PATCH/DELETE can address native default or system hooks by known ID (`GetSystemOrDefaultWebhook`, `DeleteDefaultSystemWebhook`). W `admin/hooks.go::DefaultOrSystemWebhooks` and creation routes distinguish kinds. Do not create a system hook for a selected default hook. | 3 default list/create + 2 existing ID operations / M |
| OR01 | Directory, create/read/edit organization and avatar | `/orgs`, `/orgs/{org}`, `/avatar`; A `org/{org,avatar}.go`; organization scope, native visibility/owner mutations. Org PATCH overwrites omitted strings; adapter preserves explicit intent. | 2 / S subset |
| OR02 | Members, remove member, public/concealed membership | `/orgs/{org}/{members,public_members}`; A `org/member.go`; owner removal, native self/membership visibility checks. | 1 / S subset |
| OR03 | Create/read/edit/delete/search teams and unit/repository-creation policies | `/orgs/{org}/teams`, `/teams/{id}`; A `org/team.go`, native org-owner/member gates and fixed administrator-team units. | 2 / S subset |
| OR04 | Edit all-repository/per-unit/repository-creation fields already exposed by Soda | Soda `internal/web/organizations_api.go`/`internal/forgejo/organizations.go`; `dashboard/src/teams.tsx` lacks those editors. No copied role table. | 4 / M controls |
| OR05 | Add/remove/list team members and team repositories | `/teams/{id}/{members,repos}`; native org ownership/member access and repository-organization checks. Adding an existing account to a team is not an email invitation. | 1 / S subset |
| OR06 | Invite by email/token; view pending invitation; accept/decline where supported | W `org/teams.go::{TeamsAction,TeamInvite,TeamInvitePost}`, `/org/invite/{token}`; native invitation service, expiry and membership decisions. No equivalent public API invitation flow. | 3 / M; mail conditional |
| OR07 | Organization labels, activity, blocked users, native quotas | `/orgs/{org}/{labels,activities/feeds,list_blocked,block/{name},unblock/{name},quota}`; corresponding A org handlers; owner/private/unit/quota gates. | 1 / M |
| OR08 | Organization-owned OAuth clients, client-secret rotation/grants | W `org/setting.go` and org OAuth handlers, shared native OAuth app operations. `/user/applications/oauth2` is not organization-owned client CRUD. | 3 / M |
| OR09 | Organization rename/deletion and account association effects | POST `/orgs/{org}/rename`, DELETE organization; A `org/org.go`, owner and native dependency checks. No automatic Soda account/project remapping. | 2 / D |

### Work overview, notifications, activity and graphs

**Owner U13; advanced code graph shared U09.** Tests add hidden activity, repository
visibility changes, large/empty pages, correct native totals/filter semantics,
notification object links and expensive-query limits. Do not build a Soda index.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| WK01 | My repositories/work/assigned issues/requested reviews; organization/team overview | Repository inventory, `/repos/issues/search` native actor/assignment/review filters, org/team repository APIs; A `repo/issue.go` → native issue indexer. Compose bounded queries, not a complete downloaded inventory. | 2 / S personal subset |
| WK02 | Global repository/issue/PR/user/organization/topic search | `/repos/search`, `/repos/issues/search`, `/users/search`, `/orgs`, `/topics/search`; native exploration/privacy/indexer gates. Code/commit search is separately CO17. | 2 / S subset |
| WK03 | Notification list/unread count/filter; mark read/unread/pinned; navigate thread subjects | `/notifications`, `/new`, `/threads/{id}`, `R/notifications`; A `notify/`, notification scope and actual recipient. Native subject IDs/URLs require Soda object mapping, not blind external links. | 2 / S subset |
| WK04 | Repository/org/team feeds and user heatmap | `R/activities/feeds`, org/team activity routes and AC08; native feed service with current visible resources. | 1 / S user subset, M others |
| WK05 | Period activity summary, authors, contributors, code frequency/recent commits | W `repo/{activity,contributors,code_frequency,recent_commits}.go`; native activity/statistics services/cache. `/data` web JSON exists, but is not a human OAuth API; repository/unit permissions still required. | 3 / M |
| WK06 | Multi-ref commit graph and native filtering/paging | W `repo/commit.go::Graph` → native Git graph; ordinary commit pages expose parents but are not an equivalent complete multi-ref traversal/search contract. No local clone or assumed complete page. | 3 / M full native graph |

### Actions, automation configuration and provider runners

**Owners U14/U16; local runner services/capacity stay Cockpit.** A
`repo/action.go`, org/user action handlers and shared Actions implementations;
W `repo/actions/{actions,manual,view}.go`, `repo/pull.go` trust handlers,
`shared/actions/runners.go`; native owner `services/actions/`, `models/actions/`.
Tests add disabled Actions, reader/writer/owner separation, cross-repo run/job IDs,
attempt versus job/run index, queued jobs without a task, expired logs/artifacts,
range/large-output behavior, write-only secrets, exact dispatch/no replay and
untrusted-fork approvals. Real dispatch, registration and runner jobs still need
separate scope; this source audit performs none.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| CI01 | List/filter/page runs/tasks and read a run | `R/actions/{runs,tasks}`, `/runs/{id}`; repository scope + Actions reader, native run repository match. Body totals without Link headers; respect native API page cap. | 2 / S |
| CI02 | Discover workflow files/source and manual dispatch with native inputs | Native tree/content routes plus `POST R/actions/workflows/{file}/dispatches`; native directory priority, actual ref/input validation, Actions writer/token/non-archived gate, optional `return_run_info`. | 2 / S discovery/basic dispatch; typed form schema is CI03 |
| CI03 | Read native workflow_dispatch input schema/defaults/options and enabled state | W `repo/actions/actions.go`/manual form uses native workflow parser and workflow configuration. Directory scanning does not provide this validated schema/state; no REST workflow catalog equivalent found. | 3 / M |
| CI04 | List jobs, attempts and steps, live progress, diagnostics and native allowed controls | v15 W `repo/actions/view.go::getViewResponse` mixes HTML and data. V16 adds run jobs, but not the complete step/attempt/diagnostic/control DTO. Do not expose its entire HTML-bearing response or copy scheduler decisions. | 3 v15; 2 v16 jobs + 3 remaining detail / M |
| CI05 | Read/tail/download job logs and run log archive | v15 W `repo/actions/view.go::Logs`; v16 GET `R/actions/jobs/{id}/logs?attempt=` and `/runs/{id}/logs` reuse native log service and Actions read. Range/plaintext/ZIP, expired/no-task states and actual attempt identity matter. | 3 v15; 2 v16 / M |
| CI06 | List/read/download/delete run artifacts | v15 W artifact handlers; v16 `R/actions/artifacts` and `/runs/{id}/artifacts`, individual `/zip`/DELETE. Reader for reads, writer for deletion, repository-bound aggregated identity; unconfirmed/expired artifacts are not bytes. | 3 v15; 2 v16 / M |
| CI07 | Cancel a pending/running run | v15 W `repo/actions/view.go::Cancel`; v16 `POST /runs/{id}/cancel`, token + Actions writer, repository ownership check → `services/actions.CancelRun`. Completed run remains unchanged with 204. | 3 v15; 1 v16 / M |
| CI08 | Rerun entire workflow or selected job | W `repo/actions/view.go::Rerun` → `RerunAllJobs`/`RerunJob`; native still-running/disabled/eligibility checks. No matching human API found in either release. Re-dispatch is not rerun. | 3 / M |
| CI09 | Enable/disable workflow without changing its Git contents | W `repo/actions/view.go::{EnableWorkflowFile,DisableWorkflowFile}` → native workflow config. Repository admin, not merely code writer; no API equivalent. | 3 / M |
| CI10 | Explain/approve pending untrusted PR runs; grant/revoke native Actions trust | W `repo/pull.go::{PrepareViewPullInfoActionsTrust,UpdateTrustWithPullRequestActions}` → `services/actions/trust.go`; Actions/delegated-trust predicates, not Soda operator. Generic runner/workflow tokens cannot represent the human. | 3 / M |
| CI11 | Delete a completed workflow run | `DELETE R/actions/runs/{id}` exists in v15; `reqAdmin(TypeActions)` and native `DeleteRun`, not “all run controls missing.” | 1 / M |
| CI12 | List/add/replace/delete repo/org/user secrets and variables | Native `/actions/secrets/{name}` PUT/DELETE; variable POST create, PUT replace, GET/DELETE. Repo `addActionsRoutes` uses **`reqOwner(TypeActions)`**, org uses `reqOrgOwnership`, user self; respective scopes. Web repo settings allow admin collaborators, so stock API does not cover that actor. Never read/store secret values in Soda. | 2 owner/org/self + 3 repo-admin authority parity / S subset, M full coverage |
| CI13 | Edit secret metadata/rename while preserving stored value; instance variables | W `shared/secrets/secrets.go`, W `repo/setting/variables.go` used by admin routes. Name-keyed secret PUT is replacement, not metadata-only edit; instance variable CRUD is not supplied by repo/org routes. | 3 / M |
| CI14 | List/register/read/delete provider runners; registration token and assigned jobs in native user/repo/org/site scope | Native Actions runner families; A shared/repo/org/user/admin handlers, exact scope/owner/admin gates and secret-channel delivery of new credentials. Repo owner-only API versus admin-gated web settings needs native parity review as in CI12. These APIs do not start a Soda runner service. | 2 covered actors + 3 repo-admin parity / M dashboard; Cockpit backing logic untouched |
| CI15 | Edit provider runner name/description, regenerate its credential and reset shared registration token | W `shared/actions/runners.go::{RunnerEditPost,RunnerResetRegistrationToken}` → native `Editable(ownerID,repoID)` and runner model. Existing REST registration/delete is not edit/reset. Agent-reported labels are not a web-editable field here; do not invent one. | 3 / M |

### Releases, wiki and packages

**Owner U15.** Tests add draft versus reader visibility, protected tags, omitted
versus explicitly empty fields, private/download redirects, metadata versus bytes,
native quotas, MIME/filenames, wiki path encoding/revisions and package type/owner
visibility. No publish/delete/cleanup action was executed.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| RE01 | List/read/latest/by-tag releases, including authorized drafts | `R/releases`, `/latest`, `/{id}`, `/tags/{tag}`; A `repo/release.go`, release unit read plus draft writer visibility. | 1 / S |
| RE02 | Create/edit/publish/prerelease/delete release; choose tag/target | POST/PATCH/DELETE release routes; native release writer/protected-tag/quota checks, pointer booleans. Separate delete tag/release semantics. | 2 / S subset |
| RE03 | Explicitly clear existing release notes or other fields native edit permits | A `repo/release.go::EditRelease` ignores empty strings; same in inspected v16. W `repo/release.go::EditReleasePost` assigns `rel.Note = form.Content`. The definite gap is clearing notes; do not infer that every title/tag/target clear is a valid native operation. No delete/recreate or ignored PATCH as success. | 3 / M; U17 release-clear semantic gap |
| RE04 | List/upload/read/rename/delete release assets; download bytes | Release `/assets` API; A `repo/release_attachment.go`, native attachments service; metadata versus octet-stream/native download handling, release permission/draft/size gates. | 2 / S subset; complete same-origin byte handling pending |
| WI01 | Index/current page/sidebar/footer and page revision list | `R/wiki/{pages,page/{name},revisions/{name}}`; A `repo/wiki.go`, native wiki reader and path conversion. Page response currently resolves the wiki branch, not a requested historical SHA. | 2 / M |
| WI02 | Create/edit/rename/delete wiki page with native commit message | POST `/wiki/new`, PATCH/DELETE `/wiki/page/{name}`; native wiki writer/non-archived/quota and `services/wiki`. API and web share `EditWikiPage`; no last-commit CAS input exists in either form. | 2 / M; must label native last-write behavior, not claim optimistic locking |
| WI03 | View historical wiki content/raw bytes/commit diff and restore chosen content | W `repo/wiki.go` revision/raw views and wiki commit/diff routes; native wiki repository/ref context. Main repository `/git` is not the wiki. Revision *list* API alone cannot supply historical page/diff. | 3 / M; restoration can use native edit once content is available |
| WI04 | Native wiki content search | W `repo/wiki.go::WikiSearchContent` → `services/wiki.SearchWikiContents`; no search API, no Soda wiki index. | 3 / M |
| WI05 | Rename/normalize wiki branch; delete entire native wiki | Repository PATCH `wiki_branch` calls `services/wiki.NormalizeWikiBranch`, matching native settings normalization. Whole-wiki deletion is only W `repo/setting/setting.go` case `delete-wiki` → `services/wiki.DeleteWiki`; page deletion is not equivalent. | 2 branch + 3 whole-wiki delete / M; no live deletion |
| PK01 | Owner/type/name/version package inventory and version file metadata | `/packages/{owner}`, `/{type}/{name}/{version}`, `/files`; A `packages/package.go` → native package descriptors, owner/package read. `services/convert/package.go` hides inaccessible linked repositories. Requires **package** scope. | 1 / M |
| PK02 | Download/publish using native package protocols; show copyable install commands | `routers/api/packages/api.go::CommonRoutes` under `/api/packages`, container routes under `/v2`; native package access/quota/auth. Type-specific npm/PyPI/Maven/generic/OCI/etc. handlers already provide protocol bytes/metadata. | 2 / M browser adapter/guidance; ordinary clients remain native |
| PK03 | Complete package detail: native metadata/properties/download counts and format-specific install context | D `package.go` / `convert.ToPackage{,File}` omit descriptor metadata/properties/counts used by W `user/package.go::ViewPackageVersion`. Protocol metadata can cover individual formats, but is not the complete native browser descriptor, especially distribution/file properties. | 3 / M for missing descriptor; reuse PK02 where sufficient |
| PK04 | Link/unlink repository and delete a package version | POST `/{type}/{name}/-/link/{repo}`, `/-/unlink`, DELETE version; native package writer and valid repository access. Delete does not delete the linked repository/environment. | 2 / M |
| PK05 | Configure/preview/run native owner package cleanup rules | W `user/setting/packages.go`, native package cleanup service, owner/org gates. Public package DELETE is not cleanup-rule CRUD/preview. | 3 / M |
| PK06 | Native package registry maintenance offered in settings: Cargo index initialize/rebuild, Chef key regeneration | W `user/setting/packages.go`; native owner/method-specific service/configuration, secret output handling. No generic command or filesystem endpoint. | 3 / M; preserve native conditional availability |

Package browsing is not “all API missing”; nor does the existence of a package
manager endpoint prove every format's full browser detail can be reconstructed
from it. PK03 calls for a native descriptor result, not downloading archives into
Soda and building another package index. Large streaming/output contracts still
need per-format implementation and tests. Wiki CAS would be a new native behavior,
not merely exposure of an existing web check; any stronger concurrency promise
needs separate review and shared native implementation.

### Forgejo site administration

**Owner U16, build/config owners U01/U02/U03, closure U17/U20.** API admin group
inherits `admin` scope, `reqToken`, `reqSiteAdmin`; other scoped endpoints keep
**their own** predicates. W admin routes require signed-in native site admin and
normal web request protection. Native root, Soda operator and Forgejo site admin
are separate authorities. Tests add site-admin versus operator-only, native role
revocation, self/last-admin protections, source configuration, redaction, concrete
maintenance effects and preservation of unrelated users/repos/Soda associations.

| ID | Required action / data and commands | Native interface, gate and evidence | Class / Soda |
| --- | --- | --- | --- |
| AD01 | Administration overview/statistics/runtime health | W `admin/{admin,config}.go::{Dashboard,SystemStatus,MonitorStats}` → native statistics/runtime state. `/version` and public `/settings` are insufficient; do not return raw process/config objects. | 3 / M |
| AD02 | Search/list native People; create/edit native account/status/permissions | `GET/POST /admin/users`, PATCH `/{name}`; A `admin/user.go`, D `admin_user.go`, native user services. Admin/active/prohibit-login/source fields differ from self profile; MFA read/reset is AD22. Native inventory, not Soda's local users table. | 2 / S list/create, M full edit |
| AD03 | Rename/delete account and preserve or explicitly resolve Soda associations | POST `/admin/users/{name}/rename`, DELETE account; native rename/deletion/dependency rules. No automatic Linux rename/revocation/deletion. | 2 / D |
| AD04 | Administer user Git keys and emails; search all emails | `/admin/users/{name}/{keys,emails}`, `/admin/emails{,/search}`; A `admin/{user,email}.go`. Native admin gate, explicit target and native services. | 1 / M |
| AD05 | Admin activate email and change another user's avatar | W `admin/{emails,users}.go::{ActivateEmail,AvatarPost,DeleteAvatar}`; available admin email CRUD is not every activation/avatar operation. No self-token impersonation. | 3 / M |
| AD06 | Instance organizations/repositories; create on behalf of a selected native user | `/admin/orgs`, `/admin/users/{name}/{orgs,repos}`, admin-visible `/repos/search` and scoped repo/org APIs; A admin/repo/org handlers. Paginate through supported native queries, not a local provider inventory. | 2 / M |
| AD07 | List/adopt/delete exact unadopted repositories | `/admin/unadopted`, `/{name}/{repo}`; A `admin/adopt.go` → native adoption service. Explicit destructive action, not discovery-triggered adoption. | 2 / M; no source/fixture mutation allowed by audit |
| AD08 | Instance hooks/default hooks/delivery history | HK01–HK06, A `admin/hooks.go`, W `admin/hooks.go`; site-admin not appliance operator. | 2 stock system CRUD + 3 missing native sub-actions / M |
| AD09 | Instance provider runners/jobs and configuration | CI14/CI15, A `admin/runners.go`, W shared runner handlers; native site administration, not local runner lifecycle. | 2 stock + 3 edit/reset / M |
| AD10 | Native quota rules/groups, membership, user usage and attachments/packages/artifacts | `/admin/quota/{rules,groups}`, group rules/users and `/admin/users/{name}/quota`; A `admin/quota*.go` → native quota services; gated by `setting.Quota.Enabled`. | 2 / M; never Soda environment quotas |
| AD11 | List and explicitly run named native cron tasks | `GET /admin/cron`, `POST /admin/cron/{task}`; A `admin/cron.go` → native registered task. No arbitrary command/cron editor or automatic execution. | 1 / M |
| AD12 | Native dashboard synchronize-all repository branches/tags | W `admin/admin.go::DashboardPost` specially calls `AddAllRepoBranchesToSyncQueue` and `AddAllRepoTagsToSyncQueue`; its other operations resolve registered cron tasks covered by AD11. Expose only these unmatched native commands, not generic maintenance dispatch. | 3 / M |
| AD13 | Inspect/adjust native queues; remove queued items; process stacktrace/cancellation; diagnostics | W `admin/{queue,stacktrace,diagnosis}.go`; concrete queue/process IDs and native manager. No public API equivalent. Queue changes/cancellation can affect live writes; output can disclose secrets. | 3 / M |
| AD14 | List/read/delete/clear native system notices | W `admin/notice.go` → native notice model/service. Not Soda logs or a new audit database. | 3 / M |
| AD15 | List/create/read/edit/delete native authentication sources | W `admin/auths.go` → native auth source services; type-specific LDAP/SMTP/OAuth/etc. validation, encrypted secrets and existing users. Forgejo CLI auth administration exists but is appliance-process authority, not acting web-admin delegation. | 3 / D security/config contract |
| AD16 | Inspect native configuration and change only the settings native UI actually allows | W `admin/config.go::{Config,ConfigSettings,ChangeConfig}`. `ChangeConfig` allowlists dynamic Gravatar/federated-avatar/editor-app settings; it is **not** a general app.ini editor. Redact native config DTOs; no arbitrary file/key/value proxy. | 3 / D bounded native contract |
| AD17 | Test mail/cache and conditional database self-check | W `admin/config.go::{SendTestMail,TestCache}`, `admin/admin.go::SelfCheck` (MySQL only); native implementations/configuration, explicit side effects. | 3 / M; no check executed |
| AD18 | Manage instance OAuth applications and rotate secrets/grants | W `admin/applications.go` using native `OAuth2CommonHandlers`; personal-client API is not system-app ownership. | 3 / M |
| AD19 | Instance package inventory and explicit package cleanup | W `admin/packages.go::{Packages,DeletePackageVersion,CleanupExpiredData}`. Scoped package APIs cover explicit version deletion, not complete site-wide inventory/cleanup. | 2 scoped deletes + 3 site inventory/cleanup / M |
| AD20 | Report abuse; inspect reports/content; apply native moderation decisions when enabled | W `moderation/`, `admin/reports.go` → native moderation/user/repo/issue services; native reporter versus site-admin gates and `setting.Moderation.Enabled`. No public REST report management found. | 3 / M; deletion effects still require lifecycle decision |
| AD21 | Federation settings and user-facing interaction when enabled | ActivityPub/WebFinger/nodeinfo routes are protocols, not complete admin/account UI. V16 adds user activitypub follow (see delta); native remote interaction and signature/source rules must remain native. | 2 supported protocol/client actions + 3 uncovered configuration / D |
| AD22 | Read complete native account-administration fields/MFA status, filter People by native status, explicitly reset another user's MFA | W `admin/users.go::{Users,ViewUser,EditUser,EditUserPost}` loads native MFA/detail/status-filter fields and `Reset2FA` deletes TOTP and WebAuthn credentials. In v15, A `admin/user.go::SearchUsers` only selects source/login/sort/page. Targeted U01 contradiction check against the complete v16 release notes and selected v16 handler confirms native `is_2fa_enabled` filtering via `FormOptionalBool` → `SearchUserOptions.IsTwoFactorEnabled`; reuse it rather than adding a duplicate filter. D `User` still omits MFA and several editable admin fields, and `EditUserOption` has no reset flag. Do not fetch all users and fabricate server-side filtering or use self-security authority. | 3 / D sensitive native admin contract |
| AD23 | List/set/remove native repository flags when enabled | `R/flags`, `/{flag}`; A `repo/flags.go`, repository plus admin scope, `reqToken`/`reqSiteAdmin`, `setting.Repository.EnableFlags`; W `repo/flags/` uses the native flag model. | 1 / M; do not enable the setting during audit |

### Cross-cutting Soda-only presentation and product boundaries

**Owners U01/U03/U04/U17/U18/U19/U20 and relevant feature owner.**

| ID | Required outcome | Current evidence / work | Class / Soda |
| --- | --- | --- | --- |
| UI01 | All native objects link to the corresponding Soda view | Soda `dashboard/src/markdown.tsx`, notification/activity/release/issue adapters and route mapping need complete object coverage. External website links are not the same as provider frontend links. Translate known native object identity, not arbitrary URL rewriting/fetching. | 2 / S partial; M closure |
| UI02 | Same-origin authorized images/files/assets and safe Markdown/markup | Native raw/attachment/avatar/protocol interfaces exist. Keep inert content, MIME/download policy, fixed origins/paths, native authorization and bounded streams. Native markup APIs return HTML content; they are not complete native pages or permission to render unsanitized HTML. | 2 / S Markdown/raw subset |
| UI03 | Authentication/error/loading/expiry/help/notifications in Soda | Common N/A/B matrix; provider 401/403, invalid/too-large/unavailable/incompatible states remain distinct. No bare HTML error/redirect becomes a successful API response. | 2 / S partial |
| UI04 | Close Forgejo browser ingress only after replacement workflows pass | `appliance/config/proxy.Caddyfile` still reverse-proxies full Forgejo on its separate origin; `/` still serves HTMX, React `/app/` is preview. OAuth/content/mail links and native protocols need deliberate split. | 3 native-origin-dependent auth + 2 deployment adapter / D; do not close now |
| UI05 | Preserve Soda environments, join/access, terminal and operator boundaries | Existing `internal/host/`/`project-os/`/Soda APIs own real project creation/join/access. Browser terminal still needs an existing-user workspace transport/security design, not a Forgejo API. Cockpit/Tailnet/Runners remain separate operator UI. | Outside Forgejo-gap classification; U07/U08 plus terminal owner decision; U08 acceptance unchanged |

## Native changes implied by the class-3 rows

These are **contract review boundaries**, not an approved patch list or another
feature inventory. Keep the smallest shared implementation with its existing web
caller; do not serialize `ctx.Data`, expose an arbitrary internal method, or copy
services into Soda. Native APIs must reapply resource/unit/actor/security gates.

- **AU rows:** a Forgejo-owned pre-authentication/challenge/security and consent
  boundary. Model native allowed next steps, opaque expiring single-use challenges,
  completion/revocation and typed failure without account enumeration. Share
  password/MFA/WebAuthn/activation/linking decisions. H05 threat-model review is
  required before selecting wire routes, trust between Soda and Forgejo, or token
  issuance. Presentation of credentials in Soda is not a second password authority;
  their verification/storage must remain native and transient forwarding must be
  narrowly designed. External IdP and RP-ID conflicts need explicit resolution.
- **CO05/CO07/CO09, CO17/CO18, WK05/WK06, WI03/WI04:** bounded native Git/indexer/wiki
  results before rendering. Commit/ref/path identities, binary/rename/truncation
  semantics, work/output/time/concurrency/cancellation limits must be explicit.
  Paginating lines does not bound blame/graph/search computation. Do not create a
  temporary PR, clone/cache or alternate engine to supply these operations.
- **IS13/IS14, PR02/PR06/PR07/PR10–PR12, BD rows:** expose native conversation/review,
  merge-panel and issue-project operations with original ownership checks. Keep
  hidden cards/history, pending reviews and viewed-at-SHA state native. Share native
  merge eligibility calculations rather than deriving copied rules in Soda.
- **RS03/RS12–RS16, HK03–HK06, OR06/OR08, WI05:** extend exact configuration/invitation/LFS/hook
  operations. Hook changes should share native registered handlers and event
  conversion; distinguish keep/replace/clear for secret fields. Delivery replay
  is an explicit network mutation, not a read. No unrestricted configuration proxy.
- **CI03–CI10, CI12–CI15:** reuse v16 stock interfaces where selected; expose only
  still-missing native workflow metadata, step/attempt/diagnostic/control, trust,
  runner edit/reset and secret-metadata functions. Review owner-versus-admin
  route differences natively; do not silently broaden an API gate or use an owner
  token for a collaborator. Existing web JSON contains
  rendered HTML and is not a ready-to-export stable API. Native run/job IDs are not
  web indices; deleted/expired/not-started objects need explicit semantics.
- **RE03, PK03/PK05/PK06, AD rows:** extend release field presence semantics and
  native package/admin results/commands. Redact at the native boundary, especially
  configuration, auth sources, hook deliveries, diagnostics and one-time secrets.
  Do not generalize fixed upstream administration into a privileged host shell.

Each concrete addition needs request/result/error/pagination contracts, native
permission/resource-limit tests, Soda adapter/DOM tests, source/patch provenance,
update owner and retirement condition. A small shared extraction may suffice for
one action; this list does **not** claim all work is small or already feasible.

## V16 comparison and alternatives checked

### Actual added API paths

The schema diff found these **12** paths and no removed path (15 additional
operations; method and handler changes on existing paths still require review):

| Added path under `/api/v1` | Source/authority result |
| --- | --- |
| `/actions/run` | A `misc/actions.go::GetActionsRun` requires automatic **Actions task token** identity; not a human run-list/dispatch API and not Soda's replacement authority. |
| `/admin/users/{username}/tokens` | A `admin/user_token.go` and admin route gates; native site-admin management, not ordinary self MFA/login. |
| `/admin/users/{username}/tokens/{token}` | Same distinction; must not be used to mint/revoke grants as a Soda operator shortcut. |
| `R/actions/artifacts` | Human repository Actions-reader listing; native pagination/filtering. |
| `R/actions/artifacts/{artifact_id}` | Human read; DELETE needs token/Actions writer; repository-bound aggregated artifact lookup. |
| `R/actions/artifacts/{artifact_id}/zip` | Native artifact service stream after repository and confirmation checks. |
| `R/actions/jobs/{job_id}/logs` | Actions reader; `OpenJobLogReader`, explicit attempt, plaintext/byte Range; missing/unexecuted/expired → 404. |
| `R/actions/runs/{run_id}/cancel` | Token + Actions writer, repository match; native `CancelRun`, 204 also for already complete. |
| `R/actions/runs/{run_id}/jobs` | Actions reader, verifies run belongs to repository; native job list, not full browser step/attempt state. |
| `R/actions/runs/{run_id}/logs` | Actions reader/repository match; ZIP of latest attempts via native `WriteRunLogsZip`. |
| `R/actions/runs/{run_id}/artifacts` | Actions reader, run/repository match and native artifact metadata/pagination. |
| `/user/activitypub/follow` | Conditional federation/native user action; does not implement the headless authentication/admin boundary. |

`R` in that table expands to `repos/{owner}/{repo}` (without duplicating
`/api/v1`). In v16, `W/web.go::buildAuthGroup` explicitly contains session and
optional reverse-proxy auth; `buildMixedAuthGroup` adds OAuth/Basic/tokens only for
selected mixed routes. The refactor does **not** turn all web JSON into a bearer
API. This fresh trace replaces assumptions based only on v15 `OAuth2.Verify`.

V16 A `repo/action.go` uses actual repository/unit gates, not Swagger security
advertisements. Native per-feature work remains needed, but porting its already
shared log/artifact/cancel interfaces would be more sensible than recreating
those from scratch if 15 must remain the baseline. No support/security/migration
review or maintained-version choice was completed by this comparison.

### Alternate interfaces and rationale

- `routers/init.go` mounts `/api/v1`, `/api/forgejo/v1`, `/api/internal`, package
  APIs and container registry routes. `routers/api/forgejo/v1/api.go` provides
  root/version, **not** a hidden complete headless API. `services/f3/driver/`
  supports migration, not a public acting-user board/admin interface.
- `routers/private/internal.go::CheckInternalToken` requires Forgejo's internal
  token. CLI auth/admin commands operate with native process/config authority.
  Neither is an acceptable privileged sidecar or browser-admin substitute.
- Git/SSH/LFS and package protocols remain useful for their selected operations.
  Reading/cloning everything into Soda, using runner tokens for humans, or
  pretending a standard transfer protocol manages UI-only settings is not a
  bounded adapter.
- WebAuthn and Actions have JSON web handlers; graphs have `/data`; hooks return
  AJAX results. **JSON format does not establish human API authentication or a
  complete HTML-free contract.** Native session-bound endpoints cannot be called
  by borrowing cookies. Some can share computation with new native API callers.
- Inspected v16 and earlier pinned development snapshot
  `bdc33af0c11568873c336137d404fc327ce0a40e` still lack the two U09 data contracts.
  See [integration research](forgejo-frontend-integration.md) for exact source/blob
  provenance and why HTMX would not fix an absent interface.
- Prior research found no published Forgejo decision rejecting a blame API or
  adopting a substitute API. [Gitea #34108](https://github.com/go-gitea/gitea/issues/34108)
  is not a Forgejo commitment. [Forgejo #2121](https://codeberg.org/forgejo/forgejo/issues/2121)
  and [#2395](https://codeberg.org/forgejo/forgejo/pulls/2395) demonstrate expensive
  blame/file-size concerns; [discussion #320](https://codeberg.org/forgejo/discussions/issues/320)
  is abuse/performance context, not permission to add a Soda clone backend.
- Additional public issue searches for authentication, projects, review, hooks,
  wiki, packages, Actions and headless integration were retained with their query
  URLs. They are broad, paginated keyword matches, **not exhaustive negative
  evidence or proof of maintainer intent**. No native gap is justified by an
  absent search result; the router/handler/alternate-interface trace is the basis.
  No contribution, issue or publication was made. Review relevant active reports
  during baseline selection rather than treat an open report as a reproduced bug.

## Configuration, compatibility and remaining decisions

Inspected Soda `appliance/services/forgejo.container`,
`appliance/config/forgejo.env`, `appliance/config/proxy.Caddyfile`,
`appliance/bin/soda-activate`, setup/config and native build callers. The source
still selects standalone Forgejo `15.0.7`; activation sets native `ROOT_URL`,
`DOMAIN`, Git SSH advertisement and installation lock. Tracked environment source
mostly brands/themes/cache; operator input and persisted app.ini can change other
native settings. No live secret/config/container inspection occurred here.

The selected source's `custom/conf/app.example.ini` and `modules/setting/` are
configuration evidence, not actual appliance state. Check at contract/installed
review: registration/internal/Basic/external auth, mandatory 2FA, mail/captcha,
OAuth client/refresh options, owner visibility, disabled/default repo units,
issues/pulls/wiki/projects/Actions, quota/packages/federation/moderation/stars/forks,
Git hooks, migration allowed domains, webhook allowed hosts/TLS, Git/indexer/file
limits and authentication origins. Disabled must be distinguishable from denied,
missing interface and temporary service failure. No audit request authorizes
changing these settings, enabling reverse-proxy identity or enrolling providers.

**Baseline/build decision:** review supported security baseline and migration notes,
not simply latest version or fewer missing endpoints. Full Forgejo source includes
GPL-3.0-or-later files and a GPL root license; Swagger's MIT declaration is not the
whole distribution license. Preserve corresponding source/attribution/patch notices.
The upstream Dockerfile builds Go with SQLite/bindata/timezone tags and frontend
assets; a Go-only binary is not proven equivalent packaging. H02 must feed the
existing native `forgejo.iid`/OCI/staging/seal/install consumers, preserve native
UID/data/entrypoint/Git behavior and avoid a second build/release platform.

**Compatibility decision:** H03's proposed small feature-contract revisions do not
already exist. Do not infer compatibility from `/version`, advertised paths or a
successful patch apply. H07 must test native authority/semantics as well as DTOs,
review clean rebases and retire equivalent patches deliberately. New features must
be audited in this register before implementing their screens. Missing metadata
must not globally break unrelated stock/environment access, but final selected
coverage cannot accept a placeholder.

**Lifecycle decisions:** AU22, AC02, RS10, OR09 and AD03/AD20 may change upstream
identities/ownership without changing retained project accounts or sessions. Native
operations exist in some cases; the open issue is also legitimate Soda association
behavior. Do not invent generalized remapping, deletion, synchronization or recovery
under an API-gap label. Bring the exact gap for a product decision.

## Existing source, test ownership and evidence limits

Current Soda clients/handlers remain in `internal/forgejo/` and `internal/web/`,
React in `dashboard/src/`. The source also still requires write scopes for several
read handlers (for example issue activity), unlike U09's corrected ref-read split;
least-read-scope coverage needs Soda fixes, not a Forgejo permission bypass. The connected subsets are described in
[dashboard-api.md](dashboard-api.md), not inferred complete from this register.
Keep focused existing tests and extend their concrete owners:

| Area | Existing Soda source/test owner | Native regression starting points to inspect/extend |
| --- | --- | --- |
| Auth/account | `oauth*.go`, `accounts.go`, `provider*.go`, `auth.go`, `oauth_grant_test.go`, store grant tests; `accounts.tsx` | `routers/api/v1/permissions/tests/`, `tests/integration/api_oauth2_apps_test.go`, `tests/e2e/webauthn.test.e2e.ts`, native auth/security tests; new headless challenge tests required |
| Code/copy | `history*.go`, `file_writes.go`, `repository_copy_test.go`; history/file-editor/repository-copy DOM tests | Native `api_repo_compare_test.go`, file/branch/tag/fork/migrate tests, Git/blame tests; new blame/net-diff native contracts required |
| Issues/reviews/boards | `issues*`, `issue_activity*`, `pulls*`; issue/review DOM tests | Native `api_issue_*`, `api_pull_review_test.go`, `api_pull_test.go`, `routers/web/repo/{projects,pull_review}_test.go`; board/resolve/view-state native API tests required |
| Settings/org/hooks | `repository_settings*`, `protections*`, `hooks*`, `organizations*`; settings/org/team screens | Native API collaborator/branch/org/team/hook tests plus native webhook handlers; keep omitted/clear/event/authority cases |
| Work/Actions | `work*`, `actions*`; work/actions DOM tests | `api_repo_activities_test.go`, `api_repo_actions_test.go`, native Actions service/web tests; v16 added human interfaces need OAuth/unit/range/expiry tests, not runner-token fixture reuse |
| Releases/wiki/packages/admin | `releases*`, future feature-owned clients/views | Native release/wiki/package-format/quota/admin tests; new descriptor/security/maintenance contracts need native negative tests |
| Delivery/boundary | navigation guard, `tests/installed/`, existing native build/stage checks | H02/H07 source/patch/build identity and update tests still to implement; H08 browser network/ingress/preserved populated state proof still required |

Names above identify source test owners and starting points, **not tests run in this
audit**. Existing local Go tests, TypeScript, 31 dashboard tests and dashboard/Go
builds passed during the earlier Soda-only follow-up; that evidence is not a
headless Forgejo test pass. No new build, test suite, provider operation, deployment,
reboot, data cleanup or fixture was performed for H01. Only source/research/document
consistency checks were performed; record them in the handoff.

The prior dependency investigation is unchanged: selected `react-router-dom`
7.18.3, `react-markdown` 10.1.0 and `remark-gfm` 4.0.1 have real resolved local
metadata/lockfile; full transitive license review remains U01/U02. No incidental
plugin/Go/Node/frontend dependency or Forgejo image change belongs to this audit.

**H01 result:** workflow-level coverage and source gap classification are now
recorded across the selected frontend and discovered native web surfaces, including
auth/admin risks. Concrete field-complete wire schemas, exact resource budgets,
reviewed authentication feasibility, baseline/maintainer selection, lifecycle
choices and installed conformance remain open implementation/design gates. U01/U17
are not accepted; U08 remains the only accepted, bounded milestone. Continue with
baseline/contract/H05 review, not another broad screen-writing batch or a request
to waive missing workflows through native frontend links.
