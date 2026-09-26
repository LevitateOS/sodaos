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
  and protected adapter sessions/grants, plus factory execution identity and
  run-scoped authority.
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

## Factory authority boundary

The factory runs only for a trusted team on one operator-managed
appliance. Agent instructions and repository content are untrusted inputs even
there. An agent may work in its assigned disposable workspace, but its identity
does not confer host, Soda operator, Forgejo administrator or merge authority.
Soda mediates Forgejo writes using the admitted work and acting authority; agents
must not receive a general Forgejo write token or bypass native repository
permissions. A fresh reviewer receives a separate execution identity and review
authority, without the worker's conversation, writable environment or publication
authority. Human review and merge remain Forgejo decisions.

Reusable subscription credentials are a narrow exception for private trusted work.
Keep them restricted to the assigned execution boundary and out of source, logs
and retained artifacts. A provider credential stream may be reused serially across
runs, with credential state maintained by its supported CLI. Deleting a workspace
does not revoke a provider credential; provider revocation is a separate operator
action. Code in the trusted runtime may extract its injected provider credential.
Containers share a kernel, and an allowlist does not prevent exfiltration through
an allowed service.

Keep four identities distinct: the authorizing human, the principal of one Soda
run, the Forgejo actor publishing its candidate or findings, and the provider
account supplying model access. A role is a policy template, not authority an
agent can grant itself or its subagents. An issue or label is task context rather
than an execution grant.

The [factory operator interface](../reference/factory.md) records admitted inputs
and checks active authority for publication. Cancellation or closed work withdraws
further execution and publication. New commits invalidate earlier CI and review
evidence. Cleanup operates only on recorded run-owned resources; persistent human
Projects retain their separate authority and lifetime. A fresh review boundary
constrains operations without guaranteeing independent reasoning or correct code.

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

The persistent workspace outer document is the Soda HTML shell entered at
`/workspace`. The shell path itself (`/-/soda/<framed-path>`) is an untrusted
same-origin Forgejo locator for the iframe, not an OAuth return URL and not a
Soda API path. Signed-in Forgejo
top-level pages wrap into that host with an admitted frame path. Login, logout,
signup/activate, password recovery, two-factor/passkey, provider OAuth link,
consent, callback, install, failed `soda-connect`, and any `soda-view` host
(valid, unknown, or duplicate) stay outside the shell. If a framed document lands
on those, the host replaces itself with that same-origin URL. When the frame
cannot be read (initial, cross-origin, or failed load) the host leaves the address
bar untouched. The shell does not copy credential query into `to`. The shell may
frame same-origin Forgejo; it does not embed credentials or select a
separate-origin trust model.

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
