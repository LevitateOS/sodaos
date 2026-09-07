# Native-backed Soda authentication — H05 delegation contract

**The headless transport design below is superseded as a required implementation.**
The user selected [official template overrides](dashboard-implementation-plan.md#selected-direction--official-template-overrides-partial-u01-invalidation)
with native Forgejo authentication/rendering. Preserve the native findings and
security boundaries; do not implement the proposed native session-handle APIs
merely to replace pages that will now use native templates. U01 acceptance is
withdrawn; secure integration with Soda's environment sessions still needs review.

**Corrected after the user's authority-boundary clarification.** Extend Forgejo's
interfaces; do not replace its authentication, account, consent or revocation
rules. This replaces the policy-prescriptive draft at `1275c88`, retained in Git
history. No native auth endpoint or protocol is implemented by this document.
U04 owns adapter/session integration, U05 account presentation and U16 native
administrator presentation. The [single register](forgejo-api-coverage.md) retains
AU01–AU22/AD15/AD18/AD22 and their conditional workflow requirements.

## Native constraints — retain the existing research

The selected v16.0.3 source establishes these integration constraints:

- `routers/api/shared/middleware.go::verifyAuthWithOptions` rejects inactive,
  prohibited, forced-password-change and required-but-missing-MFA actors. More
  scopes or a different user's token cannot complete those security gates.
- `routers/web/auth/{auth,password,2fa,webauthn,oauth}.go` contains native login,
  password change, challenge, consent and successful-session behavior. Extract
  necessary non-rendering operations into shared native services with existing
  web callers retained; do not copy their decisions into Soda.
- `services/auth/method/basic.go` rejects WebAuthn users. User Basic-to-PAT,
  administrator minting and borrowed Forgejo cookies are not substitutes.
- `modules/auth/webauthn/webauthn.go::Init` uses `setting.Domain` for RP-ID and the
  origin of `setting.AppURL` for allowed origins. Existing assertions are native
  second-factor flows, not evidence of passwordless/discoverable-passkey support.

These findings do not require another inventory. They require narrow native API
work and real authority/protocol tests where suitable interfaces are missing.

## Delegation, not a second authentication system

| Responsibility | Owner / conforming behavior |
| --- | --- |
| Password acceptance, validation, hashing and account eligibility | Forgejo's native verifier, account configuration and services |
| Required password change, activation and MFA/enrollment/recovery | Forgejo determines the required steps, validates proofs and changes its records |
| Challenge validity, credential counters, proof consumption, lockout and native abuse policy | Forgejo's native mechanisms; no independently chosen Soda thresholds or substitute account state machine |
| External identity-provider selection, authentication/linking and callback validation | Forgejo's configured integrations and native protocol behavior |
| OAuth client consent, scope approval, code/token issuance, expiry, refresh and revocation | Forgejo; an adapter's confidential-client credential does not authenticate a human |
| Soda browser cookie, CSRF, PKCE/state binding, protected grant storage and UI cleanup | Soda's legitimate web-adapter responsibilities; never evidence of native permissions |
| Upstream security rules/fixes | Forgejo upstream |
| Security of Soda's code and any carried native patch, compatibility tests and consuming upstream fixes | Soda maintainers; this does not transfer upstream policy ownership to Soda |

An extension may carry native pending-authentication/challenge state over a new
interface when required. That state must be governed by the same native operations
and policy as existing callers. Do not require a new transaction database, custom
factor lifecycle or independent grant engine merely because the browser is Soda.
Use the smallest explicit carrier the native implementation needs, with client/
browser/challenge/operation binding, one-use completion and secret-safe transport.
Review its exact native caller/owner before implementing it; no generic method proxy.

The earlier mandatory transaction schema/routes/state enum and independently
chosen ten-minute/five-proof/account-lockout quotas are **withdrawn as selected
requirements**, not moved to a later milestone. They were not native semantics
established by the audit. If a genuinely new transport needs an expiry or admission
bound, justify that specific transport control without changing native password,
factor, account or consent policy. A discovered upstream security gap needs a
narrow native fix/review, not an invented Soda policy engine.

Soda must still bound HTTP bodies, output and concurrent adapter work, reject
malformed input, validate callback/CSRF/PKCE bindings, protect secrets and cancel
work safely. Those controls protect the adapter; they must not impose unrelated
Linux username/password rules on Forgejo identities or pretend to grant authority.

## Native flows and Soda presentation

1. Begin using the native configured sign-in method and registered client/return
   context. Validate the browser binding without accepting a supplied native UID.
2. Submit transient credentials/proofs to the native operation. Native results
   determine whether password change, activation, MFA, enrollment, recovery or
   consent is needed. Never obtain an ordinary grant by bypassing required steps.
3. Render the supported native step in Soda. Preserve native auto-approval and
   configured policies; do not manufacture consent prompts or skip required ones.
4. Use native code/PKCE exchange and native refresh. Verify actual native subject/
   audience/scopes before binding the grant to a Soda session. Continue existing
   encrypted storage, serialized refresh and logout-winning persistence behavior.
5. Expose native security actions through operation-specific adapters. Preserve
   their native reauthentication, authorization and effects. Do not replay a lost
   password/MFA/recovery/consent mutation or infer rollback from an HTTP failure.

Forgejo-owned pages remain in Soda. A configured external IdP's own authentication
interaction is part of the upstream integration, not permission to expose Forgejo's
frontend or a new Soda choice of authentication policy. Preserve the native method
and safely return to Soda through its reviewed callback adapter. A concrete
callback/origin incompatibility is an implementation issue to resolve, not a
reason to ask the user whether to redesign OAuth or silently disable the method.

Likewise, inspect and preserve native sign-out/session/grant-revocation semantics.
Soda must clear its own browser session/private UI state when signing out. That
cleanup neither replaces a requested native revocation nor authorizes an additional
revocation affecting other sessions. Labels must distinguish the actual native
actions and effects. Do not offer competing Soda logout policies for selection.
Linux SSH sessions, keys and workloads are not browser sessions; no automatic
Linux offboarding follows from an authentication operation.

## Origins and enrolled-key preservation

Exact browser origins are per-installation operator inputs, not one global product
hostname. Preserve the installation's native RP-ID and credential validity; keep
browser, internal API and Git advertisement endpoints distinct. A required native
origin/configuration extension must continue native RP-ID/origin validation and
must not accept browser-selected trusted origins. Reject an incompatible origin
change rather than resetting enrolled keys or weakening verification.

U17/U18 must inspect the exact retained-target configuration, prove existing-key
assertions and rehearse callback/ingress behavior before an approved change.
No live RP-ID/configuration/credential inspection or mutation is claimed here.
Keep working OAuth/ingress until its conforming replacement passes; this is not a
permanent Forgejo-page fallback or permission to strand users.

## Linked identities and retained project access

Forgejo owns repository ownership, permissions and transfer/accept/reject rules.
Soda follows the native result through stable IDs; it does not ask the user to
choose a competing ownership policy or maintain a second provider-role inventory.

| Native change | Required Soda integration |
| --- | --- |
| User/repository rename | Verify the stable ID and refresh cached names/routes. Do not attach a reused name to the previous identity. Preserve the stored project-local Linux login/home. |
| Repository transfer | Resolve current native ownership and the relevant native authority for the associated operation. Do not preserve the old creator's administrative access solely through a cached owner ID. Organization ownership needs a conforming native authority lookup, not an invented successor, copied role list or blanket site/org/repository-admin elevation. |
| Account disabled/deleted or grant revoked | Honor native authentication/authorization outcomes; invalidate the appropriate Soda session/grant on confirmed invalidation. Do not turn a network failure into account deletion or assume every 401 proves global revocation. |
| Repository lookup denied/unavailable | Preserve the stable association and report the actual error. A 404 alone does not prove deletion or authorize cleanup/reattachment. |
| Native mutation succeeded but local association/cache persistence failed | Report native success plus the bounded Soda failure. No automatic replay, inverse native mutation or invented rollback. |

Soda legitimately owns environment associations, memberships and native account
provisioning. Those records do not override native identity/ownership, and Git
permissions are not themselves host-root authority. Preserve project roots,
project-local accounts/homes/development keys and workloads. Forgejo retains its
own effects on personal Git keys; do not promise Linux-key revocation from it.

**Concrete correction:** `apiCreateEnvironment` snapshots the creating human into
`Project.OwnerID`; `apiEnvironment` and `apiEnvironmentMembers` later use that ID
as authority. U07 with U04/U12 must replace reliance on that snapshot with current
trusted native authority and stable-ID checks. Preserve the record/data rather
than deleting environments. New organization-owned creation remains unfinished
source support, not justification to invent native transfer rules or waive the
required transfer workflow. Define the exact native predicate from the selected
interfaces before coding; do not equate every administrator role.

For new privileged Soda-only access/key/provisioning/terminal operations, cached
membership does not prove a native account/grant remains valid. Honor current
native validation and fail new privileged work closed on uncertainty without
removing data. U07 separately owns the browser terminal's authenticated connection
lifetime; existing SSH/workload processes are not silently terminated.

## U01 closing review — concrete native integration disposition

Reviewed the prepared v16.0.3 source after `8727233`. **The shared-service direction
is ready for implementation; stock bearer APIs alone are rejected.** This is a
source/design verdict, not a passed headless login or approval to expose an
untested patch. The first read contract is independently field-defined in the
[read contract](forgejo-read-contracts.md). No second auth policy is selected.

### Exact native owners and extraction boundaries

| Operation | Selected native owner / implementation instruction |
| --- | --- |
| Password sign-in | `routers/web/auth/auth.go::SignInPost` calls `services/auth/method.UserSignIn`, observes native CAPTCHA and `EnableInternalSignIn`, honors the source's `LocalTwoFASkipper`, and stores native `twofaUid`/remember/link state before challenges. Extract those non-rendering decisions together, not just password verification. |
| Successful authentication versus normal grant eligibility | `handleSignInFull`/`updateSession` perform native session regeneration, cleanup, locale and last-login effects. `routers/web/web.go::verifyAuthWithOptions` subsequently enforces activation/prohibition, forced password change and required MFA. A native `uid` is therefore **not** proof that an ordinary OAuth grant may be issued. Share these gates with the new native caller rather than rewriting or bypassing them. |
| Forced password / recovery | `password.go::MustChangePasswordPost` and `ResetPasswdPost` call `services/user.UpdateAuth`; native minimum/complexity/pwned checks and authorization-token consumption stay there. `commonResetPassword` checks target identity and token purpose. Preserve native reset-token consumption order and honest failed/uncertain outcomes, not a Soda promise of reusable tokens or rollback. |
| TOTP / scratch | `2fa.go::TwoFactorPost`/`TwoFactorScratchPost` validate through `models/auth.TwoFactor`, update native last-used/scratch state and finish native sign-in/linking. The current read/validate/update sequences are not proof of atomic concurrent consumption. Add native race/regression tests and any required shared-native atomic correction; serializing one Soda browser alone is insufficient. This is a concrete security review item, not a new attempt/lockout policy. |
| WebAuthn | `webauthn.go::WebAuthnLoginAssertion`/`WebAuthnLoginAssertionPost` use native session-bound `SessionData`, `BeginLogin`/`ValidateLogin`, stored-credential lookup, legacy credential migration, clone detection and sign-counter persistence. Extract the complete operation and keep its cleanup/linking/native RP-ID/origin behavior. Do not validate the response in Soda. |
| Consent / code / tokens | `oauth.go::AuthorizeOAuth`, `GrantApplicationOAuth`, `AccessTokenOAuth` and native `OAuth2Application`/`OAuth2Grant` operations own registered redirects, client validation, consent, codes, PKCE, exchange and refresh. Return typed native outcomes instead of HTML/redirect presentation. An existing confidential-client grant is reused; the native POST rejects a differently scoped existing grant. Do not silently enlarge scopes or revoke/recreate grants to make a request pass. Present native authorized-app management when its explicit action is required. |
| Logout / revocation | `auth.go::HandleSignOut` flushes/destroys the native session and clears remember/redirect cookies; `SignOut` also emits the native logout event. It does not delete OAuth grants. `models/auth/oauth2.go::RevokeOAuth2Grant` separately deletes by grant ID **and user ID**, reached through native settings handlers. Preserve both operations' distinct effects; Soda clears its own session/private UI too. No invented IdP/global logout or Linux termination follows. |
| External methods and conditional security | Existing `oauth.go`, `openid.go`, `linkaccount.go`, registration/activation and native security-setting callers remain the owners. Extract callback/link/CAPTCHA/mail/enrollment operations as their U04/U05/U16 batches are implemented; do not fork external-account linking or render provider HTML. H01 already assigns all conditional actions. |

### Selected transport boundary, not a new account state machine

Use an explicit **native-issued, confidential-client-bound session handle** to
carry native pending-auth/session context between the Soda Go adapter and the new
native operations. It is not a human bearer grant. Reuse Forgejo's configured
session storage/lifetime/regeneration/destruction mechanisms (`modules/session`,
`routers/common/middleware.go::Sessioner`, `auth.go::updateSession`) through a
small native transport adapter. Do not create another transaction database or
reassign native account/factor lifetimes. The existing middleware is cookie-bound:
it cannot simply be attached to a new token route and declared headless.

Concrete constraints for U04's native patch:

- Issue a fresh native context through an operation-specific interface after
  validating the registered confidential client and return/PKCE request. Bind the
  carrier to that client and Soda's pre-session/browser nonce. Neither a caller's
  UID nor an existing web-session cookie may initialize human authority. Reject
  regular bearer-token, cookie and Sudo substitution on this channel.
- Store native pending identity/challenge facts only in Forgejo. The adapter owns
  only the protected handle association and browser/session binding; React sees
  public native next-step data and transient form fields, never native handles,
  tokens, passwords in persisted state or session-store keys.
- Use fixed typed credential, password, TOTP, scratch, WebAuthn begin/finish,
  consent, status and sign-out operations. Their input fields are the respective
  existing native forms/protocol payloads, not a generic operation/field bag.
  Native code determines the allowed next operation on every request; UI labels
  and cached outcomes do not authorize it. Define each wire DTO alongside its
  extracted native operation/test, without another architecture proposal.
- Preserve native session regeneration and expiry; a rotated handle must remain
  bound to the same client/browser and invalidate its predecessor. Do not emulate
  regeneration merely by changing a Soda cookie. Persist the native result before
  returning success; do not replay a lost mutation/rotation. Status may report a
  native outcome but cannot synthesize a missing grant or bypass native gates.
- Reuse native code issuance only after the shared native eligibility/consent
  checks. Go retains its existing S256 exchange, actual subject/scope verification,
  grant protection and logout-winning behavior. A client secret is application
  authentication, never permission to name/impersonate an arbitrary user.
- Keep native abuse/CAPTCHA behavior and bounded secret-safe HTTP transport. Before
  exposing each endpoint, test native session-store failure, rotation, cross-client
  binding, challenge replay and direct calls that skip the preceding native step.
  Newly exposed pre-auth transport needs its own admission protections, not a
  newly invented account password/factor/lockout policy.

This resolves the U01 architecture choice: the necessary work is a bounded native
session-transport adapter and shared service extraction, not an upstream-policy
redesign. Native storage/rotation integration and exact operation DTOs are real
U04 implementation/test work, not existing capabilities advertised by this review.
Keep `auth: 0` until the implemented contract and its native tests justify changing
it. In particular, source inspection does not prove concurrent proof consumption.

### Ownership lookup disposition

The selected source already has `GET /repositories/{id}` and
`GET /users/{username}/orgs/{org}/permissions` (`org.go::GetUserOrgsPermissions`).
The latter derives `is_owner` from native `GetOrgUserMaxAuthorizeLevel`; it is
not `is_admin` or `can_create_repository`. U07/U12 should reuse these interfaces:
resolve the stored repository ID with the acting grant, verify the current user/
organization stable identities, and query the acting user's native organization
ownership where required. Never accept a caller-selected username/org as authority
or enumerate/copy teams. Require `read:repository` for repository access and the organization-permissions
route's `read:user` plus `read:organization` and native access;
a denial is not permission to consult an operator credential.

For a human owner, compare the current repository owner ID with the authenticated
native subject. For an organization, use its native ownership result, not a generic
site-admin bypass or an invented designated-human successor. Verify names against
stable IDs around the lookup, fail new privileged requests closed on unavailable/
mismatched results, and preserve data. This is request-time Soda API authority,
not a promise to atomically coordinate Forgejo transfers with Linux processes or
automatically remap old sudo groups/keys. Existing Linux access limitations remain
explicit. U07 owns regression coverage and the three source corrections remain
pending in the leading plan; U01 does not perform those code fixes.

## Implementation and review tests

- Compare each exposed operation with its native caller/configuration: valid,
  inactive/prohibited, forced-password/MFA, native denial and revoked subjects.
- Test native proof/challenge replay, stale/cross-client/cross-browser submissions,
  factor counters, configured native attempt controls and sensitive reauth; do not
  substitute tests of a separately invented Soda account-policy implementation.
- Test login-CSRF/session fixation, exact origins/RP-ID, secret-free errors/logs,
  no cookie/credential substitution, cancelled/uncertain outcomes, expiry/refresh/
  logout races and native versus local session effects.
- Test native ownership changes and name reuse: the old cached owner gains no
  continuing authority from the row, the current authorized actor can perform the
  intended operation, unrelated admin roles do not elevate, and roots/accounts/
  homes/keys/workloads remain unchanged.
- Keep conditional registration/mail/external authentication/security workflows
  required when enabled. Missing interfaces are implementation gaps, not waivers.

No native patch, new route, compatibility advertisement or live change is delivered
by this correction. U04/U05/U07/U12/U16 implement their bounded responsibilities;
U17/U18 verify and cut over. U01's closing source/design review above now satisfies
its planning-readiness exit, not any of those implementation or native exits.
