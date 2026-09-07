# Native-backed Soda authentication — H05 design review

**Draft produced during the implementation batch after `2ff9e44`.** This is a
concrete sequence/ownership/threat-model proposal, not an implemented protocol or
permission to change the installed login, RP-ID, origins or enrolled credentials.
U04 owns authentication/session integration, U05 self-account security and U16
administrator security. AU01–AU22/AD15/AD18/AD22 in the
[single register](forgejo-api-coverage.md) remain the workflow inventory.

## Native constraints and selected implementation direction

Reviewed v16.0.3 source retains the constraints found by H01:

- `routers/api/shared/middleware.go::verifyAuthWithOptions` rejects inactive,
  prohibited, forced-password-change and required-but-missing-MFA actors. Giving
  them another ordinary OAuth/PAT scope cannot finish enrollment.
- `routers/web/auth/{auth,password,2fa,webauthn,oauth}.go` owns the native login,
  password change, challenge, consent and successful-session behavior. Extract
  the needed decisions/commands inside Forgejo, retaining existing web callers;
  do not serialize whole web contexts or forward native session cookies.
- `services/auth/method/basic.go` rejects WebAuthn users. Basic/password-to-PAT or
  admin token minting is not the protocol.
- `modules/auth/webauthn/webauthn.go::Init` uses `setting.Domain` for RP-ID and the
  origin of `setting.AppURL` for allowed origins. Existing assertion/enrollment
  handlers depend on native session challenge state. Inspected assertions are a
  second factor, not evidence of discoverable/passwordless passkey support.

**Direction:** a narrow Forgejo-owned authentication transaction is needed before
an ordinary user grant exists. Native password/MFA/activation/consent verification
must determine the next permitted step. Soda presents those steps and protects its
browser requests; it cannot choose a subject, elevate privileges or mark a native
security requirement complete. This proposal does not assume the transaction can
be exposed safely by relabeling an existing web session as a bearer API.

## Proposed sequence and data ownership

The revision-1 candidate below supplies the initial route/state/binding choices;
implementation and native security review are still required before exposure.
The following operations are distinct, not a generic `call native method` endpoint.

| Operation | Inputs from Soda | Native result / invariant |
| --- | --- | --- |
| Begin | Configured confidential OAuth client identity, registered Soda return/origin context, fresh browser-bound nonce/PKCE challenge and requested scopes | Opaque short-lived authentication transaction; only native-configured methods/next-step data, no arbitrary user's security inventory |
| Verify credential | Transaction binding and transient native credential proof | Native account policy and password verifier choose password-change, MFA, activation, consent or failure; no ordinary grant before requirements are met |
| Change required password | Transaction and new-password/confirmation proof with the native required old-password/authentication context | Native validation/update/session invalidation; caller cannot clear `MustChangePassword` via an admin API |
| MFA assertion / recovery | Transaction and the selected issued native challenge response | Native TOTP/WebAuthn/scratch verification, counters and one-use consumption; recovery does not bypass remaining required policy |
| Required enrollment | Transaction and native enrollment/confirmation proof | Native seed/credential lifecycle and required-MFA policy; no reusable credential material returned after setup |
| Consent | Transaction and explicit approve/deny of native client/scopes | Native grant decision and bounded one-use authorization code bound to client/PKCE/registered return; no browser-selected subject or role |
| Exchange | Existing confidential-client code/PKCE exchange | Native token endpoint remains authoritative; introspect actual subject/audience/scopes before creating the normal Soda session |
| Refresh | Existing encrypted session-bound refresh credential | Native rotation/expiry; serialize in Soda, persist only into an existing session and never replay an uncertain refresh or pending write |
| Cancel / expire | Transaction binding, or native expiry | Invalidate pre-authentication capability; do not create a normal session or roll back independent native password/security effects |
| Sign out / revoke | Explicit local-session, native-grant or other native-session action and native-required reauth | Distinct scopes/results; no promise of terminating Linux/SSH sessions or deleting environments |

The transaction handle is a credential. Keep it out of URLs, browser-readable
state, logs, traces and error bodies. Soda may retain only the browser transaction
binding and protected native handle needed for this adapter; no passwords, MFA
proofs, TOTP seeds, copied roles or native policy transcripts in its database.
Native transaction state holds the verified subject, completed/remaining native
requirements, challenge binding/expiry/attempts and one-use completion. Prefer
reuse of existing native session/challenge primitives where their properties fit;
do not invent a second credential verifier or generalized job system.

A confidential client credential authenticates the Soda adapter, **not the user**.
It must not mint a subject/grant without native proof and consent. Bind client,
transaction, challenge, intended operation and browser nonce; reject cross-client,
cross-browser, expired, consumed, out-of-order and changed-account requests.
Native policy must be rechecked before completion, not copied at transaction start.

