# Dashboard JSON API — implemented source contracts

**Unbuilt, untested and not installed.** This describes connected source, not
milestone completion. The [core plan](dashboard-implementation-plan.md) leads;
legacy HTMX remains the default until the gated U18 cutover.

## Shared boundary

Same-origin secure HttpOnly Soda sessions are hashed in SQLite. JSON responses
are no-store; failures carry `{error:{code,message}}`, not redirects or raw
provider/native bodies. Every unsafe method requires the configured Origin,
header CSRF, JSON content type and one bounded object with explicit fields.
Bodies are limited to 64 KiB. Provider IDs are decimal strings in browser DTOs.

Provider HTTP status failures are typed and sanitized. Successful JSON must fit
the whole-response bound and contain exactly one non-null value. API reads use
a 2-MiB provider bound; OAuth responses use 64 KiB. Credentials never appear in
DTOs. No operation retries a write or substitutes an operator credential.

## Soda endpoints

| Route | Contract / authority |
| --- | --- |
| `GET /api/session` | User `{id,login,soda_display_name}`, CSRF, separate Soda-operator hint and public Forgejo origin |
| `POST /api/session/logout` | `{}`; delete session and encrypted grant together through foreign-key cascade, then expire cookie |
| `GET/PATCH /api/me/preferences` | Soda-only `display_name`, at most 200 bytes; no upstream synchronization |
| `GET/POST /api/me/development-keys` | List canonical keys; POST `public_key`; own user only, duplicate fingerprint no-op, no private keys/options or later native propagation |
| `GET /api/environments` | Trusted-team environment reservations, including incomplete ones; not upstream repository visibility |
| `POST /api/environments` | `{owner,repository}`; resolve upstream identity/owner using acting-user grant, reserve once, call real native create; creator is **not** joined |
| `GET /api/environments/{id}` | Reservation/provisioning result, nullable live observation, own login and narrow environment-administrator hint; inspect incomplete reservations too |
| `POST /api/environments/{id}/join` | `{}`; session-derived Linux identity, own public keys, fixed native account operation; membership follows native success |
| `GET /api/environments/{id}/members` | Owner/Soda operator sees permitted members; others see only their own membership |
| `GET /api/environments/{id}/connection` | Requires own membership; current IP/running state and fixed public Ed25519 host key/fingerprint; `routing_verified:false` |

Create failures return the retained environment ID and `Location` with a truthful
502/503 error. A unique repository reservation is never deleted/recreated after
failure. Normal starts/stops are not added. Cached DB IPs are not connection
proof. The helper's new fixed `/connection` operation reads only the public
Ed25519 key after verifying native project labels; it never starts a container,
reads a private key or accepts a caller-selected path.

## Acting-user Forgejo endpoints

All are individually registered and use the current session's encrypted grant.
Native permission checks remain authoritative on each call.

| Route | Input / upstream operation |
| --- | --- |
| `GET /api/forgejo/me` | Current native user, including current upstream admin hint; identity must match Soda session |
| `GET/PATCH /api/forgejo/me/settings` | `full_name,description,location,pronouns`; native user settings, not Soda profile |
| `GET/POST /api/forgejo/me/git-keys` | Native keys; POST `{title,key}`; never installs keys into projects |
| `GET/POST /api/forgejo/admin/users` | Native People inventory; POST `{login,email,password}` with required native first-password change; no Linux naming policy, no local user/session seeded |
| `GET/POST /api/forgejo/repositories` | Native `/user/repos` visible inventory; POST `{name,description,private,auto_init}` without environment side effects |
| `GET /api/forgejo/repos/{owner}/{repo}` | Typed repository metadata and actual clone URLs |
| `GET /api/forgejo/repos/{owner}/{repo}/contents?ref=...&path=...` | Slash/Unicode refs in query; path components encoded independently; bounded UTF-8 text, with safe GFM in the frontend, never active HTML |
| `GET /api/forgejo/repos/{owner}/{repo}/download?ref=...&path=...` | Fixed native raw-file endpoint; at most 8 MiB buffered before response commitment; forced octet-stream attachment and sandbox CSP, no redirects/LFS forwarding |

List routes accept page 1–1000000 and return `{items,next_page,total}`. Metadata
comes from native Link/X-Total-Count headers, not a guessed page-size cap. Totals
are decimal strings or null. Link URLs are never followed or exposed: only a
validated next page number is used to reconstruct the same explicit route.
Malformed or conflicting next-page metadata fails rather than silently truncates.

Files above 256 KiB, binary, symlink/submodule and unavailable encodings are not
rendered as text. Provider transfer errors remain errors. README/Markdown uses
react-markdown 10.1.0 and remark-gfm 4.0.1 with no raw HTML, automatic image loading
or unsafe URL schemes. Relative links retain repository/ref context. Full
ref/file/Markdown/download acceptance remains unexecuted U06 work; no complete
Forgejo rendering-extension parity is claimed.

## Repository history and writes (U09 source)

Individually registered repository subroutes now include:

