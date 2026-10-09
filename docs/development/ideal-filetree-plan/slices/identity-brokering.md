# Identity brokering

[Slice catalog and evidence scope](README.md). The cards below span current
source owners; they do not create packages, services or an approved intended model.

## I01 Enrollment and owner consent

Start, observe, cancel and complete an explicit selected-provider enrollment with owner-bound metadata and exposure consent.

- **Entrypoints:** POST/GET/DELETE identity enrollments; Controller.start_enrollment/enrollment/cancel_enrollment.
- **Owned data:** Transient enrollment ID/session presentation and completion result; owner-bound connection metadata references.
- **Authority:** Verified native owner; explicit credential-exposure consent; selected provider only.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); Native session authority.
- **Source files:** [internal/web/api/identity.go:78](../../../../internal/web/api/identity.go#L78); `rust/soda-identity/src/control.rs:166 (historical source locator)`; `rust/soda-identity/src/main.rs:245 (historical source locator)`.
- **Tests:** [internal/web/identity_native_test.go:46](../../../../internal/web/identity_native_test.go#L46) — Source-only API fixture assertions: owner/session binding and consent required before broker call; changed actor/generation/origin refused.
- **Unclear boundaries:** Owns enrollment lifecycle, while I02 owns retained encrypted credential bytes and generations. Existing providers are Codex and Muse; current registry is not authority to invent additional providers or universal enrollment semantics.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
- **Validity review:** [I01 record](../reviews/I01.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I02 Encrypted credential custody

Retain native credential bytes encrypted with connection/generation binding under the dedicated broker's private custody.

- **Entrypoints:** Store.open_encrypted/save_connection/credential; GrantCipher.seal/open; broker startup key input.
- **Owned data:** Encrypted connection credential ciphertext; connection credential generation; protected encryption key input; PostgreSQL persistence.
- **Authority:** Dedicated broker process; plaintext excluded from owner/browser metadata and ordinary application APIs.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I06](#i06-completion-revocation-and-reconciliation); PostgreSQL; AES-GCM; Restricted key file.
- **Source files:** `rust/soda-identity/src/store.rs:61 (historical source locator)`; `rust/soda-identity/src/store.rs:224 (historical source locator)`; `rust/soda-identity/src/crypto.rs:25 (historical source locator)`.
- **Tests:** `rust/soda-identity/src/crypto.rs:84 (historical source locator)` — Source-only unit assertions: authenticated round-trip; wrong binding, short ciphertext and tampering refused; `rust/soda-identity/tests/broker.rs:205 (historical source locator)` — Source-only PostgreSQL broker fixture: retained credential round-trip; fixture may return early when unavailable.
- **Unclear boundaries:** Owns ciphertext and generation, not grants or native launch authorization. Current implementation uses PostgreSQL; credentials guide still names SQLite in runtime details, so documentary storage wording needs reconciliation. No encryption or database test executed.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
- **Validity review:** [I02 record](../reviews/I02.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I03 Delegation and connection availability

Authorize explicit subscription sharing to a named member/Project and expose only the connection metadata available to that actor.

- **Entrypoints:** GET available-connections; POST identity grants; Controller.available/create_grant/revoke_grant.
- **Owned data:** Grant ID, named user/Project, revision and revocation; derived availability metadata.
- **Authority:** Connection owner consent plus current requester/recipient Project membership and repository rights.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [P03](projects.md#p03-human-membership-and-accounts); Native Forgejo repository callback; [I06](#i06-completion-revocation-and-reconciliation).
- **Source files:** [internal/web/api/identity_grants.go:47](../../../../internal/web/api/identity_grants.go#L47); `rust/soda-identity/src/control.rs:159 (historical source locator)`; `rust/soda-identity/src/control.rs:279 (historical source locator)`.
- **Tests:** [internal/web/identity_native_test.go:92](../../../../internal/web/identity_native_test.go#L92) — Source-only API fixture: named members, write authority and both confirmations required; `rust/soda-identity/tests/broker.rs:205 (historical source locator)` — Source-only broker fixture: foreign actor sees nothing without grant; shared metadata omits email; revoke removes availability.
- **Unclear boundaries:** Owns delegation state, not custody or execution reservations. Availability is derived, not evidence that execution is currently admissible. I06 performs grant revocation and resulting retirement against these owned grants.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
- **Validity review:** [I03 record](../reviews/I03.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I04 Execution admission and lease fencing

Admit one logical execution against current connection/grant authority and fence replays, conflicting reservations and late registration.

- **Entrypoints:** Runtime acquire/get_execution/close_execution; Controller.acquire/admit_execution/reserve.
- **Owned data:** Execution identity/digest/state/tombstone; lease reservation/deadline and captured connection/grant generations.
- **Authority:** Runtime-only broker API; authenticated native execution consumer; current grant and provider-specific concurrency constraints.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I05](#i05-native-binding-and-private-delivery); [I06](#i06-completion-revocation-and-reconciliation); PostgreSQL.
- **Source files:** `rust/soda-identity/src/control.rs:317 (historical source locator)`; `rust/soda-identity/src/control.rs:439 (historical source locator)`; `rust/soda-identity/src/store.rs:435 (historical source locator)`.
- **Tests:** `rust/soda-identity/tests/broker.rs:239 (historical source locator)` — Source-only PostgreSQL broker fixture: identical acquisition replays same lease; `rust/soda-identity/tests/broker.rs:359 (historical source locator)` — Source-only broker fixture: closed execution denies late register and reacquire; fixture may skip without database; `rust/soda-identity/tests/broker.rs:390 (historical source locator)` — Source-only PostgreSQL fixture: unbound reconciliation, repeated closure and terminal execution state; the registration error does not distinguish a missing lease from a fence.
- **Unclear boundaries:** Owns execution/reservation records. I05 supplies the attested binding; I06 closes the same records. Codex serialization and Muse concurrency are existing policies, not a universal-provider rule or new generic scheduler.
- **Evidence status:** Existing source coverage reused; post-reconcile closure regression mapped at 0d8d3b8e. Assertions inspected only, no tests or native execution performed.
- **Validity review:** [I04 record](../reviews/I04.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I05 Native binding and private delivery

Validate the exact native account/container/process/unit binding before credentials cross the private runtime interface.

- **Entrypoints:** Runtime register; Controller.register; host validate callbacks; Codex/Muse native adapter validation.
- **Owned data:** Attested lease binding fields; private delivery payload and protected native credential destination.
- **Authority:** Dedicated broker runtime plane; privileged host verifies native incarnation/invocation; browser input cannot attest authority.
- **Dependencies:** [I04](#i04-execution-admission-and-lease-fencing); [I02](#i02-encrypted-credential-custody); [I09](#i09-provider-execution-integration); [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P06](projects.md#p06-factory-role-accounts); systemd; Podman.
- **Source files:** `rust/soda-identity/src/control.rs:502 (historical source locator)`; `rust/soda-identity/src/runtime.rs (historical source locator)`; `rust/soda-host/src/tcodex.rs (historical source locator)`; [docs/reference/credentials.md:112 (historical line locator)](../../../reference/credentials.md).
- **Tests:** `rust/soda-host/tests/tcodex_ops_oracle.rs:206 (historical source locator)` — Oracle source assertions with fake executor: exact container, role UID/GID and systemd invocation checks precede delivery; `rust/soda-host/tests/muse_serve_oracle.rs:426 (historical source locator)` — Oracle source assertions: mismatched invocation returns stale and stops further native calls; `rust/soda-identity/tests/broker.rs:422 (historical source locator)` — Source-only late registration returns an error after unbound reconciliation and repeated closure; no native attestation exercised.
- **Unclear boundaries:** Owns binding attestation/delivery transition, not lease reservation or credential storage. Shared lease table is one state machine, not duplicate ownership. Native oracle tests prove authored call contracts only; installed attestation was not exercised.
- **Evidence status:** Existing source coverage reused; added registration assertion mapped at 0d8d3b8e. Assertions inspected only, no tests or native execution performed.
- **Validity review:** [I05 record](../reviews/I05.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I06 Completion, revocation and reconciliation

Retire leases and logical executions through verified stop/finish/return, explicit revocation and deadline reconciliation without prematurely releasing custody.

- **Entrypoints:** Controller.return_lease/end_lease/revoke/revoke_grant/reconcile/sweep; host stop/finish callbacks.
- **Owned data:** Lease terminal/returned state; execution closure; revocation transitions; custody-return result and updated generation.
- **Authority:** Owner may revoke own connection/grant; native runtime confirms retirement; unknown termination cannot imply successful return.
- **Dependencies:** [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I04](#i04-execution-admission-and-lease-fencing); [I05](#i05-native-binding-and-private-delivery); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); [I09](#i09-provider-execution-integration).
- **Source files:** `rust/soda-identity/src/control.rs:610 (historical source locator)`; `rust/soda-identity/src/control.rs:705 (historical source locator)`; `rust/soda-host/src/pfactory.rs:2537 (historical source locator)`.
- **Tests:** `rust/soda-identity/tests/broker.rs:428 (historical source locator)` — Source-only PostgreSQL broker fixture: revoke changes connection state and retires live leases; `rust/soda-identity/tests/broker.rs:306 (historical source locator)` — Source-only broker fixture: Muse return forgets lease without rotating connection generation; `rust/soda-identity/tests/broker.rs:390 (historical source locator)` — Source-only real Controller/Store with stub provider/runtime and optional PostgreSQL: reconcile unbound lease, close twice, terminal state and registration error; `rust/soda-host/src/pfactory.rs:4959 (historical source locator)` — Scripted transient-close retry in native stop adapter.
- **Unclear boundaries:** Owns closure transitions over I04's lease/execution records, not a second ledger. Mutable Codex return and immutable Muse retirement already diverge inside core code; review those concrete policies before extracting abstractions.
- **Evidence status:** Closure/retry delta mapped at 0d8d3b8e. Broker fixture assertions and host scripted tests inspected only; no tests, real native retirement or installed behavior exercised.
- **Validity review:** [I06 record](../reviews/I06.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I07 Codex adapter

Adapt the pinned native Codex enrollment protocol and mutable subscription credentials to the broker's existing provider interface.

- **Entrypoints:** Codex Provider.new/start; Session.snapshot/finish/close; native device enrollment subprocess.
- **Owned data:** Transient private Codex enrollment root/process; native auth bytes; provider-specific public enrollment presentation.
- **Authority:** Explicit selected Codex owner; configured pinned binary/version/hash; inherited credential environment filtered.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I06](#i06-completion-revocation-and-reconciliation); Native Codex executable.
- **Source files:** `rust/identity-providers/src/codex.rs:50 (historical source locator)`; `rust/identity-providers/src/codex.rs:304 (historical source locator)`; [docs/reference/credentials.md:109 (historical line locator)](../../../reference/credentials.md).
- **Tests:** `rust/identity-providers/src/codex.rs:656 (historical source locator)` — Source-only native-protocol fixture assertions: credential retention after process stop; close removes private enrollment root; `rust/identity-providers/src/codex.rs:682 (historical source locator)` — Source-only fixture assertions: unfinished enrollment canceled and root removed.
- **Unclear boundaries:** Adapter owns Codex protocol/auth representation; core owns grants, custody and lease records. Execution-side integration is I09. Browser launch's current Codex scope does not redefine the entire broker as Codex-only.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
- **Validity review:** [I07 record](../reviews/I07.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I08 Muse adapter

Adapt pinned Muse subscription device enrollment and native auth validation without exposing credential or diagnostic contents in presentation.

- **Entrypoints:** Muse Provider.new/start; Session.finish/close; credential_valid; native login subprocess.
- **Owned data:** Transient Muse enrollment root/process; native subscription auth representation and bounded public device presentation.
- **Authority:** Explicit selected Muse owner; pinned native binary/version/hash; subscription routing rather than fallback billing.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I06](#i06-completion-revocation-and-reconciliation); Native Muse executable.
- **Source files:** `rust/identity-providers/src/muse.rs:44 (historical source locator)`; `rust/identity-providers/src/muse.rs:286 (historical source locator)`; [docs/reference/credentials.md:134 (historical line locator)](../../../reference/credentials.md).
- **Tests:** `rust/identity-providers/src/muse.rs:514 (historical source locator)` — Source-only native fixture assertions: file credential backend, environment filtering, completed enrollment and no credentials/diagnostics in public presentation; `rust/identity-providers/src/muse.rs:494 (historical source locator)` — Source-only pinned-version fixture assertions: exact native version required.
- **Unclear boundaries:** Adapter owns Muse enrollment/auth semantics, while concurrency and immutable lease closure belong to I04/I06. This is an existing adapter child, not evidence that Codex assumptions can be copied to every provider.
- **Evidence status:** Representative source catalog at committed 26d420f2; assertions inspected only, no tests or native execution performed.
- **Validity review:** [I08 record](../reviews/I08.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I09 Provider execution integration

Connect human terminal, factory and explicitly registered nested native execution consumers to broker admission and attested private delivery.

- **Entrypoints:** Codex identity launch; host factory Codex/Muse adapters; soda-muse Project launch; soda-identity-compose register.
- **Owned data:** Native execution unit/process metadata and private runtime files; consumer-side execution binding; no separate connection/grant ledger.
- **Authority:** Native actor/account or fixed factory role; exact invocation; nested registration explicitly opted in.
- **Dependencies:** [I04](#i04-execution-admission-and-lease-fencing); [I05](#i05-native-binding-and-private-delivery); [I06](#i06-completion-revocation-and-reconciliation); [I07](#i07-codex-adapter); [I08](#i08-muse-adapter); [P02](projects.md#p02-profile-and-runtime-readiness); [P03](projects.md#p03-human-membership-and-accounts); [P06](projects.md#p06-factory-role-accounts); [P07](projects.md#p07-checkout-allocation-and-preparation); [S04](spaces-and-terminals.md#s04-human-terminal-lifecycle); systemd; Podman.
- **Source files:** [internal/web/api/identity_launch.go:13](../../../../internal/web/api/identity_launch.go#L13); `rust/soda-host/src/muse.rs:1523 (historical source locator)`; `rust/soda-host/src/tmuse.rs:275 (historical source locator)`; `rust/soda-identity-compose/src/main.rs:27 (historical source locator)`; `rust/soda-host/src/pfactory.rs:2043 (historical source locator)`.
- **Tests:** `rust/soda-host/tests/muse_serve_oracle.rs:416 (historical source locator)` — Oracle source assertions with fake executor/hooks: delivery echo after valid native invocation; stale invocation refused; `rust/soda-host/tests/tcodex_ops_oracle.rs:206 (historical source locator)` — Oracle source assertions: factory-native binding checks use exact role/container/unit; `rust/soda-host/src/pfactory.rs:4457 (historical source locator)` — Unit regression with fake broker/terminal: Muse harness sends one acquire request with provider_id=muse and the original execution ID. The scripted fixture still returns a Codex lease; completed phase does not establish real Muse authorization, reservation, delivery or retirement.
- **Unclear boundaries:** Dispatch supplies the selected harness and connection; host execution owns provider-matched lease acquisition and native reservation/delivery. Broker core remains the sole lease/custody owner. Current harness names also identify broker providers; that name coupling needs intended-model review for the broader provider catalog. Reservation refusal now logs its cause before the existing abandonment path; native Muse execution remains unqualified here.
- **Evidence status:** Request-provider regression evidence reused from 26d420f2; its host source anchors reconciled at 0d8d3b8e. No tests or native execution performed.
- **Validity review:** [I09 record](../reviews/I09.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.

## I10 Identity audit history

Append credential-free identity events atomically with broker state changes and preserve their immutable owner/connection history.

- **Entrypoints:** Broker/store mutations append audit events transactionally; Store.events bounded history reader has no current Rust caller or exposed route found.
- **Owned data:** identity_events append-only rows with actor, owner, connection, generation, action and time; no retained credential contents.
- **Authority:** Broker mutation authority supplies event identity; database triggers refuse UPDATE/DELETE; history read intent is owner/connection scoped in storage code.
- **Dependencies:** [I01](#i01-enrollment-and-owner-consent); [I02](#i02-encrypted-credential-custody); [I03](#i03-delegation-and-connection-availability); [I04](#i04-execution-admission-and-lease-fencing); [I06](#i06-completion-revocation-and-reconciliation); [H02](shared-supporting-slices.md#h02-storage-mechanics).
- **Source files:** `rust/soda-identity/src/schema.rs:58 (historical source locator)`; `rust/soda-identity/src/store.rs:556 (historical source locator)`; `rust/soda-identity/src/store.rs:640 (historical source locator)`; `rust/soda-identity/src/wire.rs:531 (historical source locator)`.
- **Tests:** [internal/store/identity_test.go:79](../../../../internal/store/identity_test.go#L79) — Go PostgreSQL shared-schema assertions: ordered audit actions, immutable trigger and credential absence; not direct Rust runtime proof; [internal/store/identity_test.go:101](../../../../internal/store/identity_test.go#L101) — Go PostgreSQL fixture: failed audit INSERT rolls back grant mutation; tests not run.
- **Unclear boundaries:** Audit history has its own state and append integrity; it does not acquire credential custody, grant authorization or retention policy from neighboring slices. The unused Rust history reader is an unresolved applicability seam; no public audit feature is inferred. Shared-schema/predecessor Go tests require separate port/caller classification.
- **Evidence status:** Existing append state mapped; read exposure, intended audit requirements and direct Rust proof remain unresolved.
- **Validity review:** [I10 record](../reviews/I10.md) — actual reviewed scope, findings, challenges, target allocations and remaining work.