A successful password/MFA change followed by a lost response is not automatically
undone or retried. Define an authenticated transaction-status/next-step read that
reveals only this transaction's state; no account-existence lookup by username.
Native credential updates remain native even if the browser cancels afterward.

## Threat model and required tests

| Threat / failure | Required control and proof |
| --- | --- |
| Login-CSRF / session fixation | Same-origin pre-auth requests, fresh Soda binding, CSRF checks appropriate before login, native nonce/PKCE binding, final session rotation; attacker cannot log a victim into the attacker's account |
| Replay / cross-browser transaction substitution | Native one-use challenges/completion and bounded lifetime; atomic state transitions; concurrent identical proof cannot issue duplicate authority |
| Password/account enumeration and brute force | Reuse native policy/rate limits/lockouts and enumeration-resistant errors/timing; binding attempts to a new transaction must not reset account/IP limits |
| Native role or MFA bypass | No supplied UID/admin predicate, operator token, reverse-proxy impersonation, borrowed cookie or ordinary grant before required steps; test forced password/MFA, inactive and prohibited actors |
| Stolen/overbroad client secret | Secret-file channel and narrow configured client/return/scope binding; compromised client still cannot select a subject or bypass proof; document the adapter's existing password-relay trust |
| WebAuthn origin/counter confusion | Server-issued native challenge, exact RP-ID/origin/user-verification/native counter validation; wrong origin/credential/transaction fails without deleting enrollment |
| Secret disclosure | No credential/transaction query strings, redirects, browser persistence, logs/trace/body dumps; one-time enrollment/recovery displays use no-store and disappear on completion/account change |
| Refresh/logout races | Existing protected per-session grants; logout wins in-flight refresh and browser responses; another user's session cannot inherit a grant or private draft |
| Policy changes during flow | Recheck native account/required factors/client/grant state at completion and sensitive actions; distinguish revocation from missing scope or incompatible transport |
| Cancellation/native timeout | Bound body/work/lifetime, cancel native work, retain real native outcome; no automatic replay or fabricated success from a local session row |
| Recovery/activation mail | Native signed tokens and configured expiry/policy, but Soda presentation for owned actions; no new Soda mail token authority or hidden native-page redirect |
| Sensitive self/admin operation confusion | Native self reauth and native site-admin MFA reset are different actions; resetting another user's factors never borrows self-security credentials or grants host/project privilege |

Concrete candidate TTL/attempt ceilings are specified below. They are proposed
security limits, not implemented controls or measured capacity. Targeted source
inspection did not establish a general native password-attempt limiter in the
selected login handlers; mail resend limits are not such a limiter. Therefore
“reuse native rate limits” is not sufficient: H05 must supply and test the missing
native-owned admission/attempt controls without weakening any existing policy. Pre-auth
compatibility must be checkable without first obtaining the ordinary grant that
this protocol is meant to establish; metadata still does not prove compatibility.

## U01 revision-1 protocol disposition

**Candidate for native implementation/security review, not an approved live auth
boundary.** This first contract covers sign-in and its required security gates;
registration, mail actions, authenticated self-security and admin commands remain
explicit feature-owned contracts, not arbitrary operations on this transaction.

Native prefix: `/api/forgejo/v1/soda/auth`. All requests require the configured
confidential OAuth client's authentication through a restricted secret channel
and validated registered context. Client authentication is not user HTTP Basic:
no user Basic/PAT creation, admin token or subject selection is permitted. The
native transaction handle is a separate random 256-bit capability carried only
in a redacted server-to-server header, never a URL or browser-readable value.

| Endpoint | Exact operation-specific input | Result |
| --- | --- | --- |
| `POST /transactions` | `client_id`, exact registered `redirect_uri`, canonical requested `scope`, random browser `binding` digest, S256 `code_challenge`, `nonce`, native configured `method_id` | Native opaque handle to Soda only; public state envelope below. Reject unregistered method/return or invalid native scopes; do not accept a UID or override the resolved user's native login-source binding. |
| `GET /transaction` | Handle + client/binding; no target query | Current state envelope for this transaction only; cannot reveal another account or replay a completed authorization code |
| `POST /credential` | Handle + client/binding, `expected_version`, native `username`, transient `password` | Native verifier/account policy chooses the next step; use the native sign-in form validation, including its 254/255 size limits, rather than changing Unicode/password rules |
| `POST /required-password` | Same binding/version; native `password`, `retype` fields and native required authentication context | Native password command with policy/session effects; cannot write a “password requirement passed” flag |
| `POST /totp` | Same binding/version; issued `challenge_id`, `code` | Native second-factor validation and one-use state change |
| `POST /recovery` | Same binding/version; `challenge_id`, `recovery_code` | Native scratch-code consumption; not an alternate bypass of account policy |
| `POST /webauthn` | Same binding/version; `challenge_id`, standard native assertion JSON | Native challenge/RP-ID/origin/credential/user-verification/counter checks |
| `POST /enrollment/begin` | Same binding/version; factor from the native allowed factor set | Native challenge/options or one-time enrollment display; no ordinary user grant yet |
| `POST /enrollment/finish` | Same binding/version; `challenge_id`, factor-specific native confirmation/attestation JSON | Native credential persistence and required-enrollment validation |
| `POST /consent` | Same binding/version; `approve` boolean; no client-selected subject or extra scopes | Native grant decision. On approval, one-use native authorization code to Soda only, bound to existing code/PKCE exchange; denial terminates the transaction |
| `DELETE /transaction` | Handle + client/binding | Cancel future progression; never undo a password, scratch-code or security change already committed |