- `GET /commits?ref=&path=&page=`, `GET /commits/{sha}` and `/commits/{sha}/diff`:
  native history and pinned full SHA, with unified diff bounded to 1 MiB. No local
  Git index or arbitrary ref-to-command execution.
- `GET/POST /branches`, `GET/POST /tags`: native pagination and create operations,
  with branch/tag protection and permissions remaining upstream.
- `GET /compare?base=&head=`: commits/changed files and lossless native total;
  this does not execute a merge.
- `POST/PUT /files?ref=&path=`: explicit branch/path, base64 content up to 32 KiB,
  commit message and exact loaded file SHA for updates. Native stale SHA produces
  409; no retry overwrites newer work. Creation does not accept an update SHA.
- `POST /fork`: optional personal fork name; no environment copy.
- `POST /api/forgejo/repositories/import`: credential-free HTTPS Git URL, name,
  privacy and optional separate transient credentials; owner is the acting user,
  service is native `git`, no mirror or provider-specific issue migration.

React history/ref/compare/editor/fork/import views call these operations. Unified
text diff is the current bounded presentation, not inline-review position support.
Blame and the full native import/diff detail audit remain pending. No source or
installed tests for these new views/adapters have executed.

## Issue collaboration (U10 source)

Explicit repository routes include `GET/POST /issues`, `GET/PATCH /issues/{index}`,
`GET/POST /issues/{index}/comments`, `PATCH /issue-comments/{comment}`,
`PUT /issues/{index}/labels`, `GET/POST /labels` and `/milestones`, and
label/milestone-specific PATCH routes. Native issue scope, visibility, authorship
and edit rules apply; no issue/comment/label data is stored in Soda.

IDs and issue numbers are decimal strings. Issue/comment text is bounded to
32 KiB and rendered with safe GFM. PATCH changes only explicit fields;
`milestone:"0"` removes assignment. Native issue edits are not advertised as
SHA/timestamp-preconditioned writes. Labels replace a supplied bounded ID array;
metadata forms do not acquire environment or host authority.

`/issues/{index}/reactions` supports GET/POST/DELETE for the acting user's reaction.
`/subscription` supports GET/PUT/DELETE for the session-derived username only.
`/attachments` lists native metadata and accepts a simple filename/base64 payload
up to 32 KiB, converted to the supported native multipart attachment form.
Downloads use validated-UUID links on the configured public Forgejo origin and
its own browser authentication; this is a labeled native dependency. No arbitrary
provider download URL or cookie is forwarded.

Structured templates, full comment-attachment/reaction detail and complete native
journey/permission coverage remain pending. Connected source and authored tests
are not U10 completion.

## Pull requests (U11 source)

Explicit `/pulls` list/create, `/{index}` detail, `/commits`, `/files`, `/diff`,
`/reviews`, `/reviewers` and `/merge` routes use acting-user repository scopes.
Issue/PR conversation uses the same issue-scoped comment API and React component.
`/commit-status/{sha}` lists native base-repository head statuses, not an invented
aggregate of all protection/review requirements or a CI scheduler.

Diff/files require full `head`, `base`, `merge_base` query SHAs checked before and
after retrieval. Diff is inert UTF-8 text up to 1 MiB; oversized/unsupported input
fails explicitly, never as a successful truncated review model. Review POST
requires the same snapshot and binds native `commit_id` to the displayed head.
Optional inline comments use `{path,line,body}` where line is a native new-side
file line number. Old-side mapping and existing inline-thread rendering remain
unfinished, not falsely presented as supported parity. Native submission may
leave pending comments after failure; no automatic retries are made.

Merge requires explicit snapshot and native strategy; native `head_commit_id`
preconditions are retained. Force merge, auto-merge and branch deletion are fixed
false, unknown override fields rejected, and there is no environment mutation.
Native 405 state/settings rejection maps to 409 `provider_operation_unavailable`.
Forgejo owns mergeability, protection and execution; UI reloads native results.
Authored tests are unexecuted and U11 acceptance remains pending.

## Repository webhooks (partial U12 source)

Explicit `/hooks` GET/POST and `/hooks/{hook}` GET/PATCH require acting-user native
repository scopes/permissions. Responses expose only metadata, decimal ID and
configured-value booleans: no URL (which can contain query credentials), decrypted
authorization, signing secret or provider metadata/config object. New hooks use
native Forgejo JSON payloads; no system-hook flag, delivery-test or replay endpoint
is exposed. HTTP(S) targets exclude embedded userinfo/fragments and Soda never
requests them; Forgejo owns delivery and its configured host restrictions.

PATCH reads the current authorized native hook and retains omitted target,
authorization, events and branch filter for this request only. This is necessary
because pinned upstream unconditionally replaces some omitted fields. The UI has
explicit header replacement/clear intent, never a secret readback. Native edits
remain last-write operations, not fabricated atomic versioned writes.

Pinned PATCH does not rotate signing secrets or change package/action event flags.
Unsupported rotation input is rejected; changing those flags fails explicitly,
as does silently turning an eventless hook into the native default push hook.
Native settings remain the labeled dependency for these gaps pending U17 decision.
Advanced organization/team detail coverage and all execution evidence remain pending.

