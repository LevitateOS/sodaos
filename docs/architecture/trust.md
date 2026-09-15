# Trust model

This document owns authority, identity and privilege boundaries for Soda OS.

## Distinct authorities

| Authority | Grants |
| --- | --- |
| Host root | Host OS, Podman, systemd, destructive appliance operations |
| Configured Soda operator | Appliance runner/Tailnet settings; recorded Forgejo user ID from setup |
| Forgejo site administrator | Forgejo administration |
| Organization owner/admin | Organization Forgejo authority only |
| Repository owner | May create that repository's project; not host access |
| Project member | Project-local Linux account after confirmed Join |

These authorities are not interchangeable. Owning a repository does not grant
appliance access. Arbitrary site or organization administrators are not project
root. Setup tokens are not ordinary acting-user credentials.

## Identity split

- **Forgejo** owns passwords, factors, native sessions, permissions, Git credentials
  and collaboration.
- **Soda** owns preferences, development-access public keys, environment membership
  and protected adapter sessions/grants.
- Native Git keys and collaboration remain upstream-owned.
- Soda does not maintain a second password, provider-role inventory or CI scheduler.

Stable Forgejo identity binds project membership to its original Linux login.
Native rename or transfer does not silently remap Linux users, ownership or already
installed keys. Web administration and native wheel/SSH rights are not automatically
synchronized on transfer; see [Project OS](../reference/project-os.md).

## Runtime identities

Soda service UID/GID values and native runner accounts are not developer host
onboarding. The web process's privileges and human authorization are separate
boundaries.

## Frontend and session boundary

Use stock Forgejo's native handlers, forms, scripts/styles and authentication, with
supported template hooks first and necessary targeted overrides second.

**No downstream Forgejo fork, source patch set or custom Forgejo executable.** If
supported integration cannot meet a requirement, document the actual constraint and
return for an explicit product decision. A convenience feature does not justify
taking ownership of building, shipping and maintaining modified Forgejo through
upgrades.

Do not substitute scraping, an HTML relay, borrowed cookies or weakened native
security. Same-origin composition (including a deliberate Forgejo iframe inside a
stable Soda workspace) does not select arbitrary embedding or a separate-origin
trust model.

### OAuth and adapter sessions

Keep OAuth state/PKCE/callback binding, encrypted session-bound grants, serialized
refresh, logout-winning persistence, request/response bounds and CSRF/origin checks.

- Native OAuth tokens are not native web sessions.
- Different ports do not isolate cookies.
- Root redirects to configured native Forgejo home.
- OAuth may return to a repository resolved by stored ID through the acting grant,
  never a caller-supplied URL.
- Soda's expected-user header guards page/session consistency, not native browser
  session authenticity.
- Native WebAuthn origins/RP-ID, session revocation and Git protocols stay
  upstream-owned.

Bookmark handlers redirect only to fixed native views. Private collection and
operation authority stay in protected APIs. Page loads and redirects never register
a runner, create a terminal or change project lifecycle state.

## Host helper

The root:soda Unix-socket helper exposes fixed operations, not arbitrary commands,
host Podman flags or a generic forwarding surface. Resolve actor and native target
from trusted state. Existing Linux accounts, homes and permissions remain native
facts; report incomplete provisioning honestly rather than claiming success or
destructively recreating a reservation.

## No-fork consequences

The no-fork boundary shapes integration: query-selected dashboard bodies, template
overrides, separate Soda OAuth sessions, coordinated logout and API-based identity
reads. Any future fork proposal must evaluate those existing costs together with
unmet requirements, including upstream tracking, security updates, packaging and
regression testing. Patch size does not measure ongoing commitment.

Project provisioning, privileged host operations and terminal supervision have
useful isolation independent of the no-fork constraint. Preserve those unless a
separate justification establishes otherwise.