Public state envelope: `{revision: 1, version: STRING, expires_at: RFC3339,
state: ENUM, challenge: OBJECT_OR_NULL, error: CODE_OR_NULL}`. States are
`credential`, `required_password`, `factor`, `enrollment`, `activation_required`,
`external_redirect`, `consent`, `complete`, `denied`, `expired`, `cancelled` and
`outcome_unknown`. Native policy chooses transitions. Challenge payloads are typed
by the expected native step: native factor options/WebAuthn public options,
`{client_id, name, scope, approval_required}` for consent, or native activation status.
If native policy permits existing-grant/client auto-approval, it sets
`approval_required: false`; Soda may finish that consent step without inventing a
new prompt. Only native policy chooses this, never a remembered Soda role/decision. No session cookie, native
handle, password, reusable code or copied role inventory appears in this browser
projection. One-time enrollment material is returned only by the begin command,
not by arbitrary later status reads. An external redirect needs the separately
approved IdP-origin decision and native state validation; no Forgejo-page fallback.

Native state binds handle hash, configured client, browser binding, registered
return, requested scopes, PKCE/nonce, native method/verified subject, progress,
challenge and expiry. Store it under Forgejo authority with atomic version/consume
checks; do not use ordinary web-cookie identity as its privilege. A small native
transaction record is justified by one-use authentication state, not a job engine.
Reuse native credential/challenge/grant services; extract their operations where
currently embedded in web handlers. Soda retains only its own pre-auth browser
binding and protected native handle, using purpose-separated authenticated
protection, then discards them on completion/cancellation/expiry.

**Atomicity:** claim `expected_version` before verification; concurrent/stale
submissions fail with 409 and do not verify twice. A state read never repeats a
mutation or returns a previously delivered authorization code. Native password/
recovery/credential changes must preserve their actual committed outcome. Where
an existing helper cannot participate in a combined native transaction, leave an
explicit unknown/consumed outcome on uncertainty and require fresh native proof;
do not replay it or fabricate “not applied.” That helper's crash behavior needs
focused H05 tests before exposing the operation.

**Selected initial bounds:** ten-minute absolute transaction lifetime, with each
challenge expiring no later than its existing native lifetime or that deadline;
no sliding extension on failure/status reads. At most five failed proofs per
transaction. Add native-owned aggregate limits that survive new transaction IDs:
10 failed proofs per resolved native account in 15 minutes, 60 transaction starts
and 60 proof submissions per configured client per minute (separate counters),
two simultaneous expensive verifications per
Forgejo instance, and 1,000 active transactions globally. Unresolved login names
use native normalization with protected keys; aliases must converge on the native
user after lookup. Atomic counters/expiry are native-owned, not browser or Soda
role records. Unknown-account errors must not distinguish existence. Prove these
controls under concurrent/restarted requests; do not claim an unverified cache
check is an atomic limiter.

Request JSON is strict and at most 64 KiB; native field validation may be stricter.
Responses are at most 256 KiB and no-store. Native OTP/seed/key entropy/counters,
password algorithms, code/grant expiry and account policies remain native choices,
not Soda replacements. An aborted request must not release its expensive-work
slot before the verifier actually ends. The Go/native password verifier may not
be cancellable mid-hash; a request deadline alone is not a CPU/memory bound. H05
must test the configured native algorithm's cost and preserve its native limits.

Errors: 400 malformed/unsupported proof; 401 invalid client/handle/proof with no
account enumeration; 403 native policy refusal only at a stage allowed to disclose
it; 409 stale/out-of-order/consumed submission; 410 expired transaction; 413 body
limit; 429 admission/attempt limit; 503 unavailable/unknown outcome. Public messages
are fixed, no underlying verifier/HTTP/body diagnostics. Code exchange and refresh
continue to use the existing native OAuth endpoints and their error semantics.

**Review disposition:** sufficient concrete shape for focused native extraction/
security review, but not permission to claim feasibility of every native method.
IdP product policy and named review remain human decisions; factor-policy parity,
atomic helper behavior and aggregate attempt enforcement need native tests. Do not
start a second protocol proposal or silently fall back to borrowed cookies if one
of these implementation constraints fails.