`/settings` GET/PATCH exposes bounded native description/website/visibility/default
branch, feature and merge-method fields only. Pointer patches preserve omitted
values; React sends only changed fields. Rename, transfer, archive and delete are
not accepted. `/collaborators` lists native users; `/{login}` inspects native
permission or delegates PUT/DELETE direct collaborator changes. `/deploy-keys`
lists/adds Git public keys with explicit read-only intent; individual DELETE
removes only the native deploy key. No Linux key/account operation occurs.

`/branch-protections` and `/tag-protections` list/create with fixed individual PATCH
routes. Native branch/tag PATCH was inspected: omitted fields are retained;
otherwise-ignored nested push flags require their explicit enabling parents.
Identifiers are opaque/lossless, rule path segments reject dot traversal, and
native organization team/user resolution remains upstream. Basic connected forms
exist; advanced branch allowlist/file-pattern form details remain pending. Native
423 locked-resource and 405 state/settings denials map to explicit 409 rather
than a fabricated successful change. Authored focused cases remain unexecuted.

## Organizations and teams (U12 source)

Organization-scoped acting-user routes cover `/organizations` list/create (native
`mine` filter), `/{org}` profile GET/PATCH, `/members` and `/teams`. Team routes
use the native decimal team ID, not a caller-declared Soda role or organization
membership record: `/teams/{team}` GET/PATCH, `/members` list with explicit username
PUT/DELETE, and `/repositories` list with explicit owner/repository PUT/DELETE.
Forgejo resolves real team/organization/repository authority on every request.

Organization PATCH retains omitted profile strings only in the current request
because pinned upstream otherwise clears them. Explicit email changes retain
native email semantics. Team APIs expose bounded native permission/unit fields;
otherwise-ignored repository-policy edits require explicit parent permission.
Unknown units, empty ignored override maps and limited overrides of native admin
units are rejected. These are protocol constraints, not a Soda permission engine.

Connected directory/profile/team/member/assignment forms exist; advanced team
policy editors remain unfinished. Native last-write semantics remain explicit.
No rename/delete/org-to-environment mapping or Linux offboarding synchronization
is implemented. All focused tests and native permission matrices remain unexecuted.

## My work, search, notifications and native profiles (U13)

`GET /api/forgejo/work` delegates bounded native issue/PR search and personal
assigned/created/mentioned/review-requested/reviewed booleans. Personal identity
comes from the acting grant, never a caller `uid`. `GET /repository-search`
uses native visibility/search/pagination; optional owner names resolve through
native user lookup before the exclusive owner filter. Missing owners fail closed.

`GET /notifications` lists the acting user's inbox with native all/page filters;
`PATCH /notifications/{id}` accepts read/unread/pinned. Native permissions are
retained and the native 205 acknowledgement becomes Soda 204 before authoritative
reload. Subject navigation uses fixed Soda repository routes, extracting only a
canonical subject number, not following or exposing a provider-supplied URL.

`GET /users/{login}` and `/activity` expose bounded native public-profile fields
and native permission-filtered activity, with optional date and pagination. No
email/auth-source/private security fields or remote avatar requests are added.
Activity payload text remains inert; this is not advanced graph parity. Overview
panels fail independently; superseded reads abort and discard late results.

Local Go/UI/type/build checks passed; installed two-user native proof is pending.

## OAuth, credentials and migration

`/login?return_to=%2Fapp%2F` binds `/app/` to single-use OAuth state; the legacy
return is `/projects`. Other return values are rejected. The callback remains
`/oauth/callback`. Default requested consent is `write:user write:repository write:issue write:organization write:notification`;
`administration=1` additionally requests `write:admin`, without conferring native
administrator status. Reads accept corresponding read scopes.

The selected Forgejo token response omits scopes. Soda introspects the returned
token with its confidential client, verifies active subject/audience, and stores
**actual** scopes. Existing native confidential-client consent can be reused
without expansion; the UI explains native grant revocation and fresh consent.

Schema v3 adds encrypted session grants and a key-check ciphertext, preserving
all v1/v2 product records. AES-256-GCM binds each grant to its hashed session and
provider user ID. Access/refresh/scopes/expiry are encrypted. The separately
provisioned `grant_key_file` is mandatory in production and is validated before
DB migration; a wrong existing key fails closed. Refreshes are serialized per
native user within the single dashboard process. Grant replacement is update-only
and checks a still-live session; logout wins over in-flight refresh. Ambiguous
refresh failures discard the local grant using bounded independent cleanup;
provider writes are not replayed.

Forgejo has one native grant per user/application, not per Soda session. With
native refresh-token invalidation enabled, another login/session may invalidate
an older session's refresh token. That session must reauthenticate; Soda neither
copies grants between sessions nor changes the native setting. Local logout does
not revoke Forgejo consent or existing SSH sessions.

Legacy Soda sessions remain valid for their existing Soda-local records but have
no provider grant and receive JSON reauthentication errors on provider operations.
See [credential migration and rollback](dashboard-credentials.md). These changes
have not touched the VM or established U04/U08 installed evidence.
