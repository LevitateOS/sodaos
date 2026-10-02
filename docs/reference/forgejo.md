# Forgejo customization contract

Soda ships Fountain, a maintained Forgejo 15.0 LTS fork with a native extension
host. Fountain supplies the independently consumable extension SDK and loads
administrator-installed process packages. The Soda extension is built and packaged
separately; installing or replacing that package does not rebuild Fountain.

Product surface: [Spaces](../product/spaces.md). Trust boundary:
[Trust](../architecture/trust.md).

## Image-owned presentation

Custom templates and assets ship with the Forgejo appliance image payload. Bundle
admission allows only the Soda template paths and required ancestors, with both hooks
and assets present and readable. First-install preflight refuses occupied hook/asset
destinations before writes, including symlinks and special files.

Do not deploy an unchanged copy of the upstream template tree. Prefer dedicated
template hooks; use targeted overrides only when necessary.

## Native extension presentation

Fountain owns the native page and persistent-panel host. Soda declares the Spaces
page, operator-only Runners and Tailnet pages, and preferred Workspace panel in its
extension manifest. Fountain supplies each contribution's mount context and keeps
the workspace panel mounted while native Forgejo pages change. Forgejo continues to
own browser navigation, session and page lifecycle; the extension uses the live
native authority and narrowly scoped service API described in the
[API contract](api.md). Product behavior and responsive layout live in
[Spaces](../product/spaces.md).

The Soda extension build bundles its browser modules, styles and terminal runtime
into the independently installable package. The Forgejo image payload carries
image-owned branding/templates, not a second public copy of extension UI assets.
Installing or replacing Soda therefore does not rebuild Fountain or update the
host image.

Add product UI through the declared native page or panel contribution, and retain
normal Forgejo navigation and form behavior.

## Namespace and cookies

- Browser origin is the configured `forgejo_url`.
- The browser reaches Soda product operations through the native extension
  service. The public Soda namespace serves avatars; it has no broker login
  callback. See [API](api.md).
- Product requests use the extension's supplied API base and same-origin browser
  credentials. Soda does not issue a separate browser login or cookie.
- Native Forgejo session and extension authority are validated by Fountain and the
  live SDK callback. A direct `/api/...` fetch targets Forgejo, not Soda.
- No permissive CORS or broad `/-/` proxy is used.

## Actor and repository context

Fountain supplies the extension's actor, contribution and session-generation
context. Browser-displayed IDs and generation values are labels or consistency
checks, never operation authority. Soda rechecks the current native actor and
repository permission for protected work through the SDK. The broker's separate
account custody and factory grants remain documented in
[Credentials](credentials.md); they are not browser-session substitutes.

## Integration limits

- A template cannot call Soda's database directly.
- A broker or factory grant is not a native Forgejo web session.
- Do not proxy the entire `/-/` namespace; Forgejo already owns other `/-/` routes.
- If stock configuration, templates, assets or APIs cannot meet a requirement,
  document the concrete constraint. Do not scrape or relay HTML, borrow credentials,
  or bypass the native extension host with a parallel shell.

## Username compatibility

Project OS cannot adopt colliding system accounts. Prefer preventing incompatible
new Forgejo usernames through supported upstream mechanisms or Soda-side account
integration. Existing users keep login and repository access. Silent remapping of
Linux memberships is out of scope.

## Visual identity

Forgejo customization follows the Soda brand: neutral light/dark surfaces, red
actions, square controls, Barlow Condensed headings, Barlow body and IBM Plex Mono
controls. Prefer text-led intros and restrained functional empty states over
decorative photography. Native Git status/diff colors, avatars, organization logos
and repository content retain their upstream owners.

Brand assets: [Branding](../design/branding.md).

## Source owners

- Fountain host and standalone SDK: the separate canonical Fountain repository
- Soda extension package: `appliance/soda-extension/` and its backend entry point
- Templates/assets: `appliance/forgejo/`
- Extension browser sources: `frontend/spaces/`,
  `frontend/tailnet/`; packaged by `scripts/build-soda-extension.ts`
- Presentation checks: `scripts/*forgejo*`, `tests/forgejo/`

The candidate producer accepts a clean exact-revision Fountain checkout, archives
that revision before generation or compilation, and builds its patched executable
from the extracted archive. Source and notice obligations are recorded in
[licensing](../research/licensing.md). Upstream ownership and the reproducible
maintenance procedure are defined in the
[release workflow](../development/release.md#fountain-upstream-maintenance).

## Factory API boundary

The factory integration uses stock issue, pull-request and review APIs. Review
submission includes the assigned `commit_id`; Soda must verify current run
and target authority before submitting it. The client exposes no factory merge
or commit-status write operation.

CI observations use Forgejo Actions run records filtered by `head_sha` and
`workflow_id`. The returned `commit_sha` and workflow must match the requested
candidate. A successful agent process or agent-authored status is not CI evidence.