## Decisions that must return to the product/maintainer

1. **Origin/RP-ID policy is now a concrete configuration contract, not a demand
   for one global production hostname.** SodaOS is installed with operator-supplied
   browser origins. Preserve an existing installation's native RP-ID independently
   of browser and Git-advertisement hostnames; configure only the exact eligible
   Soda origin for assertions, with temporary legacy origin retention only during
   separately tested cutover. New installations choose an eligible RP-ID from their
   declared origin; upgrades must not silently rederive/change it. Validate HTTPS,
   exact scheme/host/port and RP-ID host eligibility before enabling the new flow.
   An unrelated new domain is refused until explicit credential/origin migration
   is approved; never reset enrollment automatically. Native configuration owns
   this setting and validation, not a browser parameter. The actual hostname/RP-ID
   values are installation/rollout inputs, not a reason to repeat U01 R&D. No live
   RP-ID/credential inventory was inspected or changed; U17/U18 must verify those
   exact inputs and existing-key assertions before any ingress/configuration change.
2. **External IdP browser interaction.** Forgejo-owned pages must stay in Soda;
   an external IdP may require its own website. Confirm whether that external
   provider's own login/consent is allowed, with a registered return to Soda. If
   all external browser interaction is forbidden, that conflicts with preserving
   those enabled native methods; do not silently disable them or claim feasibility.
3. **Logout/revocation semantics.** Recommend local logout clears the current Soda
   session immediately; separately labeled native grant/session revocation follows
   native authority and affects other sessions as the provider defines. Confirm
   which explicit product action should revoke the shared application grant.
   Already authenticated Linux shells/workloads are not part of that operation.
4. **Named security/update review.** Assign a human reviewer/maintainer for this
   native pre-authentication boundary and the selected source-release cadence.
   Repository milestone ownership is not a person's commitment or a security review.

## Linked identities and retained project access

Source already stores a member's project-local login independently of their current
Forgejo username (`internal/store/members.go`, `internal/web/environments_api.go`).
Preserve that association; do not rename/recreate Linux accounts on a native name
change or interpret a reused username as the former identity. Native resource
lookup/ownership checks must use stable IDs and fail on ambiguous associations.

U01 technical dispositions (implementation belongs to the existing U owners):

| Native change | Soda consequence |
| --- | --- |
| User/repository rename with the same native ID | Resolve by stable native identity, verify the returned ID, then refresh only cached display/route names. Keep stored project-local login/home/account. Never follow a reused name onto a different native ID. |
| Native account disabled/deleted or grant revoked | Deny new protected browser operations when native validation fails; invalidate affected Soda sessions/grants on confirmed account/grant invalidation, not on network uncertainty. Preserve existing project roots and project-local development SSH keys/accounts/workloads. Forgejo still owns effects on its personal Git keys. No claim that native account deletion revokes Linux access. |
| Repository deleted, hidden from the actor or lookup unavailable | Keep the association by ID and show unavailable/denied as appropriate. A 404 is not proof of deletion and cannot authorize row/root cleanup or reattachment to a reused name. |
| Native operation succeeded but Soda cache/association persistence failed | Report native success plus the bounded Soda failure; no automatic retry, inverse native mutation or invented rollback. Subsequent trusted ID reads may repair display data, not grant new administration. |
| Human-to-human or organization repository transfer | **Human decision still required:** specify project-admin succession independently of Linux account preservation, including which human administers an organization-linked environment. Do not derive host authority from site/org administration. |

This is not already dynamic succession: `apiCreateEnvironment` stores the creating
human as `Project.OwnerID`; `apiEnvironmentMembers` checks that stored ID (or the
configured Soda operator), rather than the repository's current owner. Existing
organization-owned environment creation is explicitly rejected. Thus a transfer
policy cannot be called implemented by merely updating a cached repository name.
U05/U07/U12/U16 must implement the chosen bounded authority/association change and
native denial/name-reuse/session tests. The decision must not become a generalized
Linux offboarding/reconciliation service or a waiver of native transfer/delete UI.

For Soda-only protected routes, provider-independent cached membership is not proof
that a native account/grant remains valid. The U04/U07 implementation must perform
current native validation before new privileged access/key/terminal/provisioning
operations; any finite read-session freshness must be explicit and tested. Native
unavailability fails those new privileged operations closed without deleting data.
Already established native SSH/workload sessions remain outside browser revocation;
U07 separately owns the browser terminal's authenticated connection lifetime.

No native authentication patch, new auth route, compatibility advertisement, live
configuration mutation or security-key reset is implemented by this design. Keep
working OAuth/login and direct ingress unchanged until an approved replacement
passes isolated native/browser proof. U04/U05/U16 and U17 remain incomplete.
