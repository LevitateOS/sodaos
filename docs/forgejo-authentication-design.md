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

Exact routes/wire revisions and storage reuse require native review before coding.
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

Choose concrete transaction TTL/attempt budgets from the native mechanism and
review them with measured tests, including distributed/concurrent requests. No
numeric limit is claimed implemented or sufficient by this draft. Pre-auth
compatibility must be checkable without first obtaining the ordinary grant that
this protocol is meant to establish; metadata still does not prove compatibility.

## Decisions that must return to the product/maintainer

1. **Production browser origin and enrolled-key preservation.** Recommend keeping
   the existing RP-ID where the final Soda host is eligible for it, and adding only
   the exact Soda origin through a reviewed native configuration/interface change.
   Origins differing only by port can share an RP-ID, but the native origin check
   still needs explicit support. Unrelated domains cannot simply reuse enrolled
   credentials by changing the UI. No live RP-ID/credential inventory was inspected
   or modified in this batch; do not assume the test `localhost` arrangement is the
   production contract. Confirm the intended production Soda origin and the RP-ID
   that must remain valid before implementing this native configuration change.
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

Repository transfer, organization ownership, account deletion/disablement and
revocation still require explicit decisions about **Soda's legitimate records and
new browser access**, not a generalized Linux offboarding subsystem. Recommend
preserving existing roots/accounts/data and refusing ambiguous new privileged
operations while showing an honest association error. Do not silently select that
as final transfer/deletion policy, change existing project administrator authority,
or block ordinary unrelated native operations to hide the decision. U05/U07/U12/
U16 retain these decisions under the leading plan.

No native authentication patch, new auth route, compatibility advertisement, live
configuration mutation or security-key reset is implemented by this design. Keep
working OAuth/login and direct ingress unchanged until an approved replacement
passes isolated native/browser proof. U04/U05/U16 and U17 remain incomplete.
