# Forgejo customization contract

Soda customizes stock Forgejo through supported template hooks, assets and APIs.
There is no Forgejo fork, source patch set or custom Forgejo executable.

Product surface: [Spaces](../product/spaces.md). Trust boundary:
[Trust](../architecture/trust.md).

## Image-owned presentation

Custom templates and assets ship with the Forgejo appliance image payload. Bundle
admission allows only the Soda template paths and required ancestors, with both hooks
and assets present and readable. First-install preflight refuses occupied hook/asset
destinations before writes, including symlinks and special files.

Do not deploy an unchanged copy of the upstream template tree. Prefer dedicated
template hooks; use targeted overrides only when necessary.

## Shared presentation components

Spaces mounts Lit components into Forgejo's real document for the native drawer
and dashboard/admin Soda views. The persistent workspace host is a dedicated Soda
HTML document that frames native Forgejo same-origin; it does not replace Forgejo
handlers, cookies or chrome. Native Forgejo retains header, profile menu, forms,
routing, notifications and beforeunload behavior inside that frame.

Framed Forgejo documents must not mount a nested Spaces drawer. Signed-in top-level
Forgejo documents wrap into `/-/soda/workspace` except auth/recovery flows,
login/OAuth/install, failed `soda-connect`, and any `soda-view` host (valid,
unknown, or duplicate — the dashboard answers those). Product host contract:
[Spaces](../product/spaces.md).

Responsive workspace rules:

- Desktop drawer begins near half width with a measured native-left minimum.
- Compact projections switch Forge/Terminal surfaces deliberately.
- Preserve merged native compact repository header/action disclosure rather than
  inventing schematic chrome.

## Namespace and cookies

- Browser origin is the configured `forgejo_url`.
- Soda API/login/callback paths mount at `/-/soda/`.
- Caddy forwards that prefix unchanged; Go rejects unprefixed aliases and
  encoded-path tricks.
- Soda uses its own path-scoped cookies and CSRF token with exact Forgejo Origin
  and same-origin fetch metadata.
- Native Forgejo cookies/CSRF are not Soda substitutes.
- `fetch('/api/...')` targets Forgejo, not Soda. No permissive CORS.

## Actor and repository context

Native templates expose signed-user and repository identity. Use escaped template
data attributes and keep IDs as strings in JavaScript. Repository-scoped reads and
join authorization check the acting grant's subject, consent and native visibility.

`X-Soda-Expected-User-ID` is a consistency guard for page/session alignment, not
proof of the live native browser session. OAuth returns use repository IDs resolved
through the acting grant, never caller-supplied redirect URLs.

## Integration limits

- A template cannot call Soda's database directly.
- An OAuth grant is not a native web session; different ports do not isolate cookies.
- Do not proxy the entire `/-/` namespace; Forgejo already owns other `/-/` routes.
- If stock configuration, templates, assets or APIs cannot meet a requirement,
  document the concrete constraint. Do not scrape, relay HTML, borrow credentials
  or introduce a replacement frontend.

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

- Templates/assets: `appliance/forgejo/`
- Browser hooks: `assets/branding/forgejo/`, `frontend/spaces/`, `frontend/runners/`
- Presentation checks: `scripts/*forgejo*`, `tests/forgejo/`
