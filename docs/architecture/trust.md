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
  and product grants, plus factory execution identity and run-scoped authority.
- **The installed Soda extension** carries only the live native actor and bounded
  request authority needed by the current browser operation. The Soda service
  checks its product policy and does not receive Forgejo cookies or session IDs.
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
there. People authorize the repository's factory policy and the account access it
uses. Eligible issues then proceed without a separate human admission or merge
decision for every issue. Who may establish that policy and sponsor each authority
remains a decision identified in the [product contract](../product/overview.md#rules-to-settle-before-implementation).

An agent may work in its assigned environment, but its execution identity does not
confer host, Soda operator or Forgejo administrator authority. Soda mediates native
writes and automatic merge within the authorized repository scope. Agents cannot
grant themselves broader access, receive general Forgejo write authority or bypass
native repository policy. Issue content, comments and labels cannot enlarge the
factory's configured authority.

Review is assigned to another agent session with its own execution identity in
the same repository container, as required by the
[factory lifecycle](../product/overview.md#software-factory-workflow). Distinct
assignments do not make a shared writable environment a security isolation boundary
or guarantee independent reasoning. Review context, checkout separation and the
division of correction work remain to be designed; the older fresh-container rule
does not decide them.

Reusable subscription credentials are a narrow exception for private trusted work.
Keep them restricted to the assigned execution boundary and out of source, logs
and retained artifacts. A provider credential stream may be reused serially across
runs, with credential state maintained by its supported CLI. Deleting a workspace
does not revoke a provider credential; provider revocation is a separate operator
action. Code in the trusted runtime may extract its injected provider credential.
Containers share a kernel, and an allowlist does not prevent exfiltration through
an allowed service.

Keep four identities distinct: the authorizing human, the principal of one Soda
run, the native actor publishing a candidate, review or merge, and the provider
account supplying model access. A role is a policy template, not authority an
agent can grant itself or its subagents. An issue may trigger work under an
existing policy; it is not an execution grant on its own.

Publication and merge require active authority. Revocation, cancellation or closed
work withdraws further execution, publication and merge authority. Review and CI
must cover the exact candidate; new commits invalidate earlier evidence. Native
repository state and policy must still permit the merge when it occurs. Cleanup
operates only on recorded run-owned resources; a run does not acquire ownership of
a persistent environment merely by using it.

The [factory reference](../reference/factory.md) describes how the existing
operator command enforces its narrower admission and publication model. It does
not impose manual admission, human merge or disposable containers on the target
product.

## Fountain consumption boundary

Fountain is a generic Forgejo extension platform. It owns native identity,
sessions, repository permissions and supported ways for extensions to act with
bounded authority. Soda owns its factory policy, environments and Identity Broker;
those product concepts do not belong in Fountain.

If Soda appears to need an authentication or native-authorization workaround,
first check the selected Fountain version, its supported contract and the actual
Soda caller. If correct consumption cannot satisfy the requirement, the missing generic
capability is a Fountain shortcoming to resolve there. Soda must not compensate
with borrowed cookies, synthetic native sessions, login relays, impersonation or
a parallel OAuth login. This applies to background factory operations as well as
browser contributions; background authority cannot depend on keeping a person's
browser session alive.

Provider enrollment and account custody remain legitimate Identity Broker duties,
separate from native forge sign-in. The concrete grant protocol and unattended
publication/merge integration must be settled against Fountain's generic contract
before implementation; this document does not assume that capability is already
available.

## Frontend and session boundary

Forgejo's small maintained extension layer owns native contribution routes,
session admission, scoped callbacks and one generic persistent browser host.
Installed extensions are administrator-trusted code: backend processes share the
Forgejo operating-system identity and browser assets run on its origin. Process
separation manages lifetime and failure; it does not sandbox extensions or make
their JavaScript safe to install.

Forgejo validates its native session, account state, request origin and core
permission before creating scoped extension authority. Cookies and raw session
IDs remain in Forgejo. A signed-in page, navigation context, repository shown in
the frame or opaque display binding does not grant an extension broader access.
Host callbacks name bounded operations and recheck current authority. Soda also
checks its own membership, configured operator identity, broker consent and
project grants. A Forgejo site administrator is not automatically the Soda
operator.

Multi-request flows use a host-verified binding to one native session generation.
It conveys continuity, not authentication. Logout, session regeneration or an
account switch invalidates the old flow; the next step cannot silently adopt the
new actor. Terminal streams retain authorization for their lifetime and close on
invalid session or lost repository/membership permission. Native state checks are
decisive; observations may accelerate cancellation but do not grant or preserve
access.

Bookmark handlers lead to fixed native views. Page loading, redirects and opening
the persistent workspace do not register a runner, create a terminal or change
project lifecycle state. Native Forgejo authorization protects each view; Soda's
operation rules still govern every private read and mutation.

The Identity Broker holds Codex and Muse subscription credentials and factory
grants. Forgejo account sign-in, WebAuthn and Git authentication remain native;
Soda does not hold Forgejo OAuth grants or mediate project/worker Git requests.
Factory publication uses separately configured actor credentials. See
[Credentials](../reference/credentials.md) and the [factory interface](../reference/factory.md).

## Host helper

The root:soda Unix-socket helper exposes fixed operations, not arbitrary commands,
host Podman flags or a generic forwarding surface. Resolve actor and native target
from trusted state. Existing Linux accounts, homes and permissions remain native
facts; report incomplete provisioning honestly rather than claiming success or
destructively recreating a reservation.

## Maintaining the Forgejo extension layer

The native extension layer adds a maintained Forgejo fork. Keep its source change
small, attributable to the pinned upstream LTS, and reproducible in the appliance
image. Track security and maintenance updates. Verify session admission, operation
permissions, callback bounds, package lifecycle and native routes against the
selected upstream version before updating its pin. Keep the runtime and Soda
service boundaries independent of this maintenance obligation.

Project provisioning, privileged host operations and terminal supervision have
their own Soda and host ownership. The Forgejo process and its trusted extensions
do not gain the Soda database, broker secrets or privileged helper sockets.
